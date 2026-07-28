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

	// ImageExists reports whether an image is present in the local Docker cache.
	ImageExists(image string) bool

	// NetworkExists reports whether a user-defined Docker network exists.
	NetworkExists(name string) (bool, error)

	// EnsureNetwork creates a user-defined bridge network if absent. It is
	// idempotent and safe to call concurrently.
	EnsureNetwork(name string) error

	// RemoveNetwork removes a user-defined Docker network. A missing network is
	// not an error.
	RemoveNetwork(name string) error

	// ConnectNetwork attaches an existing container to a network with optional
	// DNS aliases. It is idempotent if the container is already connected.
	ConnectNetwork(network, container string, aliases []string) error

	// Run creates and starts a container with the given configuration.
	// Returns the container ID on success.
	Run(config ContainerConfig) (string, error)

	// Stop stops a running container with the specified timeout.
	// The timeout is in seconds - Docker will wait this long before forcefully killing.
	Stop(containerID string, timeoutSeconds int) error

	// Start starts an existing stopped container.
	// Returns an error if the container doesn't exist or cannot be started.
	Start(containerID string) error

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

	// Exec runs a command inside a running container.
	// Returns the exit code, command output, and any error.
	Exec(containerName string, command []string) (int, string, error)

	// ExecShell runs a shell command inside a running container using sh -c.
	// Returns the exit code, command output, and any error.
	ExecShell(containerName string, shellCommand string) (int, string, error)
}
