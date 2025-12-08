/**
 * CLI Reference Generator
 * 
 * Generates reference pages from cli/docs/ at build time.
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

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

interface CommandInfo {
  name: string;
  description: string;
  usage: string;
  flags: Flag[];
  examples: Example[];
  hasDetailedDoc: boolean;
  specContent: string;
}

interface Flag {
  flag: string;
  short: string;
  type: string;
  default: string;
  description: string;
}

interface Example {
  description: string;
  command: string;
}

const CLI_DOCS_DIR = path.resolve(__dirname, '../..', 'cli/docs');
const COMMANDS_DIR = path.join(CLI_DOCS_DIR, 'commands');
const OUTPUT_DIR = path.resolve(__dirname, '../src/pages/reference/cli');
const CONTENT_DIR = path.resolve(__dirname, '../src/content/cli-reference');

// Commands to exclude from documentation (internal/hidden commands)
const EXCLUDED_COMMANDS = ['listen'];

/**
 * Discovers commands dynamically from cli-reference.md Commands Overview table
 * and cli/docs/commands/ directory.
 */
function discoverCommands(cliReference: string): string[] {
  const commands = new Set<string>();
  
  // Method 1: Scan cli/docs/commands/ directory for .md files (most reliable)
  if (fs.existsSync(COMMANDS_DIR)) {
    const files = fs.readdirSync(COMMANDS_DIR);
    for (const file of files) {
      if (file.endsWith('.md')) {
        const cmdName = file.replace('.md', '');
        if (!EXCLUDED_COMMANDS.includes(cmdName)) {
          commands.add(cmdName);
        }
      }
    }
  }
  
  // Method 2: Find all ## `azd app <command>` section headers in cli-reference.md
  const sectionRegex = /^## `azd app (\w+)`/gm;
  let match;
  while ((match = sectionRegex.exec(cliReference)) !== null) {
    const cmdName = match[1];
    if (!EXCLUDED_COMMANDS.includes(cmdName)) {
      commands.add(cmdName);
    }
  }
  
  // Sort commands: priority order first, then alphabetically
  const priorityOrder = ['reqs', 'deps', 'run', 'start', 'stop', 'restart', 'health', 'logs', 'info', 'test'];
  const sortedCommands = Array.from(commands).sort((a, b) => {
    const aIndex = priorityOrder.indexOf(a);
    const bIndex = priorityOrder.indexOf(b);
    if (aIndex !== -1 && bIndex !== -1) return aIndex - bIndex;
    if (aIndex !== -1) return -1;
    if (bIndex !== -1) return 1;
    return a.localeCompare(b);
  });
  
  return sortedCommands;
}

function parseFlags(content: string): Flag[] {
  const flags: Flag[] = [];
  
  // Only parse tables in the ### Flags section
  const flagsSectionMatch = content.match(/### Flags\s*([\s\S]*?)(?=###|$)/);
  if (!flagsSectionMatch) return flags;
  
  const flagsSection = flagsSectionMatch[1];
  
  // Match 5-column flag tables (Flag | Short | Type | Default | Description)
  const table5ColRegex = /\|\s*`([^`]+)`\s*\|\s*(`[^`]*`|)\s*\|\s*([^|]*)\|\s*([^|]*)\|\s*([^|]*)\|/g;
  let match;
  
  while ((match = table5ColRegex.exec(flagsSection)) !== null) {
    const flag = match[1].trim();
    // Skip header row or if it doesn't look like a flag
    if (flag === 'Flag' || flag.includes('---') || !flag.startsWith('--')) continue;
    
    flags.push({
      flag,
      short: match[2].trim().replace(/`/g, ''),
      type: match[3].trim().replace(/`/g, ''),
      default: match[4].trim().replace(/`/g, ''),
      description: match[5].trim()
    });
  }
  
  // Match 4-column flag tables (Flag | Type | Default | Description) - no Short column
  const table4ColRegex = /\|\s*`(--[^`]+)`\s*\|\s*([^|]*)\|\s*([^|]*)\|\s*([^|]*)\|/g;
  
  while ((match = table4ColRegex.exec(flagsSection)) !== null) {
    const flag = match[1].trim();
    // Skip header row
    if (flag === 'Flag' || flag.includes('---')) continue;
    // Skip if we already have this flag (from 5-column table)
    if (flags.some(f => f.flag === flag)) continue;
    
    flags.push({
      flag,
      short: '',
      type: match[2].trim().replace(/`/g, ''),
      default: match[3].trim().replace(/`/g, ''),
      description: match[4].trim()
    });
  }
  
  return flags;
}

function parseExamples(content: string): Example[] {
  const examples: Example[] = [];
  
  // Match code blocks with comments
  const exampleRegex = /```bash\n([\s\S]*?)```/g;
  let match;
  
  while ((match = exampleRegex.exec(content)) !== null) {
    const block = match[1].trim();
    const lines = block.split('\n');
    
    let i = 0;
    while (i < lines.length) {
      const line = lines[i];
      if (line.startsWith('#')) {
        i++;
        continue; // Skip comments
      }
      if (line.startsWith('azd app')) {
        // Handle multi-line commands with backslash continuation
        let fullCommand = line.trim();
        while (fullCommand.endsWith('\\') && i + 1 < lines.length) {
          i++;
          fullCommand = fullCommand.slice(0, -1).trim() + ' ' + lines[i].trim();
        }
        examples.push({
          description: '',
          command: fullCommand
        });
      }
      i++;
    }
  }
  
  return examples.slice(0, 10); // Limit to 10 examples
}

function parseCommandFromReference(content: string, commandName: string): CommandInfo | null {
  // Find the command section
  const sectionRegex = new RegExp(`## \`azd app ${commandName}\`([\\s\\S]*?)(?=## \`azd app |## Exit Codes|$)`);
  const match = content.match(sectionRegex);
  
  if (!match) return null;
  
  const section = match[1];
  
  // Extract description (first paragraph after heading)
  const descMatch = section.match(/\n\n([^#\n][^\n]+)/);
  const description = descMatch ? descMatch[1].trim() : '';
  
  // Extract usage
  const usageMatch = section.match(/```bash\nazd app (\w+)([^`]*)?```/);
  const usage = usageMatch ? `azd app ${usageMatch[1]}${usageMatch[2] || ''}`.trim() : `azd app ${commandName} [flags]`;
  
  // Read full spec content if available
  const specPath = path.join(COMMANDS_DIR, `${commandName}.md`);
  const hasDetailedDoc = fs.existsSync(specPath);
  const specContent = hasDetailedDoc ? fs.readFileSync(specPath, 'utf-8') : '';
  
  return {
    name: commandName,
    description,
    usage,
    flags: parseFlags(section),
    examples: parseExamples(section),
    hasDetailedDoc,
    specContent
  };
}

function generateCommandPage(command: CommandInfo): string {
  // Generate the flag rows at generation time, not runtime
  const flagRows = command.flags.map(f => `
      <tr class="border-t border-neutral-200 dark:border-neutral-700">
        <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400 bg-transparent">${f.flag}</code></td>
        <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">${f.short ? `<code class="bg-transparent">${f.short}</code>` : '-'}</td>
        <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">${f.type || '-'}</td>
        <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">${f.default || '-'}</td>
        <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">${f.description}</td>
      </tr>`).join('');

  const flagsTable = command.flags.length > 0 ? `
<div class="overflow-x-auto my-8">
  <table class="min-w-full text-sm rounded-lg overflow-hidden border border-neutral-200 dark:border-neutral-700">
    <thead>
      <tr class="bg-neutral-200 dark:bg-neutral-700">
        <th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Flag</th>
        <th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Short</th>
        <th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Type</th>
        <th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Default</th>
        <th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Description</th>
      </tr>
    </thead>
    <tbody class="bg-neutral-100 dark:bg-neutral-800">${flagRows}
    </tbody>
  </table>
</div>` : '';

  // Create example code variables in frontmatter
  const exampleCodes = command.examples.map((e, i) => 
    `const example${i} = ${JSON.stringify(e.command)};`
  ).join('\n');

  const examplesSection = command.examples.length > 0 ? `
<h2 class="text-2xl font-bold mt-12 mb-6">Examples</h2>
<div class="space-y-4">
  ${command.examples.map((_, i) => `<Code code={example${i}} lang="bash" frame="terminal" />`).join('\n  ')}
</div>` : '';

  // Create usage code variable in frontmatter
  const usageCode = `const usageCode = ${JSON.stringify(command.usage)};`;

  return `---
import Layout from '../../../components/Layout.astro';
import { Code } from 'astro-expressive-code/components';
${usageCode}
${exampleCodes}
---

<Layout title="${command.name} - CLI Reference" description="${command.description}">
  <div class="max-w-4xl mx-auto px-4 py-12">
    <!-- Breadcrumb -->
    <nav class="text-sm mb-8">
      <ol class="flex items-center gap-2 text-neutral-500">
        <li><a href="/azd-app/" class="hover:text-blue-500">Home</a></li>
        <li>/</li>
        <li><a href="/azd-app/reference/cli/" class="hover:text-blue-500">CLI Reference</a></li>
        <li>/</li>
        <li class="text-neutral-900 dark:text-white">${command.name}</li>
      </ol>
    </nav>

    <!-- Header -->
    <div class="mb-8">
      <h1 class="text-4xl font-bold mb-4">azd app ${command.name}</h1>
      <p class="text-xl text-neutral-700 dark:text-neutral-300">${command.description}</p>
    </div>

    <!-- Usage -->
    <h2 class="text-2xl font-bold mt-8 mb-4">Usage</h2>
    <Code code={usageCode} lang="bash" frame="terminal" />

    <!-- Flags -->
    ${command.flags.length > 0 ? '<h2 class="text-2xl font-bold mt-12 mb-4">Flags</h2>' : ''}
    ${flagsTable}

    <!-- Examples -->
    ${examplesSection}

    <!-- Link to detailed docs -->
    ${command.hasDetailedDoc ? `
    <div class="mt-12 p-6 bg-blue-100 dark:bg-blue-900/40 rounded-lg border border-blue-300 dark:border-blue-700">
      <h3 class="text-lg font-semibold mb-2 text-neutral-900 dark:text-neutral-100">📚 Detailed Documentation</h3>
      <p class="text-neutral-700 dark:text-neutral-300 mb-4">
        For complete documentation including flows, diagrams, and advanced usage, see the full command specification.
      </p>
      <a 
        href="https://github.com/jongio/azd-app/blob/main/cli/docs/commands/${command.name}.md"
        target="_blank"
        rel="noopener noreferrer"
        class="inline-flex items-center gap-2 text-blue-700 dark:text-blue-400 hover:underline"
      >
        View full ${command.name} specification →
      </a>
    </div>` : ''}

    <!-- Navigation -->
    <div class="mt-12 pt-8 border-t border-neutral-200 dark:border-neutral-700">
      <div class="flex justify-between">
        <a href="/azd-app/reference/cli/" class="text-blue-600 dark:text-blue-400 hover:underline">
          ← Back to CLI Reference
        </a>
      </div>
    </div>
  </div>

  <script>
    document.querySelectorAll('.copy-button').forEach(button => {
      button.addEventListener('click', async () => {
        const code = button.getAttribute('data-code');
        if (code) {
          await navigator.clipboard.writeText(code);
          button.textContent = 'Copied!';
          setTimeout(() => { button.textContent = 'Copy'; }, 2000);
        }
      });
    });
  </script>
</Layout>
`;
}

function generateIndexPage(commands: CommandInfo[]): string {
  const commandCards = commands.map(cmd => `
    <a href="/azd-app/reference/cli/${cmd.name}/" class="block p-6 bg-neutral-100 dark:bg-neutral-800 rounded-lg border border-neutral-200 dark:border-neutral-700 hover:border-blue-500 dark:hover:border-blue-500 transition-colors">
      <div class="flex items-start justify-between mb-2">
        <code class="text-lg font-semibold text-blue-600 dark:text-blue-400">azd app ${cmd.name}</code>
        ${cmd.hasDetailedDoc ? '<span class="text-xs px-2 py-1 bg-green-200 dark:bg-green-900 text-green-800 dark:text-green-300 rounded">Full Docs</span>' : ''}
      </div>
      <p class="text-neutral-700 dark:text-neutral-300">${cmd.description}</p>
      <div class="mt-4 text-sm text-neutral-600 dark:text-neutral-400">
        ${cmd.flags.length} flags • ${cmd.examples.length} examples
      </div>
    </a>`).join('\n');

  return `---
import Layout from '../../../components/Layout.astro';
import { Code } from 'astro-expressive-code/components';

const quickRefCode = \`# Check prerequisites
azd app reqs

# Install dependencies
azd app deps

# Start development environment
azd app run

# Monitor service health
azd app health --stream

# View logs
azd app logs --follow

# Show running services
azd app info

# Start MCP server for AI debugging
azd app mcp serve\`;
---

<Layout title="CLI Reference" description="Complete reference for all azd app commands and flags">
  <div class="max-w-6xl mx-auto px-4 py-12">
    <!-- Header -->
    <div class="mb-12">
      <h1 class="text-4xl font-bold mb-4">CLI Reference</h1>
      <p class="text-xl text-neutral-700 dark:text-neutral-300">
        Complete reference for all <code class="px-2 py-1 bg-neutral-100 dark:bg-neutral-800 rounded">azd app</code> commands and flags.
      </p>
    </div>

    <!-- Global Flags -->
    <section class="mb-12">
      <h2 class="text-2xl font-bold mb-6">Global Flags</h2>
      <p class="text-neutral-700 dark:text-neutral-300 mb-4">
        These flags are available for all commands:
      </p>
      <div class="overflow-x-auto">
        <table class="min-w-full text-sm rounded-lg overflow-hidden border border-neutral-200 dark:border-neutral-700">
          <thead>
            <tr class="bg-neutral-200 dark:bg-neutral-700">
              <th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Flag</th>
              <th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Short</th>
              <th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Description</th>
            </tr>
          </thead>
          <tbody class="bg-neutral-100 dark:bg-neutral-800">
            <tr class="border-t border-neutral-200 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400 bg-transparent">--output</code></td>
              <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300"><code class="bg-transparent">-o</code></td>
              <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">Output format (default, json)</td>
            </tr>
            <tr class="border-t border-neutral-200 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400 bg-transparent">--debug</code></td>
              <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">-</td>
              <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">Enable debug logging</td>
            </tr>
            <tr class="border-t border-neutral-200 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400 bg-transparent">--structured-logs</code></td>
              <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">-</td>
              <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">Enable structured JSON logging to stderr</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Commands -->
    <section>
      <h2 class="text-2xl font-bold mb-6">Commands</h2>
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        ${commandCards}
      </div>
    </section>

    <!-- Quick Reference -->
    <section class="mt-12">
      <h2 class="text-2xl font-bold mb-6">Quick Reference</h2>
      <Code code={quickRefCode} lang="bash" frame="terminal" />
    </section>

    <!-- Environment Variables -->
    <section class="mt-12">
      <h2 class="text-2xl font-bold mb-6">Environment Variables</h2>
      <p class="text-neutral-700 dark:text-neutral-300 mb-4">
        When running through <code class="px-2 py-1 bg-neutral-100 dark:bg-neutral-800 rounded">azd app &lt;command&gt;</code>, 
        these Azure environment variables are automatically available:
      </p>
      <div class="overflow-x-auto">
        <table class="min-w-full text-sm rounded-lg overflow-hidden border border-neutral-200 dark:border-neutral-700">
          <thead>
            <tr class="bg-neutral-200 dark:bg-neutral-700">
              <th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Variable</th>
              <th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Description</th>
            </tr>
          </thead>
          <tbody class="bg-neutral-100 dark:bg-neutral-800">
            <tr class="border-t border-neutral-200 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400 bg-transparent">AZURE_SUBSCRIPTION_ID</code></td>
              <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">Current Azure subscription</td>
            </tr>
            <tr class="border-t border-neutral-200 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400 bg-transparent">AZURE_RESOURCE_GROUP_NAME</code></td>
              <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">Target resource group</td>
            </tr>
            <tr class="border-t border-neutral-200 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400 bg-transparent">AZURE_ENV_NAME</code></td>
              <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">Environment name</td>
            </tr>
            <tr class="border-t border-neutral-200 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400 bg-transparent">AZURE_LOCATION</code></td>
              <td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">Azure region</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- MCP Integration -->
    <section class="mt-12 p-6 bg-purple-100 dark:bg-purple-900/40 rounded-lg border border-purple-300 dark:border-purple-700">
      <div class="flex items-start gap-4">
        <span class="text-3xl">🤖</span>
        <div>
          <h3 class="text-xl font-bold mb-2 text-neutral-900 dark:text-neutral-100">AI-Powered Debugging with MCP</h3>
          <p class="text-neutral-700 dark:text-neutral-300 mb-4">
            The <code class="px-2 py-1 bg-purple-200/50 dark:bg-purple-800/50 rounded">azd app mcp</code> command 
            enables AI assistants like GitHub Copilot to interact with your running services.
          </p>
          <a href="/azd-app/mcp/" class="inline-flex items-center gap-2 text-purple-700 dark:text-purple-400 font-medium hover:underline">
            Learn about MCP integration →
          </a>
        </div>
      </div>
    </section>
  </div>
</Layout>
`;
}

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
  const discoveredCommands = discoverCommands(cliReference);
  console.log(`  📋 Discovered ${discoveredCommands.length} commands: ${discoveredCommands.join(', ')}\n`);
  
  // Parse each command
  const commands: CommandInfo[] = [];
  
  for (const cmdName of discoveredCommands) {
    const cmd = parseCommandFromReference(cliReference, cmdName);
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
