package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestDoctorServicePathChecks(t *testing.T) {
	dir := t.TempDir()
	mkdirDoctorService(t, dir, "api")
	cfg := &service.AzureYaml{Services: map[string]service.Service{
		"api": {Project: "./api"},
		"web": {Project: "./missing"},
	}}
	checks := doctorServicePathChecks(dir, cfg)
	assertDoctorCheck(t, checks, "api", "service.project", doctorPass)
	assertDoctorCheck(t, checks, "web", "service.project", doctorFail)
}

func TestDoctorPortChecks(t *testing.T) {
	cfg := &service.AzureYaml{Services: map[string]service.Service{
		"api":   {Ports: []string{"8080:80"}},
		"web":   {Ports: []string{"8080:80"}},
		"admin": {Ports: []string{"70000:80"}},
	}}
	checks := doctorPortChecks(cfg)
	assertDoctorCheck(t, checks, "api", "port.valid", doctorPass)
	assertDoctorCheck(t, checks, "web", "port.unique", doctorFail)
	assertDoctorCheck(t, checks, "admin", "port.valid", doctorFail)
}

func TestDoctorDetectTools(t *testing.T) {
	dir := t.TempDir()
	mkdirDoctorService(t, dir, "web")
	if err := os.WriteFile(filepath.Join(dir, "web", "pnpm-lock.yaml"), []byte("lock"), 0o600); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	if got := doctorDetectNodePackageManager(filepath.Join(dir, "web")); got != "pnpm" {
		t.Fatalf("package manager = %q, want pnpm", got)
	}
}

func assertDoctorCheck(t *testing.T, checks []doctorCheck, serviceName, checkID, severity string) {
	t.Helper()
	for _, check := range checks {
		if check.Service == serviceName && check.CheckID == checkID && check.Severity == severity {
			return
		}
	}
	t.Fatalf("missing doctor check %s/%s/%s in %#v", serviceName, checkID, severity, checks)
}

func mkdirDoctorService(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, name), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", name, err)
	}
}
