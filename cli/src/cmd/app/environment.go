package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/jongio/azd-core/env"
)

var environmentListRunner env.CommandRunner = &env.DefaultCommandRunner{}

type azdEnvironment struct {
	Name      string `json:"Name"`
	IsDefault bool   `json:"IsDefault"`
}

type missingEnvironmentError struct {
	name      string
	available []azdEnvironment
	cause     error
}

func (e *missingEnvironmentError) Error() string {
	var message strings.Builder
	fmt.Fprintf(&message, "environment %q does not exist.\n\nAvailable environments:", e.name)
	if len(e.available) == 0 {
		message.WriteString(" none")
	}
	for _, available := range e.available {
		fmt.Fprintf(&message, "\n  %s", available.Name)
		if available.IsDefault {
			message.WriteString(" (default)")
		}
	}
	fmt.Fprintf(&message, "\n\nCreate it with: azd env new %s\nRun without -e to use the default environment.", e.name)
	return message.String()
}

func (e *missingEnvironmentError) Unwrap() error {
	return e.cause
}

func loadSelectedEnvironment(ctx context.Context, name string) error {
	loadErr := env.LoadAzdEnvironment(ctx, name)
	if loadErr == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if ctx.Err() != nil || !errors.As(loadErr, &exitErr) {
		return loadErr
	}

	output, err := environmentListRunner.Run(ctx, "azd", "env", "list", "--output", "json")
	if err != nil {
		return fmt.Errorf("%w; unable to list available environments: %w", loadErr, err)
	}
	var available []azdEnvironment
	if err := json.Unmarshal(output, &available); err != nil {
		return fmt.Errorf("%w; unable to parse available environments: %w", loadErr, err)
	}
	if available == nil {
		return fmt.Errorf("%w; invalid environment list: expected an array", loadErr)
	}
	for _, candidate := range available {
		if candidate.Name == "" {
			return fmt.Errorf("%w; invalid environment list: an environment has no name", loadErr)
		}
		if candidate.Name == name {
			return loadErr
		}
	}
	sort.Slice(available, func(i, j int) bool { return available[i].Name < available[j].Name })
	return &missingEnvironmentError{name: name, available: available, cause: loadErr}
}
