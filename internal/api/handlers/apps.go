package handlers

import (
	"bufio"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"nas-controller/internal/docker"
	"nas-controller/internal/models"
	"nas-controller/internal/services"
)

type AppHandler struct {
	appManager   *services.AppManager
	buildService *services.BuildService
	dockerClient *docker.Client
	dataDir      string
}

func NewAppHandler(
	appManager *services.AppManager,
	buildService *services.BuildService,
	dockerClient *docker.Client,
	dataDir string,
) *AppHandler {
	return &AppHandler{
		appManager:   appManager,
		buildService: buildService,
		dockerClient: dockerClient,
		dataDir:      dataDir,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *AppHandler) ListApps(c *gin.Context) {
	apps, err := h.appManager.GetAllApps()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Enrich with uptime info
	ctx := context.Background()
	for _, app := range apps {
		if app.Status == models.StatusRunning && app.ContainerID != "" {
			uptime, _ := h.appManager.GetContainerUptime(ctx, app.ID)
			app.LastBuildDuration = uptime // Reuse field for uptime in list view
		}
	}

	c.JSON(http.StatusOK, apps)
}

func (h *AppHandler) GetApp(c *gin.Context) {
	id := c.Param("id")
	app, err := h.appManager.GetApp(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}

	// Get uptime if running
	if app.Status == models.StatusRunning && app.ContainerID != "" {
		uptime, _ := h.appManager.GetContainerUptime(context.Background(), app.ID)
		// Return uptime separately
		c.JSON(http.StatusOK, gin.H{
			"app":    app,
			"uptime": uptime,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"app": app})
}

func (h *AppHandler) CloneRepo(c *gin.Context) {
	var req models.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.appManager.CloneAndValidate(req.RepoURL, req.Branch)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AppHandler) CreateApp(c *gin.Context) {
	var req struct {
		RepoURL string                      `json:"repoUrl" binding:"required"`
		Branch  string                      `json:"branch" binding:"required"`
		Config  models.ConfigureAppRequest  `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	app, err := h.appManager.CreateApp(req.RepoURL, req.Branch, &req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Auto-trigger build and start in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		if err := h.appManager.BuildApp(ctx, app.ID, nil); err != nil {
			log.Printf("Auto-build failed for %s: %v", app.Name, err)
			return
		}
		if err := h.appManager.StartApp(ctx, app.ID); err != nil {
			log.Printf("Auto-start failed for %s: %v", app.Name, err)
		}
	}()

	c.JSON(http.StatusCreated, app)
}

func (h *AppHandler) UpdateApp(c *gin.Context) {
	id := c.Param("id")
	app, err := h.appManager.GetApp(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}

	var req models.ConfigureAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Name != "" {
		app.Name = req.Name
	}
	if req.DockerfilePath != "" {
		app.DockerfilePath = req.DockerfilePath
	}
	if req.BuildContext != "" {
		app.BuildContext = req.BuildContext
	}
	if req.InternalPort > 0 {
		app.InternalPort = req.InternalPort
	}
	if req.ExternalPort > 0 {
		app.ExternalPort = req.ExternalPort
	}
	if req.Env != nil {
		app.Env = req.Env
	}
	if req.BuildArgs != nil {
		app.BuildArgs = req.BuildArgs
	}

	if err := h.appManager.UpdateApp(app); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, app)
}

func (h *AppHandler) DeleteApp(c *gin.Context) {
	id := c.Param("id")

	if err := h.appManager.DeleteApp(context.Background(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "app deleted"})
}

func (h *AppHandler) BuildApp(c *gin.Context) {
	id := c.Param("id")

	if h.buildService.IsBuilding() {
		c.JSON(http.StatusConflict, gin.H{"error": "another build is in progress"})
		return
	}

	// Start build in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		h.appManager.BuildApp(ctx, id, nil)
	}()

	c.JSON(http.StatusAccepted, gin.H{"message": "build started"})
}

func (h *AppHandler) StartApp(c *gin.Context) {
	id := c.Param("id")

	if err := h.appManager.StartApp(context.Background(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "app started"})
}

func (h *AppHandler) StopApp(c *gin.Context) {
	id := c.Param("id")

	if err := h.appManager.StopApp(context.Background(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "app stopped"})
}

func (h *AppHandler) RestartApp(c *gin.Context) {
	id := c.Param("id")

	if err := h.appManager.RestartApp(context.Background(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "app restarted"})
}

func (h *AppHandler) PullAndRebuild(c *gin.Context) {
	id := c.Param("id")

	if h.buildService.IsBuilding() {
		c.JSON(http.StatusConflict, gin.H{"error": "another build is in progress"})
		return
	}

	// Start in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		h.appManager.PullAndRebuild(ctx, id, nil)
	}()

	c.JSON(http.StatusAccepted, gin.H{"message": "pull and rebuild started"})
}

func (h *AppHandler) CheckUpdate(c *gin.Context) {
	id := c.Param("id")

	result, err := h.appManager.CheckAppUpdate(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AppHandler) GetLogs(c *gin.Context) {
	id := c.Param("id")
	lines := c.DefaultQuery("lines", "100")

	app, err := h.appManager.GetApp(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}

	if app.ContainerID == "" {
		c.JSON(http.StatusOK, gin.H{"logs": ""})
		return
	}

	logs, err := h.dockerClient.GetContainerLogs(context.Background(), app.ContainerID, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer logs.Close()

	data, _ := io.ReadAll(logs)
	// Strip Docker log header bytes
	cleanLogs := stripDockerLogHeaders(data)

	c.JSON(http.StatusOK, gin.H{"logs": string(cleanLogs)})
}

func (h *AppHandler) ClearLogs(c *gin.Context) {
	id := c.Param("id")

	// Clear build logs
	h.buildService.ClearBuildLog(id)

	c.JSON(http.StatusOK, gin.H{"message": "logs cleared"})
}

func (h *AppHandler) GetBuildLogs(c *gin.Context) {
	id := c.Param("id")

	logs, err := h.buildService.GetBuildLog(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (h *AppHandler) GetAppIcon(c *gin.Context) {
	id := c.Param("id")

	// Check for cached icon
	iconPath := filepath.Join(h.dataDir, "icons", id+".png")
	if _, err := os.Stat(iconPath); err == nil {
		c.File(iconPath)
		return
	}

	// Return default icon
	c.JSON(http.StatusNotFound, gin.H{"error": "no icon"})
}

func (h *AppHandler) StreamLogs(c *gin.Context) {
	id := c.Param("id")

	app, err := h.appManager.GetApp(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		return
	}

	if app.ContainerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container not running"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logs, err := h.dockerClient.StreamContainerLogs(ctx, app.ContainerID)
	if err != nil {
		return
	}
	defer logs.Close()

	scanner := bufio.NewScanner(logs)
	for scanner.Scan() {
		line := stripDockerLogHeaders(scanner.Bytes())
		if err := conn.WriteMessage(websocket.TextMessage, line); err != nil {
			return
		}
	}
}

func (h *AppHandler) StreamBuild(c *gin.Context) {
	id := c.Param("id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	app, err := h.appManager.GetApp(id)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error: app not found"))
		return
	}

	progressChan := make(chan services.BuildProgress, 100)
	defer close(progressChan)

	// Start build
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		h.appManager.BuildApp(ctx, app.ID, progressChan)
	}()

	// Stream progress
	for progress := range progressChan {
		if progress.Error != "" {
			conn.WriteMessage(websocket.TextMessage, []byte("ERROR: "+progress.Error))
		}
		if progress.Message != "" {
			conn.WriteMessage(websocket.TextMessage, []byte(progress.Message))
		}
		if progress.Complete {
			if progress.Success {
				conn.WriteMessage(websocket.TextMessage, []byte("\n\nBuild completed successfully!"))
			} else {
				conn.WriteMessage(websocket.TextMessage, []byte("\n\nBuild failed!"))
			}
			return
		}
	}
}

// stripDockerLogHeaders removes the 8-byte header from Docker log lines
func stripDockerLogHeaders(data []byte) []byte {
	var result []byte
	for len(data) > 0 {
		if len(data) >= 8 {
			// Skip header
			size := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
			data = data[8:]
			if size > 0 && len(data) >= size {
				result = append(result, data[:size]...)
				data = data[size:]
			} else {
				result = append(result, data...)
				break
			}
		} else {
			result = append(result, data...)
			break
		}
	}
	return result
}

func parseInt(s string, defaultVal int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultVal
}
