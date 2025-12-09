// Package docker provides a Docker client abstraction using the docker CLI.
package docker

import (
	"fmt"
	"regexp"
)

// ContainerConfig holds configuration for creating a container.
type ContainerConfig struct {
	// Name is the container name (optional, Docker will generate one if empty)
	Name string

	// Image is the Docker image to run (e.g., "mcr.microsoft.com/azure-storage/azurite")
	Image string

	// Ports contains the port mappings for the container
	Ports []PortMapping

	// Environment contains environment variables to set in the container
	Environment map[string]string
}

// PortMapping represents a host:container port mapping.
type PortMapping struct {
	// HostPort is the port on the host machine (0 = auto-assign)
	HostPort int

	// ContainerPort is the port inside the container
	ContainerPort int

	// Protocol is the network protocol ("tcp" or "udp"), defaults to "tcp"
	Protocol string

	// BindIP is the IP address to bind to on the host (e.g., "127.0.0.1")
	BindIP string
}

// Container represents a Docker container.
type Container struct {
	// ID is the container's unique identifier
	ID string

	// Name is the container's name
	Name string

	// Image is the image the container was created from
	Image string

	// Status is the container's current status (e.g., "running", "exited")
	Status string
}

// DefaultProtocol is the default protocol for port mappings.
const DefaultProtocol = "tcp"

// imageNameRegex validates Docker image names.
// Pattern: [registry/][namespace/]name[:tag][@digest]
// Examples: nginx, nginx:latest, docker.io/library/nginx:1.21, mcr.microsoft.com/azure-storage/azurite
var imageNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*[a-zA-Z0-9](:[a-zA-Z0-9._-]+)?(@sha256:[a-fA-F0-9]{64})?$`)

// containerNameRegex validates Docker container names.
// Pattern: [a-zA-Z0-9][a-zA-Z0-9_.-]* (must start with alphanumeric)
var containerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// ValidateImageName checks if an image name is valid and safe.
// Returns an error if the image name contains potentially dangerous characters
// or doesn't match the expected Docker image name format.
func ValidateImageName(image string) error {
	if image == "" {
		return fmt.Errorf("image name cannot be empty")
	}
	if len(image) > 255 {
		return fmt.Errorf("image name too long (max 255 characters)")
	}
	if !imageNameRegex.MatchString(image) {
		return fmt.Errorf("invalid image name %q: must match pattern [registry/][namespace/]name[:tag][@digest]", image)
	}
	return nil
}

// ValidateContainerName checks if a container name is valid.
func ValidateContainerName(name string) error {
	if name == "" {
		return nil // Empty name is valid (Docker will generate one)
	}
	if len(name) > 128 {
		return fmt.Errorf("container name too long (max 128 characters)")
	}
	if !containerNameRegex.MatchString(name) {
		return fmt.Errorf("invalid container name %q: must start with alphanumeric and contain only [a-zA-Z0-9_.-]", name)
	}
	return nil
}

// GetProtocol returns the protocol for a port mapping, defaulting to "tcp".
func (p PortMapping) GetProtocol() string {
	if p.Protocol == "" {
		return DefaultProtocol
	}
	return p.Protocol
}

// Validate checks if the port mapping is valid.
func (p PortMapping) Validate() error {
	// HostPort 0 is valid (auto-assign)
	if p.HostPort < 0 || p.HostPort > 65535 {
		return fmt.Errorf("invalid host port %d: must be 0-65535", p.HostPort)
	}
	if p.ContainerPort < 1 || p.ContainerPort > 65535 {
		return fmt.Errorf("invalid container port %d: must be 1-65535", p.ContainerPort)
	}
	if p.Protocol != "" && p.Protocol != "tcp" && p.Protocol != "udp" {
		return fmt.Errorf("invalid protocol %q: must be 'tcp' or 'udp'", p.Protocol)
	}
	return nil
}

// Validate checks if the container configuration is valid.
func (c ContainerConfig) Validate() error {
	if err := ValidateImageName(c.Image); err != nil {
		return err
	}
	if err := ValidateContainerName(c.Name); err != nil {
		return err
	}
	for i, port := range c.Ports {
		if err := port.Validate(); err != nil {
			return fmt.Errorf("port mapping %d: %w", i, err)
		}
	}
	return nil
}
