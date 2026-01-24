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
 * Also tries to auto-detect Log Analytics workspace GUID if missing
 */
async function loadAzdEnvironment(cwd: string): Promise<Record<string, string>> {
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
  
  // If AZURE_LOG_ANALYTICS_WORKSPACE_GUID is missing, try to auto-detect it
  if (!envVars['AZURE_LOG_ANALYTICS_WORKSPACE_GUID'] && 
      envVars['AZURE_SUBSCRIPTION_ID'] && 
      envVars['AZURE_RESOURCE_GROUP']) {
    try {
      const workspaceGUID = await detectLogAnalyticsWorkspace(
        envVars['AZURE_SUBSCRIPTION_ID'],
        envVars['AZURE_RESOURCE_GROUP']
      );
      if (workspaceGUID) {
        envVars['AZURE_LOG_ANALYTICS_WORKSPACE_GUID'] = workspaceGUID;
        console.log(`   ℹ️  Auto-detected AZURE_LOG_ANALYTICS_WORKSPACE_GUID from Azure`);
      }
    } catch (error) {
      // Silently fail - auto-detection is optional
      console.log(`   ⚠️  Could not auto-detect Log Analytics workspace (Azure logs may not work)`);
    }
  }
  
  return envVars;
}

/**
 * Try to detect Log Analytics workspace GUID from Azure CLI
 * Returns the workspace GUID (customer ID) if found, empty string otherwise
 * Matches the implementation in cli/src/internal/azure/standalone_logs.go
 */
async function detectLogAnalyticsWorkspace(subscriptionId: string, resourceGroup: string): Promise<string> {
  try {
    const { spawnSync } = await import('child_process');
    
    // Build command matching Go implementation: az monitor log-analytics workspace list --resource-group <rg> --query "[0].customerId" -o tsv
    const args = [
      'monitor', 'log-analytics', 'workspace', 'list',
      '--resource-group', resourceGroup,
      '--query', '[0].customerId',
      '-o', 'tsv'
    ];
    
    // Add subscription if provided (for better reliability)
    if (subscriptionId) {
      args.splice(args.length - 2, 0, '--subscription', subscriptionId);
    }
    
    const result = spawnSync('az', args, {
      encoding: 'utf-8',
      stdio: ['pipe', 'pipe', 'pipe'],
      timeout: 30000 // 30 second timeout like Go code
    });
    
    if (result.error || result.status !== 0) {
      return '';
    }
    
    const output = (result.stdout || '').trim();
    
    // Validate it looks like a GUID (basic check - GUIDs are typically 36 chars with dashes)
    if (output && output.length > 0 && output !== 'None' && output !== 'null' && output.length >= 32) {
      return output;
    }
  } catch (error) {
    // Azure CLI command failed - workspace might not exist, not accessible, or not provisioned
    // This is expected for local-only projects or when Azure resources aren't deployed
  }
  
  return '';
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

export async function startProcess(
  command: string,
  args: string[],
  cwd: string,
  name: string,
  onOutput?: (line: string) => void,
  processes?: ChildProcess[]
): Promise<ChildProcess> {
  console.log(`🚀 Starting ${name}...`);
  console.log(`   Command: ${command} ${args.join(' ')}`);
  console.log(`   Dir: ${cwd}`);

  // Load azd environment variables from .azure/{env-name}/.env if it exists
  const azdEnv = await loadAzdEnvironment(cwd);
  const mergedEnv = { ...process.env, ...azdEnv };

  const isWindows = process.platform === 'win32';
  const proc = spawn(command, args, {
    cwd,
    stdio: ['pipe', 'pipe', 'pipe'], // Use 'pipe' for stdin to handle interactive prompts
    shell: isWindows,
    detached: !isWindows,
    env: mergedEnv,
  });

  // Handle port conflict prompts automatically
  let outputBuffer = '';
  let portConflictDetected = false;
  const handleOutput = (data: Buffer) => {
    outputBuffer += data.toString();
    const lines = outputBuffer.split('\n');
    outputBuffer = lines.pop() || ''; // Keep incomplete line in buffer
    
    lines.forEach((line: string) => {
      if (line.trim()) {
        console.log(`   [${name}] ${line}`);
        onOutput?.(line);
        
        // Auto-respond to port conflict prompts
        // Detect port conflict by looking for "requires port" message
        if (line.includes('requires port') && line.includes('configured in azure.yaml')) {
          portConflictDetected = true;
        }
        
        // When we see "Cancel" option, the prompt is complete and waiting for input
        if (portConflictDetected && line.includes('4) Cancel')) {
          // Option 2: Kill the process using the port (most reliable for screenshots)
          setTimeout(() => {
            if (proc.stdin && !proc.stdin.destroyed) {
              proc.stdin.write('2\n');
              console.log(`   [${name}] Auto-selected option 2: Kill the process using the port`);
              portConflictDetected = false;
            }
          }, 100); // Small delay to ensure prompt is ready
        }
      }
    });
  };

  proc.stdout?.on('data', handleOutput);
  proc.stderr?.on('data', handleOutput);

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
