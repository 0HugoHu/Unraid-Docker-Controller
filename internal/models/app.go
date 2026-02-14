package models

import (
	"time"
)

type AppStatus string

const (
	StatusStopped      AppStatus = "stopped"
	StatusRunning      AppStatus = "running"
	StatusBuilding     AppStatus = "building"
	StatusBuildFailed  AppStatus = "build-failed"
	StatusStarting     AppStatus = "starting"
	StatusError        AppStatus = "error"
)

type App struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description string            `json:"description"`
	Icon        string            `json:"icon"`
	RepoURL     string            `json:"repoUrl"`
	Branch      string            `json:"branch"`
	LastCommit  string            `json:"lastCommit"`
	LastPulled  *time.Time        `json:"lastPulled"`

	DockerfilePath string         `json:"dockerfilePath"`
	BuildContext   string         `json:"buildContext"`
	BuildArgs      map[string]string `json:"buildArgs"`

	ImageName     string         `json:"imageName"`
	ContainerName string         `json:"containerName"`
	ContainerID   string         `json:"containerId"`
	InternalPort  int            `json:"internalPort"`
	ExternalPort  int            `json:"externalPort"`
	RestartPolicy string         `json:"restartPolicy"`

	Env     map[string]string `json:"env"`
	Volumes []string          `json:"volumes"`

	Status            AppStatus  `json:"status"`
	LastBuild         *time.Time `json:"lastBuild"`
	LastBuildDuration string     `json:"lastBuildDuration"`
	LastBuildSuccess  bool       `json:"lastBuildSuccess"`
	ImageSize         int64      `json:"imageSize"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AppManifest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Icon        string            `json:"icon"`
	DefaultPort int               `json:"defaultPort"`
	Env         map[string]string `json:"env"`
	Volumes     []string          `json:"volumes,omitempty"`
}

type CreateAppRequest struct {
	RepoURL string `json:"repoUrl" binding:"required"`
	Branch  string `json:"branch" binding:"required"`
}

type ConfigureAppRequest struct {
	Name           string            `json:"name"`
	DockerfilePath string            `json:"dockerfilePath"`
	BuildContext   string            `json:"buildContext"`
	InternalPort   int               `json:"internalPort"`
	ExternalPort   int               `json:"externalPort"`
	Env            map[string]string `json:"env"`
	BuildArgs      map[string]string `json:"buildArgs"`
	Volumes        []string          `json:"volumes,omitempty"`
}

type CloneResult struct {
	Slug           string      `json:"slug"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	HasDockerfile  bool        `json:"hasDockerfile"`
	DockerfilePath string      `json:"dockerfilePath"`
	Manifest       *AppManifest `json:"manifest"`
	SuggestedPort  int         `json:"suggestedPort"`
}
