/**
 * Screenshot I/O Module
 * 
 * Handles file system operations, directory management, image optimization,
 * and external process management.
 */

import * as fs from 'fs';
import * as path from 'path';
import { spawn, type ChildProcess, execSync } from 'child_process';

/**
 * Load azd environment variables from .azure/{env-name}/.env
 * Returns empty object if no default environment is found
 */
function loadAzdEnvironment(cwd: string): Record<string, string> {
  const azdDir = path.join(cwd, '.azure');
  if (!fs.existsSync(azdDir)) {
    return {};
  }

  // Find the default environment by checking .azure/{env-name}/.env files
  // The default is marked in .azure/config.json
  const configPath = path.join(azdDir, 'config.json');
  let defaultEnvName: string | undefined;
  
  if (fs.existsSync(configPath)) {
    try {
      const config = JSON.parse(fs.readFileSync(configPath, 'utf-8'));
      defaultEnvName = config.defaultEnvironment;
    } catch {
      // Ignore parse errors
    }
  }

  // If no default found in config, look for environments
  if (!defaultEnvName) {
    const envDirs = fs.readdirSync(azdDir, { withFileTypes: true })
      .filter(d => d.isDirectory())
      .map(d => d.name);
    
    // Check each for .env file and use the first one found
    for (const envDir of envDirs) {
      const envPath = path.join(azdDir, envDir, '.env');
      if (fs.existsSync(envPath)) {
        defaultEnvName = envDir;
        break;
      }
    }
  }

  if (!defaultEnvName) {
    return {};
  }

  const envPath = path.join(azdDir, defaultEnvName, '.env');
  if (!fs.existsSync(envPath)) {
    return {};
  }

  // Parse .env file
  const envVars: Record<string, string> = {};
  const envContent = fs.readFileSync(envPath, 'utf-8');
  
  for (const line of envContent.split('\n')) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) {
      continue;
    }
    
    const match = trimmed.match(/^([^=]+)=(.*)$/);
    if (match) {
      const key = match[1].trim();
      let value = match[2].trim();
      
      // Remove quotes if present
      if ((value.startsWith('"') && value.endsWith('"')) || 
          (value.startsWith("'") && value.endsWith("'"))) {
        value = value.slice(1, -1);
      }
      
      envVars[key] = value;
    }
  }

  console.log(`   ℹ️  Loaded ${Object.keys(envVars).length} environment variables from .azure/${defaultEnvName}/.env`);
  return envVars;
}

export async function ensureDir(dir: string): Promise<void> {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
}

export async function findAzdAppBinary(cliDir: string): Promise<string> {
  // Look for the built binary - prefer the NEWEST one to avoid stale binary issues
  const binDir = path.join(cliDir, 'bin');
  const isWindows = process.platform === 'win32';
  
  if (fs.existsSync(binDir)) {
    const files = fs.readdirSync(binDir);
    const ext = isWindows ? '.exe' : '';
    const platformArch = isWindows ? 'windows-amd64' : `${process.platform}-${process.arch === 'x64' ? 'amd64' : process.arch}`;
    
    // Find all matching binaries for this platform (excluding .old files)
    const candidates = files
      .filter(f => 
        (f === `azd-app${ext}` || f.includes(platformArch)) &&
        (isWindows ? f.endsWith('.exe') : !f.includes('.')) &&
        !f.endsWith('.old')
      )
      .map(f => ({
        name: f,
        path: path.join(binDir, f),
        mtime: fs.statSync(path.join(binDir, f)).mtime.getTime()
      }))
      .sort((a, b) => b.mtime - a.mtime); // Sort by newest first
    
    if (candidates.length > 0) {
      const newest = candidates[0];
      console.log(`  Found ${candidates.length} candidate(s), using newest: ${newest.name}`);
      if (candidates.length > 1) {
        const oldest = candidates[candidates.length - 1];
        const ageMinutes = Math.round((newest.mtime - oldest.mtime) / 60000);
        if (ageMinutes > 5) {
          console.log(`  ⚠️  Warning: ${oldest.name} is ${ageMinutes} minutes older - consider cleaning up stale binaries`);
        }
      }
      return newest.path;
    }
  }
  
  // Fall back to azd-app in PATH (if installed)
  return 'azd-app';
}

export async function optimizeImages(screenshotsDir: string): Promise<void> {
  console.log('\n🔧 Optimizing images...');

  // Check if sharp is available for optimization
  try {
    // Dynamic require to handle missing module gracefully
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const sharp = require('sharp');

    const files = fs.readdirSync(screenshotsDir).filter((f) => f.endsWith('.png'));

    for (const file of files) {
      const filePath = path.join(screenshotsDir, file);
      const originalSize = fs.statSync(filePath).size;

      // Optimize with sharp
      const optimized = await sharp(filePath)
        .png({ quality: 80, compressionLevel: 9 })
        .toBuffer();

      fs.writeFileSync(filePath, optimized);

      const newSize = fs.statSync(filePath).size;
      const savings = (((originalSize - newSize) / originalSize) * 100).toFixed(1);
      console.log(`  ✓ ${file}: ${(newSize / 1024).toFixed(1)} KB (${savings}% smaller)`);
    }
  } catch {
    console.log('  ⚠️ sharp not available, skipping optimization');
    console.log('  Install with: pnpm add -D sharp');
  }
}

export function startProcess(
  command: string,
  args: string[],
  cwd: string,
  name: string,
  onOutput?: (line: string) => void,
  processes?: ChildProcess[]
): ChildProcess {
  console.log(`🚀 Starting ${name}...`);
  console.log(`   Command: ${command} ${args.join(' ')}`);
  console.log(`   Dir: ${cwd}`);

  // Load azd environment variables from .azure/{env-name}/.env if it exists
  const azdEnv = loadAzdEnvironment(cwd);
  const mergedEnv = { ...process.env, ...azdEnv };

  const isWindows = process.platform === 'win32';
  const proc = spawn(command, args, {
    cwd,
    stdio: ['ignore', 'pipe', 'pipe'],
    shell: isWindows,
    detached: !isWindows,
    env: mergedEnv,
  });

  proc.stdout?.on('data', (data) => {
    const lines = data.toString().trim().split('\n');
    lines.forEach((line: string) => {
      if (line.trim()) {
        console.log(`   [${name}] ${line}`);
        onOutput?.(line);
      }
    });
  });

  proc.stderr?.on('data', (data) => {
    const lines = data.toString().trim().split('\n');
    lines.forEach((line: string) => {
      if (line.trim()) {
        console.log(`   [${name}] ${line}`);
        onOutput?.(line);
      }
    });
  });

  if (processes) {
    processes.push(proc);
  }
  return proc;
}

export function killProcess(proc: ChildProcess): void {
  if (!proc.killed) {
    const isWindows = process.platform === 'win32';
    if (isWindows) {
      // On Windows, use taskkill to kill process tree
      try {
        execSync(`taskkill /pid ${proc.pid} /T /F`, { stdio: 'ignore' });
      } catch {
        proc.kill('SIGTERM');
      }
    } else {
      // On Unix, kill the process group
      try {
        process.kill(-proc.pid!, 'SIGTERM');
      } catch {
        proc.kill('SIGTERM');
      }
    }
  }
}

export async function waitForUrl(url: string, timeout = 30000): Promise<boolean> {
  const start = Date.now();
  const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));
  
  while (Date.now() - start < timeout) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        return true;
      }
    } catch {
      // Service not ready yet
    }
    await sleep(500);
  }
  return false;
}
