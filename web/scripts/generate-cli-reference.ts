/**
 * CLI Reference Generator
 * 
 * Main orchestration script that generates reference pages from cli/docs/ at build time.
 * Parses cli-reference.md and individual command docs to create:
 * - /reference/cli/index.astro (overview)
 * - /reference/cli/[command].astro (individual command pages)
 * 
 * Commands are discovered dynamically from:
 * 1. The "Commands Overview" table in cli-reference.md
 * 2. Individual .md files in cli/docs/commands/
 */

import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';
import { discoverCommands, parseCommandFromReference, type CommandInfo } from './cli-parser.js';
import { generateCommandPage, generateIndexPage } from './cli-template.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const CLI_DOCS_DIR = path.resolve(__dirname, '../..', 'cli/docs');
const COMMANDS_DIR = path.join(CLI_DOCS_DIR, 'commands');
const OUTPUT_DIR = path.resolve(__dirname, '../src/pages/reference/cli');



async function main() {
  console.log('🔧 Generating CLI reference pages...\n');
  
  // Read the main CLI reference
  const cliReferencePath = path.join(CLI_DOCS_DIR, 'cli-reference.md');
  if (!fs.existsSync(cliReferencePath)) {
    console.error('❌ cli-reference.md not found at:', cliReferencePath);
    process.exit(1);
  }
  
  const cliReference = fs.readFileSync(cliReferencePath, 'utf-8');
  
  // Discover commands dynamically
  const discoveredCommands = discoverCommands(cliReference, COMMANDS_DIR);
  console.log(`  📋 Discovered ${discoveredCommands.length} commands: ${discoveredCommands.join(', ')}\n`);
  
  // Parse each command
  const commands: CommandInfo[] = [];
  
  for (const cmdName of discoveredCommands) {
    const cmd = parseCommandFromReference(cliReference, cmdName, COMMANDS_DIR);
    if (cmd) {
      commands.push(cmd);
      console.log(`  ✓ Parsed: ${cmdName} (${cmd.flags.length} flags, ${cmd.examples.length} examples)`);
    } else {
      console.log(`  ⚠ Skipped: ${cmdName} (not found in cli-reference.md)`);
    }
  }
  
  // Ensure output directory exists
  if (!fs.existsSync(OUTPUT_DIR)) {
    fs.mkdirSync(OUTPUT_DIR, { recursive: true });
  }
  
  // Generate index page
  const indexPage = generateIndexPage(commands);
  fs.writeFileSync(path.join(OUTPUT_DIR, 'index.astro'), indexPage);
  console.log(`\n  ✓ Generated: reference/cli/index.astro`);
  
  // Generate individual command pages
  for (const cmd of commands) {
    const page = generateCommandPage(cmd);
    fs.writeFileSync(path.join(OUTPUT_DIR, `${cmd.name}.astro`), page);
    console.log(`  ✓ Generated: reference/cli/${cmd.name}.astro`);
  }
  
  console.log(`\n✅ Generated ${commands.length + 1} CLI reference pages`);
}

main().catch(err => {
  console.error('Error generating CLI reference:', err);
  process.exit(1);
});
