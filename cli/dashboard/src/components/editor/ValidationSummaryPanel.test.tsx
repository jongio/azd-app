/**
 * Validation Summary Panel Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ValidationSummaryPanel } from './ValidationSummaryPanel'
import type { ValidationError } from '@/lib/editor/validation-types'

describe('ValidationSummaryPanel', () => {
  const mockErrors: ValidationError[] = [
    {
      level: 'error',
      message: 'Service name is required',
      path: 'services.api',
      rule: 'required',
    },
    {
      level: 'error',
      message: 'Circular dependency detected',
      path: 'services',
      rule: 'circular-dependency',
      context: 'Dependencies must not form a cycle',
    },
  ]

  const mockWarnings: ValidationError[] = [
    {
      level: 'warning',
      message: 'Port 8080 is used by multiple services',
      path: 'services',
      rule: 'port-conflict',
      context: 'Multiple services using the same port may cause runtime conflicts',
    },
  ]

  const mockInfo: ValidationError[] = [
    {
      level: 'info',
      message: "Service 'api' is missing a health check",
      path: 'services.api',
      rule: 'recommended-healthcheck',
      context: 'Consider adding a health check to monitor service availability',
    },
  ]

  it('should render "No validation issues" when no errors exist', () => {
    render(
      <ValidationSummaryPanel
        errors={[]}
        warnings={[]}
        info={[]}
      />
    )

    expect(screen.getByText(/No validation issues/i)).toBeInTheDocument()
  })

  it('should display error count in header', () => {
    render(
      <ValidationSummaryPanel
        errors={mockErrors}
        warnings={[]}
        info={[]}
      />
    )

    expect(screen.getByText(/2 errors/i)).toBeInTheDocument()
  })

  it('should display warning count in header', () => {
    render(
      <ValidationSummaryPanel
        errors={[]}
        warnings={mockWarnings}
        info={[]}
      />
    )

    expect(screen.getByText(/1 warning/i)).toBeInTheDocument()
  })

  it('should display info count in header', () => {
    render(
      <ValidationSummaryPanel
        errors={[]}
        warnings={[]}
        info={mockInfo}
      />
    )

    expect(screen.getByText(/1 suggestion/i)).toBeInTheDocument()
  })

  it('should display combined counts when multiple types exist', () => {
    render(
      <ValidationSummaryPanel
        errors={mockErrors}
        warnings={mockWarnings}
        info={mockInfo}
      />
    )

    // Check that all counts are present in the document
    expect(screen.getByText(/2 errors/i)).toBeInTheDocument()
    expect(screen.getByText(/1 warning/i)).toBeInTheDocument()
    expect(screen.getByText(/1 suggestion/i)).toBeInTheDocument()
  })

  it('should render error messages', () => {
    render(
      <ValidationSummaryPanel
        errors={mockErrors}
        warnings={[]}
        info={[]}
      />
    )

    expect(screen.getByText('Service name is required')).toBeInTheDocument()
    expect(screen.getByText('Circular dependency detected')).toBeInTheDocument()
  })

  it('should render paths for errors', () => {
    render(
      <ValidationSummaryPanel
        errors={mockErrors}
        warnings={[]}
        info={[]}
      />
    )

    // Use getAllByText since multiple elements may have "Path: services"
    const paths = screen.getAllByText(/Path:/i)
    expect(paths.length).toBeGreaterThan(0)
    expect(screen.getByText('Path: services.api')).toBeInTheDocument()
    expect(screen.getByText('Path: services')).toBeInTheDocument()
  })

  it('should render context when provided', () => {
    render(
      <ValidationSummaryPanel
        errors={mockErrors}
        warnings={[]}
        info={[]}
      />
    )

    expect(screen.getByText('Dependencies must not form a cycle')).toBeInTheDocument()
  })

  it('should expand/collapse error section', () => {
    render(
      <ValidationSummaryPanel
        errors={mockErrors}
        warnings={[]}
        info={[]}
      />
    )

    const errorButton = screen.getByRole('button', { name: /Errors \(2\)/i })
    expect(errorButton).toHaveAttribute('aria-expanded', 'true')

    // Initially expanded, errors should be visible
    expect(screen.getByText('Service name is required')).toBeInTheDocument()

    // Click to collapse
    fireEvent.click(errorButton)
    expect(errorButton).toHaveAttribute('aria-expanded', 'false')

    // Errors should be hidden
    expect(screen.queryByText('Service name is required')).not.toBeInTheDocument()

    // Click to expand again
    fireEvent.click(errorButton)
    expect(errorButton).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Service name is required')).toBeInTheDocument()
  })

  it('should call onItemClick when clicking an error', () => {
    const onItemClick = vi.fn()

    render(
      <ValidationSummaryPanel
        errors={mockErrors}
        warnings={[]}
        info={[]}
        onItemClick={onItemClick}
      />
    )

    const errorButton = screen.getByRole('listitem', { name: /Service name is required/i })
    fireEvent.click(errorButton)

    expect(onItemClick).toHaveBeenCalledWith('services.api')
  })

  it('should not be clickable when onItemClick is not provided', () => {
    render(
      <ValidationSummaryPanel
        errors={mockErrors}
        warnings={[]}
        info={[]}
      />
    )

    const errorButton = screen.getByRole('listitem', { name: /Service name is required/i })
    expect(errorButton).toBeDisabled()
  })

  it('should have proper ARIA labels', () => {
    render(
      <ValidationSummaryPanel
        errors={mockErrors}
        warnings={mockWarnings}
        info={mockInfo}
      />
    )

    expect(screen.getByRole('region', { name: 'Validation Summary' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Collapse Errors/i })).toBeInTheDocument()
  })

  it('should display warnings section', () => {
    render(
      <ValidationSummaryPanel
        errors={[]}
        warnings={mockWarnings}
        info={[]}
      />
    )

    expect(screen.getByText(/Warnings \(1\)/i)).toBeInTheDocument()
    expect(screen.getByText('Port 8080 is used by multiple services')).toBeInTheDocument()
  })

  it('should display info section (initially collapsed)', () => {
    render(
      <ValidationSummaryPanel
        errors={[]}
        warnings={[]}
        info={mockInfo}
      />
    )

    const infoButton = screen.getByRole('button', { name: /Suggestions \(1\)/i })
    expect(infoButton).toHaveAttribute('aria-expanded', 'false')

    // Info items should not be visible initially
    expect(screen.queryByText("Service 'api' is missing a health check")).not.toBeInTheDocument()

    // Click to expand
    fireEvent.click(infoButton)
    expect(infoButton).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText("Service 'api' is missing a health check")).toBeInTheDocument()
  })

  it('should handle all three severity levels together', () => {
    render(
      <ValidationSummaryPanel
        errors={mockErrors}
        warnings={mockWarnings}
        info={mockInfo}
      />
    )

    expect(screen.getByText(/Errors \(2\)/i)).toBeInTheDocument()
    expect(screen.getByText(/Warnings \(1\)/i)).toBeInTheDocument()
    expect(screen.getByText(/Suggestions \(1\)/i)).toBeInTheDocument()
  })

  it('should apply custom className', () => {
    const { container } = render(
      <ValidationSummaryPanel
        errors={mockErrors}
        warnings={[]}
        info={[]}
        className="custom-class"
      />
    )

    expect(container.firstChild).toHaveClass('custom-class')
  })

  it('should use correct icons for each severity level', () => {
    render(
      <ValidationSummaryPanel
        errors={mockErrors}
        warnings={mockWarnings}
        info={mockInfo}
      />
    )

    // All sections should have their icons (we can't directly test SVG, but the sections should render)
    expect(screen.getByRole('button', { name: /Errors \(2\)/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Warnings \(1\)/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Suggestions \(1\)/i })).toBeInTheDocument()
  })

  it('should handle empty path gracefully', () => {
    const errorsWithoutPath: ValidationError[] = [
      {
        level: 'error',
        message: 'General validation error',
        path: '',
        rule: 'validation',
      },
    ]

    render(
      <ValidationSummaryPanel
        errors={errorsWithoutPath}
        warnings={[]}
        info={[]}
      />
    )

    expect(screen.getByText('General validation error')).toBeInTheDocument()
    // Path should not be displayed when empty
    expect(screen.queryByText(/Path:/i)).not.toBeInTheDocument()
  })
})
