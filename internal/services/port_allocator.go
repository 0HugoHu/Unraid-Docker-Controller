package services

import (
	"fmt"
	"net"
	"sync"

	"nas-controller/internal/database"
	"nas-controller/internal/docker"
)

const (
	PortRangeStart = 13001
	PortRangeEnd   = 13999
)

type PortAllocator struct {
	db           *database.DB
	dockerClient *docker.Client
	mu           sync.Mutex
}

func NewPortAllocator(db *database.DB, dockerClient *docker.Client) *PortAllocator {
	return &PortAllocator{
		db:           db,
		dockerClient: dockerClient,
	}
}

func (p *PortAllocator) AllocatePort() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	usedPorts, err := p.db.GetUsedPorts()
	if err != nil {
		return 0, fmt.Errorf("failed to get used ports: %v", err)
	}

	usedSet := make(map[int]bool)
	for _, port := range usedPorts {
		usedSet[port] = true
	}

	for port := PortRangeStart; port <= PortRangeEnd; port++ {
		if usedSet[port] {
			continue
		}
		if !p.isPortInUse(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", PortRangeStart, PortRangeEnd)
}

func (p *PortAllocator) IsPortAvailable(port int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	usedPorts, err := p.db.GetUsedPorts()
	if err != nil {
		return false
	}

	for _, used := range usedPorts {
		if used == port {
			return false
		}
	}

	return !p.isPortInUse(port)
}

func (p *PortAllocator) FindNextAvailable(preferredPort int) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	usedPorts, err := p.db.GetUsedPorts()
	if err != nil {
		return 0, err
	}

	usedSet := make(map[int]bool)
	for _, port := range usedPorts {
		usedSet[port] = true
	}

	// Try preferred port first
	if preferredPort >= PortRangeStart && preferredPort <= PortRangeEnd {
		if !usedSet[preferredPort] && !p.isPortInUse(preferredPort) {
			return preferredPort, nil
		}
	}

	// Find next available
	for port := PortRangeStart; port <= PortRangeEnd; port++ {
		if usedSet[port] {
			continue
		}
		if !p.isPortInUse(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports")
}

func (p *PortAllocator) isPortInUse(port int) bool {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return true
	}
	listener.Close()
	return false
}

func (p *PortAllocator) GetUsedPorts() ([]int, error) {
	return p.db.GetUsedPorts()
}
