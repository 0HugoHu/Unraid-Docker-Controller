package services

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

type AuthService struct {
	dataDir      string
	passwordFile string
}

func NewAuthService(dataDir string) *AuthService {
	return &AuthService{
		dataDir:      dataDir,
		passwordFile: filepath.Join(dataDir, "password.txt"),
	}
}

func (s *AuthService) EnsurePassword() (string, bool, error) {
	// Check if password file exists
	if _, err := os.Stat(s.passwordFile); os.IsNotExist(err) {
		// Generate new password
		password := generateRandomPassword(16)
		if err := os.WriteFile(s.passwordFile, []byte(password), 0600); err != nil {
			return "", false, err
		}
		return password, true, nil
	}

	// Read existing password
	data, err := os.ReadFile(s.passwordFile)
	if err != nil {
		return "", false, err
	}

	return strings.TrimSpace(string(data)), false, nil
}

func (s *AuthService) ValidatePassword(password string) bool {
	storedPassword, _, err := s.EnsurePassword()
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(password), []byte(storedPassword)) == 1
}

func (s *AuthService) UpdatePassword(currentPassword, newPassword string) error {
	if !s.ValidatePassword(currentPassword) {
		return os.ErrPermission
	}

	return os.WriteFile(s.passwordFile, []byte(newPassword), 0600)
}

func (s *AuthService) GenerateSessionToken() string {
	return generateRandomPassword(32)
}

func generateRandomPassword(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}
