/**
 * Command Validation Script
 *
 * This script validates that all CLI commands in cli/docs/commands/
 * have corresponding documentation coverage in the website.
 *
 * Coverage can be in:
 * - web/src/content/commands/*.md (reference docs)
 * - web/src/pages/tour/*.mdx (tutorial coverage)
 * - web/src/pages/reference/commands/ (generated pages)
 *
 * Exit codes:
 * - 0: All commands have documentation coverage
 * - 1: One or more commands are missing documentation
 */

import * as fs from "node:fs";
import * as path from "node:path";

// Resolve paths relative to project root (web's parent)
const webRoot = path.resolve(import.meta.dirname, "..");
const projectRoot = path.resolve(webRoot, "..");

const CLI_COMMANDS_DIR = path.join(projectRoot, "cli", "docs", "commands");
const CONTENT_COMMANDS_DIR = path.join(webRoot, "src", "content", "commands");
const TOUR_PAGES_DIR = path.join(webRoot, "src", "pages", "tour");
const REFERENCE_CLI_DIR = path.join(
  webRoot,
  "src",
  "pages",
  "reference",
  "cli"
);
const EXCLUDE_FILE = path.join(webRoot, "scripts", ".exclude-commands");

interface ValidationResult {
  command: string;
  covered: boolean;
  coverageLocation?: string;
}

/**
 * Get list of commands to exclude from validation
 */
function getExcludedCommands(): Set<string> {
  const excluded = new Set<string>();

  if (fs.existsSync(EXCLUDE_FILE)) {
    const content = fs.readFileSync(EXCLUDE_FILE, "utf-8");
    const lines = content.split("\n");

    for (const line of lines) {
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
 * Get all command names from CLI docs
 */
function getCliCommands(): string[] {
  if (!fs.existsSync(CLI_COMMANDS_DIR)) {
    console.warn(`⚠️  Warning: CLI commands directory not found: ${CLI_COMMANDS_DIR}`);
    return [];
  }

  const files = fs.readdirSync(CLI_COMMANDS_DIR);
  const mdFiles = files.filter((f) => f.endsWith(".md"));

  if (mdFiles.length === 0) {
    console.warn("⚠️  Warning: No command documentation files found in cli/docs/commands/");
    return [];
  }

  return mdFiles.map((f) => path.basename(f, ".md"));
}

/**
 * Check if a command is covered in content/commands
 */
function checkContentCommands(command: string): string | null {
  const mdPath = path.join(CONTENT_COMMANDS_DIR, `${command}.md`);
  const mdxPath = path.join(CONTENT_COMMANDS_DIR, `${command}.mdx`);

  if (fs.existsSync(mdPath)) {
    return `content/commands/${command}.md`;
  }
  if (fs.existsSync(mdxPath)) {
    return `content/commands/${command}.mdx`;
  }
  return null;
}

/**
 * Check if a command is covered in tour pages
 * Tour pages may cover commands within their content
 */
function checkTourPages(command: string): string | null {
  if (!fs.existsSync(TOUR_PAGES_DIR)) {
    return null;
  }

  const files = fs.readdirSync(TOUR_PAGES_DIR);

  for (const file of files) {
    if (file.endsWith(".mdx") || file.endsWith(".astro")) {
      const filePath = path.join(TOUR_PAGES_DIR, file);
      const content = fs.readFileSync(filePath, "utf-8");

      // Check if the tour page covers this command
      // Look for patterns like "azd app <command>" or command name in title/headings
      const patterns = [
        new RegExp(`azd\\s+app\\s+${command}\\b`, "i"),
        new RegExp(`#.*\\b${command}\\b`, "i"),
        new RegExp(`title:.*\\b${command}\\b`, "i"),
        new RegExp(`command:\\s*["']?${command}["']?`, "i"),
      ];

      for (const pattern of patterns) {
        if (pattern.test(content)) {
          return `pages/tour/${file}`;
        }
      }
    }
  }

  return null;
}

/**
 * Check if a command is covered in reference/cli pages (generated)
 */
function checkReferenceCli(command: string): string | null {
  if (!fs.existsSync(REFERENCE_CLI_DIR)) {
    return null;
  }

  const files = fs.readdirSync(REFERENCE_CLI_DIR);

  // Check for direct command page
  const possibleNames = [
    `${command}.astro`,
    `${command}.mdx`,
    `${command}.md`,
  ];

  for (const name of possibleNames) {
    if (files.includes(name)) {
      return `pages/reference/cli/${name}`;
    }
  }

  return null;
}

/**
 * Validate all commands have documentation coverage
 */
function validateCommands(): ValidationResult[] {
  const commands = getCliCommands();
  const excludedCommands = getExcludedCommands();
  const results: ValidationResult[] = [];

  for (const command of commands) {
    if (excludedCommands.has(command)) {
      console.log(`ℹ️  Skipping excluded command: ${command}`);
      continue;
    }

    let coverageLocation: string | null = null;

    // Check each coverage location
    coverageLocation = checkContentCommands(command);
    if (!coverageLocation) {
      coverageLocation = checkTourPages(command);
    }
    if (!coverageLocation) {
      coverageLocation = checkReferenceCli(command);
    }

    results.push({
      command,
      covered: coverageLocation !== null,
      coverageLocation: coverageLocation ?? undefined,
    });
  }

  return results;
}

/**
 * Main entry point
 */
function main(): void {
  console.log("🔍 Validating CLI command documentation coverage...\n");

  const results = validateCommands();

  if (results.length === 0) {
    console.log("⚠️  Warning: No commands found to validate");
    console.log("   This may be expected if cli/docs/commands/ is empty");
    process.exit(0);
  }

  const covered = results.filter((r) => r.covered);
  const missing = results.filter((r) => !r.covered);

  // Report covered commands
  if (covered.length > 0) {
    console.log(`✅ Commands with documentation (${covered.length}):`);
    for (const result of covered) {
      console.log(`   • ${result.command} → ${result.coverageLocation}`);
    }
    console.log();
  }

  // Report missing commands
  if (missing.length > 0) {
    console.log(`❌ Commands missing documentation (${missing.length}):`);
    for (const result of missing) {
      console.log(`   Missing documentation for command: ${result.command}`);
    }
    console.log();
    console.log("To fix this, add documentation in one of these locations:");
    console.log("   • web/src/content/commands/<command>.md");
    console.log("   • web/src/pages/tour/<step>.mdx (covering the command)");
    console.log("   • web/src/pages/reference/cli/<command>.astro");
    console.log();
    console.log("To exclude a command from validation, add it to:");
    console.log("   web/scripts/.exclude-commands");
    console.log();
    process.exit(1);
  }

  console.log(`✅ All ${results.length} commands have documentation coverage!`);
  process.exit(0);
}

main();
