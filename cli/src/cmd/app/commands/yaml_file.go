package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func marshalYAMLNode(node *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(node); err != nil {
		_ = enc.Close()
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeFileAtomic(path string, data []byte) error {
	cleanPath := filepath.Clean(path)
	mode := os.FileMode(0o644)
	if info, err := os.Stat(cleanPath); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat %s: %w", cleanPath, err)
	}

	tmpPath := cleanPath + ".tmp"
	// #nosec G306 -- azure.yaml needs to remain readable when written.
	if err := os.WriteFile(tmpPath, data, mode); err != nil {
		return fmt.Errorf("failed to write temporary file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, cleanPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize %s: %w", cleanPath, err)
	}
	return nil
}
