/**
 * Tests for LogsPaneModeBar component
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LogsPaneModeBar } from './LogsPaneAzureControls'

describe('LogsPaneModeBar', () => {
  it('shows diagnostics button in Azure mode when callback is provided', () => {
    const onOpenDiagnostics = vi.fn()
    
    render(
      <LogsPaneModeBar
        isCollapsed={false}
        logMode="azure"
        isModeSwitching={false}
        onOpenDiagnostics={onOpenDiagnostics}
      />
    )

    const button = screen.getByRole('button', { name: /diagnostics/i })
    expect(button).toBeInTheDocument()
  })

  it('calls onOpenDiagnostics when diagnostics button is clicked', async () => {
    const user = userEvent.setup()
    const onOpenDiagnostics = vi.fn()
    
    render(
      <LogsPaneModeBar
        isCollapsed={false}
        logMode="azure"
        isModeSwitching={false}
        onOpenDiagnostics={onOpenDiagnostics}
      />
    )

    const button = screen.getByRole('button', { name: /diagnostics/i })
    await user.click(button)
    
    expect(onOpenDiagnostics).toHaveBeenCalledTimes(1)
  })

  it('does not show diagnostics button in local mode', () => {
    const onOpenDiagnostics = vi.fn()
    
    render(
      <LogsPaneModeBar
        isCollapsed={false}
        logMode="local"
        isModeSwitching={false}
        onOpenDiagnostics={onOpenDiagnostics}
      />
    )

    const button = screen.queryByRole('button', { name: /diagnostics/i })
    expect(button).not.toBeInTheDocument()
  })

  it('does not show diagnostics button when callback is not provided', () => {
    render(
      <LogsPaneModeBar
        isCollapsed={false}
        logMode="azure"
        isModeSwitching={false}
      />
    )

    const button = screen.queryByRole('button', { name: /diagnostics/i })
    expect(button).not.toBeInTheDocument()
  })

  it('does not render when collapsed', () => {
    const onOpenDiagnostics = vi.fn()
    
    const { container } = render(
      <LogsPaneModeBar
        isCollapsed={true}
        logMode="azure"
        isModeSwitching={false}
        onOpenDiagnostics={onOpenDiagnostics}
      />
    )

    expect(container.firstChild).toBeNull()
  })
})
