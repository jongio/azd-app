package executor

import "testing"

func TestRejectLeadingDash(t *testing.T) {
	tests := []struct {
		name      string
		arg       string
		wantError bool
	}{
		// --- Valid inputs: must pass ---
		{name: "empty string", arg: "", wantError: false},
		{name: "simple filename", arg: "main.py", wantError: false},
		{name: "relative path", arg: "src/main.py", wantError: false},
		{name: "dotslash relative", arg: "./src/main.py", wantError: false},
		{name: "absolute path unix", arg: "/home/user/app/main.py", wantError: false},
		{name: "csproj filename", arg: "MyProject.csproj", wantError: false},
		{name: "csproj relative", arg: "src/MyProject.csproj", wantError: false},
		{name: "nested relative", arg: "src/agent/agent.py", wantError: false},
		{name: "windows absolute", arg: `C:\code\app\main.py`, wantError: false},
		{name: "underscore prefix", arg: "__main__.py", wantError: false},

		// --- Injection attempts: must be rejected ---
		{name: "single dash c", arg: "-c", wantError: true},
		{name: "double dash flag", arg: "--flag", wantError: true},
		{name: "double dash project", arg: "--project", wantError: true},
		{name: "single dash alone", arg: "-", wantError: true},
		{name: "dash verbosity", arg: "--verbosity:q", wantError: true},
		{name: "dash exec", arg: "-exec", wantError: true},
		{name: "double dash empty", arg: "--", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RejectLeadingDash(tt.arg)
			if tt.wantError && err == nil {
				t.Errorf("RejectLeadingDash(%q) = nil, want error", tt.arg)
			}
			if !tt.wantError && err != nil {
				t.Errorf("RejectLeadingDash(%q) = %v, want nil", tt.arg, err)
			}
		})
	}
}
