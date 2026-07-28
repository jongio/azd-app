package docsgate

import (
	"regexp"
	"strings"
)

var (
	// commandHeadingPattern matches a per-command section heading such as
	// "## `azd app support-bundle`". The character class deliberately allows
	// hyphens and spaces so hyphenated commands and subcommands both parse.
	commandHeadingPattern = regexp.MustCompile("^## `azd app ([a-z0-9][a-z0-9 -]*)`\\s*$")

	// headingPattern matches any ATX heading so section scanning knows where a
	// section ends.
	headingPattern = regexp.MustCompile(`^(#{1,6}) `)

	// fencePattern matches the opening or closing line of a fenced code block.
	fencePattern = regexp.MustCompile("^\\s*(```|~~~)")

	// tableFlagPattern matches a table cell that documents a flag, such as
	// "`--service`". Only the first two cells of a row are checked, which keeps
	// a flag mentioned in some other row's description column from counting as
	// documented. The capture excludes the leading dashes so it matches the bare
	// names the CLI metadata reports.
	tableFlagPattern = regexp.MustCompile("^`--([a-z0-9][a-z0-9-]*)`$")

	// tableNamePattern matches a table cell holding a backticked name rather
	// than a flag, such as the "`list`" cells of a Subcommands table and the
	// rows of the Commands Overview table.
	tableNamePattern = regexp.MustCompile("^`([a-z0-9][a-z0-9 -]*)`$")

	// specLinkPattern captures the detail doc path from an overview row.
	specLinkPattern = regexp.MustCompile(`\(commands/([a-z0-9-]+)\.md\)`)
)

// Section is one per-command section of cli-reference.md.
type Section struct {
	// Name is the command as written in the heading, such as "notifications list".
	Name string
	// Line is the 1-based line number of the heading, for actionable errors.
	Line int
	// Flags holds every flag documented in a table row inside the section.
	Flags map[string]bool
	// Names holds the non-flag backticked names in the section's table rows.
	// Subcommands appear here, which is how a parent section documents them.
	Names map[string]bool
}

// Reference is the parsed view of cli/docs/cli-reference.md.
type Reference struct {
	// GlobalFlags holds the flags documented in the Global Flags table. A flag
	// listed here counts as documented for every command.
	GlobalFlags map[string]bool
	// Sections maps a command slug to its parsed section.
	Sections map[string]*Section
	// Overview maps a command slug to the detail doc slug its overview row links
	// to. An empty value means the row exists but links to nothing.
	Overview map[string]string
}

// ParseReference reads cli-reference.md into the shape the coverage rules need.
//
// The parser walks lines once and tracks which section it is inside rather than
// running a regex over the whole document, because sections are delimited by the
// next heading of equal or higher level and that is awkward to express as a
// single expression over a 40 KB file.
func ParseReference(content string) *Reference {
	ref := &Reference{
		GlobalFlags: map[string]bool{},
		Sections:    map[string]*Section{},
		Overview:    map[string]string{},
	}

	var current *Section
	inGlobalFlags := false
	inOverview := false
	inFence := false

	for i, line := range strings.Split(content, "\n") {
		// Fenced code blocks are full of shell comments, and a shell comment is
		// indistinguishable from a markdown heading. Skipping fenced content
		// keeps "# Check health of all services" from closing the section it
		// documents.
		if fencePattern.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		if m := headingPattern.FindStringSubmatch(line); m != nil {
			level := len(m[1])

			// Any H2 closes the previous command section and both trailer states.
			if level <= 2 {
				current = nil
				inGlobalFlags = false
				inOverview = false
			} else if level == 3 {
				// An H3 closes only the Global Flags subsection.
				inGlobalFlags = false
			}

			switch {
			case commandHeadingPattern.MatchString(line):
				name := commandHeadingPattern.FindStringSubmatch(line)[1]
				current = &Section{
					Name:  name,
					Line:  i + 1,
					Flags: map[string]bool{},
					Names: map[string]bool{},
				}
				ref.Sections[slugify(name)] = current
			case strings.TrimSpace(line) == "### Global Flags":
				inGlobalFlags = true
			case strings.TrimSpace(line) == "## Commands Overview":
				inOverview = true
			}
			continue
		}

		if !strings.HasPrefix(line, "|") {
			continue
		}

		cells := tableCells(line)
		if len(cells) == 0 {
			continue
		}

		if inOverview {
			if m := tableNamePattern.FindStringSubmatch(cells[0]); m != nil {
				slug := slugify(m[1])
				spec := ""
				if link := specLinkPattern.FindStringSubmatch(line); link != nil {
					spec = link[1]
				}
				ref.Overview[slug] = spec
			}
			continue
		}

		// A flag is documented by the first cell of a flags table, or by the
		// second cell of a table whose first column names the subcommand the
		// flag belongs to. Looking no further keeps a flag named in a
		// description column from counting as documented.
		for _, cell := range cells[:min(2, len(cells))] {
			m := tableFlagPattern.FindStringSubmatch(cell)
			if m == nil {
				continue
			}
			switch {
			case inGlobalFlags:
				ref.GlobalFlags[m[1]] = true
			case current != nil:
				current.Flags[m[1]] = true
			}
		}

		if m := tableNamePattern.FindStringSubmatch(cells[0]); m != nil && current != nil {
			current.Names[slugify(m[1])] = true
		}
	}

	return ref
}

// tableCells splits a markdown table row into trimmed cell values. Cells past
// the second are only used for link extraction, so a pipe inside a description
// splitting a trailing cell in two is harmless.
func tableCells(line string) []string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") {
		return nil
	}
	parts := strings.Split(strings.Trim(trimmed, "|"), "|")
	cells := make([]string, len(parts))
	for i, part := range parts {
		cells[i] = strings.TrimSpace(part)
	}
	return cells
}

// slugify converts a command name to the slug used for section lookups and
// detail doc filenames: "notifications list" becomes "notifications-list".
func slugify(name string) string {
	return strings.Join(strings.Fields(name), "-")
}
