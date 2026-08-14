package rpc

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// ProjectSource is the narrow interface ProjectHandler needs to satisfy
// GetProject. The dashboard wires it to service.ParseAzureYaml; tests
// inject a stub. Returning the parsed AzureYaml (rather than just the
// project name) keeps the door open for future ProjectService RPCs that
// expose more of the manifest without forcing a re-parse.
type ProjectSource interface {
	ParseAzureYaml(workingDir string) (*service.AzureYaml, error)
}

// ProjectSourceFunc adapts a plain function to the ProjectSource
// interface, so production code can wire `service.ParseAzureYaml`
// directly without an extra wrapper struct.
type ProjectSourceFunc func(workingDir string) (*service.AzureYaml, error)

// ParseAzureYaml implements ProjectSource by delegating to the func.
func (f ProjectSourceFunc) ParseAzureYaml(workingDir string) (*service.AzureYaml, error) {
	return f(workingDir)
}

// ProjectHandler implements azdappv1connect.ProjectServiceHandler.
//
// Wire shape: GetProjectResponse mirrors the legacy /api/project JSON
// (`name`, `dir`). Adding fields here means a proto change first;
// keeping the response narrow protects clients from accidental schema
// drift through the migration window.
type ProjectHandler struct {
	source     ProjectSource
	projectDir string
}

// Compile-time interface conformance.
var _ azdappv1connect.ProjectServiceHandler = (*ProjectHandler)(nil)

// NewProjectHandler constructs a ProjectHandler. Both deps are required:
// a nil source means GetProject can't read the manifest, and an empty
// projectDir means ParseAzureYaml can't resolve a path. Both are
// programming errors at construction time, so we panic rather than
// surface NPEs on the first request.
func NewProjectHandler(source ProjectSource, projectDir string) *ProjectHandler {
	if source == nil {
		panic("rpc: NewProjectHandler called with nil ProjectSource")
	}
	if projectDir == "" {
		panic("rpc: NewProjectHandler called with empty projectDir")
	}
	return &ProjectHandler{source: source, projectDir: projectDir}
}

// GetProject returns the azure.yaml-derived project name plus the
// resolved working directory. Mirrors the legacy GET /api/project
// payload byte-for-byte (modulo proto3 JSON wire encoding).
func (h *ProjectHandler) GetProject(
	_ context.Context,
	_ *connect.Request[v1.GetProjectRequest],
) (*connect.Response[v1.GetProjectResponse], error) {
	azureYaml, err := h.source.ParseAzureYaml(h.projectDir)
	if err != nil {
		// Surface as Internal because parse failures here mean the
		// dashboard's project context is broken (missing azure.yaml,
		// malformed manifest, IO error). A more granular code (e.g.
		// FailedPrecondition for "no azure.yaml in this tree") would be
		// a behavior change vs. the legacy 500, defer that until a
		// concrete consumer needs it.
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.GetProjectResponse{
		Name: azureYaml.Name,
		Dir:  h.projectDir,
	}), nil
}
