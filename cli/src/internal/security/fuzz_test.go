package security

import "testing"

// FuzzValidatePath tests ValidatePath with random inputs to ensure it never panics.
func FuzzValidatePath(f *testing.F) {
	// Seed with known test cases
	f.Add("")
	f.Add("/valid/path")
	f.Add("../../../etc/passwd")
	f.Add("../../..")
	f.Add("./normal/path")
	f.Add("C:\\Windows\\System32")
	f.Add("/tmp/../etc/passwd")
	f.Add("path/with/../../traversal")

	f.Fuzz(func(t *testing.T, path string) {
		// Function should never panic
		_ = ValidatePath(path)
	})
}

// FuzzSanitizeScriptName tests SanitizeScriptName with random inputs.
func FuzzSanitizeScriptName(f *testing.F) {
	// Seed with known test cases
	f.Add("valid-script")
	f.Add("dev")
	f.Add("start")
	f.Add("build:prod")
	f.Add("rm -rf /")
	f.Add("test; echo 'pwned'")
	f.Add("$(whoami)")
	f.Add("`cat /etc/passwd`")
	f.Add("script|grep secret")

	f.Fuzz(func(t *testing.T, name string) {
		// Function should never panic
		_ = SanitizeScriptName(name)
	})
}

// FuzzValidatePackageManager tests ValidatePackageManager with random inputs.
func FuzzValidatePackageManager(f *testing.F) {
	// Seed with known test cases
	f.Add("npm")
	f.Add("pnpm")
	f.Add("yarn")
	f.Add("pip")
	f.Add("poetry")
	f.Add("uv")
	f.Add("dotnet")
	f.Add("invalid")
	f.Add("rm")
	f.Add("")
	f.Add("npm && rm -rf /")

	f.Fuzz(func(t *testing.T, pm string) {
		// Function should never panic
		_ = ValidatePackageManager(pm)
	})
}
