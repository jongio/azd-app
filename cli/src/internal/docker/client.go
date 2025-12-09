package docker

import "io"

// Client provides Docker container operations using the docker CLI.
type Client interface {
	// IsAvailable checks if Docker is installed and running.
	// Returns true if docker is accessible and the daemon is responding.
	IsAvailable() bool

	// Pull downloads an image if not present locally.
	// Returns an error if the image cannot be found or downloaded.
	Pull(image string) error

	// Run creates and starts a container with the given configuration.
	// Returns the container ID on success.
	Run(config ContainerConfig) (string, error)

	// Stop stops a running container with the specified timeout.
	// The timeout is in seconds - Docker will wait this long before forcefully killing.
	Stop(containerID string, timeoutSeconds int) error

	// Remove removes a container.
	// The container must be stopped first.
	Remove(containerID string) error

	// Logs returns a reader for the container's stdout/stderr stream.
	// The caller is responsible for closing the returned ReadCloser.
	Logs(containerID string) (io.ReadCloser, error)

	// Inspect returns detailed information about a container.
	// Returns nil and an error if the container is not found.
	Inspect(containerID string) (*Container, error)

	// IsRunning checks if a container is currently running.
	// Returns false if the container doesn't exist or is not running.
	IsRunning(containerID string) bool

	// InspectByName finds and inspects a container by name.
	// Returns the container if found (running or not), nil if not found.
	InspectByName(containerName string) (*Container, error)
}
