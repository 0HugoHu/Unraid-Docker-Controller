package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"nas-controller/internal/database"
	"nas-controller/internal/docker"
	"nas-controller/internal/models"
)

type AppManager struct {
	db            *database.DB
	dockerClient  *docker.Client
	gitService    *GitService
	buildService  *BuildService
	portAllocator *PortAllocator
	dataDir       string
}

func NewAppManager(
	db *database.DB,
	dockerClient *docker.Client,
	gitService *GitService,
	buildService *BuildService,
	portAllocator *PortAllocator,
	dataDir string,
) *AppManager {
	return &AppManager{
		db:            db,
		dockerClient:  dockerClient,
		gitService:    gitService,
		buildService:  buildService,
		portAllocator: portAllocator,
		dataDir:       dataDir,
	}
}

func (m *AppManager) CloneAndValidate(repoURL string, branch string) (*models.CloneResult, error) {
	return m.gitService.CloneRepo(repoURL, branch)
}

func (m *AppManager) CreateApp(repoURL string, branch string, config *models.ConfigureAppRequest) (*models.App, error) {
	// Get clone result info
	cloneResult, err := m.gitService.CloneRepo(repoURL, branch)
	if err != nil {
		// Try to use existing repo if already cloned
		slug := m.gitService.extractSlug(repoURL)
		if slug == "" {
			return nil, err
		}
		repoPath := m.gitService.GetRepoPath(slug)
		if _, statErr := os.Stat(repoPath); os.IsNotExist(statErr) {
			return nil, err
		}
		// Repo exists, continue
		cloneResult = &models.CloneResult{
			Slug:           slug,
			Name:           config.Name,
			DockerfilePath: config.DockerfilePath,
			HasDockerfile:  true,
		}
	}

	// Allocate port
	port, err := m.portAllocator.AllocatePort()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate port: %v", err)
	}

	// Override with config if provided
	if config.ExternalPort > 0 {
		if m.portAllocator.IsPortAvailable(config.ExternalPort) {
			port = config.ExternalPort
		}
	}

	name := cloneResult.Name
	if config.Name != "" {
		name = config.Name
	}

	dockerfilePath := cloneResult.DockerfilePath
	if config.DockerfilePath != "" {
		dockerfilePath = config.DockerfilePath
	}

	buildContext := "."
	if config.BuildContext != "" {
		buildContext = config.BuildContext
	}

	internalPort := 80
	if config.InternalPort > 0 {
		internalPort = config.InternalPort
	} else if cloneResult.Manifest != nil && cloneResult.Manifest.DefaultPort > 0 {
		internalPort = cloneResult.Manifest.DefaultPort
	}

	env := make(map[string]string)
	if cloneResult.Manifest != nil && cloneResult.Manifest.Env != nil {
		for k, v := range cloneResult.Manifest.Env {
			env[k] = v
		}
	}
	if config.Env != nil {
		for k, v := range config.Env {
			env[k] = v
		}
	}

	buildArgs := make(map[string]string)
	if config.BuildArgs != nil {
		buildArgs = config.BuildArgs
	}

	now := time.Now()
	commit, _ := m.gitService.GetLastCommit(cloneResult.Slug)

	app := &models.App{
		ID:             uuid.New().String(),
		Name:           name,
		Slug:           cloneResult.Slug,
		Description:    cloneResult.Description,
		RepoURL:        repoURL,
		Branch:         branch,
		LastCommit:     commit,
		LastPulled:     &now,
		DockerfilePath: dockerfilePath,
		BuildContext:   buildContext,
		BuildArgs:      buildArgs,
		ImageName:      fmt.Sprintf("%s:latest", cloneResult.Slug),
		ContainerName:  cloneResult.Slug,
		InternalPort:   internalPort,
		ExternalPort:   port,
		RestartPolicy:  "unless-stopped",
		Env:            env,
		Status:         models.StatusStopped,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := m.db.CreateApp(app); err != nil {
		return nil, fmt.Errorf("failed to save app: %v", err)
	}

	return app, nil
}

func (m *AppManager) BuildApp(ctx context.Context, appID string, progressChan chan<- BuildProgress) error {
	app, err := m.db.GetApp(appID)
	if err != nil {
		return fmt.Errorf("app not found: %v", err)
	}

	// Update status to building
	app.Status = models.StatusBuilding
	m.db.UpdateApp(app)

	repoPath := m.gitService.GetRepoPath(app.Slug)
	buildContext := filepath.Join(repoPath, app.BuildContext)

	startTime := time.Now()
	err = m.buildService.BuildApp(ctx, app, buildContext, progressChan)
	duration := time.Since(startTime)

	app.LastBuild = &startTime
	app.LastBuildDuration = duration.Round(time.Second).String()

	if err != nil {
		app.Status = models.StatusBuildFailed
		app.LastBuildSuccess = false
		m.db.UpdateApp(app)
		return err
	}

	app.Status = models.StatusStopped
	app.LastBuildSuccess = true

	// Get image size
	if size, err := m.dockerClient.GetImageSize(ctx, app.ImageName); err == nil {
		app.ImageSize = size
	}

	m.db.UpdateApp(app)
	return nil
}

func (m *AppManager) StartApp(ctx context.Context, appID string) error {
	app, err := m.db.GetApp(appID)
	if err != nil {
		return fmt.Errorf("app not found: %v", err)
	}

	// Check if container exists
	existing, _ := m.dockerClient.GetContainerByName(ctx, app.ContainerName)
	if existing != nil {
		// Remove old container
		m.dockerClient.StopContainer(ctx, existing.ID)
		m.dockerClient.RemoveContainer(ctx, existing.ID, true)
	}

	// Check port availability and reassign if needed
	if !m.portAllocator.IsPortAvailable(app.ExternalPort) {
		newPort, err := m.portAllocator.FindNextAvailable(app.ExternalPort)
		if err != nil {
			return fmt.Errorf("no available ports: %v", err)
		}
		app.ExternalPort = newPort
		m.db.UpdateApp(app)
	}

	// Create container
	containerID, err := m.dockerClient.CreateContainer(
		ctx,
		app.ContainerName,
		app.ImageName,
		app.InternalPort,
		app.ExternalPort,
		app.Env,
		app.RestartPolicy,
	)
	if err != nil {
		app.Status = models.StatusError
		m.db.UpdateApp(app)
		return fmt.Errorf("failed to create container: %v", err)
	}

	app.ContainerID = containerID
	app.Status = models.StatusStarting
	m.db.UpdateApp(app)

	// Start container
	if err := m.dockerClient.StartContainer(ctx, containerID); err != nil {
		app.Status = models.StatusError
		m.db.UpdateApp(app)
		return fmt.Errorf("failed to start container: %v", err)
	}

	app.Status = models.StatusRunning
	m.db.UpdateApp(app)

	return nil
}

func (m *AppManager) StopApp(ctx context.Context, appID string) error {
	app, err := m.db.GetApp(appID)
	if err != nil {
		return fmt.Errorf("app not found: %v", err)
	}

	if app.ContainerID != "" {
		if err := m.dockerClient.StopContainer(ctx, app.ContainerID); err != nil {
			// Container might not exist, try by name
			if container, _ := m.dockerClient.GetContainerByName(ctx, app.ContainerName); container != nil {
				m.dockerClient.StopContainer(ctx, container.ID)
			}
		}
	}

	app.Status = models.StatusStopped
	m.db.UpdateApp(app)

	return nil
}

func (m *AppManager) RestartApp(ctx context.Context, appID string) error {
	if err := m.StopApp(ctx, appID); err != nil {
		// Ignore stop errors
	}
	return m.StartApp(ctx, appID)
}

func (m *AppManager) DeleteApp(ctx context.Context, appID string) error {
	app, err := m.db.GetApp(appID)
	if err != nil {
		return fmt.Errorf("app not found: %v", err)
	}

	// Stop and remove container
	if app.ContainerID != "" {
		m.dockerClient.StopContainer(ctx, app.ContainerID)
		m.dockerClient.RemoveContainer(ctx, app.ContainerID, true)
	}

	// Also try by name
	if container, _ := m.dockerClient.GetContainerByName(ctx, app.ContainerName); container != nil {
		m.dockerClient.StopContainer(ctx, container.ID)
		m.dockerClient.RemoveContainer(ctx, container.ID, true)
	}

	// Remove image
	m.dockerClient.RemoveImage(ctx, app.ImageName)

	// Remove repo
	m.gitService.RemoveRepo(app.Slug)

	// Remove build logs
	m.buildService.ClearBuildLog(app.ID)

	// Remove from database
	return m.db.DeleteApp(appID)
}

func (m *AppManager) PullAndRebuild(ctx context.Context, appID string, progressChan chan<- BuildProgress) error {
	app, err := m.db.GetApp(appID)
	if err != nil {
		return fmt.Errorf("app not found: %v", err)
	}

	wasRunning := app.Status == models.StatusRunning

	// Stop container if running (must stop before rebuild to free the container name)
	if wasRunning {
		m.StopApp(ctx, appID)
	}

	// Pull latest changes
	commit, err := m.gitService.PullRepo(app.Slug, app.Branch)
	if err != nil {
		return fmt.Errorf("failed to pull repo: %v", err)
	}

	now := time.Now()
	app.LastCommit = commit[:8]
	app.LastPulled = &now
	m.db.UpdateApp(app)

	// Rebuild
	if err := m.BuildApp(ctx, appID, progressChan); err != nil {
		return err
	}

	// Auto-restart if was running
	if wasRunning {
		return m.StartApp(ctx, appID)
	}
	return nil
}

func (m *AppManager) CheckAppUpdate(appID string) (*UpdateCheckResult, error) {
	app, err := m.db.GetApp(appID)
	if err != nil {
		return nil, fmt.Errorf("app not found: %v", err)
	}
	return m.gitService.CheckForUpdates(app.Slug, app.Branch)
}

func (m *AppManager) GetApp(appID string) (*models.App, error) {
	return m.db.GetApp(appID)
}

func (m *AppManager) GetAllApps() ([]*models.App, error) {
	return m.db.GetAllApps()
}

func (m *AppManager) UpdateApp(app *models.App) error {
	app.UpdatedAt = time.Now()
	return m.db.UpdateApp(app)
}

func (m *AppManager) ReconcileStates() error {
	ctx := context.Background()
	apps, err := m.db.GetAllApps()
	if err != nil {
		return err
	}

	for _, app := range apps {
		if app.ContainerID == "" && app.ContainerName != "" {
			// Try to find container by name
			container, _ := m.dockerClient.GetContainerByName(ctx, app.ContainerName)
			if container != nil {
				app.ContainerID = container.ID
			}
		}

		if app.ContainerID != "" {
			status, err := m.dockerClient.GetContainerStatus(ctx, app.ContainerID)
			if err != nil {
				app.Status = models.StatusStopped
				app.ContainerID = ""
			} else if status == "running" {
				app.Status = models.StatusRunning
			} else {
				app.Status = models.StatusStopped
			}
		} else {
			if app.Status == models.StatusRunning || app.Status == models.StatusStarting {
				app.Status = models.StatusStopped
			}
		}

		m.db.UpdateApp(app)
	}

	return nil
}

func (m *AppManager) GetContainerUptime(ctx context.Context, appID string) (string, error) {
	app, err := m.db.GetApp(appID)
	if err != nil {
		return "", err
	}

	if app.ContainerID == "" {
		return "", nil
	}

	return m.dockerClient.GetContainerUptime(ctx, app.ContainerID)
}
