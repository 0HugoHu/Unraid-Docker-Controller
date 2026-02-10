package handlers

import (
	"context"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"nas-controller/internal/database"
	"nas-controller/internal/docker"
	"nas-controller/internal/services"
)

const Version = "1.0.0"

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
