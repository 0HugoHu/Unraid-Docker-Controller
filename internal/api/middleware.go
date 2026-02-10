package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"nas-controller/internal/database"
)

type AuthMiddleware struct {
	db *database.DB
}

func NewAuthMiddleware(db *database.DB) *AuthMiddleware {
	return &AuthMiddleware{db: db}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check session cookie
		token, err := c.Cookie("session")
		if err != nil {
			// Also check Authorization header
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if !m.db.ValidateSession(token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) AuthenticateWS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For WebSocket, check query param
		token := c.Query("token")
		if token == "" {
			// Also check cookie
			token, _ = c.Cookie("session")
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if !m.db.ValidateSession(token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
			return
		}

		c.Next()
	}
}
