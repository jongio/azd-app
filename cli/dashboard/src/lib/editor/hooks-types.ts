/**
 * Hooks Types - Type definitions for Azure YAML lifecycle hooks
 */

/**
 * Lifecycle event types supported by Azure DevCenter
 */
export type LifecycleEvent =
  | 'preprovision'
  | 'postprovision'
  | 'preinfracreate'
  | 'postinfracreate'
  | 'preinfradelete'
  | 'postinfradelete'
  | 'predown'
  | 'postdown'
  | 'preup'
  | 'postup'
  | 'prepackage'
  | 'postpackage'
  | 'prepublish'
  | 'postpublish'
  | 'predeploy'
  | 'postdeploy'
  | 'prerestore'
  | 'postrestore'
  | 'prerun'
  | 'postrun'

/**
 * Shell types supported by hooks
 */
export type ShellType = 'sh' | 'bash' | 'pwsh' | 'powershell' | 'cmd'

/**
 * Platform types for platform-specific overrides
 */
export type Platform = 'windows' | 'posix' | 'all'

/**
 * Platform-specific hook override
 */
export interface PlatformHookOverride {
  /** Script or command to execute */
  run?: string
  
  /** Shell to use for execution */
  shell?: ShellType
  
  /** Continue on error */
  continueOnError?: boolean
  
  /** Interactive mode */
  interactive?: boolean
}

/**
 * Hook configuration
 */
export interface HookConfig {
  /** Script or command to execute */
  run?: string
  
  /** Shell to use for execution */
  shell?: ShellType
  
  /** Continue on error */
  continueOnError?: boolean
  
  /** Interactive mode */
  interactive?: boolean
  
  /** Windows-specific override */
  windows?: PlatformHookOverride
  
  /** POSIX (Linux/macOS) override */
  posix?: PlatformHookOverride
}

/**
 * Hooks configuration (all lifecycle events)
 */
export interface HooksConfig {
  preprovision?: HookConfig | HookConfig[]
  postprovision?: HookConfig | HookConfig[]
  preinfracreate?: HookConfig | HookConfig[]
  postinfracreate?: HookConfig | HookConfig[]
  preinfradelete?: HookConfig | HookConfig[]
  postinfradelete?: HookConfig | HookConfig[]
  predown?: HookConfig | HookConfig[]
  postdown?: HookConfig | HookConfig[]
  preup?: HookConfig | HookConfig[]
  postup?: HookConfig | HookConfig[]
  prepackage?: HookConfig | HookConfig[]
  postpackage?: HookConfig | HookConfig[]
  prepublish?: HookConfig | HookConfig[]
  postpublish?: HookConfig | HookConfig[]
  predeploy?: HookConfig | HookConfig[]
  postdeploy?: HookConfig | HookConfig[]
  prerestore?: HookConfig | HookConfig[]
  postrestore?: HookConfig | HookConfig[]
  prerun?: HookConfig | HookConfig[]
  postrun?: HookConfig | HookConfig[]
}

/**
 * Form data for hook editor
 */
export interface HookFormData {
  /** Lifecycle event */
  event: LifecycleEvent
  
  /** Enabled/disabled state */
  enabled: boolean
  
  /** Script command */
  run: string
  
  /** Shell selection */
  shell: ShellType
  
  /** Working directory (optional) */
  workingDirectory?: string
  
  /** Environment variables (optional) */
  environmentVariables?: Record<string, string>
  
  /** Continue on error */
  continueOnError: boolean
  
  /** Interactive mode */
  interactive: boolean
  
  /** Platform-specific overrides */
  platforms: {
    windows: boolean
    windowsRun?: string
    windowsShell?: ShellType
    windowsContinueOnError?: boolean
    windowsInteractive?: boolean
    posix: boolean
    posixRun?: string
    posixShell?: ShellType
    posixContinueOnError?: boolean
    posixInteractive?: boolean
  }
}

/**
 * Hook display info for timeline
 */
export interface HookDisplayInfo {
  /** Lifecycle event */
  event: LifecycleEvent
  
  /** Display name */
  displayName: string
  
  /** Category (provision, deploy, etc.) */
  category: string
  
  /** Description */
  description: string
  
  /** Whether hook is configured */
  configured: boolean
  
  /** Whether hook is enabled (if configured) */
  enabled: boolean
  
  /** Script summary (first 50 chars) */
  scriptSummary?: string
  
  /** Has platform-specific overrides */
  hasPlatformOverrides: boolean
  
  /** Has Windows override */
  hasWindows: boolean
  
  /** Has POSIX override */
  hasPosix: boolean
}

/**
 * Lifecycle event metadata
 */
export interface LifecycleEventMeta {
  event: LifecycleEvent
  displayName: string
  category: string
  description: string
  order: number
}

/**
 * All lifecycle events with metadata
 */
export const LIFECYCLE_EVENTS: LifecycleEventMeta[] = [
  {
    event: 'preprovision',
    displayName: 'Pre Provision',
    category: 'Provision',
    description: 'Before infrastructure provisioning',
    order: 1,
  },
  {
    event: 'postprovision',
    displayName: 'Post Provision',
    category: 'Provision',
    description: 'After infrastructure provisioning',
    order: 2,
  },
  {
    event: 'preinfracreate',
    displayName: 'Pre Infra Create',
    category: 'Infrastructure',
    description: 'Before infrastructure creation',
    order: 3,
  },
  {
    event: 'postinfracreate',
    displayName: 'Post Infra Create',
    category: 'Infrastructure',
    description: 'After infrastructure creation',
    order: 4,
  },
  {
    event: 'predeploy',
    displayName: 'Pre Deploy',
    category: 'Deployment',
    description: 'Before application deployment',
    order: 5,
  },
  {
    event: 'postdeploy',
    displayName: 'Post Deploy',
    category: 'Deployment',
    description: 'After application deployment',
    order: 6,
  },
  {
    event: 'predown',
    displayName: 'Pre Down',
    category: 'Teardown',
    description: 'Before infrastructure teardown',
    order: 7,
  },
  {
    event: 'postdown',
    displayName: 'Post Down',
    category: 'Teardown',
    description: 'After infrastructure teardown',
    order: 8,
  },
  {
    event: 'preinfradelete',
    displayName: 'Pre Infra Delete',
    category: 'Infrastructure',
    description: 'Before infrastructure deletion',
    order: 9,
  },
  {
    event: 'postinfradelete',
    displayName: 'Post Infra Delete',
    category: 'Infrastructure',
    description: 'After infrastructure deletion',
    order: 10,
  },
  {
    event: 'preup',
    displayName: 'Pre Up',
    category: 'Up',
    description: 'Before up command',
    order: 11,
  },
  {
    event: 'postup',
    displayName: 'Post Up',
    category: 'Up',
    description: 'After up command',
    order: 12,
  },
  {
    event: 'prepackage',
    displayName: 'Pre Package',
    category: 'Package',
    description: 'Before package creation',
    order: 13,
  },
  {
    event: 'postpackage',
    displayName: 'Post Package',
    category: 'Package',
    description: 'After package creation',
    order: 14,
  },
  {
    event: 'prepublish',
    displayName: 'Pre Publish',
    category: 'Publish',
    description: 'Before publish',
    order: 15,
  },
  {
    event: 'postpublish',
    displayName: 'Post Publish',
    category: 'Publish',
    description: 'After publish',
    order: 16,
  },
  {
    event: 'prerestore',
    displayName: 'Pre Restore',
    category: 'Restore',
    description: 'Before environment restore',
    order: 17,
  },
  {
    event: 'postrestore',
    displayName: 'Post Restore',
    category: 'Restore',
    description: 'After environment restore',
    order: 18,
  },
  {
    event: 'prerun',
    displayName: 'Pre Run',
    category: 'Run',
    description: 'Before run command (azd app)',
    order: 19,
  },
  {
    event: 'postrun',
    displayName: 'Post Run',
    category: 'Run',
    description: 'After run command (azd app)',
    order: 20,
  },
]

/**
 * Get lifecycle event metadata
 */
export function getLifecycleEventMeta(event: LifecycleEvent): LifecycleEventMeta {
  const meta = LIFECYCLE_EVENTS.find((e) => e.event === event)
  if (!meta) {
    throw new Error(`Unknown lifecycle event: ${event}`)
  }
  return meta
}

/**
 * Convert hook config to form data
 */
export function hookToFormData(event: LifecycleEvent, config: HookConfig): HookFormData {
  return {
    event,
    enabled: true,
    run: config.run || '',
    shell: config.shell || 'sh',
    continueOnError: config.continueOnError ?? false,
    interactive: config.interactive ?? false,
    platforms: {
      windows: !!config.windows,
      windowsRun: config.windows?.run,
      windowsShell: config.windows?.shell,
      windowsContinueOnError: config.windows?.continueOnError,
      windowsInteractive: config.windows?.interactive,
      posix: !!config.posix,
      posixRun: config.posix?.run,
      posixShell: config.posix?.shell,
      posixContinueOnError: config.posix?.continueOnError,
      posixInteractive: config.posix?.interactive,
    },
  }
}

/**
 * Convert form data to hook config
 */
export function formDataToHook(data: HookFormData): HookConfig | null {
  if (!data.enabled) {
    return null
  }

  const config: HookConfig = {}

  // Base configuration (only if no platform overrides or both platforms specified)
  if (!data.platforms.windows && !data.platforms.posix) {
    config.run = data.run
    config.shell = data.shell
    config.continueOnError = data.continueOnError
    config.interactive = data.interactive
  } else if (data.platforms.windows && data.platforms.posix) {
    // Both platforms specified - don't include base config
  } else {
    // Only one platform specified - include base config
    config.run = data.run
    config.shell = data.shell
    config.continueOnError = data.continueOnError
    config.interactive = data.interactive
  }

  // Windows override
  if (data.platforms.windows && data.platforms.windowsRun) {
    config.windows = {
      run: data.platforms.windowsRun,
      shell: data.platforms.windowsShell,
      continueOnError: data.platforms.windowsContinueOnError,
      interactive: data.platforms.windowsInteractive,
    }
  }

  // POSIX override
  if (data.platforms.posix && data.platforms.posixRun) {
    config.posix = {
      run: data.platforms.posixRun,
      shell: data.platforms.posixShell,
      continueOnError: data.platforms.posixContinueOnError,
      interactive: data.platforms.posixInteractive,
    }
  }

  return config
}

/**
 * Get default hook form data
 */
export function getDefaultHookFormData(event: LifecycleEvent): HookFormData {
  return {
    event,
    enabled: false,
    run: '',
    shell: 'sh',
    continueOnError: false,
    interactive: false,
    platforms: {
      windows: false,
      posix: false,
    },
  }
}

/**
 * Get hook display info for timeline
 */
export function getHookDisplayInfo(
  event: LifecycleEvent,
  config: HookConfig | HookConfig[] | undefined
): HookDisplayInfo {
  const meta = getLifecycleEventMeta(event)
  
  if (!config) {
    return {
      event,
      displayName: meta.displayName,
      category: meta.category,
      description: meta.description,
      configured: false,
      enabled: false,
      hasPlatformOverrides: false,
      hasWindows: false,
      hasPosix: false,
    }
  }

  // Handle array of hooks (take first one for display)
  const hookConfig = Array.isArray(config) ? config[0] : config

  const scriptSummary = hookConfig.run
    ? hookConfig.run.length > 50
      ? `${hookConfig.run.substring(0, 47)}...`
      : hookConfig.run
    : undefined

  return {
    event,
    displayName: meta.displayName,
    category: meta.category,
    description: meta.description,
    configured: true,
    enabled: true,
    scriptSummary,
    hasPlatformOverrides: !!(hookConfig.windows || hookConfig.posix),
    hasWindows: !!hookConfig.windows,
    hasPosix: !!hookConfig.posix,
  }
}

/**
 * Validate shell type
 */
export function isValidShell(shell: string): shell is ShellType {
  return ['sh', 'bash', 'pwsh', 'powershell', 'cmd'].includes(shell)
}

/**
 * Check if hook has cross-platform coverage
 */
export function hasCrossPlatformCoverage(config: HookConfig): boolean {
  // Has base config OR both windows and posix
  if (config.run && !config.windows && !config.posix) {
    return true // Base config works on all platforms
  }
  
  return !!(config.windows && config.posix)
}

/**
 * Get platform coverage warning message
 */
export function getPlatformCoverageWarning(config: HookConfig): string | null {
  if (hasCrossPlatformCoverage(config)) {
    return null
  }

  if (config.windows && !config.posix) {
    return 'This hook only runs on Windows. Add POSIX override for Linux/macOS support.'
  }

  if (config.posix && !config.windows) {
    return 'This hook only runs on Linux/macOS. Add Windows override for Windows support.'
  }

  return 'This hook has no configuration.'
}
