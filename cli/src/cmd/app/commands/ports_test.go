package commands

import (
	"encoding/json"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestCollectPortReportBasic(t *testing.T) {
	y := &service.AzureYaml{Services: map[string]service.Service{
		"web":    {Ports: []string{"3000:8080"}},
		"worker": {Docker: &service.DockerConfig{}, Ports: []string{"8080"}}, // host auto-assigned
		"db":     {},                                                         // no ports
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
	if c.Port != 3000 {
		t.Errorf("expected conflict on port 3000, got %d", c.Port)
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
		"a": {Docker: &service.DockerConfig{}, Ports: []string{"8080"}},
		"b": {Docker: &service.DockerConfig{}, Ports: []string{"8080"}},
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

	if len(r.conflicts) != 1 || r.conflicts[0].Port != 3000 {
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

func TestPortReportJSONShape(t *testing.T) {
	y := &service.AzureYaml{Services: map[string]service.Service{
		"web": {Ports: []string{"3000:8080"}},
		"api": {Ports: []string{"3000:9090"}},
	}}

	r := collectPortReport(y)

	data, err := json.Marshal(r.services)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]struct {
		Ports []struct {
			Host      string `json:"host"`
			HostPort  int    `json:"hostPort"`
			Container int    `json:"container"`
			Protocol  string `json:"protocol"`
			Conflict  bool   `json:"conflict"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	web, ok := decoded["web"]
	if !ok || len(web.Ports) != 1 {
		t.Fatalf("expected web with one port, got %+v", decoded)
	}
	if web.Ports[0].Host != "3000" || !web.Ports[0].Conflict {
		t.Errorf("expected web port 3000 flagged as conflict, got %+v", web.Ports[0])
	}
}

func TestConflictSummary(t *testing.T) {
	got := conflictSummary([]portConflict{
		{Port: 3000, Owners: []string{"web", "api"}},
		{Port: 8080, Owners: []string{"svc"}},
	})
	want := "3000 (web, api); 8080 (svc)"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
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
