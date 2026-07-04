package commands

import (
	"fmt"
	"strconv"
	"strings"
)

// parseScaleFlags parses repeated --scale entries in name=count format.
func parseScaleFlags(entries []string) (map[string]int, error) {
	if len(entries) == 0 {
		return map[string]int{}, nil
	}

	scale := make(map[string]int, len(entries))

	for _, rawEntry := range entries {
		entry := strings.TrimSpace(rawEntry)
		if entry == "" {
			return nil, fmt.Errorf("empty scale entry")
		}

		nameRaw, countRaw, found := strings.Cut(entry, "=")
		if !found {
			return nil, fmt.Errorf("expected name=count format, got %q", entry)
		}

		name := strings.TrimSpace(nameRaw)
		if name == "" {
			return nil, fmt.Errorf("service name is required in %q", entry)
		}

		if _, exists := scale[name]; exists {
			return nil, fmt.Errorf("duplicate scale entry for service %q", name)
		}

		countText := strings.TrimSpace(countRaw)
		count, err := strconv.Atoi(countText)
		if err != nil {
			return nil, fmt.Errorf("invalid instance count %q for service %q: %w", countText, name, err)
		}

		if count < 1 {
			return nil, fmt.Errorf("instance count for service %q must be at least 1", name)
		}

		scale[name] = count
	}

	return scale, nil
}
