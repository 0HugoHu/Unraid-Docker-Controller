package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nas-controller/internal/docker"
	"nas-controller/internal/models"
)

type BuildService struct {
	dockerClient *docker.Client
	dataDir      string
	logsDir      string
	building     bool
	buildMu      sync.Mutex
	buildCancel  context.CancelFunc
}

type BuildProgress struct {
	AppID    string `json:"appId"`
	Message  string `json:"message"`
	Error    string `json:"error"`
	Complete bool   `json:"complete"`
	Success  bool   `json:"success"`
}

func NewBuildService(dockerClient *docker.Client, dataDir string) *BuildService {
	logsDir := filepath.Join(dataDir, "logs")
	os.MkdirAll(logsDir, 0755)

	return &BuildService{
		dockerClient: dockerClient,
		dataDir:      dataDir,
		logsDir:      logsDir,
	}
}

func (s *BuildService) IsBuilding() bool {
	s.buildMu.Lock()
	defer s.buildMu.Unlock()
	return s.building
}

func (s *BuildService) BuildApp(ctx context.Context, app *models.App, repoPath string, progressChan chan<- BuildProgress) error {
	s.buildMu.Lock()
	if s.building {
		s.buildMu.Unlock()
		return fmt.Errorf("another build is in progress")
	}
	s.building = true

	// Create cancelable context
	buildCtx, cancel := context.WithCancel(ctx)
	s.buildCancel = cancel
	s.buildMu.Unlock()

	defer func() {
		s.buildMu.Lock()
		s.building = false
		s.buildCancel = nil
		s.buildMu.Unlock()
	}()

	// Create log file
	logPath := filepath.Join(s.logsDir, fmt.Sprintf("build-%s.log", app.ID))
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %v", err)
	}
	defer logFile.Close()

	// Create multi-writer for both log file and progress channel
	writer := &buildLogWriter{
		appID:        app.ID,
		logFile:      logFile,
		progressChan: progressChan,
	}

	startTime := time.Now()

	sendProgress := func(msg string) {
		if progressChan != nil {
			progressChan <- BuildProgress{
				AppID:   app.ID,
				Message: msg,
			}
		}
	}

	sendProgress(fmt.Sprintf("Starting build for %s\n", app.Name))
	sendProgress(fmt.Sprintf("Context: %s\n", repoPath))
	sendProgress(fmt.Sprintf("Dockerfile: %s\n", app.DockerfilePath))
	sendProgress(fmt.Sprintf("Image: %s\n\n", app.ImageName))

	// Build the image
	err = s.dockerClient.BuildImage(
		buildCtx,
		repoPath,
		app.DockerfilePath,
		app.ImageName,
		app.BuildArgs,
		writer,
	)

	duration := time.Since(startTime)

	if err != nil {
		errMsg := fmt.Sprintf("\n\nBuild failed: %v\n", err)
		writer.Write([]byte(errMsg))

		if progressChan != nil {
			progressChan <- BuildProgress{
				AppID:    app.ID,
				Error:    err.Error(),
				Complete: true,
				Success:  false,
			}
		}
		return err
	}

	successMsg := fmt.Sprintf("\n\nBuild completed successfully in %s\n", duration.Round(time.Second))
	writer.Write([]byte(successMsg))

	if progressChan != nil {
		progressChan <- BuildProgress{
			AppID:    app.ID,
			Message:  successMsg,
			Complete: true,
			Success:  true,
		}
	}

	return nil
}

func (s *BuildService) CancelBuild() {
	s.buildMu.Lock()
	defer s.buildMu.Unlock()

	if s.buildCancel != nil {
		s.buildCancel()
	}
}

func (s *BuildService) GetBuildLog(appID string) (string, error) {
	logPath := filepath.Join(s.logsDir, fmt.Sprintf("build-%s.log", appID))
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func (s *BuildService) ClearBuildLog(appID string) error {
	logPath := filepath.Join(s.logsDir, fmt.Sprintf("build-%s.log", appID))
	return os.Remove(logPath)
}

func (s *BuildService) GetLogsSize() (int64, error) {
	var size int64
	err := filepath.Walk(s.logsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func (s *BuildService) ClearAllLogs() error {
	entries, err := os.ReadDir(s.logsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		os.Remove(filepath.Join(s.logsDir, entry.Name()))
	}
	return nil
}

type buildLogWriter struct {
	appID        string
	logFile      io.Writer
	progressChan chan<- BuildProgress
}

func (w *buildLogWriter) Write(p []byte) (n int, err error) {
	n, err = w.logFile.Write(p)
	if w.progressChan != nil && len(p) > 0 {
		w.progressChan <- BuildProgress{
			AppID:   w.appID,
			Message: string(p),
		}
	}
	return n, err
}
