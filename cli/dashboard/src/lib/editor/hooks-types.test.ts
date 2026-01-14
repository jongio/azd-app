/**
 * Tests for hooks-types
 */

import { describe, it, expect } from 'vitest'
import {
  getLifecycleEventMeta,
  hookToFormData,
  formDataToHook,
  getDefaultHookFormData,
  getHookDisplayInfo,
  hasCrossPlatformCoverage,
  getPlatformCoverageWarning,
  isValidShell,
  LIFECYCLE_EVENTS,
} from './hooks-types'
import type { HookConfig, LifecycleEvent } from './hooks-types'

describe('hooks-types', () => {
  describe('LIFECYCLE_EVENTS', () => {
    it('should contain all 20 lifecycle events', () => {
      expect(LIFECYCLE_EVENTS).toHaveLength(20)
    })

    it('should have unique event names', () => {
      const events = LIFECYCLE_EVENTS.map((e) => e.event)
      const uniqueEvents = new Set(events)
      expect(uniqueEvents.size).toBe(events.length)
    })

    it('should have sequential order numbers', () => {
      const orders = LIFECYCLE_EVENTS.map((e) => e.order)
      expect(orders).toEqual([...Array(20).keys()].map((i) => i + 1))
    })
  })

  describe('getLifecycleEventMeta', () => {
    it('should return metadata for valid event', () => {
      const meta = getLifecycleEventMeta('preprovision')
      expect(meta.event).toBe('preprovision')
      expect(meta.displayName).toBe('Pre Provision')
      expect(meta.category).toBe('Provision')
      expect(meta.description).toBeTruthy()
    })

    it('should throw for invalid event', () => {
      expect(() => getLifecycleEventMeta('invalid' as LifecycleEvent)).toThrow()
    })
  })

  describe('hookToFormData', () => {
    it('should convert simple hook config to form data', () => {
      const config: HookConfig = {
        run: './setup.sh',
        shell: 'bash',
        continueOnError: true,
        interactive: false,
      }

      const formData = hookToFormData('preprovision', config)

      expect(formData.event).toBe('preprovision')
      expect(formData.enabled).toBe(true)
      expect(formData.run).toBe('./setup.sh')
      expect(formData.shell).toBe('bash')
      expect(formData.continueOnError).toBe(true)
      expect(formData.interactive).toBe(false)
      expect(formData.platforms.windows).toBe(false)
      expect(formData.platforms.posix).toBe(false)
    })

    it('should convert hook with Windows override', () => {
      const config: HookConfig = {
        run: './setup.sh',
        shell: 'sh',
        windows: {
          run: '.\\setup.ps1',
          shell: 'pwsh',
          continueOnError: false,
        },
      }

      const formData = hookToFormData('predeploy', config)

      expect(formData.platforms.windows).toBe(true)
      expect(formData.platforms.windowsRun).toBe('.\\setup.ps1')
      expect(formData.platforms.windowsShell).toBe('pwsh')
      expect(formData.platforms.windowsContinueOnError).toBe(false)
    })

    it('should convert hook with POSIX override', () => {
      const config: HookConfig = {
        run: 'echo "default"',
        posix: {
          run: './posix-script.sh',
          shell: 'bash',
        },
      }

      const formData = hookToFormData('postdeploy', config)

      expect(formData.platforms.posix).toBe(true)
      expect(formData.platforms.posixRun).toBe('./posix-script.sh')
      expect(formData.platforms.posixShell).toBe('bash')
    })

    it('should handle empty config', () => {
      const config: HookConfig = {}

      const formData = hookToFormData('prerun', config)

      expect(formData.event).toBe('prerun')
      expect(formData.enabled).toBe(true)
      expect(formData.run).toBe('')
      expect(formData.shell).toBe('sh')
    })
  })

  describe('formDataToHook', () => {
    it('should return null when disabled', () => {
      const formData = getDefaultHookFormData('preprovision')
      formData.enabled = false

      const config = formDataToHook(formData)

      expect(config).toBeNull()
    })

    it('should convert form data to simple hook config', () => {
      const formData = getDefaultHookFormData('predeploy')
      formData.enabled = true
      formData.run = './build.sh'
      formData.shell = 'bash'
      formData.continueOnError = true

      const config = formDataToHook(formData)

      expect(config).toEqual({
        run: './build.sh',
        shell: 'bash',
        continueOnError: true,
        interactive: false,
      })
    })

    it('should convert form data with Windows override', () => {
      const formData = getDefaultHookFormData('preprovision')
      formData.enabled = true
      formData.run = './default.sh'
      formData.shell = 'sh'
      formData.platforms.windows = true
      formData.platforms.windowsRun = '.\\windows.ps1'
      formData.platforms.windowsShell = 'pwsh'

      const config = formDataToHook(formData)

      expect(config?.windows).toEqual({
        run: '.\\windows.ps1',
        shell: 'pwsh',
        continueOnError: undefined,
        interactive: undefined,
      })
    })

    it('should not include base config when both platforms specified', () => {
      const formData = getDefaultHookFormData('postprovision')
      formData.enabled = true
      formData.run = './default.sh' // Should be ignored
      formData.platforms.windows = true
      formData.platforms.windowsRun = '.\\windows.ps1'
      formData.platforms.posix = true
      formData.platforms.posixRun = './posix.sh'

      const config = formDataToHook(formData)

      expect(config?.run).toBeUndefined()
      expect(config?.windows).toBeTruthy()
      expect(config?.posix).toBeTruthy()
    })

    it('should not create platform override without run command', () => {
      const formData = getDefaultHookFormData('prerestore')
      formData.enabled = true
      formData.run = './script.sh'
      formData.platforms.windows = true
      // windowsRun is empty/undefined

      const config = formDataToHook(formData)

      expect(config?.windows).toBeUndefined()
    })
  })

  describe('getDefaultHookFormData', () => {
    it('should return disabled form data by default', () => {
      const formData = getDefaultHookFormData('prerun')

      expect(formData.event).toBe('prerun')
      expect(formData.enabled).toBe(false)
      expect(formData.run).toBe('')
      expect(formData.shell).toBe('sh')
      expect(formData.continueOnError).toBe(false)
      expect(formData.interactive).toBe(false)
    })
  })

  describe('getHookDisplayInfo', () => {
    it('should return unconfigured info when config is undefined', () => {
      const info = getHookDisplayInfo('preprovision', undefined)

      expect(info.event).toBe('preprovision')
      expect(info.configured).toBe(false)
      expect(info.enabled).toBe(false)
      expect(info.hasPlatformOverrides).toBe(false)
    })

    it('should return configured info with script summary', () => {
      const config: HookConfig = {
        run: './very-long-script-name-that-will-be-truncated.sh',
        shell: 'bash',
      }

      const info = getHookDisplayInfo('predeploy', config)

      expect(info.configured).toBe(true)
      expect(info.enabled).toBe(true)
      // Script is 53 chars, should be truncated to 50 with "..."
      expect(info.scriptSummary).toBeTruthy()
      expect(info.scriptSummary?.length).toBeLessThanOrEqual(50)
      if (config.run && config.run.length > 50) {
        expect(info.scriptSummary).toMatch(/\.\.\./)
      }
    })

    it('should detect platform overrides', () => {
      const config: HookConfig = {
        run: './default.sh',
        windows: {
          run: '.\\windows.ps1',
        },
        posix: {
          run: './posix.sh',
        },
      }

      const info = getHookDisplayInfo('postdeploy', config)

      expect(info.hasPlatformOverrides).toBe(true)
      expect(info.hasWindows).toBe(true)
      expect(info.hasPosix).toBe(true)
    })

    it('should handle array of hooks (use first)', () => {
      const configs: HookConfig[] = [
        { run: './first.sh', shell: 'bash' },
        { run: './second.sh', shell: 'sh' },
      ]

      const info = getHookDisplayInfo('preup', configs)

      expect(info.configured).toBe(true)
      expect(info.scriptSummary).toBe('./first.sh')
    })
  })

  describe('hasCrossPlatformCoverage', () => {
    it('should return true for base config without platform overrides', () => {
      const config: HookConfig = {
        run: './script.sh',
        shell: 'bash',
      }

      expect(hasCrossPlatformCoverage(config)).toBe(true)
    })

    it('should return true when both Windows and POSIX overrides exist', () => {
      const config: HookConfig = {
        windows: {
          run: '.\\windows.ps1',
        },
        posix: {
          run: './posix.sh',
        },
      }

      expect(hasCrossPlatformCoverage(config)).toBe(true)
    })

    it('should return false when only Windows override exists', () => {
      const config: HookConfig = {
        windows: {
          run: '.\\windows.ps1',
        },
      }

      expect(hasCrossPlatformCoverage(config)).toBe(false)
    })

    it('should return false when only POSIX override exists', () => {
      const config: HookConfig = {
        posix: {
          run: './posix.sh',
        },
      }

      expect(hasCrossPlatformCoverage(config)).toBe(false)
    })
  })

  describe('getPlatformCoverageWarning', () => {
    it('should return null for cross-platform coverage', () => {
      const config: HookConfig = {
        run: './script.sh',
      }

      expect(getPlatformCoverageWarning(config)).toBeNull()
    })

    it('should warn when only Windows override exists', () => {
      const config: HookConfig = {
        windows: {
          run: '.\\windows.ps1',
        },
      }

      const warning = getPlatformCoverageWarning(config)

      expect(warning).toContain('Windows')
      expect(warning).toContain('POSIX')
    })

    it('should warn when only POSIX override exists', () => {
      const config: HookConfig = {
        posix: {
          run: './posix.sh',
        },
      }

      const warning = getPlatformCoverageWarning(config)

      expect(warning).toContain('Linux/macOS')
      expect(warning).toContain('Windows')
    })
  })

  describe('isValidShell', () => {
    it('should return true for valid shells', () => {
      expect(isValidShell('sh')).toBe(true)
      expect(isValidShell('bash')).toBe(true)
      expect(isValidShell('pwsh')).toBe(true)
      expect(isValidShell('powershell')).toBe(true)
      expect(isValidShell('cmd')).toBe(true)
    })

    it('should return false for invalid shells', () => {
      expect(isValidShell('zsh')).toBe(false)
      expect(isValidShell('fish')).toBe(false)
      expect(isValidShell('invalid')).toBe(false)
      expect(isValidShell('')).toBe(false)
    })
  })
})
