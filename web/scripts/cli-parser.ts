/**
 * CLI Parser Module
 * 
 * Parses CLI documentation markdown files to extract command information,
 * flags, examples, and metadata. Reusable across different CLI documentation formats.
 */

import * as fs from 'fs';
import * as path from 'path';

export interface CommandInfo {
  name: string;
  description: string;
  usage: string;
  flags: Flag[];
  examples: Example[];
  hasDetailedDoc: boolean;
  specContent: string;
}

export interface Flag {
  flag: string;
  short: string;
  type: string;
  default: string;
  description: string;
}

export interface Example {
  description: string;
  command: string;
}

/**
 * Discovers commands dynamically from cli-reference.md Commands Overview table
 * and cli/docs/commands/ directory.
 */
export function discoverCommands(cliReference: string, commandsDir: string): string[] {
  const commands = new Set<string>();
  
  // Method 1: Scan cli/docs/commands/ directory for .md files (most reliable)
  if (fs.existsSync(commandsDir)) {
    const files = fs.readdirSync(commandsDir);
    for (const file of files) {
      if (file.endsWith('.md')) {
        const cmdName = file.replace('.md', '');
        commands.add(cmdName);
      }
    }
  }
  
  // Method 2: Find all ## `azd app <command>` section headers in cli-reference.md
  const sectionRegex = /^## `azd app (\w+)`/gm;
  let match;
  while ((match = sectionRegex.exec(cliReference)) !== null) {
    const cmdName = match[1];
    commands.add(cmdName);
  }
  
  // Sort commands: priority order first, then alphabetically
  const priorityOrder = ['reqs', 'deps', 'add', 'run', 'start', 'stop', 'restart', 'health', 'logs', 'info', 'test'];
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

/**
 * Parses flag tables from markdown content.
 * Handles both 5-column (Flag | Short | Type | Default | Description)
 * and 4-column (Flag | Type | Default | Description) formats.
 */
export function parseFlags(content: string): Flag[] {
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

/**
 * Parses example code blocks from markdown content.
 * Extracts bash commands, handling multi-line commands with backslash continuation.
 */
export function parseExamples(content: string): Example[] {
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

/**
 * Parses command information from the main cli-reference.md file.
 * Extracts description, usage, flags, and examples for a specific command.
 */
export function parseCommandFromReference(
  content: string, 
  commandName: string,
  commandsDir: string
): CommandInfo | null {
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
  const specPath = path.join(commandsDir, `${commandName}.md`);
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
