package service

import (
	"testing"
)

func TestGetServiceProjectDir(t *testing.T) {
	tests := []struct {
		name       string
		service    Service
		workingDir string
		expected   string
	}{
		{
			name: "service with project specified",
			service: Service{
				Project: "/path/to/project",
			},
			workingDir: "/working/dir",
			expected:   "/path/to/project",
		},
		{
			name: "service without project uses working dir",
			service: Service{
				Project: "",
			},
			workingDir: "/working/dir",
			expected:   "/working/dir",
		},
		{
			name:       "service with empty project uses working dir",
			service:    Service{},
			workingDir: "/another/working/dir",
			expected:   "/another/working/dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetServiceProjectDir(tt.service, tt.workingDir)
			if result != tt.expected {
				t.Errorf("GetServiceProjectDir() = %v, want %v", result, tt.expected)
			}
		})
	}
}
