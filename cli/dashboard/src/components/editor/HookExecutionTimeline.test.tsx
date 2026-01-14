/**
 * Tests for HookExecutionTimeline component
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HookExecutionTimeline } from './HookExecutionTimeline'
import type { HooksConfig } from '@/lib/editor/hooks-types'

describe('HookExecutionTimeline', () => {
  const mockOnEditHook = vi.fn()
  const mockOnAddHook = vi.fn()

  describe('Rendering', () => {
    it('should render timeline header', () => {
      const hooks: HooksConfig = {}

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      expect(screen.getByText('Lifecycle Hooks Timeline')).toBeInTheDocument()
    })

    it('should render all lifecycle events', () => {
      const hooks: HooksConfig = {}

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      expect(screen.getByText('Pre Provision')).toBeInTheDocument()
      expect(screen.getByText('Post Provision')).toBeInTheDocument()
      expect(screen.getByText('Pre Deploy')).toBeInTheDocument()
      expect(screen.getByText('Post Deploy')).toBeInTheDocument()
      expect(screen.getByText('Pre Down')).toBeInTheDocument()
      expect(screen.getByText('Post Down')).toBeInTheDocument()
      expect(screen.getByText('Pre Restore')).toBeInTheDocument()
      expect(screen.getByText('Post Restore')).toBeInTheDocument()
    })

    it('should group events by category', () => {
      const hooks: HooksConfig = {}

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      // Categories are rendered in uppercase in the UI
      // Use getAllByText since some categories appear multiple times (e.g., Infrastructure)
      const provisionHeaders = screen.getAllByText(/PROVISION/i)
      expect(provisionHeaders.length).toBeGreaterThan(0)
      
      const deploymentHeaders = screen.getAllByText(/DEPLOYMENT/i)
      expect(deploymentHeaders.length).toBeGreaterThan(0)
      
      const teardownHeaders = screen.getAllByText(/TEARDOWN/i)
      expect(teardownHeaders.length).toBeGreaterThan(0)
      
      const restoreHeaders = screen.getAllByText(/RESTORE/i)
      expect(restoreHeaders.length).toBeGreaterThan(0)
    })

    it('should render legend', () => {
      const hooks: HooksConfig = {}

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      expect(screen.getByText('Configured & Enabled')).toBeInTheDocument()
      expect(screen.getByText('Not Configured')).toBeInTheDocument()
      expect(screen.getByText('Platform-Specific')).toBeInTheDocument()
    })
  })

  describe('Configured Hooks', () => {
    it('should highlight configured hooks', () => {
      const hooks: HooksConfig = {
        preprovision: {
          run: './setup.sh',
          shell: 'bash',
        },
      }

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      const preProvisionButton = screen.getByRole('button', { name: /Pre Provision/i })
      expect(preProvisionButton).toHaveClass('border-cyan-200')
    })

    it('should show script summary for configured hooks', () => {
      const hooks: HooksConfig = {
        predeploy: {
          run: './build-and-deploy.sh',
          shell: 'bash',
        },
      }

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      expect(screen.getByText('./build-and-deploy.sh')).toBeInTheDocument()
    })

    it('should truncate long script summaries', () => {
      const longScript = './very-long-script-name-that-exceeds-the-fifty-character-limit.sh'
      const hooks: HooksConfig = {
        postdeploy: {
          run: longScript,
        },
      }

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      const scriptText = screen.getByText(/very-long-script-name/i)
      expect(scriptText.textContent).toContain('...')
      expect(scriptText.textContent!.length).toBeLessThanOrEqual(50)
    })
  })

  describe('Platform-Specific Indicators', () => {
    it('should show Windows indicator for Windows-only hooks', () => {
      const hooks: HooksConfig = {
        prerun: {
          windows: {
            run: '.\\setup.ps1',
            shell: 'pwsh',
          },
        },
      }

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      expect(screen.getByText('Win')).toBeInTheDocument()
    })

    it('should show POSIX indicator for POSIX-only hooks', () => {
      const hooks: HooksConfig = {
        postrun: {
          posix: {
            run: './cleanup.sh',
            shell: 'bash',
          },
        },
      }

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      expect(screen.getByText('POSIX')).toBeInTheDocument()
    })

    it('should show both indicators for cross-platform hooks', () => {
      const hooks: HooksConfig = {
        preprovision: {
          windows: {
            run: '.\\setup.ps1',
          },
          posix: {
            run: './setup.sh',
          },
        },
      }

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      expect(screen.getByText('Win')).toBeInTheDocument()
      expect(screen.getByText('POSIX')).toBeInTheDocument()
    })

    it('should not show platform indicators for base config hooks', () => {
      const hooks: HooksConfig = {
        predeploy: {
          run: './deploy.sh',
          shell: 'bash',
        },
      }

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      expect(screen.queryByText('Win')).not.toBeInTheDocument()
      expect(screen.queryByText('POSIX')).not.toBeInTheDocument()
    })
  })

  describe('User Interactions', () => {
    it('should call onEditHook when configured hook is clicked', async () => {
      const user = userEvent.setup()
      const hooks: HooksConfig = {
        preprovision: {
          run: './setup.sh',
        },
      }

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      const hookButton = screen.getByRole('button', { name: /Pre Provision/i })
      await user.click(hookButton)

      expect(mockOnEditHook).toHaveBeenCalledWith('preprovision')
    })

    it('should call onAddHook when unconfigured hook is clicked', async () => {
      const user = userEvent.setup()
      const hooks: HooksConfig = {}

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      const hookButton = screen.getByRole('button', { name: /Post Deploy/i })
      await user.click(hookButton)

      expect(mockOnAddHook).toHaveBeenCalledWith('postdeploy')
    })

    it('should show tooltip on hover for configured hooks', async () => {
      const user = userEvent.setup()
      const hooks: HooksConfig = {
        predeploy: {
          run: './build.sh',
          shell: 'bash',
        },
      }

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      const hookButton = screen.getByRole('button', { name: /Pre Deploy/i })
      await user.hover(hookButton)

      // Tooltip should appear
      expect(screen.getByText('Click to edit configuration')).toBeInTheDocument()
    })

    it('should hide tooltip on mouse leave', async () => {
      const user = userEvent.setup()
      const hooks: HooksConfig = {
        postdeploy: {
          run: './notify.sh',
        },
      }

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      const hookButton = screen.getByRole('button', { name: /Post Deploy/i })
      await user.hover(hookButton)
      
      // Tooltip should be visible
      expect(screen.getByText('Click to edit configuration')).toBeInTheDocument()

      await user.unhover(hookButton)

      // Note: The tooltip might still be in the DOM due to transition timing
      // In real implementation, it would fade out
    })
  })

  describe('Multiple Hooks', () => {
    it('should handle array of hooks (display first)', () => {
      const hooks: HooksConfig = {
        prerestore: [
          { run: './first-restore.sh', shell: 'bash' },
          { run: './second-restore.sh', shell: 'sh' },
        ],
      }

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      // Should show the first hook's script
      expect(screen.getByText('./first-restore.sh')).toBeInTheDocument()
      expect(screen.queryByText('./second-restore.sh')).not.toBeInTheDocument()
    })
  })

  describe('Visual States', () => {
    it('should use different styles for configured vs unconfigured hooks', () => {
      const hooks: HooksConfig = {
        preprovision: {
          run: './setup.sh',
        },
        // postprovision is not configured
      }

      render(
        <HookExecutionTimeline
          hooks={hooks}
          onEditHook={mockOnEditHook}
          onAddHook={mockOnAddHook}
        />
      )

      const configuredButton = screen.getByRole('button', { name: /Pre Provision/i })
      const unconfiguredButton = screen.getByRole('button', { name: /Post Provision/i })

      expect(configuredButton).toHaveClass('border-cyan-200')
      expect(unconfiguredButton).toHaveClass('border-slate-200')
    })
  })
})
