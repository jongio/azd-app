// Package yamlutil provides utilities for manipulating YAML files while preserving
// formatting, comments, and structure. It uses text-based manipulation to guarantee
// zero data loss when updating YAML configuration files.
package yamlutil

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ArrayAppendOptions configures how array items should be appended to a YAML section.
type ArrayAppendOptions struct {
	SectionKey string                              // The YAML key to find (e.g., "reqs")
	Items      []map[string]any                    // Items to append
	ItemIDKey  string                              // Key to use for deduplication (e.g., "id")
	FormatItem func(map[string]any, string) string // Custom formatter for items
}

// AppendToArraySection appends items to a YAML array section while preserving all
// comments, formatting, and other content in the file.
//
// This uses text-based manipulation to guarantee zero data loss, only appending
// new items to the specified array section.
func AppendToArraySection(content string, opts ArrayAppendOptions) (string, int, error) {
	lines := strings.Split(content, "\n")

	// Parse YAML to find existing items (read-only)
	existingIDs, err := getExistingIDs(content, opts.SectionKey, opts.ItemIDKey)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse existing items: %w", err)
	}

	// Filter items to add (skip duplicates)
	var toAdd []map[string]any
	for _, item := range opts.Items {
		if id, ok := item[opts.ItemIDKey].(string); ok {
			if !existingIDs[id] {
				toAdd = append(toAdd, item)
			}
		}
	}

	// Nothing to add
	if len(toAdd) == 0 {
		return content, 0, nil
	}

	// Find the section location and indentation
	sectionInfo, err := findSection(lines, opts.SectionKey)
	if err != nil {
		// If section not found, create it at the end of the file
		return appendNewSection(content, opts, toAdd)
	}

	// Handle inline empty array (e.g., "reqs: []")
	if sectionInfo.hasInlineVal {
		return replaceInlineEmptyArray(lines, sectionInfo, opts, toAdd)
	}

	// Find the end of the array
	lastLineIdx, arrayIndent := findLastArrayLine(lines, sectionInfo)

	// Build new items as YAML text
	newItemsText := buildItemsYaml(toAdd, arrayIndent, opts.FormatItem)

	// Insert the new items and reconstruct the file
	result := insertLines(lines, lastLineIdx, newItemsText)

	return result, len(toAdd), nil
}

// replaceInlineEmptyArray handles the case where a section has an inline empty array (e.g., "reqs: []").
// It replaces the inline array with a proper multi-line array format.
func replaceInlineEmptyArray(lines []string, section *sectionInfo, opts ArrayAppendOptions, items []map[string]any) (string, int, error) {
	// Get the section line and extract the key part
	sectionLine := lines[section.lineIdx]
	keyPart := opts.SectionKey + ":"

	// Find where the key ends and replace everything after with just the colon
	keyIdx := strings.Index(sectionLine, keyPart)
	if keyIdx == -1 {
		return "", 0, fmt.Errorf("unexpected: section key not found in line")
	}

	// Replace the line with just the section key (removing the inline value)
	newSectionLine := sectionLine[:keyIdx+len(keyPart)]
	lines[section.lineIdx] = newSectionLine

	// Determine indentation for array items (section indent + 2 spaces)
	arrayIndent := section.indent + "  "

	// Build new items as YAML text
	newItemsText := buildItemsYaml(items, arrayIndent, opts.FormatItem)

	// Insert the new items after the section line
	result := insertLines(lines, section.lineIdx, newItemsText)

	return result, len(items), nil
}

// appendNewSection creates a new section at the end of the file with the specified items.
func appendNewSection(content string, opts ArrayAppendOptions, items []map[string]any) (string, int, error) {
	var builder strings.Builder
	builder.WriteString(content)

	// Ensure there's a newline before the new section
	if !strings.HasSuffix(content, "\n") {
		builder.WriteString("\n")
	}

	// Add blank line before new section for readability
	if strings.TrimSpace(content) != "" {
		builder.WriteString("\n")
	}

	// Add section header
	builder.WriteString(opts.SectionKey)
	builder.WriteString(":\n")

	// Add items with default indentation (2 spaces)
	arrayIndent := "  "
	itemsText := buildItemsYaml(items, arrayIndent, opts.FormatItem)
	builder.WriteString(itemsText)

	return builder.String(), len(items), nil
}

// sectionInfo holds information about a YAML section location.
type sectionInfo struct {
	lineIdx      int    // Line index where the section key appears
	indent       string // Indentation of the section key line
	hasInlineVal bool   // True if section has inline value (e.g., "reqs: []")
}

// getExistingIDs parses the YAML to extract existing item IDs.
func getExistingIDs(content, sectionKey, idKey string) (map[string]bool, error) {
	var root map[string]any
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		return nil, err
	}

	existingIDs := make(map[string]bool)

	if section, ok := root[sectionKey].([]any); ok {
		for _, item := range section {
			if itemMap, ok := item.(map[string]any); ok {
				if id, ok := itemMap[idKey].(string); ok {
					existingIDs[id] = true
				}
			}
		}
	}

	return existingIDs, nil
}

// findSection locates the specified section in the YAML lines.
func findSection(lines []string, sectionKey string) (*sectionInfo, error) {
	searchKey := sectionKey + ":"

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == searchKey || strings.HasPrefix(trimmed, searchKey+" ") {
			indent := getIndentation(line)
			// Check if the section has an inline value (e.g., "reqs: []" or "reqs: {}")
			hasInlineVal := strings.HasPrefix(trimmed, searchKey+" ")
			return &sectionInfo{
				lineIdx:      i,
				indent:       indent,
				hasInlineVal: hasInlineVal,
			}, nil
		}
	}

	return nil, fmt.Errorf("section %q not found in YAML", sectionKey)
}

// findLastArrayLine finds the last line of the array and determines indentation.
func findLastArrayLine(lines []string, section *sectionInfo) (int, string) {
	lastLineIdx := section.lineIdx
	arrayIndent := ""

	for i := section.lineIdx + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check if this is an array item
		if isArrayItem(trimmed) {
			if arrayIndent == "" {
				arrayIndent = getIndentation(line)
			}
			// Scan for all properties of this item
			lastLineIdx = scanArrayItem(lines, i, arrayIndent)
			continue
		}

		// Check if we've reached a new section
		if isNewSection(line, section.indent, trimmed) {
			break
		}
	}

	// Use default indentation if no array items found
	if arrayIndent == "" {
		arrayIndent = section.indent + "  "
	}

	return lastLineIdx, arrayIndent
}

// isArrayItem checks if a line is a YAML array item.
func isArrayItem(trimmed string) bool {
	return strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "-\t")
}

// getIndentation extracts the leading whitespace from a line.
func getIndentation(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// isNewSection checks if a line represents a new top-level section.
func isNewSection(line, baseIndent, trimmed string) bool {
	if trimmed == "" {
		return false
	}
	lineIndent := getIndentation(line)
	return len(lineIndent) <= len(baseIndent)
}

// scanArrayItem scans from an array item start to find its last line.
func scanArrayItem(lines []string, startIdx int, arrayIndent string) int {
	lastIdx := startIdx

	for j := startIdx + 1; j < len(lines); j++ {
		innerLine := lines[j]
		innerTrimmed := strings.TrimSpace(innerLine)

		// Skip empty lines and comments
		if innerTrimmed == "" || strings.HasPrefix(innerTrimmed, "#") {
			continue
		}

		// Another array item means we're done with this one
		if isArrayItem(innerTrimmed) {
			break
		}

		// Check if this line is part of the array item (more indented)
		lineIndent := getIndentation(innerLine)
		if len(lineIndent) <= len(arrayIndent) {
			// Less indented = new section, stop here
			break
		}

		// This line is part of the current item
		lastIdx = j
	}

	return lastIdx
}

// buildItemsYaml generates YAML text for new items using the custom formatter.
func buildItemsYaml(items []map[string]any, arrayIndent string, formatter func(map[string]any, string) string) string {
	var builder strings.Builder

	for _, item := range items {
		if formatter != nil {
			builder.WriteString(formatter(item, arrayIndent))
		}
	}

	return builder.String()
}

// insertLines inserts new lines into the content at the specified position.
func insertLines(lines []string, lastLineIdx int, newText string) string {
	if newText == "" {
		return strings.Join(lines, "\n")
	}

	// Pre-allocate result slice
	newLines := strings.Split(strings.TrimRight(newText, "\n"), "\n")
	result := make([]string, 0, len(lines)+len(newLines))

	// Lines before and including last line
	result = append(result, lines[:lastLineIdx+1]...)

	// New lines
	result = append(result, newLines...)

	// Remaining lines
	result = append(result, lines[lastLineIdx+1:]...)

	return strings.Join(result, "\n")
}
