package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"nas-controller/internal/database"
	"nas-controller/internal/services"
)

type AuthHandler struct {
	db          *database.DB
	authService *services.AuthService
}

func NewAuthHandler(db *database.DB, authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		db:          db,
		authService: authService,
	}
}

type LoginRequest struct {
	Password string `json:"password" binding:"required"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=8"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
		return
	}

	if !h.authService.ValidatePassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}

	// Clean up expired sessions
	h.db.CleanupExpiredSessions()

	// Create new session
	token := h.authService.GenerateSessionToken()
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days

	if err := h.db.CreateSession(token, expiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	// Set cookie
	c.SetCookie("session", token, 7*24*60*60, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"expiresAt": expiresAt,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token, err := c.Cookie("session")
	if err == nil {
		h.db.DeleteSession(token)
	}

	c.SetCookie("session", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AuthHandler) Check(c *gin.Context) {
	token, err := c.Cookie("session")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"authenticated": false})
		return
	}

	valid := h.db.ValidateSession(token)
	c.JSON(http.StatusOK, gin.H{"authenticated": valid})
}

func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.authService.UpdatePassword(req.CurrentPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}
