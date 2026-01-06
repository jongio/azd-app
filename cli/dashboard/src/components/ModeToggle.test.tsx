/**
 * ModeToggle Component Tests
 * 
 * Tests the log source mode toggle component that switches between local and Azure log sources.
 * Verifies button states, accessibility, and user interaction handling.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ModeToggle, type ModeToggleProps } from './ModeToggle'

describe('ModeToggle', () => {
  const defaultProps: ModeToggleProps = {
    mode: 'local',
    azureEnabled: true,
    azureStatus: 'connected',
    onModeChange: vi.fn(),
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.clearAllTimers()
  })

  describe('Basic Rendering', () => {
    it('renders both local and Azure buttons', () => {
      render(<ModeToggle {...defaultProps} />)
      
      expect(screen.getByRole('button', { name: 'View local logs' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'View Azure logs' })).toBeInTheDocument()
    })

    it('shows labels when showLabels is true and size is not compact', () => {
      render(<ModeToggle {...defaultProps} showLabels={true} size="standard" />)
      
      expect(screen.getByText('Local')).toBeInTheDocument()
      expect(screen.getByText('Azure')).toBeInTheDocument()
    })

    it('hides labels when size is compact', () => {
      render(<ModeToggle {...defaultProps} showLabels={true} size="compact" />)
      
      expect(screen.queryByText('Local')).not.toBeInTheDocument()
      expect(screen.queryByText('Azure')).not.toBeInTheDocument()
    })

    it('applies correct aria-pressed attributes', () => {
      render(<ModeToggle {...defaultProps} mode="local" />)
      
      const localButton = screen.getByRole('button', { name: 'View local logs' })
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      
      expect(localButton).toHaveAttribute('aria-pressed', 'true')
      expect(azureButton).toHaveAttribute('aria-pressed', 'false')
    })
  })

  describe('Mode Switching', () => {
    it('calls onModeChange when clicking local button from Azure mode', async () => {
      const onModeChange = vi.fn()
      const user = userEvent.setup()
      
      render(<ModeToggle {...defaultProps} mode="azure" onModeChange={onModeChange} />)
      
      const localButton = screen.getByRole('button', { name: 'View local logs' })
      await user.click(localButton)
      
      expect(onModeChange).toHaveBeenCalledWith('local')
    })

    it('calls onModeChange when clicking Azure button from local mode', async () => {
      const onModeChange = vi.fn()
      const user = userEvent.setup()
      
      render(<ModeToggle {...defaultProps} mode="local" onModeChange={onModeChange} />)
      
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      await user.click(azureButton)
      
      expect(onModeChange).toHaveBeenCalledWith('azure')
    })

    it('does not call onModeChange when clicking already selected mode', async () => {
      const onModeChange = vi.fn()
      const user = userEvent.setup()
      
      render(<ModeToggle {...defaultProps} mode="local" onModeChange={onModeChange} />)
      
      const localButton = screen.getByRole('button', { name: 'View local logs' })
      await user.click(localButton)
      
      expect(onModeChange).not.toHaveBeenCalled()
    })

    it('supports keyboard navigation with arrow keys', () => {
      const onModeChange = vi.fn()
      
      const { rerender } = render(<ModeToggle {...defaultProps} mode="local" onModeChange={onModeChange} />)
      
      const radioGroup = screen.getByRole('group', { name: 'Log source' })
      
      // Trigger keydown directly on the radiogroup using fireEvent
      // Arrow keys toggle between modes
      fireEvent.keyDown(radioGroup, { key: 'ArrowRight' })
      expect(onModeChange).toHaveBeenCalledWith('azure')
      
      onModeChange.mockClear()
      
      // Re-render with azure mode to test the other direction
      rerender(<ModeToggle {...defaultProps} mode="azure" onModeChange={onModeChange} />)
      const radioGroupAzure = screen.getByRole('group', { name: 'Log source' })
      fireEvent.keyDown(radioGroupAzure, { key: 'ArrowLeft' })
      expect(onModeChange).toHaveBeenCalledWith('local')
    })
  })

  describe('Setup Guide Integration', () => {
    it('calls onOpenSetupGuide when Azure clicked while disabled', async () => {
      const onOpenSetupGuide = vi.fn()
      const onModeChange = vi.fn()
      const user = userEvent.setup()
      
      render(
        <ModeToggle
          {...defaultProps}
          mode="local"
          azureEnabled={false}
          onOpenSetupGuide={onOpenSetupGuide}
          onModeChange={onModeChange}
        />
      )
      
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      await user.click(azureButton)
      
      expect(onOpenSetupGuide).toHaveBeenCalledTimes(1)
    })

    it('does not change mode when Azure clicked while disabled', async () => {
      const onOpenSetupGuide = vi.fn()
      const onModeChange = vi.fn()
      const user = userEvent.setup()
      
      render(
        <ModeToggle
          {...defaultProps}
          mode="local"
          azureEnabled={false}
          onOpenSetupGuide={onOpenSetupGuide}
          onModeChange={onModeChange}
        />
      )
      
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      await user.click(azureButton)
      
      // Should not call onModeChange
      expect(onModeChange).not.toHaveBeenCalled()
      // Should call onOpenSetupGuide instead
      expect(onOpenSetupGuide).toHaveBeenCalled()
    })

    it('changes mode when Azure clicked while enabled', async () => {
      const onOpenSetupGuide = vi.fn()
      const onModeChange = vi.fn()
      const user = userEvent.setup()
      
      render(
        <ModeToggle
          {...defaultProps}
          mode="local"
          azureEnabled={true}
          onOpenSetupGuide={onOpenSetupGuide}
          onModeChange={onModeChange}
        />
      )
      
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      await user.click(azureButton)
      
      // Should call onModeChange
      expect(onModeChange).toHaveBeenCalledWith('azure')
      // Should not call onOpenSetupGuide
      expect(onOpenSetupGuide).not.toHaveBeenCalled()
    })

    it('shows setup tooltip when Azure is disabled', () => {
      render(
        <ModeToggle
          {...defaultProps}
          azureEnabled={false}
        />
      )
      
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      expect(azureButton).toHaveAttribute('title', 'Click to set up Azure logs')
    })

    it('does not call onOpenSetupGuide when undefined', async () => {
      const onModeChange = vi.fn()
      const user = userEvent.setup()
      
      render(
        <ModeToggle
          {...defaultProps}
          mode="local"
          azureEnabled={false}
          onModeChange={onModeChange}
          onOpenSetupGuide={undefined}
        />
      )
      
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      await user.click(azureButton)
      
      // Should not throw and should not change mode
      expect(onModeChange).not.toHaveBeenCalled()
    })

    it('allows clicking Azure button when disabled', async () => {
      const onOpenSetupGuide = vi.fn()
      const user = userEvent.setup()
      
      render(
        <ModeToggle
          {...defaultProps}
          azureEnabled={false}
          onOpenSetupGuide={onOpenSetupGuide}
        />
      )
      
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      
      // Button should not be disabled
      expect(azureButton).not.toBeDisabled()
      
      await user.click(azureButton)
      
      // Should trigger setup guide
      expect(onOpenSetupGuide).toHaveBeenCalled()
    })
  })

  describe('Azure Connection Status', () => {
    it('shows warning indicator when disconnected with connectionMessage', () => {
      render(
        <ModeToggle
          {...defaultProps}
          azureStatus="disconnected"
          connectionMessage="Connection lost"
          showStatus={true}
        />
      )
      
      const warningIndicator = screen.getByTitle('Connection lost')
      expect(warningIndicator).toBeInTheDocument()
    })

    it('shows warning indicator when Azure is disabled with connectionMessage', () => {
      render(
        <ModeToggle
          {...defaultProps}
          azureEnabled={false}
          azureStatus="disabled"
          connectionMessage="Azure not configured"
          showStatus={true}
        />
      )
      
      const warningIndicator = screen.getByTitle('Azure not configured')
      expect(warningIndicator).toBeInTheDocument()
    })

    it('does not show status indicator when showStatus is false', () => {
      render(
        <ModeToggle
          {...defaultProps}
          azureStatus="disconnected"
          connectionMessage="Connection lost"
          showStatus={false}
        />
      )
      
      expect(screen.queryByTitle('Connection lost')).not.toBeInTheDocument()
    })

    it('applies correct icon color for connected status', () => {
      render(
        <ModeToggle
          {...defaultProps}
          mode="azure"
          azureEnabled={true}
          azureStatus="connected"
          showStatus={true}
        />
      )
      
      // Check for emerald color class on Azure icon when connected
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      const azureIcon = azureButton.querySelector('svg')
      if (azureIcon?.className instanceof SVGAnimatedString) {
        expect(azureIcon.className.baseVal).toContain('text-emerald-500')
      }
    })
  })

  describe('Loading State', () => {
    it('disables buttons when isLoading is true', () => {
      render(<ModeToggle {...defaultProps} isLoading={true} />)
      
      const localButton = screen.getByRole('button', { name: 'View local logs' })
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      
      expect(localButton).toBeDisabled()
      expect(azureButton).toBeDisabled()
    })

    it('shows loading spinner on Azure button when switching to Azure', () => {
      render(<ModeToggle {...defaultProps} mode="azure" isLoading={true} />)
      
      // Loading spinner should be visible
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      expect(azureButton.querySelector('.animate-spin')).toBeInTheDocument()
    })

    it('does not call onModeChange when loading', async () => {
      const onModeChange = vi.fn()
      const user = userEvent.setup()
      
      render(<ModeToggle {...defaultProps} isLoading={true} onModeChange={onModeChange} />)
      
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      await user.click(azureButton)
      
      expect(onModeChange).not.toHaveBeenCalled()
    })
  })

  describe('Accessibility', () => {
    it('has proper radiogroup role and label', () => {
      render(<ModeToggle {...defaultProps} />)
      
      const radioGroup = screen.getByRole('group', { name: 'Log source' })
      expect(radioGroup).toBeInTheDocument()
    })

    it('announces mode changes to screen readers', async () => {
      const onModeChange = vi.fn()
      const user = userEvent.setup()
      
      render(<ModeToggle {...defaultProps} mode="local" onModeChange={onModeChange} />)
      
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      await user.click(azureButton)
      
      // Check for announcement
      const announcement = screen.getByRole('status')
      await waitFor(() => {
        expect(announcement).toHaveTextContent('Switched to Azure logs')
      })
    })

    it('supports focus-visible styles', () => {
      render(<ModeToggle {...defaultProps} />)
      
      const localButton = screen.getByRole('button', { name: 'View local logs' })
      
      expect(localButton.className).toContain('focus-visible:outline-none')
      expect(localButton.className).toContain('focus-visible:ring-2')
    })
  })

  describe('Size Variants', () => {
    it('applies compact size classes', () => {
      render(<ModeToggle {...defaultProps} size="compact" />)
      
      const localButton = screen.getByRole('button', { name: 'View local logs' })
      expect(localButton.className).toContain('p-2')
    })

    it('applies standard size classes', () => {
      render(<ModeToggle {...defaultProps} size="standard" />)
      
      const localButton = screen.getByRole('button', { name: 'View local logs' })
      expect(localButton.className).toContain('px-3')
    })

    it('applies large size classes', () => {
      render(<ModeToggle {...defaultProps} size="large" />)
      
      const localButton = screen.getByRole('button', { name: 'View local logs' })
      expect(localButton.className).toContain('px-4')
    })
  })

  describe('Tooltip Behavior', () => {
    it('shows compact mode tooltip for local button when size is compact', () => {
      render(<ModeToggle {...defaultProps} size="compact" />)
      
      const localButton = screen.getByRole('button', { name: 'View local logs' })
      expect(localButton).toHaveAttribute('title', 'Local logs')
    })

    it('shows compact mode tooltip for Azure button when size is compact and configured', () => {
      render(<ModeToggle {...defaultProps} size="compact" azureEnabled={true} />)
      
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      expect(azureButton).toHaveAttribute('title', 'Azure logs')
    })

    it('does not show tooltip for local button when size is not compact', () => {
      render(<ModeToggle {...defaultProps} size="standard" />)
      
      const localButton = screen.getByRole('button', { name: 'View local logs' })
      expect(localButton).not.toHaveAttribute('title')
    })
  })

  describe('Custom Class Names', () => {
    it('applies custom className to container', () => {
      const { container } = render(<ModeToggle {...defaultProps} className="custom-class" />)
      
      const radioGroup = container.querySelector('.custom-class')
      expect(radioGroup).toBeInTheDocument()
    })
  })

  describe('Edge Cases', () => {
    it('handles undefined onModeChange gracefully', async () => {
      const user = userEvent.setup()
      
      render(<ModeToggle {...defaultProps} onModeChange={undefined} />)
      
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      
      // Should not throw - just verify the click works
      await user.click(azureButton)
      // If we get here without throwing, test passes
      expect(azureButton).toBeInTheDocument()
    })

    it('cleans up timeout on unmount', async () => {
      const user = userEvent.setup()
      const onModeChange = vi.fn()
      
      const { unmount } = render(<ModeToggle {...defaultProps} onModeChange={onModeChange} />)
      
      // Trigger a mode change to create a timeout
      const azureButton = screen.getByRole('button', { name: 'View Azure logs' })
      await user.click(azureButton)
      
      // Unmount should clean up without errors
      expect(() => unmount()).not.toThrow()
    })
  })
})

