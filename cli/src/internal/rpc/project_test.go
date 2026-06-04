package rpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// stubProjectSource lets tests dictate the GetProject outcome without
// touching the filesystem. The handler must remain agnostic to where
// ParseAzureYaml comes from, so the test boundary is the
// ProjectSource interface, not service.ParseAzureYaml itself.
type stubProjectSource struct {
	yaml *service.AzureYaml
	err  error

	gotWorkingDir string
	calls         int
}

func (s *stubProjectSource) ParseAzureYaml(workingDir string) (*service.AzureYaml, error) {
	s.calls++
	s.gotWorkingDir = workingDir
	return s.yaml, s.err
}

// newProjectTestServer wires a ProjectHandler behind an httptest server
// and returns the configured stub plus a Connect client. Tests mutate
// the stub to drive each scenario.
func newProjectTestServer(t *testing.T, projectDir string, source *stubProjectSource) (azdappv1connect.ProjectServiceClient, func()) {
	t.Helper()
	mgr := broadcast.New()

	mux := http.NewServeMux()
	if err := Mount(mux, Dependencies{
		Broadcast:            mgr,
		Project:              source,
		ProjectDir:           projectDir,
		AllowUnauthenticated: true,
	}); err != nil {
		t.Fatalf("Mount: %v", err)
	}

	srv := httptest.NewServer(mux)
	client := azdappv1connect.NewProjectServiceClient(srv.Client(), srv.URL)
	return client, func() {
		srv.Close()
		mgr.StopAll()
	}
}

func TestGetProjectReturnsNameAndDir(t *testing.T) {
	source := &stubProjectSource{
		yaml: &service.AzureYaml{Name: "fullstack-demo"},
	}
	client, cleanup := newProjectTestServer(t, "/abs/projects/fullstack", source)
	defer cleanup()

	resp, err := client.GetProject(context.Background(), connect.NewRequest(&v1.GetProjectRequest{}))
	if err != nil {
		t.Fatalf("GetProject error: %v", err)
	}
	if got := resp.Msg.GetName(); got != "fullstack-demo" {
		t.Errorf("Name=%q want fullstack-demo", got)
	}
	if got := resp.Msg.GetDir(); got != "/abs/projects/fullstack" {
		t.Errorf("Dir=%q want /abs/projects/fullstack", got)
	}
	if source.calls != 1 {
		t.Errorf("ParseAzureYaml call count=%d want 1", source.calls)
	}
	if source.gotWorkingDir != "/abs/projects/fullstack" {
		t.Errorf("ParseAzureYaml workingDir=%q want /abs/projects/fullstack", source.gotWorkingDir)
	}
}

func TestGetProjectReturnsInternalOnParseFailure(t *testing.T) {
	source := &stubProjectSource{err: errors.New("azure.yaml: malformed YAML")}
	client, cleanup := newProjectTestServer(t, "/abs/projects/broken", source)
	defer cleanup()

	_, err := client.GetProject(context.Background(), connect.NewRequest(&v1.GetProjectRequest{}))
	if err == nil {
		t.Fatal("expected error from GetProject; got nil")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Errorf("error code=%v want Internal; full err=%v", got, err)
	}
}

func TestGetProjectAcceptsEmptyProjectName(t *testing.T) {
	// azure.yaml without a `name:` key is valid YAML but yields the
	// proto zero value. Verify we propagate it instead of synthesising
	// a placeholder.
	source := &stubProjectSource{yaml: &service.AzureYaml{}}
	client, cleanup := newProjectTestServer(t, "/abs/projects/anon", source)
	defer cleanup()

	resp, err := client.GetProject(context.Background(), connect.NewRequest(&v1.GetProjectRequest{}))
	if err != nil {
		t.Fatalf("GetProject error: %v", err)
	}
	if got := resp.Msg.GetName(); got != "" {
		t.Errorf("Name=%q want empty string", got)
	}
	if got := resp.Msg.GetDir(); got != "/abs/projects/anon" {
		t.Errorf("Dir=%q want /abs/projects/anon", got)
	}
}

func TestNewProjectHandlerPanicsOnNilSource(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when source is nil")
		}
	}()
	_ = NewProjectHandler(nil, "/abs/projects/x")
}

func TestNewProjectHandlerPanicsOnEmptyProjectDir(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when projectDir is empty")
		}
	}()
	_ = NewProjectHandler(ProjectSourceFunc(func(string) (*service.AzureYaml, error) {
		return &service.AzureYaml{}, nil
	}), "")
}

func TestProjectServiceNotMountedWithoutSource(t *testing.T) {
	// Mount with no Project source: GetProject path must 404 so callers
	// can detect "feature off" without ambiguous Connect errors.
	mgr := broadcast.New()
	defer mgr.StopAll()

	mux := http.NewServeMux()
	if err := Mount(mux, Dependencies{Broadcast: mgr, AllowUnauthenticated: true}); err != nil {
		t.Fatalf("Mount: %v", err)
	}

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + azdappv1connect.ProjectServiceGetProjectProcedure)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status=%d want 404 when ProjectService is not mounted", resp.StatusCode)
	}
}
