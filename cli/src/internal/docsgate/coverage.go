package docsgate

import (
	"fmt"
	"sort"
)

// Structural rule identifiers. These compare repository state against the live
// command tree, so they cannot be skipped.
const (
	// RuleCommandUndocumented fires when a top level command has no section in
	// cli-reference.md.
	RuleCommandUndocumented = "command-undocumented"
	// RuleCommandMissingOverview fires when a top level command is absent from
	// the Commands Overview table, which is how readers discover it.
	RuleCommandMissingOverview = "command-missing-overview"
	// RuleCommandMissingDetailDoc fires when a top level command has no file in
	// cli/docs/commands.
	RuleCommandMissingDetailDoc = "command-missing-detail-doc"
	// RuleSubcommandUndocumented fires when a subcommand appears in neither its
	// parent's section nor a section of its own.
	RuleSubcommandUndocumented = "subcommand-undocumented"
	// RuleFlagUndocumented fires when a visible flag appears in neither its
	// command's documentation nor the Global Flags table.
	RuleFlagUndocumented = "flag-undocumented"
	// RuleDocOrphaned fires when documentation describes a command the CLI no
	// longer ships.
	RuleDocOrphaned = "doc-orphaned"
)

// CheckCoverage compares the live command tree against the parsed docs.
//
// The rules follow the shape cli-reference.md already has. Top level commands
// carry the documentation burden: a section, an overview row, and a detail doc.
// Subcommands are documented inside their parent, so they only need to be named
// there. Flags may be documented in their own command's section, in the root
// command's section, or in the Global Flags table.
//
// detailDocs is the set of slugs found in cli/docs/commands, derived from the
// filenames rather than from the overview links so a deleted file is caught even
// when the link survives.
func CheckCoverage(md *Metadata, ref *Reference, detailDocs map[string]bool) []Finding {
	var findings []Finding

	for _, cmd := range md.DocumentableCommands() {
		findings = append(findings, checkCommand(cmd, ref, detailDocs)...)
		findings = append(findings, checkFlags(cmd, ref)...)
	}

	findings = append(findings, checkOrphans(md, ref, detailDocs)...)

	sortFindings(findings)
	return findings
}

// checkCommand verifies a command is discoverable in the docs.
func checkCommand(cmd CommandRef, ref *Reference, detailDocs map[string]bool) []Finding {
	if !cmd.TopLevel() {
		return checkSubcommand(cmd, ref)
	}

	var findings []Finding
	slug := cmd.Slug()

	if _, ok := ref.Sections[slug]; !ok {
		findings = append(findings, Finding{
			Rule:    RuleCommandUndocumented,
			Command: cmd.Name(),
			File:    referencePath,
			Message: fmt.Sprintf("add a \"## `azd app %s`\" section documenting the command and its flags", cmd.Name()),
		})
	}

	if _, ok := ref.Overview[slug]; !ok {
		findings = append(findings, Finding{
			Rule:    RuleCommandMissingOverview,
			Command: cmd.Name(),
			File:    referencePath,
			Message: "add a row to the Commands Overview table linking to commands/" + slug + ".md",
		})
	}

	if !detailDocs[slug] {
		findings = append(findings, Finding{
			Rule:    RuleCommandMissingDetailDoc,
			Command: cmd.Name(),
			File:    fmt.Sprintf("%s/%s.md", commandsDir, slug),
			Message: "create the detail spec for this command",
		})
	}

	return findings
}

// checkSubcommand verifies a subcommand is named somewhere a reader will find
// it: a Subcommands row in the root command's section, or a section of its own.
func checkSubcommand(cmd CommandRef, ref *Reference) []Finding {
	if _, ok := ref.Sections[cmd.Slug()]; ok {
		return nil
	}
	if section, ok := ref.Sections[cmd.Root()]; ok && namesSubcommand(section, cmd) {
		return nil
	}
	return []Finding{{
		Rule:    RuleSubcommandUndocumented,
		Command: cmd.Name(),
		File:    referencePath,
		Message: fmt.Sprintf("add a Subcommands row to the \"## `azd app %s`\" section: | `%s` | %s |",
			cmd.Root(), cmd.Leaf(), cmd.Short),
	}}
}

// namesSubcommand reports whether a section's tables name the subcommand,
// either by its leaf word or by its full path.
func namesSubcommand(section *Section, cmd CommandRef) bool {
	return section.Names[cmd.Leaf()] || section.Names[cmd.Slug()]
}

// checkFlags verifies every visible flag is documented somewhere that applies to
// the command.
func checkFlags(cmd CommandRef, ref *Reference) []Finding {
	sections := applicableSections(cmd, ref)

	// Without any section to document them in, the missing section is the only
	// finding worth reporting. Listing every flag of a brand new command would
	// bury it.
	if len(sections) == 0 {
		return nil
	}
	target := sections[0]

	var findings []Finding
	for _, flag := range cmd.DocumentedFlags() {
		if ref.GlobalFlags[flag.Name] || documentedIn(sections, flag.Name) {
			continue
		}
		findings = append(findings, Finding{
			Rule:    RuleFlagUndocumented,
			Command: cmd.Name(),
			Flag:    "--" + flag.Name,
			File:    fmt.Sprintf("%s:%d", referencePath, target.Line),
			Message: fmt.Sprintf("add a Flags table row: | `--%s` | | %s | | %s |", flag.Name, flag.Type, flag.Description),
		})
	}
	return findings
}

// applicableSections returns the sections that may document a command's flags,
// most specific first: the command's own section, then its root command's.
func applicableSections(cmd CommandRef, ref *Reference) []*Section {
	var out []*Section
	if s, ok := ref.Sections[cmd.Slug()]; ok {
		out = append(out, s)
	}
	if !cmd.TopLevel() {
		if s, ok := ref.Sections[cmd.Root()]; ok {
			out = append(out, s)
		}
	}
	return out
}

// documentedIn reports whether any section has a Flags row for the flag.
func documentedIn(sections []*Section, flag string) bool {
	for _, s := range sections {
		if s.Flags[flag] {
			return true
		}
	}
	return false
}

// checkOrphans reports documentation for commands the CLI no longer ships.
//
// The known set includes hidden and built-in commands so documenting `mcp` stays
// legal, while a doc for a deleted command is still caught.
func checkOrphans(md *Metadata, ref *Reference, detailDocs map[string]bool) []Finding {
	known := md.KnownCommandSlugs()
	var findings []Finding

	for slug, section := range ref.Sections {
		if known[slug] {
			continue
		}
		findings = append(findings, Finding{
			Rule:    RuleDocOrphaned,
			Command: section.Name,
			File:    fmt.Sprintf("%s:%d", referencePath, section.Line),
			Message: "the CLI no longer ships this command; remove the section or restore the command",
		})
	}

	for slug := range detailDocs {
		if known[slug] {
			continue
		}
		findings = append(findings, Finding{
			Rule:    RuleDocOrphaned,
			Command: slug,
			File:    fmt.Sprintf("%s/%s.md", commandsDir, slug),
			Message: "the CLI no longer ships this command; remove the doc or restore the command",
		})
	}

	return findings
}

// sortFindings gives the report a stable order so CI output can be diffed
// between runs.
func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		if a.Command != b.Command {
			return a.Command < b.Command
		}
		if a.Flag != b.Flag {
			return a.Flag < b.Flag
		}
		return a.File < b.File
	})
}
