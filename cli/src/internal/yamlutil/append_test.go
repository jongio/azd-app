package yamlutil

import (
	"fmt"
	"strings"
	"testing"
)

func TestAppendToArraySection(t *testing.T) {
	t.Run("appends items to existing array", func(t *testing.T) {
		content := `# Config file
name: test

# Items section
items:
  - id: item1
    value: foo
  - id: item2
    value: bar

other: data
`

		opts := ArrayAppendOptions{
			SectionKey: "items",
			ItemIDKey:  "id",
			Items: []map[string]any{
				{"id": "item3", "value": "baz"},
			},
			FormatItem: func(item map[string]any, indent string) string {
				return indent + "- id: " + item["id"].(string) + "\n" +
					indent + "  value: " + item["value"].(string) + "\n"
			},
		}

		result, added, err := AppendToArraySection(content, opts)
		if err != nil {
			t.Fatalf("AppendToArraySection failed: %v", err)
		}

		if added != 1 {
			t.Errorf("Expected added=1, got %d", added)
		}

		if !strings.Contains(result, "id: item3") {
			t.Error("Expected item3 to be added")
		}

		if !strings.Contains(result, "# Config file") {
			t.Error("Expected comments to be preserved")
		}

		if !strings.Contains(result, "other: data") {
			t.Error("Expected other sections to be preserved")
		}
	})

	t.Run("skips duplicate items", func(t *testing.T) {
		content := `name: test
items:
  - id: item1
    value: foo
`

		opts := ArrayAppendOptions{
			SectionKey: "items",
			ItemIDKey:  "id",
			Items: []map[string]any{
				{"id": "item1", "value": "duplicate"},
			},
			FormatItem: func(item map[string]any, indent string) string {
				return indent + "- id: " + item["id"].(string) + "\n"
			},
		}

		result, added, err := AppendToArraySection(content, opts)
		if err != nil {
			t.Fatalf("AppendToArraySection failed: %v", err)
		}

		if added != 0 {
			t.Errorf("Expected added=0 for duplicate, got %d", added)
		}

		if result != content {
			t.Error("Content should remain unchanged for duplicates")
		}
	})

	t.Run("preserves inline comments", func(t *testing.T) {
		content := `name: test
items:
  # First item
  - id: item1
    value: foo
  # Second item
  - id: item2
    value: bar
`

		opts := ArrayAppendOptions{
			SectionKey: "items",
			ItemIDKey:  "id",
			Items: []map[string]any{
				{"id": "item3", "value": "baz"},
			},
			FormatItem: func(item map[string]any, indent string) string {
				return indent + "- id: " + item["id"].(string) + "\n" +
					indent + "  value: " + item["value"].(string) + "\n"
			},
		}

		result, _, err := AppendToArraySection(content, opts)
		if err != nil {
			t.Fatalf("AppendToArraySection failed: %v", err)
		}

		if !strings.Contains(result, "# First item") {
			t.Error("Expected first item comment to be preserved")
		}

		if !strings.Contains(result, "# Second item") {
			t.Error("Expected second item comment to be preserved")
		}
	})

	t.Run("handles empty array", func(t *testing.T) {
		content := `name: test
items:
other: data
`

		opts := ArrayAppendOptions{
			SectionKey: "items",
			ItemIDKey:  "id",
			Items: []map[string]any{
				{"id": "item1", "value": "foo"},
			},
			FormatItem: func(item map[string]any, indent string) string {
				return indent + "- id: " + item["id"].(string) + "\n" +
					indent + "  value: " + item["value"].(string) + "\n"
			},
		}

		result, added, err := AppendToArraySection(content, opts)
		if err != nil {
			t.Fatalf("AppendToArraySection failed: %v", err)
		}

		if added != 1 {
			t.Errorf("Expected added=1, got %d", added)
		}

		if !strings.Contains(result, "id: item1") {
			t.Error("Expected item1 to be added")
		}
	})

	t.Run("creates section when missing", func(t *testing.T) {
		content := `name: test
other: data
`

		opts := ArrayAppendOptions{
			SectionKey: "missing",
			ItemIDKey:  "id",
			Items: []map[string]any{
				{"id": "item1", "value": "test"},
			},
			FormatItem: func(item map[string]any, arrayIndent string) string {
				return fmt.Sprintf("%s- id: %s\n%s  value: %s\n",
					arrayIndent, item["id"], arrayIndent, item["value"])
			},
		}

		result, added, err := AppendToArraySection(content, opts)
		if err != nil {
			t.Fatalf("AppendToArraySection failed: %v", err)
		}

		if added != 1 {
			t.Errorf("Expected added=1, got %d", added)
		}

		if !strings.Contains(result, "missing:") {
			t.Error("Expected 'missing:' section header to be created")
		}

		if !strings.Contains(result, "id: item1") {
			t.Error("Expected item1 to be added")
		}
	})

	t.Run("handles multiple items at once", func(t *testing.T) {
		content := `name: test

items:
  - id: item1
    value: a
`

		opts := ArrayAppendOptions{
			SectionKey: "items",
			ItemIDKey:  "id",
			Items: []map[string]any{
				{"id": "item2", "value": "b"},
				{"id": "item3", "value": "c"},
				{"id": "item4", "value": "d"},
			},
			FormatItem: func(item map[string]any, indent string) string {
				return indent + "- id: " + item["id"].(string) + "\n" +
					indent + "  value: " + item["value"].(string) + "\n"
			},
		}

		result, added, err := AppendToArraySection(content, opts)
		if err != nil {
			t.Fatalf("AppendToArraySection failed: %v", err)
		}

		if added != 3 {
			t.Errorf("Expected added=3, got %d", added)
		}

		if !strings.Contains(result, "id: item2") {
			t.Error("Expected item2 to be added")
		}
		if !strings.Contains(result, "id: item3") {
			t.Error("Expected item3 to be added")
		}
		if !strings.Contains(result, "id: item4") {
			t.Error("Expected item4 to be added")
		}
	})

	t.Run("handles deeply indented arrays", func(t *testing.T) {
		content := `name: test
config:
  database:
    items:
      - id: item1
        value: foo
other: data
`

		opts := ArrayAppendOptions{
			SectionKey: "items",
			ItemIDKey:  "id",
			Items: []map[string]any{
				{"id": "item2", "value": "bar"},
			},
			FormatItem: func(item map[string]any, indent string) string {
				return indent + "- id: " + item["id"].(string) + "\n" +
					indent + "  value: " + item["value"].(string) + "\n"
			},
		}

		result, added, err := AppendToArraySection(content, opts)
		if err != nil {
			t.Fatalf("AppendToArraySection failed: %v", err)
		}

		if added != 1 {
			t.Errorf("Expected added=1, got %d", added)
		}

		if !strings.Contains(result, "id: item2") {
			t.Error("Expected item2 to be added")
		}

		// Verify indentation is correct (6 spaces for items under config.database.items)
		lines := strings.Split(result, "\n")
		foundItem2 := false
		for _, line := range lines {
			if strings.Contains(line, "id: item2") {
				if !strings.HasPrefix(line, "      - ") {
					t.Errorf("Expected 6-space indent for item2, got: %q", line)
				}
				foundItem2 = true
				break
			}
		}
		if !foundItem2 {
			t.Error("Could not find item2 in result")
		}
	})

	t.Run("handles all duplicates scenario", func(t *testing.T) {
		content := `name: test

items:
  - id: item1
  - id: item2
`

		opts := ArrayAppendOptions{
			SectionKey: "items",
			ItemIDKey:  "id",
			Items: []map[string]any{
				{"id": "item1"},
				{"id": "item2"},
			},
			FormatItem: func(item map[string]any, indent string) string {
				return indent + "- id: " + item["id"].(string) + "\n"
			},
		}

		result, added, err := AppendToArraySection(content, opts)
		if err != nil {
			t.Fatalf("AppendToArraySection failed: %v", err)
		}

		if added != 0 {
			t.Errorf("Expected added=0, got %d", added)
		}

		// Result should be unchanged
		if result != content {
			t.Error("Expected content to be unchanged when all items are duplicates")
		}
	})

	t.Run("preserves trailing content after array", func(t *testing.T) {
		content := `name: test

items:
  - id: item1

# Important section below
services:
  - name: api

# Footer comment
version: 1.0
`

		opts := ArrayAppendOptions{
			SectionKey: "items",
			ItemIDKey:  "id",
			Items: []map[string]any{
				{"id": "item2"},
			},
			FormatItem: func(item map[string]any, indent string) string {
				return indent + "- id: " + item["id"].(string) + "\n"
			},
		}

		result, added, err := AppendToArraySection(content, opts)
		if err != nil {
			t.Fatalf("AppendToArraySection failed: %v", err)
		}

		if added != 1 {
			t.Errorf("Expected added=1, got %d", added)
		}

		if !strings.Contains(result, "# Important section below") {
			t.Error("Expected trailing section comment to be preserved")
		}
		if !strings.Contains(result, "# Footer comment") {
			t.Error("Expected footer comment to be preserved")
		}
		if !strings.Contains(result, "version: 1.0") {
			t.Error("Expected trailing content to be preserved")
		}
	})

	t.Run("handles inline empty array", func(t *testing.T) {
		content := `name: test
items: []
services:
  api: {}
`

		opts := ArrayAppendOptions{
			SectionKey: "items",
			ItemIDKey:  "id",
			Items: []map[string]any{
				{"id": "item1", "value": "foo"},
			},
			FormatItem: func(item map[string]any, indent string) string {
				return indent + "- id: " + item["id"].(string) + "\n" +
					indent + "  value: " + item["value"].(string) + "\n"
			},
		}

		result, added, err := AppendToArraySection(content, opts)
		if err != nil {
			t.Fatalf("AppendToArraySection failed: %v", err)
		}

		if added != 1 {
			t.Errorf("Expected added=1, got %d", added)
		}

		// Should not have "items: []" anymore
		if strings.Contains(result, "items: []") {
			t.Error("Expected 'items: []' to be replaced with actual array")
		}

		if !strings.Contains(result, "- id: item1") {
			t.Error("Expected item1 to be added")
		}

		// Should preserve services section
		if !strings.Contains(result, "services:") {
			t.Error("Expected services section to be preserved")
		}
	})

	t.Run("handles inline empty array with indentation", func(t *testing.T) {
		content := `name: test
config:
  items: []
  other: value
`

		opts := ArrayAppendOptions{
			SectionKey: "items",
			ItemIDKey:  "id",
			Items: []map[string]any{
				{"id": "item1"},
			},
			FormatItem: func(item map[string]any, indent string) string {
				return indent + "- id: " + item["id"].(string) + "\n"
			},
		}

		result, added, err := AppendToArraySection(content, opts)
		if err != nil {
			t.Fatalf("AppendToArraySection failed: %v", err)
		}

		if added != 1 {
			t.Errorf("Expected added=1, got %d", added)
		}

		// Should not have "items: []" anymore
		if strings.Contains(result, "items: []") {
			t.Error("Expected 'items: []' to be replaced with actual array")
		}

		// Verify indentation is correct (4 spaces for items under config)
		lines := strings.Split(result, "\n")
		foundItem := false
		for _, line := range lines {
			if strings.Contains(line, "- id: item1") {
				// Should have 4 spaces (2 for config + 2 for array item)
				if !strings.HasPrefix(line, "    - ") {
					t.Errorf("Expected 4-space indent for item1, got: %q", line)
				}
				foundItem = true
				break
			}
		}
		if !foundItem {
			t.Error("Could not find item1 in result")
		}

		// Should preserve other value
		if !strings.Contains(result, "other: value") {
			t.Error("Expected 'other: value' to be preserved")
		}
	})

	t.Run("handles inline empty object treated as empty array", func(t *testing.T) {
		// This tests that inline {} is also handled (should be replaced)
		content := `name: test
reqs: {}
`

		opts := ArrayAppendOptions{
			SectionKey: "reqs",
			ItemIDKey:  "name",
			Items: []map[string]any{
				{"name": "node", "minVersion": "20.0.0"},
			},
			FormatItem: func(item map[string]any, indent string) string {
				return indent + "- name: " + item["name"].(string) + "\n" +
					indent + "  minVersion: \"" + item["minVersion"].(string) + "\"\n"
			},
		}

		result, added, err := AppendToArraySection(content, opts)
		if err != nil {
			t.Fatalf("AppendToArraySection failed: %v", err)
		}

		if added != 1 {
			t.Errorf("Expected added=1, got %d", added)
		}

		// Should not have "reqs: {}" anymore
		if strings.Contains(result, "reqs: {}") {
			t.Error("Expected 'reqs: {}' to be replaced with actual array")
		}

		if !strings.Contains(result, "- name: node") {
			t.Error("Expected node to be added")
		}
	})
}

func TestFindSection(t *testing.T) {
	tests := []struct {
		name       string
		lines      []string
		sectionKey string
		wantIdx    int
		wantIndent string
		wantErr    bool
	}{
		{
			name:       "finds top-level section",
			lines:      []string{"name: test", "items:", "  - id: 1"},
			sectionKey: "items",
			wantIdx:    1,
			wantIndent: "",
			wantErr:    false,
		},
		{
			name:       "finds indented section",
			lines:      []string{"outer:", "  items:", "    - id: 1"},
			sectionKey: "items",
			wantIdx:    1,
			wantIndent: "  ",
			wantErr:    false,
		},
		{
			name:       "returns error for missing section",
			lines:      []string{"name: test"},
			sectionKey: "missing",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := findSection(tt.lines, tt.sectionKey)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if info.lineIdx != tt.wantIdx {
				t.Errorf("lineIdx = %d, want %d", info.lineIdx, tt.wantIdx)
			}

			if info.indent != tt.wantIndent {
				t.Errorf("indent = %q, want %q", info.indent, tt.wantIndent)
			}
		})
	}
}

func TestGetIndentation(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"no indent", ""},
		{"  two spaces", "  "},
		{"    four spaces", "    "},
		{"\ttab", "\t"},
		{"  \tmixed", "  \t"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := getIndentation(tt.line)
			if got != tt.want {
				t.Errorf("getIndentation(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsArrayItem(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"- item", true},
		{"-\titem", true},
		{"  - item", false}, // Already trimmed in actual usage
		{"not an item", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := isArrayItem(tt.line)
			if got != tt.want {
				t.Errorf("isArrayItem(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
