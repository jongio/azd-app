package detector

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkFindNodeProjects benchmarks the Node.js project detection.
func BenchmarkFindNodeProjects(b *testing.B) {
	// Create a temporary directory structure for testing
	tmpDir := b.TempDir()

	// Create a simple project structure
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		b.Fatal(err)
	}

	packageJSON := filepath.Join(projectDir, "package.json")
	if err := os.WriteFile(packageJSON, []byte(`{"name":"test"}`), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FindNodeProjects(tmpDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFindPythonProjects benchmarks the Python project detection.
func BenchmarkFindPythonProjects(b *testing.B) {
	tmpDir := b.TempDir()

	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		b.Fatal(err)
	}

	reqFile := filepath.Join(projectDir, "requirements.txt")
	if err := os.WriteFile(reqFile, []byte("flask==2.0.0"), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FindPythonProjects(tmpDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFindDotnetProjects benchmarks the .NET project detection.
func BenchmarkFindDotnetProjects(b *testing.B) {
	tmpDir := b.TempDir()

	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		b.Fatal(err)
	}

	csproj := filepath.Join(projectDir, "test.csproj")
	if err := os.WriteFile(csproj, []byte(`<Project Sdk="Microsoft.NET.Sdk"></Project>`), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FindDotnetProjects(tmpDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDetectNodePackageManager benchmarks package manager detection.
func BenchmarkDetectNodePackageManager(b *testing.B) {
	tmpDir := b.TempDir()

	packageJSON := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(packageJSON, []byte(`{"packageManager":"pnpm@8.15.0"}`), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DetectNodePackageManager(tmpDir)
	}
}

// BenchmarkFindAzureYaml benchmarks azure.yaml search.
func BenchmarkFindAzureYaml(b *testing.B) {
	tmpDir := b.TempDir()

	azureYaml := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYaml, []byte("name: test"), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FindAzureYaml(tmpDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}
