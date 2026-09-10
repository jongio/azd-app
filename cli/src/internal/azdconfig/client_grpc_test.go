package azdconfig

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"google.golang.org/grpc"
)

// stubUserConfigServer answers GetSection with a canned payload so the real
// Client can be driven end to end over gRPC. Only GetSection is implemented;
// the embedded Unimplemented base returns Unimplemented for everything else,
// which is the correct answer for a method this test does not exercise.
type stubUserConfigServer struct {
	azdext.UnimplementedUserConfigServiceServer

	section []byte
	found   bool
	gotPath string
}

func (s *stubUserConfigServer) GetSection(
	_ context.Context,
	req *azdext.GetUserConfigSectionRequest,
) (*azdext.GetUserConfigSectionResponse, error) {
	s.gotPath = req.Path
	return &azdext.GetUserConfigSectionResponse{Section: s.section, Found: s.found}, nil
}

// newTestClient starts an in-process gRPC server backed by stub and returns a
// Client wired to it. The server and connection are torn down with the test.
func newTestClient(t *testing.T, stub *stubUserConfigServer) *Client {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := grpc.NewServer()
	azdext.RegisterUserConfigServiceServer(server, stub)
	go func() {
		// Serve returns once the listener closes during cleanup.
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	azdClient, err := azdext.NewAzdClient(azdext.WithAddress(listener.Addr().String()))
	if err != nil {
		t.Fatalf("dial stub server: %v", err)
	}
	t.Cleanup(azdClient.Close)

	return &Client{azdClient: azdClient, ctx: context.Background()}
}

// TestClientGetAllServicePortsDecodesKeysAndValues covers the production
// adapter rather than InMemoryClient: the type switch over the decoded JSON
// values and the decodeSegment call applied to each key. A service named
// "foo.bar" is stored percent-encoded, so the caller must get the dotted name
// back.
func TestClientGetAllServicePortsDecodesKeysAndValues(t *testing.T) {
	// Numbers arrive as float64 through encoding/json; ports written by older
	// builds arrive as strings. Both must decode, and both keys must be
	// percent-decoded.
	section, err := json.Marshal(map[string]any{
		"foo%2Ebar":     3000,
		"api":           3001,
		"web%2Efront":   "3002",
		"ignored":       true,
		"not%2Danumber": "abc",
	})
	if err != nil {
		t.Fatalf("marshal section: %v", err)
	}

	stub := &stubUserConfigServer{section: section, found: true}
	client := newTestClient(t, stub)

	ports, err := client.GetAllServicePorts("hash123")
	if err != nil {
		t.Fatalf("GetAllServicePorts: %v", err)
	}

	want := map[string]int{
		"foo.bar":   3000,
		"api":       3001,
		"web.front": 3002,
	}
	if len(ports) != len(want) {
		t.Fatalf("ports = %#v, want %#v", ports, want)
	}
	for name, port := range want {
		if ports[name] != port {
			t.Errorf("ports[%q] = %d, want %d", name, ports[name], port)
		}
	}

	if stub.gotPath != projectConfigPath("hash123", "ports") {
		t.Errorf("requested path = %q, want %q", stub.gotPath, projectConfigPath("hash123", "ports"))
	}
}

func TestClientGetAllServicePortsWhenSectionMissing(t *testing.T) {
	client := newTestClient(t, &stubUserConfigServer{found: false})

	ports, err := client.GetAllServicePorts("hash123")
	if err != nil {
		t.Fatalf("GetAllServicePorts: %v", err)
	}
	if ports == nil {
		t.Fatal("expected an empty map rather than nil, so callers can index it")
	}
	if len(ports) != 0 {
		t.Fatalf("ports = %#v, want empty", ports)
	}
}

func TestClientGetAllServicePortsRejectsMalformedSection(t *testing.T) {
	client := newTestClient(t, &stubUserConfigServer{section: []byte("{not json"), found: true})

	if _, err := client.GetAllServicePorts("hash123"); err == nil {
		t.Fatal("expected an error for a section that is not valid JSON")
	}
}
