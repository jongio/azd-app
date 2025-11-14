package installer

import "fmt"

// DependencyInstallError represents an error during dependency installation.
type DependencyInstallError struct {
	ProjectType    string
	ProjectDir     string
	PackageManager string
	Command        string
	Err            error
}

// Error implements the error interface.
func (e *DependencyInstallError) Error() string {
	return fmt.Sprintf("failed to install %s dependencies in %s using %s (command: %s): %v",
		e.ProjectType, e.ProjectDir, e.PackageManager, e.Command, e.Err)
}

// Unwrap returns the underlying error.
func (e *DependencyInstallError) Unwrap() error {
	return e.Err
}

// VirtualEnvError represents an error during virtual environment setup.
type VirtualEnvError struct {
	ProjectDir string
	Tool       string
	Err        error
}

// Error implements the error interface.
func (e *VirtualEnvError) Error() string {
	return fmt.Sprintf("failed to create virtual environment in %s using %s: %v",
		e.ProjectDir, e.Tool, e.Err)
}

// Unwrap returns the underlying error.
func (e *VirtualEnvError) Unwrap() error {
	return e.Err
}
