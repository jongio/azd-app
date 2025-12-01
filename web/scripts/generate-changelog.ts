/**
 * Changelog Generator
 * 
 * Parses cli/CHANGELOG.md and generates version history pages:
 * - /reference/changelog/index.astro (full history)
 * - /reference/whats-new/index.astro (latest 3 versions)
 */

import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

interface ReleaseEntry {
  version: string;
  date: string;
  changes: ChangeEntry[];
  isUnreleased?: boolean;
}

interface ChangeEntry {
  type: 'feature' | 'fix' | 'breaking' | 'chore' | 'other';
  message: string;
  prNumber?: string;
  commitHash?: string;
}

const CHANGELOG_PATH = path.resolve(__dirname, '../..', 'cli/CHANGELOG.md');
const OUTPUT_DIR = path.resolve(__dirname, '../src/pages/reference');

function parseChangeType(line: string): ChangeEntry['type'] {
  const lowerLine = line.toLowerCase();
  if (lowerLine.includes('breaking')) return 'breaking';
  if (lowerLine.includes('feat:') || lowerLine.includes('feature')) return 'feature';
  if (lowerLine.includes('fix:') || lowerLine.includes('fixed')) return 'fix';
  if (lowerLine.includes('chore:') || lowerLine.includes('ci:') || lowerLine.includes('docs:')) return 'chore';
  return 'other';
}

function parseChangelog(content: string): ReleaseEntry[] {
  const releases: ReleaseEntry[] = [];
  const lines = content.split('\n');
  
  let currentRelease: ReleaseEntry | null = null;
  
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    
    // Match version headers like "## [0.6.1] - 2025-11-29" or "## [Unreleased]"
    const versionMatch = line.match(/^## \[([^\]]+)\](?:\s*-\s*(\d{4}-\d{2}-\d{2}))?/);
    
    if (versionMatch) {
      // Save previous release
      if (currentRelease && currentRelease.changes.length > 0) {
        releases.push(currentRelease);
      }
      
      const version = versionMatch[1];
      const date = versionMatch[2] || '';
      
      currentRelease = {
        version,
        date,
        changes: [],
        isUnreleased: version.toLowerCase() === 'unreleased'
      };
      continue;
    }
    
    // Match change items (lines starting with -)
    if (currentRelease && line.trim().startsWith('-')) {
      const changeText = line.trim().slice(1).trim();
      
      // Skip empty lines
      if (!changeText) continue;
      
      // Extract PR number if present
      const prMatch = changeText.match(/\(#(\d+)\)/);
      const prNumber = prMatch ? prMatch[1] : undefined;
      
      // Extract commit hash if present
      const commitMatch = changeText.match(/\(([a-f0-9]{7})\)$/);
      const commitHash = commitMatch ? commitMatch[1] : undefined;
      
      currentRelease.changes.push({
        type: parseChangeType(changeText),
        message: changeText,
        prNumber,
        commitHash
      });
    }
  }
  
  // Save last release
  if (currentRelease && currentRelease.changes.length > 0) {
    releases.push(currentRelease);
  }
  
  // Filter out duplicates and sort by version
  const seen = new Set<string>();
  return releases.filter(r => {
    if (seen.has(r.version)) return false;
    seen.add(r.version);
    return true;
  });
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

function formatChangeMessage(change: ChangeEntry): string {
  let message = escapeHtml(change.message);
  
  // Link PR numbers to GitHub
  if (change.prNumber) {
    message = message.replace(
      `(#${change.prNumber})`,
      `(<a href="https://github.com/jongio/azd-app/pull/${change.prNumber}" class="text-blue-600 dark:text-blue-400 hover:underline" target="_blank">#${change.prNumber}</a>)`
    );
  }
  
  // Link commit hashes to GitHub
  if (change.commitHash) {
    message = message.replace(
      `(${change.commitHash})`,
      `(<a href="https://github.com/jongio/azd-app/commit/${change.commitHash}" class="text-neutral-500 hover:underline" target="_blank">${change.commitHash}</a>)`
    );
  }
  
  return message;
}

function getTypeIcon(type: ChangeEntry['type']): string {
  switch (type) {
    case 'breaking': return '⚠️';
    case 'feature': return '✨';
    case 'fix': return '🐛';
    case 'chore': return '🔧';
    default: return '📝';
  }
}

function getTypeColor(type: ChangeEntry['type']): string {
  switch (type) {
    case 'breaking': return 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300';
    case 'feature': return 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300';
    case 'fix': return 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300';
    case 'chore': return 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400';
    default: return 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300';
  }
}

function generateReleaseHtml(release: ReleaseEntry, isLatest: boolean = false): string {
  const changes = release.changes.map(change => `
        <li class="flex items-start gap-3 py-2">
          <span class="shrink-0 text-lg">${getTypeIcon(change.type)}</span>
          <span class="text-neutral-700 dark:text-neutral-300">${formatChangeMessage(change)}</span>
        </li>`).join('');

  return `
    <article class="mb-12 ${isLatest ? 'pb-8 border-b-2 border-blue-200 dark:border-blue-800' : ''}">
      <header class="flex items-center gap-4 mb-4">
        <h2 class="text-2xl font-bold ${release.isUnreleased ? 'text-neutral-500' : 'text-neutral-900 dark:text-white'}">
          ${release.isUnreleased ? '🚧 Unreleased' : `v${release.version}`}
        </h2>
        ${release.date ? `<time class="text-sm text-neutral-500" datetime="${release.date}">${release.date}</time>` : ''}
        ${isLatest ? '<span class="px-2 py-1 text-xs font-medium bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 rounded">Latest</span>' : ''}
      </header>
      <ul class="space-y-1">
        ${changes}
      </ul>
    </article>`;
}

function generateChangelogPage(releases: ReleaseEntry[]): string {
  const releasesHtml = releases
    .filter(r => !r.isUnreleased)
    .map((r, i) => generateReleaseHtml(r, i === 0))
    .join('\n');

  return `---
import Layout from '../../../components/Layout.astro';
---

<Layout title="Changelog" description="Complete version history of azd-app CLI">
  <div class="max-w-4xl mx-auto px-4 py-12">
    <!-- Header -->
    <div class="mb-12">
      <h1 class="text-4xl font-bold mb-4">Changelog</h1>
      <p class="text-xl text-neutral-600 dark:text-neutral-400">
        Complete version history of the azd-app CLI extension.
      </p>
      <div class="mt-4 flex gap-4">
        <a 
          href="/azd-app/reference/whats-new/" 
          class="text-blue-600 dark:text-blue-400 hover:underline"
        >
          What's New (latest releases) →
        </a>
        <a 
          href="https://github.com/jongio/azd-app/releases" 
          target="_blank"
          rel="noopener noreferrer"
          class="text-neutral-600 dark:text-neutral-400 hover:underline"
        >
          GitHub Releases →
        </a>
      </div>
    </div>

    <!-- Release List -->
    <div class="space-y-8">
      ${releasesHtml}
    </div>
  </div>
</Layout>
`;
}

function generateWhatsNewPage(releases: ReleaseEntry[]): string {
  const latestReleases = releases
    .filter(r => !r.isUnreleased)
    .slice(0, 3);
    
  const releasesHtml = latestReleases
    .map((r, i) => generateReleaseHtml(r, i === 0))
    .join('\n');

  return `---
import Layout from '../../../components/Layout.astro';
---

<Layout title="What's New" description="Latest updates to azd-app CLI">
  <div class="max-w-4xl mx-auto px-4 py-12">
    <!-- Header -->
    <div class="mb-12">
      <h1 class="text-4xl font-bold mb-4">What's New</h1>
      <p class="text-xl text-neutral-600 dark:text-neutral-400">
        Latest updates and features in the azd-app CLI extension.
      </p>
      <div class="mt-4">
        <a 
          href="/azd-app/reference/changelog/" 
          class="text-blue-600 dark:text-blue-400 hover:underline"
        >
          View full changelog →
        </a>
      </div>
    </div>

    <!-- Highlights -->
    <div class="mb-12 p-6 bg-linear-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
      <h2 class="text-xl font-bold mb-4">🎉 Recent Highlights</h2>
      <ul class="space-y-2 text-neutral-700 dark:text-neutral-300">
        <li class="flex items-start gap-2">
          <span class="text-green-500">✓</span>
          <span><strong>MCP Server</strong> - AI assistants can now debug your services</span>
        </li>
        <li class="flex items-start gap-2">
          <span class="text-green-500">✓</span>
          <span><strong>Production Health Checks</strong> - Circuit breaker, rate limiting, caching</span>
        </li>
        <li class="flex items-start gap-2">
          <span class="text-green-500">✓</span>
          <span><strong>Azure Functions & Logic Apps</strong> - Full support for serverless workloads</span>
        </li>
        <li class="flex items-start gap-2">
          <span class="text-green-500">✓</span>
          <span><strong>Lifecycle Hooks</strong> - Run scripts before/after services start</span>
        </li>
      </ul>
    </div>

    <!-- Latest Releases -->
    <div class="space-y-8">
      ${releasesHtml}
    </div>

    <!-- CTA -->
    <div class="mt-12 p-6 bg-neutral-100 dark:bg-neutral-800 rounded-lg text-center">
      <h3 class="text-lg font-semibold mb-2">Ready to try it?</h3>
      <p class="text-neutral-600 dark:text-neutral-400 mb-4">
        Get started with the latest version of azd-app in minutes.
      </p>
      <a 
        href="/azd-app/quick-start/"
        class="inline-flex items-center gap-2 px-6 py-3 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 transition-colors"
      >
        Quick Start Guide →
      </a>
    </div>
  </div>
</Layout>
`;
}

async function main() {
  console.log('📋 Generating changelog pages...\n');
  
  // Read changelog
  if (!fs.existsSync(CHANGELOG_PATH)) {
    console.error('❌ CHANGELOG.md not found at:', CHANGELOG_PATH);
    process.exit(1);
  }
  
  const content = fs.readFileSync(CHANGELOG_PATH, 'utf-8');
  const releases = parseChangelog(content);
  
  console.log(`  ✓ Parsed ${releases.length} releases`);
  
  // Show latest 3
  const latest = releases.filter(r => !r.isUnreleased).slice(0, 3);
  for (const r of latest) {
    console.log(`    - v${r.version} (${r.date}): ${r.changes.length} changes`);
  }
  
  // Create output directories
  const changelogDir = path.join(OUTPUT_DIR, 'changelog');
  const whatsNewDir = path.join(OUTPUT_DIR, 'whats-new');
  
  if (!fs.existsSync(changelogDir)) {
    fs.mkdirSync(changelogDir, { recursive: true });
  }
  if (!fs.existsSync(whatsNewDir)) {
    fs.mkdirSync(whatsNewDir, { recursive: true });
  }
  
  // Generate pages
  fs.writeFileSync(path.join(changelogDir, 'index.astro'), generateChangelogPage(releases));
  console.log(`\n  ✓ Generated: reference/changelog/index.astro`);
  
  fs.writeFileSync(path.join(whatsNewDir, 'index.astro'), generateWhatsNewPage(releases));
  console.log(`  ✓ Generated: reference/whats-new/index.astro`);
  
  console.log(`\n✅ Generated 2 changelog pages`);
}

main().catch(err => {
  console.error('Error generating changelog:', err);
  process.exit(1);
});
