/**
 * Command Validation Script
 *
 * Validates that every command the CLI ships reaches the website.
 *
 * The chain of custody has two links, and this script owns the second:
 *
 *   1. `mage docsGate` compares the live `azd app metadata` command tree
 *      against cli/docs/cli-reference.md and fails when they disagree.
 *   2. This script compares cli-reference.md against what the website
 *      generator can actually read out of it.
 *
 * Enumerating from the reference rather than from cli/docs/commands/ is the
 * point. The old direction asked "does the website cover this file?", so a
 * command with no file was invisible and a stale page counted as coverage.
 *
 * It deliberately reads the reference rather than the generated pages under
 * web/src/pages/reference/cli/. Those are produced by `generate:cli`, which
 * runs after this script, so checking them would either pass on stale output
 * or fail on a clean checkout.
 *
 * Exit codes:
 * - 0: every shipped command reaches the website
 * - 1: one or more commands would be missing or incomplete
 */

import * as fs from "node:fs";
import * as path from "node:path";
import {
  discoverCommands,
  extractCommandSection,
  parseCommandFromReference,
  parseCommandsOverview,
  parseFlags,
} from "./cli-parser.js";

// Resolve paths relative to project root (web's parent)
const webRoot = path.resolve(import.meta.dirname, "..");
const projectRoot = path.resolve(webRoot, "..");

const CLI_REFERENCE = path.join(projectRoot, "cli", "docs", "cli-reference.md");
const CLI_COMMANDS_DIR = path.join(projectRoot, "cli", "docs", "commands");
const CONTENT_COMMANDS_DIR = path.join(webRoot, "src", "content", "commands");
const EXCLUDE_FILE = path.join(webRoot, "scripts", ".exclude-commands");

interface Problem {
  command: string;
  rule: string;
  detail: string;
  fix: string;
}

/**
 * Get list of commands to exclude from validation
 */
function getExcludedCommands(): Set<string> {
  const excluded = new Set<string>();

  if (fs.existsSync(EXCLUDE_FILE)) {
    const content = fs.readFileSync(EXCLUDE_FILE, "utf-8");

    for (const line of content.split("\n")) {
      const trimmed = line.trim();
      // Skip comments and empty lines
      if (trimmed && !trimmed.startsWith("#")) {
        excluded.add(trimmed);
      }
    }
  }

  return excluded;
}

/**
 * Checks one command from the reference all the way to a renderable page.
 */
function checkCommand(
  command: string,
  reference: string,
  discovered: Set<string>
): Problem[] {
  const problems: Problem[] = [];

  if (!discovered.has(command)) {
    problems.push({
      command,
      rule: "not-discovered",
      detail: "the generator does not see this command, so no page will be built",
      fix: `add a "## \`azd app ${command}\`" section to cli/docs/cli-reference.md or a cli/docs/commands/${command}.md spec`,
    });
    return problems;
  }

  const parsed = parseCommandFromReference(reference, command, CLI_COMMANDS_DIR);
  if (!parsed) {
    problems.push({
      command,
      rule: "no-section",
      detail: "the Commands Overview lists this command but it has no reference section",
      fix: `add a "## \`azd app ${command}\`" section to cli/docs/cli-reference.md`,
    });
    return problems;
  }

  if (!parsed.description) {
    problems.push({
      command,
      rule: "no-description",
      detail: "the page would render with an empty summary",
      fix: `add a one line summary directly under the "## \`azd app ${command}\`" heading`,
    });
  }

  // A section that documents flags must yield flags. When the two disagree the
  // table has drifted into a shape the parser cannot read, and the flags would
  // vanish from the website without anything failing.
  const section = extractCommandSection(reference, command) ?? "";
  if (/^### Flags\s*$/m.test(section) && parseFlags(section).length === 0) {
    problems.push({
      command,
      rule: "flags-unparsed",
      detail: "the section has a Flags heading but no row the generator can read",
      fix: "use | `--flag` | Short | Type | Default | Description | rows so the flags reach the website",
    });
  }

  return problems;
}

/**
 * Reports hand-authored website pages for commands the CLI no longer ships.
 *
 * `known` must include excluded commands. Exclusion means "do not validate the
 * documentation for this command", not "this command was removed", so filtering
 * excluded commands out here would report their pages as orphans and fail the
 * build on exactly the configuration .exclude-commands exists to permit.
 */
function checkOrphans(known: Set<string>): Problem[] {
  if (!fs.existsSync(CONTENT_COMMANDS_DIR)) {
    return [];
  }

  const problems: Problem[] = [];
  for (const file of fs.readdirSync(CONTENT_COMMANDS_DIR)) {
    if (!file.endsWith(".md") && !file.endsWith(".mdx")) continue;

    const command = path.basename(file, path.extname(file));
    if (known.has(command)) continue;

    problems.push({
      command,
      rule: "orphaned",
      detail: `src/content/commands/${file} documents a command the CLI no longer ships`,
      fix: "remove the page or restore the command",
    });
  }
  return problems;
}

/**
 * Main entry point
 */
function main(): void {
  console.log("🔍 Validating CLI command documentation coverage...\n");

  if (!fs.existsSync(CLI_REFERENCE)) {
    console.error(`❌ CLI reference not found: ${CLI_REFERENCE}`);
    process.exit(1);
  }

  const reference = fs.readFileSync(CLI_REFERENCE, "utf-8");
  const excluded = getExcludedCommands();
  const overview = parseCommandsOverview(reference);
  const shipped = overview.filter((c) => !excluded.has(c));

  if (shipped.length === 0) {
    console.error("❌ No commands found in the Commands Overview table of cli/docs/cli-reference.md");
    console.error("   Validating against an empty list would pass while proving nothing.");
    process.exit(1);
  }

  const discovered = new Set(discoverCommands(reference, CLI_COMMANDS_DIR));

  const problems: Problem[] = [];
  for (const command of [...shipped].sort()) {
    problems.push(...checkCommand(command, reference, discovered));
  }
  problems.push(...checkOrphans(new Set([...overview, ...excluded])));

  const brokenCommands = new Set(problems.map((p) => p.command));
  const healthy = shipped.filter((c) => !brokenCommands.has(c));

  if (healthy.length > 0) {
    console.log(`✅ Commands reaching the website (${healthy.length}):`);
    console.log(`   ${healthy.join(", ")}\n`);
  }

  if (problems.length === 0) {
    console.log(`✅ All ${shipped.length} shipped commands reach the website.`);
    process.exit(0);
  }

  console.log(`❌ Problems found (${problems.length}):\n`);
  for (const problem of problems) {
    console.log(`   [${problem.rule}] ${problem.command}`);
    console.log(`      ${problem.detail}`);
    console.log(`      fix: ${problem.fix}\n`);
  }
  console.log("The Commands Overview table in cli/docs/cli-reference.md is the source of truth.");
  console.log("To exclude a command from validation, add it to web/scripts/.exclude-commands");
  process.exit(1);
}

main();
