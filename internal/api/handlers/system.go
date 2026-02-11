package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"nas-controller/internal/database"
	"nas-controller/internal/docker"
	"nas-controller/internal/services"
)

const Version = "1.0.0"

const defaultControllerRepo = "https://github.com/0HugoHu/Unraid-Docker-Controller.git"

type SystemHandler struct {
	dockerClient *docker.Client
	buildService *services.BuildService
	db           *database.DB
	dataDir      string
}

func NewSystemHandler(
	dockerClient *docker.Client,
	buildService *services.BuildService,
	db *database.DB,
	dataDir string,
) *SystemHandler {
	return &SystemHandler{
		dockerClient: dockerClient,
		buildService: buildService,
		db:           db,
		dataDir:      dataDir,
	}
}

func (h *SystemHandler) GetInfo(c *gin.Context) {
	ctx := context.Background()

	dockerInfo, _ := h.dockerClient.GetDockerInfo(ctx)

	apps, _ := h.db.GetAllApps()
	runningCount := 0
	for _, app := range apps {
		if app.Status == "running" {
			runningCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"version":     Version,
		"totalApps":   len(apps),
		"runningApps": runningCount,
		"docker":      dockerInfo,
	})
}

func (h *SystemHandler) GetStorage(c *gin.Context) {
	// Get database size
	dbPath := filepath.Join(h.dataDir, "controller.db")
	dbInfo, _ := os.Stat(dbPath)
	dbSize := int64(0)
	if dbInfo != nil {
		dbSize = dbInfo.Size()
	}

	// Get repos size
	reposSize := int64(0)
	reposDir := filepath.Join(h.dataDir, "repos")
	filepath.Walk(reposDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			reposSize += info.Size()
		}
		return nil
	})

	// Get logs size
	logsSize, _ := h.buildService.GetLogsSize()

	// Get Docker images size
	imagesSize := int64(0)
	apps, _ := h.db.GetAllApps()
	for _, app := range apps {
		imagesSize += app.ImageSize
	}

	c.JSON(http.StatusOK, gin.H{
		"database":    dbSize,
		"repositories": reposSize,
		"logs":        logsSize,
		"images":      imagesSize,
		"total":       dbSize + reposSize + logsSize + imagesSize,
	})
}

func (h *SystemHandler) GetPorts(c *gin.Context) {
	usedPorts, _ := h.db.GetUsedPorts()

	c.JSON(http.StatusOK, gin.H{
		"usedPorts": usedPorts,
		"range": gin.H{
			"start": services.PortRangeStart,
			"end":   services.PortRangeEnd,
		},
	})
}

func (h *SystemHandler) PruneImages(c *gin.Context) {
	ctx := context.Background()

	reclaimed, err := h.dockerClient.PruneImages(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "images pruned",
		"spaceReclaimed": reclaimed,
	})
}

func (h *SystemHandler) ClearAllLogs(c *gin.Context) {
	if err := h.buildService.ClearAllLogs(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all logs cleared"})
}

func (h *SystemHandler) CheckSelfUpdate(c *gin.Context) {
	var req struct {
		RepoURL string `json:"repoUrl"`
		Branch  string `json:"branch"`
	}
	c.ShouldBindJSON(&req)

	if req.RepoURL == "" {
		req.RepoURL = defaultControllerRepo
	}
	if req.Branch == "" {
		req.Branch = "main"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	srcDir := filepath.Join(h.dataDir, "controller-src")

	if _, err := os.Stat(filepath.Join(srcDir, ".git")); os.IsNotExist(err) {
		// No local source yet — clone it to check
		os.RemoveAll(srcDir)
		cmd := exec.CommandContext(ctx, "git", "clone", "--branch", req.Branch, "--depth", "1", req.RepoURL, srcDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("git clone failed: %s", string(output))})
			return
		}
		// First time — always report as update available
		cmd = exec.CommandContext(ctx, "git", "-C", srcDir, "rev-parse", "HEAD")
		commitOut, _ := cmd.Output()
		commit := strings.TrimSpace(string(commitOut))
		short := commit
		if len(short) > 8 {
			short = short[:8]
		}
		c.JSON(http.StatusOK, gin.H{
			"hasUpdate":    true,
			"localCommit":  "none",
			"remoteCommit": short,
		})
		return
	}

	// Update remote URL
	exec.CommandContext(ctx, "git", "-C", srcDir, "remote", "set-url", "origin", req.RepoURL).Run()

	// Get local HEAD before fetch
	cmd := exec.CommandContext(ctx, "git", "-C", srcDir, "rev-parse", "HEAD")
	localOut, err := cmd.Output()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get local commit: %v", err)})
		return
	}
	localCommit := strings.TrimSpace(string(localOut))

	// Fetch remote
	cmd = exec.CommandContext(ctx, "git", "-C", srcDir, "fetch", "origin", req.Branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("git fetch failed: %s", string(output))})
		return
	}

	// Get remote HEAD
	cmd = exec.CommandContext(ctx, "git", "-C", srcDir, "rev-parse", fmt.Sprintf("origin/%s", req.Branch))
	remoteOut, err := cmd.Output()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get remote commit: %v", err)})
		return
	}
	remoteCommit := strings.TrimSpace(string(remoteOut))

	localShort := localCommit
	if len(localShort) > 8 {
		localShort = localShort[:8]
	}
	remoteShort := remoteCommit
	if len(remoteShort) > 8 {
		remoteShort = remoteShort[:8]
	}

	c.JSON(http.StatusOK, gin.H{
		"hasUpdate":    localCommit != remoteCommit,
		"localCommit":  localShort,
		"remoteCommit": remoteShort,
	})
}

func (h *SystemHandler) SelfUpdate(c *gin.Context) {
	var req struct {
		RepoURL string `json:"repoUrl"`
		Branch  string `json:"branch"`
	}
	c.ShouldBindJSON(&req)

	if req.RepoURL == "" {
		req.RepoURL = defaultControllerRepo
	}
	if req.Branch == "" {
		req.Branch = "main"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Step 1: Clone or pull the controller source
	srcDir := filepath.Join(h.dataDir, "controller-src")
	if _, err := os.Stat(filepath.Join(srcDir, ".git")); err == nil {
		exec.CommandContext(ctx, "git", "-C", srcDir, "remote", "set-url", "origin", req.RepoURL).Run()
		cmd := exec.CommandContext(ctx, "git", "-C", srcDir, "fetch", "origin", req.Branch)
		if output, err := cmd.CombinedOutput(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("git fetch failed: %s", string(output))})
			return
		}
		cmd = exec.CommandContext(ctx, "git", "-C", srcDir, "reset", "--hard", fmt.Sprintf("origin/%s", req.Branch))
		if output, err := cmd.CombinedOutput(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("git reset failed: %s", string(output))})
			return
		}
	} else {
		os.RemoveAll(srcDir)
		cmd := exec.CommandContext(ctx, "git", "clone", "--branch", req.Branch, "--depth", "1", req.RepoURL, srcDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("git clone failed: %s", string(output))})
			return
		}
	}

	// Step 2: Build new image
	imageName := "nas-controller:latest"
	log.Printf("Self-update: building new image %s from %s", imageName, srcDir)
	if err := h.dockerClient.BuildImage(ctx, srcDir, "./Dockerfile", imageName, nil, io.Discard); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("image build failed: %v", err)})
		return
	}

	// Step 3: Inspect self to get current container config
	selfInfo, err := h.dockerClient.InspectSelf(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to inspect self: %v", err)})
		return
	}

	containerName := strings.TrimPrefix(selfInfo.Name, "/")
	containerID := selfInfo.ID

	// Extract port bindings
	var portArgs []string
	for containerPort, bindings := range selfInfo.HostConfig.PortBindings {
		for _, binding := range bindings {
			hostPort := binding.HostPort
			if binding.HostIP != "" && binding.HostIP != "0.0.0.0" {
				portArgs = append(portArgs, fmt.Sprintf("-p %s:%s:%s", binding.HostIP, hostPort, containerPort))
			} else {
				portArgs = append(portArgs, fmt.Sprintf("-p %s:%s", hostPort, containerPort))
			}
		}
	}

	// Extract volume mounts
	var volumeArgs []string
	for _, mount := range selfInfo.Mounts {
		if mount.Type == "bind" {
			volumeArgs = append(volumeArgs, fmt.Sprintf("-v %s:%s", mount.Source, mount.Destination))
		} else if mount.Type == "volume" {
			volumeArgs = append(volumeArgs, fmt.Sprintf("-v %s:%s", mount.Name, mount.Destination))
		}
	}

	// Extract env vars
	var envArgs []string
	for _, env := range selfInfo.Config.Env {
		envArgs = append(envArgs, fmt.Sprintf("-e %s", env))
	}

	// Build the swap script
	createArgs := []string{
		"docker create",
		fmt.Sprintf("--name %s", containerName),
		"--restart unless-stopped",
	}
	createArgs = append(createArgs, portArgs...)
	createArgs = append(createArgs, volumeArgs...)
	createArgs = append(createArgs, envArgs...)
	createArgs = append(createArgs, imageName)

	swapScript := fmt.Sprintf(
		"sleep 3 && docker stop %s && docker rm %s && %s && docker start %s",
		containerID,
		containerID,
		strings.Join(createArgs, " "),
		containerName,
	)

	// Step 4: Spawn helper container to perform the swap
	cmd := exec.CommandContext(ctx, "docker", "run", "-d", "--rm",
		"--name", "nas-controller-updater",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"docker:cli",
		"sh", "-c", swapScript,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to spawn updater: %s", string(output))})
		return
	}

	log.Printf("Self-update: helper container spawned, controller will restart shortly")
	c.JSON(http.StatusOK, gin.H{"message": "Update in progress. Controller will restart shortly."})
}
