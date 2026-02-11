package services

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"nas-controller/internal/models"
)

type GitService struct {
	dataDir  string
	reposDir string
}

func NewGitService(dataDir string) *GitService {
	reposDir := filepath.Join(dataDir, "repos")
	os.MkdirAll(reposDir, 0755)

	return &GitService{
		dataDir:  dataDir,
		reposDir: reposDir,
	}
}

func (s *GitService) CloneRepo(repoURL string, branch string) (*models.CloneResult, error) {
	// Extract slug from URL
	slug := s.extractSlug(repoURL)
	if slug == "" {
		return nil, fmt.Errorf("invalid repository URL")
	}

	repoPath := filepath.Join(s.reposDir, slug)

	// Remove existing repo if exists
	os.RemoveAll(repoPath)

	// Clone the repository
	cmd := exec.Command("git", "clone", "--branch", branch, "--depth", "1", repoURL, repoPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git clone failed: %s, output: %s", err, string(output))
	}

	// Check for Dockerfile
	dockerfilePath := "./Dockerfile"
	hasDockerfile := false

	possiblePaths := []string{
		filepath.Join(repoPath, "Dockerfile"),
		filepath.Join(repoPath, "dockerfile"),
		filepath.Join(repoPath, "docker", "Dockerfile"),
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			hasDockerfile = true
			rel, _ := filepath.Rel(repoPath, p)
			dockerfilePath = "./" + filepath.ToSlash(rel)
			break
		}
	}

	if !hasDockerfile {
		os.RemoveAll(repoPath)
		return nil, fmt.Errorf("no Dockerfile found in repository. Please add a Dockerfile to your repo")
	}

	// Read manifest if exists
	var manifest *models.AppManifest
	manifestPath := filepath.Join(repoPath, "nas-controller.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		manifest = &models.AppManifest{}
		json.Unmarshal(data, manifest)
	}

	// Determine name
	name := slug
	description := ""
	if manifest != nil && manifest.Name != "" {
		name = manifest.Name
	}
	if manifest != nil && manifest.Description != "" {
		description = manifest.Description
	}

	result := &models.CloneResult{
		Slug:           slug,
		Name:           name,
		Description:    description,
		HasDockerfile:  hasDockerfile,
		DockerfilePath: dockerfilePath,
		Manifest:       manifest,
		SuggestedPort:  80,
	}

	if manifest != nil && manifest.DefaultPort > 0 {
		result.SuggestedPort = manifest.DefaultPort
	}

	return result, nil
}

func (s *GitService) PullRepo(slug string, branch string) (string, error) {
	repoPath := filepath.Join(s.reposDir, slug)

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return "", fmt.Errorf("repository not found")
	}

	// Fetch and reset to origin
	cmd := exec.Command("git", "-C", repoPath, "fetch", "origin", branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git fetch failed: %s, output: %s", err, string(output))
	}

	cmd = exec.Command("git", "-C", repoPath, "reset", "--hard", fmt.Sprintf("origin/%s", branch))
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git reset failed: %s, output: %s", err, string(output))
	}

	// Get latest commit hash
	cmd = exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %s", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (s *GitService) GetRepoPath(slug string) string {
	return filepath.Join(s.reposDir, slug)
}

func (s *GitService) GetLastCommit(slug string) (string, error) {
	repoPath := filepath.Join(s.reposDir, slug)

	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output))[:8], nil
}

func (s *GitService) RemoveRepo(slug string) error {
	repoPath := filepath.Join(s.reposDir, slug)
	return os.RemoveAll(repoPath)
}

func (s *GitService) extractSlug(repoURL string) string {
	// Handle various GitHub URL formats
	re := regexp.MustCompile(`github\.com[/:]([^/]+)/([^/.]+)`)
	matches := re.FindStringSubmatch(repoURL)
	if len(matches) >= 3 {
		return strings.ToLower(matches[2])
	}
	return ""
}

type UpdateCheckResult struct {
	HasUpdate    bool   `json:"hasUpdate"`
	LocalCommit  string `json:"localCommit"`
	RemoteCommit string `json:"remoteCommit"`
}

func (s *GitService) CheckForUpdates(slug string, branch string) (*UpdateCheckResult, error) {
	repoPath := filepath.Join(s.reposDir, slug)

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository not found")
	}

	// Get local HEAD
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	localOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get local commit: %v", err)
	}
	localCommit := strings.TrimSpace(string(localOutput))

	// Fetch remote
	cmd = exec.Command("git", "-C", repoPath, "fetch", "origin", branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git fetch failed: %s", string(output))
	}

	// Get remote HEAD
	cmd = exec.Command("git", "-C", repoPath, "rev-parse", fmt.Sprintf("origin/%s", branch))
	remoteOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote commit: %v", err)
	}
	remoteCommit := strings.TrimSpace(string(remoteOutput))

	return &UpdateCheckResult{
		HasUpdate:    localCommit != remoteCommit,
		LocalCommit:  localCommit[:8],
		RemoteCommit: remoteCommit[:8],
	}, nil
}

func (s *GitService) GetReposSize() (int64, error) {
	var size int64
	err := filepath.Walk(s.reposDir, func(path string, info os.FileInfo, err error) error {
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
