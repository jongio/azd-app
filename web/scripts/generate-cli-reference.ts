/**
 * CLI Reference Generator
 * 
 * Generates reference pages from cli/docs/ at build time.
 * Parses cli-reference.md and individual command docs to create:
 * - /reference/cli/index.astro (overview)
 * - /reference/cli/[command].astro (individual command pages)
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

// Commands to document (order matters for navigation)
const COMMANDS = [
  'reqs',
  'deps', 
  'run',
  'health',
  'logs',
  'info',
  'mcp',
  'notifications',
  'version'
];

function parseFlags(content: string): Flag[] {
  const flags: Flag[] = [];
  
  // Match flag tables
  const tableRegex = /\|\s*`([^`]+)`\s*\|\s*([^|]*)\|\s*([^|]*)\|\s*([^|]*)\|\s*([^|]*)\|/g;
  let match;
  
  while ((match = tableRegex.exec(content)) !== null) {
    const flag = match[1].trim();
    // Skip header row
    if (flag === 'Flag' || flag.includes('---')) continue;
    
    flags.push({
      flag,
      short: match[2].trim().replace(/`/g, ''),
      type: match[3].trim().replace(/`/g, ''),
      default: match[4].trim().replace(/`/g, ''),
      description: match[5].trim()
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
    
    for (const line of lines) {
      if (line.startsWith('#')) {
        continue; // Skip comments for now
      }
      if (line.startsWith('azd app')) {
        examples.push({
          description: '',
          command: line.trim()
        });
      }
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
  
  return {
    name: commandName,
    description,
    usage,
    flags: parseFlags(section),
    examples: parseExamples(section),
    hasDetailedDoc: fs.existsSync(path.join(COMMANDS_DIR, `${commandName}.md`))
  };
}

function generateCommandPage(command: CommandInfo): string {
  const flagsTable = command.flags.length > 0 ? `
<div class="overflow-x-auto my-8">
  <table class="min-w-full text-sm">
    <thead>
      <tr class="border-b border-neutral-200 dark:border-neutral-700">
        <th class="text-left py-3 px-4 font-semibold">Flag</th>
        <th class="text-left py-3 px-4 font-semibold">Short</th>
        <th class="text-left py-3 px-4 font-semibold">Type</th>
        <th class="text-left py-3 px-4 font-semibold">Default</th>
        <th class="text-left py-3 px-4 font-semibold">Description</th>
      </tr>
    </thead>
    <tbody>
      ${command.flags.map(f => `
      <tr class="border-b border-neutral-100 dark:border-neutral-800">
        <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400">${f.flag}</code></td>
        <td class="py-3 px-4">${f.short ? `<code>${f.short}</code>` : '-'}</td>
        <td class="py-3 px-4 text-neutral-600 dark:text-neutral-400">${f.type || '-'}</td>
        <td class="py-3 px-4 text-neutral-600 dark:text-neutral-400">${f.default || '-'}</td>
        <td class="py-3 px-4">${f.description}</td>
      </tr>`).join('')}
    </tbody>
  </table>
</div>` : '';

  const examplesSection = command.examples.length > 0 ? `
<h2 class="text-2xl font-bold mt-12 mb-6">Examples</h2>
<div class="space-y-4">
  ${command.examples.map(e => `
  <div class="bg-neutral-900 rounded-lg overflow-hidden">
    <div class="flex items-center justify-between px-4 py-2 bg-neutral-800">
      <span class="text-neutral-400 text-sm">bash</span>
      <button 
        class="copy-button text-neutral-400 hover:text-white text-sm"
        data-code="${e.command.replace(/"/g, '&quot;')}"
      >
        Copy
      </button>
    </div>
    <pre class="p-4 overflow-x-auto"><code class="text-green-400">${e.command}</code></pre>
  </div>`).join('')}
</div>` : '';

  return `---
import Layout from '../../../components/Layout.astro';
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
      <p class="text-xl text-neutral-600 dark:text-neutral-400">${command.description}</p>
    </div>

    <!-- Usage -->
    <h2 class="text-2xl font-bold mt-8 mb-4">Usage</h2>
    <div class="bg-neutral-900 rounded-lg p-4 overflow-x-auto">
      <code class="text-green-400">${command.usage}</code>
    </div>

    <!-- Flags -->
    ${command.flags.length > 0 ? '<h2 class="text-2xl font-bold mt-12 mb-4">Flags</h2>' : ''}
    ${flagsTable}

    <!-- Examples -->
    ${examplesSection}

    <!-- Link to detailed docs -->
    ${command.hasDetailedDoc ? `
    <div class="mt-12 p-6 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
      <h3 class="text-lg font-semibold mb-2">📚 Detailed Documentation</h3>
      <p class="text-neutral-600 dark:text-neutral-400 mb-4">
        For complete documentation including flows, diagrams, and advanced usage, see the full command specification.
      </p>
      <a 
        href="https://github.com/jongio/azd-app/blob/main/cli/docs/commands/${command.name}.md"
        target="_blank"
        rel="noopener noreferrer"
        class="inline-flex items-center gap-2 text-blue-600 dark:text-blue-400 hover:underline"
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
    <a href="/azd-app/reference/cli/${cmd.name}/" class="block p-6 bg-white dark:bg-neutral-800 rounded-lg border border-neutral-200 dark:border-neutral-700 hover:border-blue-500 dark:hover:border-blue-500 transition-colors">
      <div class="flex items-start justify-between mb-2">
        <code class="text-lg font-semibold text-blue-600 dark:text-blue-400">azd app ${cmd.name}</code>
        ${cmd.hasDetailedDoc ? '<span class="text-xs px-2 py-1 bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300 rounded">Full Docs</span>' : ''}
      </div>
      <p class="text-neutral-600 dark:text-neutral-400">${cmd.description}</p>
      <div class="mt-4 text-sm text-neutral-500">
        ${cmd.flags.length} flags • ${cmd.examples.length} examples
      </div>
    </a>`).join('\n');

  return `---
import Layout from '../../../components/Layout.astro';
---

<Layout title="CLI Reference" description="Complete reference for all azd app commands and flags">
  <div class="max-w-6xl mx-auto px-4 py-12">
    <!-- Header -->
    <div class="mb-12">
      <h1 class="text-4xl font-bold mb-4">CLI Reference</h1>
      <p class="text-xl text-neutral-600 dark:text-neutral-400">
        Complete reference for all <code class="px-2 py-1 bg-neutral-100 dark:bg-neutral-800 rounded">azd app</code> commands and flags.
      </p>
    </div>

    <!-- Global Flags -->
    <section class="mb-12">
      <h2 class="text-2xl font-bold mb-6">Global Flags</h2>
      <p class="text-neutral-600 dark:text-neutral-400 mb-4">
        These flags are available for all commands:
      </p>
      <div class="overflow-x-auto">
        <table class="min-w-full text-sm bg-white dark:bg-neutral-800 rounded-lg overflow-hidden">
          <thead>
            <tr class="bg-neutral-50 dark:bg-neutral-900">
              <th class="text-left py-3 px-4 font-semibold">Flag</th>
              <th class="text-left py-3 px-4 font-semibold">Short</th>
              <th class="text-left py-3 px-4 font-semibold">Description</th>
            </tr>
          </thead>
          <tbody>
            <tr class="border-t border-neutral-100 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400">--output</code></td>
              <td class="py-3 px-4"><code>-o</code></td>
              <td class="py-3 px-4">Output format (default, json)</td>
            </tr>
            <tr class="border-t border-neutral-100 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400">--debug</code></td>
              <td class="py-3 px-4">-</td>
              <td class="py-3 px-4">Enable debug logging</td>
            </tr>
            <tr class="border-t border-neutral-100 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400">--structured-logs</code></td>
              <td class="py-3 px-4">-</td>
              <td class="py-3 px-4">Enable structured JSON logging to stderr</td>
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
      <div class="bg-neutral-900 rounded-lg p-6 overflow-x-auto">
        <pre class="text-sm"><code class="text-green-400"># Check prerequisites
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
azd app mcp serve</code></pre>
      </div>
    </section>

    <!-- Environment Variables -->
    <section class="mt-12">
      <h2 class="text-2xl font-bold mb-6">Environment Variables</h2>
      <p class="text-neutral-600 dark:text-neutral-400 mb-4">
        When running through <code class="px-2 py-1 bg-neutral-100 dark:bg-neutral-800 rounded">azd app &lt;command&gt;</code>, 
        these Azure environment variables are automatically available:
      </p>
      <div class="overflow-x-auto">
        <table class="min-w-full text-sm bg-white dark:bg-neutral-800 rounded-lg overflow-hidden">
          <thead>
            <tr class="bg-neutral-50 dark:bg-neutral-900">
              <th class="text-left py-3 px-4 font-semibold">Variable</th>
              <th class="text-left py-3 px-4 font-semibold">Description</th>
            </tr>
          </thead>
          <tbody>
            <tr class="border-t border-neutral-100 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400">AZURE_SUBSCRIPTION_ID</code></td>
              <td class="py-3 px-4">Current Azure subscription</td>
            </tr>
            <tr class="border-t border-neutral-100 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400">AZURE_RESOURCE_GROUP_NAME</code></td>
              <td class="py-3 px-4">Target resource group</td>
            </tr>
            <tr class="border-t border-neutral-100 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400">AZURE_ENV_NAME</code></td>
              <td class="py-3 px-4">Environment name</td>
            </tr>
            <tr class="border-t border-neutral-100 dark:border-neutral-700">
              <td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400">AZURE_LOCATION</code></td>
              <td class="py-3 px-4">Azure region</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- MCP Integration -->
    <section class="mt-12 p-6 bg-gradient-to-r from-purple-50 to-blue-50 dark:from-purple-900/20 dark:to-blue-900/20 rounded-lg border border-purple-200 dark:border-purple-800">
      <div class="flex items-start gap-4">
        <span class="text-3xl">🤖</span>
        <div>
          <h3 class="text-xl font-bold mb-2">AI-Powered Debugging with MCP</h3>
          <p class="text-neutral-600 dark:text-neutral-400 mb-4">
            The <code class="px-2 py-1 bg-white/50 dark:bg-neutral-800 rounded">azd app mcp</code> command 
            enables AI assistants like GitHub Copilot to interact with your running services.
          </p>
          <a href="/azd-app/mcp/" class="inline-flex items-center gap-2 text-purple-600 dark:text-purple-400 font-medium hover:underline">
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
  
  // Parse each command
  const commands: CommandInfo[] = [];
  
  for (const cmdName of COMMANDS) {
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
