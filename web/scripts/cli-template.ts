/**
 * CLI Template Module
 * 
 * Generates Astro template pages for CLI reference documentation.
 * Creates individual command pages and the main index/overview page.
 */

import type { CommandInfo } from './cli-parser.js';

const HIDDEN_FROM_INDEX = ['listen'];

function generateFlagsTable(command: CommandInfo): string {
  if (command.flags.length === 0) return '';
  const rows = command.flags.map(f => `<tr class="border-t border-neutral-200 dark:border-neutral-700"><td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400 bg-transparent">${f.flag}</code></td><td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">${f.short ? `<code class="bg-transparent">${f.short}</code>` : '-'}</td><td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">${f.type || '-'}</td><td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">${f.default || '-'}</td><td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">${f.description}</td></tr>`).join('');
  return `<h2 class="text-2xl font-bold mt-12 mb-4">Flags</h2><div class="overflow-x-auto my-8"><table class="min-w-full text-sm rounded-lg overflow-hidden border border-neutral-200 dark:border-neutral-700"><thead><tr class="bg-neutral-200 dark:bg-neutral-700"><th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Flag</th><th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Short</th><th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Type</th><th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Default</th><th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Description</th></tr></thead><tbody class="bg-neutral-100 dark:bg-neutral-800">${rows}</tbody></table></div>`;
}

function generateExamplesSection(command: CommandInfo): string {
  if (command.examples.length === 0) return '';
  return `<h2 class="text-2xl font-bold mt-12 mb-6">Examples</h2><div class="space-y-4">${command.examples.map((_, i) => `<Code code={example${i}} lang="bash" frame="terminal" />`).join('\n  ')}</div>`;
}

function generateDetailedDocsLink(command: CommandInfo): string {
  if (!command.hasDetailedDoc) return '';
  return `<div class="mt-12 p-6 bg-blue-100 dark:bg-blue-900/40 rounded-lg border border-blue-300 dark:border-blue-700"><h3 class="text-lg font-semibold mb-2 text-neutral-900 dark:text-neutral-100">📚 Detailed Documentation</h3><p class="text-neutral-700 dark:text-neutral-300 mb-4">For complete documentation including flows, diagrams, and advanced usage, see the full command specification.</p><a href="https://github.com/jongio/azd-app/blob/main/cli/docs/commands/${command.name}.md" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-2 text-blue-700 dark:text-blue-400 hover:underline">View full ${command.name} specification →</a></div>`;
}

export function generateCommandPage(command: CommandInfo): string {
  const exampleCodes = command.examples.map((e, i) => 
    `const example${i} = ${JSON.stringify(e.command)};`).join('\n');

  return `---
import Layout from '../../../components/Layout.astro';
import { Code } from 'astro-expressive-code/components';
const usageCode = ${JSON.stringify(command.usage)};
${exampleCodes}
---

<Layout title="${command.name} - CLI Reference" description="${command.description}">
  <div class="max-w-4xl mx-auto px-4 py-12">
    <nav class="text-sm mb-8">
      <ol class="flex items-center gap-2 text-neutral-500">
        <li><a href="/azd-app/" class="hover:text-blue-500">Home</a></li>
        <li>/</li>
        <li><a href="/azd-app/reference/cli/" class="hover:text-blue-500">CLI Reference</a></li>
        <li>/</li>
        <li class="text-neutral-900 dark:text-white">${command.name}</li>
      </ol>
    </nav>

    <div class="mb-8">
      <h1 class="text-4xl font-bold mb-4">azd app ${command.name}</h1>
      <p class="text-xl text-neutral-700 dark:text-neutral-300">${command.description}</p>
    </div>

    <h2 class="text-2xl font-bold mt-8 mb-4">Usage</h2>
    <Code code={usageCode} lang="bash" frame="terminal" />

    ${generateFlagsTable(command)}

    ${generateExamplesSection(command)}

    ${generateDetailedDocsLink(command)}

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

function generateGlobalFlagsTable(): string {
  const flags = [['--output', '-o', 'Output format (default, json)'], ['--debug', '-', 'Enable debug logging'], ['--structured-logs', '-', 'Enable structured JSON logging to stderr'], ['--cwd', '-C', 'Sets the current working directory']];
  const rows = flags.map(([flag, short, desc]) => `<tr class="border-t border-neutral-200 dark:border-neutral-700"><td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400 bg-transparent">${flag}</code></td><td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">${short !== '-' ? `<code class="bg-transparent">${short}</code>` : '-'}</td><td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">${desc}</td></tr>`).join('');
  return `<section class="mb-12"><h2 class="text-2xl font-bold mb-6">Global Flags</h2><p class="text-neutral-700 dark:text-neutral-300 mb-4">These flags are available for all commands:</p><div class="overflow-x-auto"><table class="min-w-full text-sm rounded-lg overflow-hidden border border-neutral-200 dark:border-neutral-700"><thead><tr class="bg-neutral-200 dark:bg-neutral-700"><th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Flag</th><th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Short</th><th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Description</th></tr></thead><tbody class="bg-neutral-100 dark:bg-neutral-800">${rows}</tbody></table></div></section>`;
}

function generateEnvVarsTable(): string {
  const vars = [['AZURE_SUBSCRIPTION_ID', 'Current Azure subscription'], ['AZURE_RESOURCE_GROUP_NAME', 'Target resource group'], ['AZURE_ENV_NAME', 'Environment name'], ['AZURE_LOCATION', 'Azure region']];
  const rows = vars.map(([name, desc]) => `<tr class="border-t border-neutral-200 dark:border-neutral-700"><td class="py-3 px-4"><code class="text-blue-600 dark:text-blue-400 bg-transparent">${name}</code></td><td class="py-3 px-4 text-neutral-700 dark:text-neutral-300">${desc}</td></tr>`).join('');
  return `<section class="mt-12"><h2 class="text-2xl font-bold mb-6">Environment Variables</h2><p class="text-neutral-700 dark:text-neutral-300 mb-4">When running through <code class="px-2 py-1 bg-neutral-100 dark:bg-neutral-800 rounded">azd app &lt;command&gt;</code>, these Azure environment variables are automatically available:</p><div class="overflow-x-auto"><table class="min-w-full text-sm rounded-lg overflow-hidden border border-neutral-200 dark:border-neutral-700"><thead><tr class="bg-neutral-200 dark:bg-neutral-700"><th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Variable</th><th class="text-left py-3 px-4 font-semibold text-neutral-900 dark:text-neutral-100">Description</th></tr></thead><tbody class="bg-neutral-100 dark:bg-neutral-800">${rows}</tbody></table></div></section>`;
}

export function generateIndexPage(commands: CommandInfo[]): string {
  const visibleCommands = commands.filter(cmd => !HIDDEN_FROM_INDEX.includes(cmd.name));
  const commandCards = visibleCommands.map(cmd => `
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
    <div class="mb-12">
      <h1 class="text-4xl font-bold mb-4">CLI Reference</h1>
      <p class="text-xl text-neutral-700 dark:text-neutral-300">
        Complete reference for all <code class="px-2 py-1 bg-neutral-100 dark:bg-neutral-800 rounded">azd app</code> commands and flags.
      </p>
    </div>

    ${generateGlobalFlagsTable()}

    <section>
      <h2 class="text-2xl font-bold mb-6">Commands</h2>
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        ${commandCards}
      </div>
    </section>

    <section class="mt-12">
      <h2 class="text-2xl font-bold mb-6">Quick Reference</h2>
      <Code code={quickRefCode} lang="bash" frame="terminal" />
    </section>

    ${generateEnvVarsTable()}

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
