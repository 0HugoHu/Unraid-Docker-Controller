package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
)

type Client struct {
	cli *client.Client
}

type BuildMessage struct {
	Stream      string `json:"stream"`
	Error       string `json:"error"`
	ErrorDetail struct {
		Message string `json:"message"`
	} `json:"errorDetail"`
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker: %v", err)
	}

	return &Client{cli: cli}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

func (c *Client) BuildImage(ctx context.Context, contextPath string, dockerfilePath string, imageName string, buildArgs map[string]string, logWriter io.Writer) error {
	// Create tar archive of the build context
	tar, err := archive.TarWithOptions(contextPath, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("failed to create build context: %v", err)
	}
	defer tar.Close()

	// Convert build args
	args := make(map[string]*string)
	for k, v := range buildArgs {
		val := v
		args[k] = &val
	}

	opts := types.ImageBuildOptions{
		Dockerfile: dockerfilePath,
		Tags:       []string{imageName},
		BuildArgs:  args,
		Remove:     true,
		ForceRemove: true,
	}

	resp, err := c.cli.ImageBuild(ctx, tar, opts)
	if err != nil {
		return fmt.Errorf("failed to build image: %v", err)
	}
	defer resp.Body.Close()

	// Stream build output
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var msg BuildMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Error != "" {
			return fmt.Errorf("build error: %s", msg.Error)
		}
		if msg.Stream != "" && logWriter != nil {
			logWriter.Write([]byte(msg.Stream))
		}
	}

	return nil
}

func (c *Client) CreateContainer(ctx context.Context, name string, imageName string, internalPort int, externalPort int, env map[string]string, restartPolicy string) (string, error) {
	// Convert env map to slice
	envSlice := make([]string, 0, len(env))
	for k, v := range env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}

	// Port bindings
	portStr := fmt.Sprintf("%d/tcp", internalPort)
	exposedPorts := nat.PortSet{
		nat.Port(portStr): struct{}{},
	}
	portBindings := nat.PortMap{
		nat.Port(portStr): []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: strconv.Itoa(externalPort),
			},
		},
	}

	// Restart policy
	var restartPolicyConfig container.RestartPolicy
	switch restartPolicy {
	case "always":
		restartPolicyConfig = container.RestartPolicy{Name: "always"}
	case "unless-stopped":
		restartPolicyConfig = container.RestartPolicy{Name: "unless-stopped"}
	case "on-failure":
		restartPolicyConfig = container.RestartPolicy{Name: "on-failure", MaximumRetryCount: 3}
	default:
		restartPolicyConfig = container.RestartPolicy{Name: "no"}
	}

	config := &container.Config{
		Image:        imageName,
		Env:          envSlice,
		ExposedPorts: exposedPorts,
	}

	hostConfig := &container.HostConfig{
		PortBindings:  portBindings,
		RestartPolicy: restartPolicyConfig,
	}

	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, &network.NetworkingConfig{}, nil, name)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	return c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
}

func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	timeout := 30
	return c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	return c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	})
}

func (c *Client) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	info, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}

	if info.State.Running {
		return "running", nil
	}
	return "stopped", nil
}

func (c *Client) GetContainerLogs(ctx context.Context, containerID string, tail string) (io.ReadCloser, error) {
	return c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	})
}

func (c *Client) StreamContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	return c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	})
}

func (c *Client) RemoveImage(ctx context.Context, imageName string) error {
	_, err := c.cli.ImageRemove(ctx, imageName, image.RemoveOptions{Force: true, PruneChildren: true})
	return err
}

func (c *Client) GetImageSize(ctx context.Context, imageName string) (int64, error) {
	inspect, _, err := c.cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return 0, err
	}
	return inspect.Size, nil
}

func (c *Client) PruneImages(ctx context.Context) (uint64, error) {
	report, err := c.cli.ImagesPrune(ctx, filters.Args{})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

func (c *Client) GetContainerByName(ctx context.Context, name string) (*types.Container, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	searchName := "/" + name
	for _, cont := range containers {
		for _, n := range cont.Names {
			if n == searchName {
				return &cont, nil
			}
		}
	}
	return nil, nil
}

func (c *Client) GetDockerInfo(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.cli.Info(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"containers":      info.Containers,
		"containersRunning": info.ContainersRunning,
		"images":          info.Images,
		"serverVersion":   info.ServerVersion,
		"memoryTotal":     info.MemTotal,
	}, nil
}

func (c *Client) IsPortInUse(port int) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return true
	}
	listener.Close()
	return false
}

func (c *Client) GetContainerUptime(ctx context.Context, containerID string) (string, error) {
	info, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}

	if !info.State.Running {
		return "", nil
	}

	startTime, err := time.Parse(time.RFC3339Nano, info.State.StartedAt)
	if err != nil {
		return "", err
	}

	duration := time.Since(startTime)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours), nil
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes), nil
	}
	return fmt.Sprintf("%dm", minutes), nil
}

func (c *Client) InspectSelf(ctx context.Context) (types.ContainerJSON, error) {
	hostname, _ := os.Hostname() // Container ID in Docker
	return c.cli.ContainerInspect(ctx, hostname)
}
