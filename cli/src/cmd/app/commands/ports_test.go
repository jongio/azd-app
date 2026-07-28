package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"
)

func TestCollectPortReportBasic(t *testing.T) {
	y := &service.AzureYaml{Services: map[string]service.Service{
		"web":    {Ports: []string{"3000:8080"}},
		"worker": {Docker: &service.DockerConfig{Image: "nginx"}, Ports: []string{"8080"}}, // host auto-assigned
		"db":     {},                                                                       // no ports
	}}

	r := collectPortReport(y)

	if len(r.order) != 3 || r.order[0] != "db" || r.order[1] != "web" || r.order[2] != "worker" {
		t.Fatalf("expected sorted order [db web worker], got %v", r.order)
	}
	if len(r.conflicts) != 0 {
		t.Errorf("expected no conflicts, got %v", r.conflicts)
	}

	web := r.services["web"].Ports
	if len(web) != 1 {
		t.Fatalf("expected 1 web binding, got %d", len(web))
	}
	if web[0].Host != "3000" || web[0].HostPort != 3000 || web[0].Container != 8080 || web[0].Protocol != "tcp" || web[0].Conflict {
		t.Errorf("unexpected web binding: %+v", web[0])
	}

	worker := r.services["worker"].Ports
	if len(worker) != 1 || worker[0].Host != "auto" || worker[0].HostPort != 0 {
		t.Errorf("expected worker host auto, got %+v", worker)
	}

	if len(r.services["db"].Ports) != 0 {
		t.Errorf("expected db to have no ports, got %+v", r.services["db"].Ports)
	}
}

func TestCollectPortReportConflict(t *testing.T) {
	y := &service.AzureYaml{Services: map[string]service.Service{
		"web": {Ports: []string{"3000:8080"}},
		"api": {Ports: []string{"3000:9090"}},
	}}

	r := collectPortReport(y)

	if len(r.conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %v", r.conflicts)
	}
	c := r.conflicts[0]
	if c.HostPort != 3000 || c.Protocol != "tcp" {
		t.Errorf("expected conflict on 3000/tcp, got %+v", c)
	}
	if len(c.Owners) != 2 {
		t.Errorf("expected 2 owners, got %v", c.Owners)
	}
	if !r.services["web"].Ports[0].Conflict || !r.services["api"].Ports[0].Conflict {
		t.Error("expected both conflicting bindings to be marked")
	}
}

func TestCollectPortReportAutoPortsNeverConflict(t *testing.T) {
	y := &service.AzureYaml{Services: map[string]service.Service{
		"a": {Docker: &service.DockerConfig{Image: "nginx"}, Ports: []string{"8080"}},
		"b": {Docker: &service.DockerConfig{Image: "nginx"}, Ports: []string{"8080"}},
	}}

	r := collectPortReport(y)

	if len(r.conflicts) != 0 {
		t.Errorf("auto-assigned ports must not conflict, got %v", r.conflicts)
	}
	if r.services["a"].Ports[0].Conflict || r.services["b"].Ports[0].Conflict {
		t.Error("auto bindings should not be flagged as conflicts")
	}
}

func TestCollectPortReportSameServiceDuplicate(t *testing.T) {
	y := &service.AzureYaml{Services: map[string]service.Service{
		"web": {Ports: []string{"3000:8080", "3000:9090"}},
	}}

	r := collectPortReport(y)

	if len(r.conflicts) != 1 || r.conflicts[0].HostPort != 3000 {
		t.Fatalf("expected a conflict on 3000 within one service, got %v", r.conflicts)
	}
	if len(r.conflicts[0].Owners) != 1 || r.conflicts[0].Owners[0] != "web" {
		t.Errorf("expected deduped owner [web], got %v", r.conflicts[0].Owners)
	}
	for _, b := range r.services["web"].Ports {
		if !b.Conflict {
			t.Errorf("expected both web bindings flagged, got %+v", b)
		}
	}
}

func TestCollectPortReportUDP(t *testing.T) {
	y := &service.AzureYaml{Services: map[string]service.Service{
		"dns": {Ports: []string{"5300:53/udp"}},
	}}

	r := collectPortReport(y)

	b := r.services["dns"].Ports[0]
	if b.Protocol != "udp" || b.HostPort != 5300 || b.Container != 53 {
		t.Errorf("unexpected udp binding: %+v", b)
	}
}

func TestCollectPortReportConflictIdentityMatrix(t *testing.T) {
	tests := []struct {
		name          string
		ports         map[string][]string
		wantConflicts int
		wantMarked    []string
	}{
		{
			name: "same IP port and protocol conflicts",
			ports: map[string][]string{
				"api": {"127.0.0.1:3000:8080/tcp"},
				"web": {"127.0.0.1:3000:9090/tcp"},
			},
			wantConflicts: 1,
			wantMarked:    []string{"api", "web"},
		},
		{
			name: "different specific IPs do not conflict",
			ports: map[string][]string{
				"api": {"127.0.0.1:3000:8080/tcp"},
				"web": {"192.168.1.5:3000:9090/tcp"},
			},
			wantConflicts: 0,
		},
		{
			name: "IPv4 wildcard conflicts with specific IPv4",
			ports: map[string][]string{
				"api": {"0.0.0.0:3000:8080/tcp"},
				"web": {"127.0.0.1:3000:9090/tcp"},
			},
			wantConflicts: 1,
			wantMarked:    []string{"api", "web"},
		},
		{
			name: "empty wildcard conflicts with any family",
			ports: map[string][]string{
				"api": {"3000:8080/tcp"},
				"web": {"[::1]:3000:9090/tcp"},
			},
			wantConflicts: 1,
			wantMarked:    []string{"api", "web"},
		},
		{
			name: "IPv4 wildcard does not conflict with specific IPv6",
			ports: map[string][]string{
				"api": {"0.0.0.0:3000:8080/tcp"},
				"web": {"[::1]:3000:9090/tcp"},
			},
			wantConflicts: 0,
		},
		{
			name: "IPv6 wildcard conflicts with specific IPv6",
			ports: map[string][]string{
				"api": {"[::]:3000:8080/tcp"},
				"web": {"[::1]:3000:9090/tcp"},
			},
			wantConflicts: 1,
			wantMarked:    []string{"api", "web"},
		},
		{
			name: "same IP and port with different protocol does not conflict",
			ports: map[string][]string{
				"api": {"127.0.0.1:3000:8080/tcp"},
				"web": {"127.0.0.1:3000:9090/udp"},
			},
			wantConflicts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			services := make(map[string]service.Service, len(tt.ports))
			for name, ports := range tt.ports {
				services[name] = service.Service{Ports: ports}
			}

			r := collectPortReport(&service.AzureYaml{Services: services})
			if len(r.conflicts) != tt.wantConflicts {
				t.Fatalf("conflicts = %v, want %d", r.conflicts, tt.wantConflicts)
			}

			for _, name := range tt.wantMarked {
				if !r.services[name].Ports[0].Conflict {
					t.Fatalf("expected %s binding to be marked as conflicting", name)
				}
			}
		})
	}
}

func TestPortReportTextIncludesBindIP(t *testing.T) {
	r := portReport{
		services: map[string]servicePorts{
			"api": {
				Ports: []portBinding{
					{Host: "8080", HostPort: 8080, BindIP: "127.0.0.1", Container: 80, Protocol: "tcp"},
					{Host: "9090", HostPort: 9090, Container: 90, Protocol: "tcp"},
				},
			},
		},
		order: []string{"api"},
	}

	if err := cliout.SetFormat("default"); err != nil {
		t.Fatalf("SetFormat(default): %v", err)
	}
	t.Cleanup(func() { _ = cliout.SetFormat("default") })

	out, err := captureStdout(t, func() error {
		printPortReport(r)
		return nil
	})
	if err != nil {
		t.Fatalf("printPortReport: %v", err)
	}
	if !strings.Contains(out, "127.0.0.1:8080 -> 80/tcp") {
		t.Fatalf("expected bind IP in text output, got %q", out)
	}
	if !strings.Contains(out, "9090 -> 90/tcp") {
		t.Fatalf("expected empty bind IP to keep current text output, got %q", out)
	}
	if strings.Contains(out, ":9090 -> 90/tcp") {
		t.Fatalf("empty bind IP should not render a prefix, got %q", out)
	}
}

func TestPortReportJSONShape(t *testing.T) {
	y := &service.AzureYaml{Services: map[string]service.Service{
		"web": {Ports: []string{"127.0.0.1:3000:8080"}},
		"api": {Ports: []string{"127.0.0.1:3000:9090"}},
		"db":  {Ports: []string{"5432:5432"}},
	}}

	r := collectPortReport(y)

	data, err := json.Marshal(newPortJSONReport(r))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded struct {
		Services map[string]struct {
			Ports []struct {
				Host      string `json:"host"`
				HostPort  int    `json:"hostPort"`
				BindIP    string `json:"bindIP"`
				Container int    `json:"container"`
				Protocol  string `json:"protocol"`
				Conflict  bool   `json:"conflict"`
			} `json:"ports"`
		} `json:"services"`
		Conflicts []portConflict `json:"conflicts"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	web, ok := decoded.Services["web"]
	if !ok || len(web.Ports) != 1 {
		t.Fatalf("expected web with one port, got %+v", decoded)
	}
	if web.Ports[0].Host != "3000" || web.Ports[0].BindIP != "127.0.0.1" || !web.Ports[0].Conflict {
		t.Errorf("expected web port 3000 flagged as conflict, got %+v", web.Ports[0])
	}
	if _, ok := decoded.Services["db"]; !ok {
		t.Fatalf("expected db service, got %+v", decoded.Services)
	}
	if strings.Contains(string(data), `"bindIP":""`) {
		t.Fatalf("empty bindIP should be omitted, got %s", data)
	}
	if len(decoded.Conflicts) != 1 {
		t.Fatalf("expected top-level conflicts array with one entry, got %+v", decoded.Conflicts)
	}

	noConflict := collectPortReport(&service.AzureYaml{Services: map[string]service.Service{
		"api": {Ports: []string{"127.0.0.1:3000:8080"}},
		"web": {Ports: []string{"192.168.1.5:3000:9090"}},
	}})
	noConflictData, err := json.Marshal(newPortJSONReport(noConflict))
	if err != nil {
		t.Fatalf("marshal no-conflict report failed: %v", err)
	}
	if !strings.Contains(string(noConflictData), `"conflicts":[]`) {
		t.Fatalf("expected empty top-level conflicts array, got %s", noConflictData)
	}
}

func TestRunPortsJSONConflictOutputIsValidJSON(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writePortsAzureYaml(t, dir, `name: ports-test
services:
  api:
    project: ./api
    ports:
      - "127.0.0.1:3000:8080"
  web:
    project: ./web
    ports:
      - "127.0.0.1:3000:9090"
`)

	originalFormat := cliout.GetFormat()
	t.Cleanup(func() { _ = cliout.SetFormat(string(originalFormat)) })
	if err := cliout.SetFormat("json"); err != nil {
		t.Fatalf("SetFormat(json): %v", err)
	}

	cmd := NewPortsCommand()
	out, err := captureStdout(t, func() error { return runPorts(cmd, nil) })
	if err == nil {
		t.Fatal("expected conflict error")
	}

	var decoded portJSONReport
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("stdout must be valid JSON, got %q: %v", out, err)
	}
	if len(decoded.Conflicts) != 1 {
		t.Fatalf("expected one top-level conflict, got %+v", decoded.Conflicts)
	}
	if !cmd.SilenceErrors {
		t.Fatal("JSON conflict path should silence redundant cobra error rendering")
	}
}

func TestConflictSummary(t *testing.T) {
	got := conflictSummary([]portConflict{
		{HostPort: 3000, Protocol: "tcp", Owners: []string{"web", "api"}},
		{BindIP: "127.0.0.1", HostPort: 8080, Protocol: "udp", Owners: []string{"svc"}},
	})
	want := "3000/tcp (web, api); 127.0.0.1:8080/udp (svc)"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func writePortsAzureYaml(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(content), 0o600); err != nil {
		t.Fatalf("write azure.yaml: %v", err)
	}
}

func TestProtocolOrDefault(t *testing.T) {
	if protocolOrDefault("") != "tcp" {
		t.Error("empty protocol should default to tcp")
	}
	if protocolOrDefault("udp") != "udp" {
		t.Error("explicit protocol should be preserved")
	}
}

func TestDedupeStrings(t *testing.T) {
	got := dedupeStrings([]string{"a", "b", "a", "c", "b"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}
