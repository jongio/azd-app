package commands

import (
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// doctorCheckRef identifies a check by service, id, and severity for assertions.
type doctorCheckRef struct {
	service  string
	checkID  string
	severity string
}

func TestDoctorPortChecksTable(t *testing.T) {
	tests := []struct {
		name     string
		services map[string]service.Service
		want     []doctorCheckRef
		notWant  []doctorCheckRef
	}{
		{
			name: "container single port never conflicts",
			services: map[string]service.Service{
				"api": {Image: "nginx:latest", Ports: []string{"8080"}},
				"web": {Image: "nginx:latest", Ports: []string{"8080"}},
			},
			want: []doctorCheckRef{
				{"api", "port.valid", doctorPass},
				{"web", "port.valid", doctorPass},
			},
			notWant: []doctorCheckRef{{"web", "port.unique", doctorFail}},
		},
		{
			name: "non-container single port is a host port",
			services: map[string]service.Service{
				"api": {Language: "go", Ports: []string{"3000"}},
				"web": {Language: "go", Ports: []string{"3000"}},
			},
			want: []doctorCheckRef{
				{"api", "port.valid", doctorPass},
				{"web", "port.unique", doctorFail},
			},
		},
		{
			name: "different protocols do not conflict",
			services: map[string]service.Service{
				"api": {Language: "go", Ports: []string{"3000/tcp"}},
				"web": {Language: "go", Ports: []string{"3000/udp"}},
			},
			want: []doctorCheckRef{
				{"api", "port.valid", doctorPass},
				{"web", "port.valid", doctorPass},
			},
			notWant: []doctorCheckRef{{"web", "port.unique", doctorFail}},
		},
		{
			name: "same protocol conflicts",
			services: map[string]service.Service{
				"api": {Language: "go", Ports: []string{"3000/udp"}},
				"web": {Language: "go", Ports: []string{"3000/udp"}},
			},
			want: []doctorCheckRef{{"web", "port.unique", doctorFail}},
		},
		{
			name: "explicit mapping conflict is detected",
			services: map[string]service.Service{
				"api": {Image: "nginx:latest", Ports: []string{"8080:80"}},
				"web": {Image: "nginx:latest", Ports: []string{"8080:9090"}},
			},
			want: []doctorCheckRef{
				{"api", "port.valid", doctorPass},
				{"web", "port.unique", doctorFail},
			},
		},
		{
			name: "out of range host port fails",
			services: map[string]service.Service{
				"admin": {Language: "go", Ports: []string{"70000:80"}},
			},
			want: []doctorCheckRef{{"admin", "port.valid", doctorFail}},
		},
		{
			name: "unparseable port fails",
			services: map[string]service.Service{
				"admin": {Language: "go", Ports: []string{"not-a-port"}},
			},
			want: []doctorCheckRef{{"admin", "port.valid", doctorFail}},
		},
		{
			name: "zero host port on a non-container service fails",
			services: map[string]service.Service{
				"admin": {Language: "go", Ports: []string{"0"}},
			},
			want: []doctorCheckRef{{"admin", "port.valid", doctorFail}},
		},
		{
			name: "negative host port fails",
			services: map[string]service.Service{
				"admin": {Language: "go", Ports: []string{"-1:80"}},
			},
			want: []doctorCheckRef{{"admin", "port.valid", doctorFail}},
		},
		{
			name: "ipv6 bind form is parsed",
			services: map[string]service.Service{
				"api": {Image: "nginx:latest", Ports: []string{"[::1]:3000:8080"}},
			},
			want: []doctorCheckRef{{"api", "port.valid", doctorPass}},
		},
		{
			name: "bind ip mapping conflicts on the same host port",
			services: map[string]service.Service{
				"api": {Image: "nginx:latest", Ports: []string{"127.0.0.1:3000:8080"}},
				"web": {Image: "nginx:latest", Ports: []string{"3000:8080"}},
			},
			want: []doctorCheckRef{{"web", "port.unique", doctorFail}},
		},
		{
			name: "container service running a local command uses host port semantics",
			services: map[string]service.Service{
				"api": {Docker: &service.DockerConfig{Image: "nginx:latest"}, Command: "npm run dev", Ports: []string{"3000"}},
				"web": {Language: "go", Ports: []string{"3000"}},
			},
			want: []doctorCheckRef{{"web", "port.unique", doctorFail}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checks := doctorPortChecks(&service.AzureYaml{Services: tt.services})
			for _, want := range tt.want {
				assertDoctorCheck(t, checks, want.service, want.checkID, want.severity)
			}
			for _, notWant := range tt.notWant {
				assertNoDoctorCheck(t, checks, notWant.service, notWant.checkID, notWant.severity)
			}
		})
	}
}

func TestDoctorPortChecksWarnsWhenNoPortsDeclared(t *testing.T) {
	checks := doctorPortChecks(&service.AzureYaml{Services: map[string]service.Service{
		"api": {Language: "go"},
	}})
	assertDoctorCheck(t, checks, "", "port.declared", doctorWarn)
}

func assertNoDoctorCheck(t *testing.T, checks []doctorCheck, serviceName, checkID, severity string) {
	t.Helper()
	for _, check := range checks {
		if check.Service == serviceName && check.CheckID == checkID && check.Severity == severity {
			t.Fatalf("unexpected doctor check %s/%s/%s in %#v", serviceName, checkID, severity, checks)
		}
	}
}
