// Package executor provides safe command execution with output monitoring and timeout handling.
// This file contains the argv injection guard (CWE-88).
package executor

import (
	"fmt"
	"strings"
)

// RejectLeadingDash guards exec.Command argv against leading-dash injection (CWE-88).
//
// User-controlled values from azure.yaml (entrypoint, project path, etc.) that begin
// with '-' would be parsed as CLI flags by the subprocess rather than treated as data.
// For example, an entrypoint of "-c" would be interpreted as the Python -c flag and
// could execute arbitrary code supplied by a further azure.yaml field.
//
// Legitimate paths and entry points derived from azure.yaml never start with '-':
// they are relative paths (src/main.py), absolute paths (/abs/path), or named scripts.
// Any value starting with '-' is therefore structurally invalid and is rejected.
func RejectLeadingDash(arg string) error {
	if strings.HasPrefix(arg, "-") {
		return fmt.Errorf("argument %q is invalid: values from azure.yaml must not start with '-' (CWE-88 argv injection guard)", arg)
	}
	return nil
}
