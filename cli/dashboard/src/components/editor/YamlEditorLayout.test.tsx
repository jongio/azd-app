/**
 * YamlEditorLayout Component Tests
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { YamlEditorLayout } from './YamlEditorLayout'

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value.toString()
    },
    removeItem: (key: string) => {
      delete store[key]
    },
    clear: () => {
      store = {}
    },
  }
})()

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
})

describe('YamlEditorLayout', () => {
  beforeEach(() => {
    localStorageMock.clear()
    // Mock window.innerWidth
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: 1920,
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders header, sidebar, content, and preview', () => {
    render(
      <YamlEditorLayout
        header={<div>Header</div>}
        sidebar={<div>Sidebar</div>}
        content={<div>Content</div>}
        preview={<div>Preview</div>}
      />
    )

    expect(screen.getByText('Header')).toBeInTheDocument()
    expect(screen.getByText('Sidebar')).toBeInTheDocument()
    expect(screen.getByText('Content')).toBeInTheDocument()
    expect(screen.getByText('Preview')).toBeInTheDocument()
  })

  it('renders footer when provided', () => {
    render(
      <YamlEditorLayout
        header={<div>Header</div>}
        sidebar={<div>Sidebar</div>}
        content={<div>Content</div>}
        footer={<div>Footer</div>}
      />
    )

    expect(screen.getByText('Footer')).toBeInTheDocument()
  })

  it('hides preview when isPreviewVisible is false', () => {
    render(
      <YamlEditorLayout
        header={<div>Header</div>}
        sidebar={<div>Sidebar</div>}
        content={<div>Content</div>}
        preview={<div>Preview</div>}
        isPreviewVisible={false}
      />
    )

    expect(screen.queryByText('Preview')).not.toBeInTheDocument()
  })

  it('applies collapsed sidebar styles when isSidebarCollapsed is true', () => {
    const { container } = render(
      <YamlEditorLayout
        header={<div>Header</div>}
        sidebar={<div>Sidebar</div>}
        content={<div>Content</div>}
        isSidebarCollapsed={true}
      />
    )

    const sidebar = container.querySelector('aside')
    expect(sidebar).toHaveClass('w-12')
  })

  it('applies expanded sidebar styles when isSidebarCollapsed is false', () => {
    const { container } = render(
      <YamlEditorLayout
        header={<div>Header</div>}
        sidebar={<div>Sidebar</div>}
        content={<div>Content</div>}
        isSidebarCollapsed={false}
      />
    )

    const sidebar = container.querySelector('aside')
    expect(sidebar).toHaveClass('w-64')
  })

  it('loads preview width from localStorage', () => {
    localStorageMock.setItem('azd-editor-preview-width', '50')
    
    const { container } = render(
      <YamlEditorLayout
        header={<div>Header</div>}
        sidebar={<div>Sidebar</div>}
        content={<div>Content</div>}
        preview={<div>Preview</div>}
      />
    )

    const preview = container.querySelector('aside:last-child')
    expect(preview).toHaveStyle({ flex: '0 0 50%' })
  })

  it('uses default preview width when localStorage is empty', () => {
    const { container } = render(
      <YamlEditorLayout
        header={<div>Header</div>}
        sidebar={<div>Sidebar</div>}
        content={<div>Content</div>}
        preview={<div>Preview</div>}
      />
    )

    const preview = container.querySelector('aside:last-child')
    expect(preview).toHaveStyle({ flex: '0 0 40%' })
  })

  it('clamps preview width to valid range (20-60%)', () => {
    localStorageMock.setItem('azd-editor-preview-width', '10')
    
    const { container } = render(
      <YamlEditorLayout
        header={<div>Header</div>}
        sidebar={<div>Sidebar</div>}
        content={<div>Content</div>}
        preview={<div>Preview</div>}
      />
    )

    const preview = container.querySelector('aside:last-child')
    // Should be clamped to 20%
    expect(preview).toHaveStyle({ flex: '0 0 20%' })
  })

  it('applies custom className', () => {
    const { container } = render(
      <YamlEditorLayout
        header={<div>Header</div>}
        sidebar={<div>Sidebar</div>}
        content={<div>Content</div>}
        className="custom-layout-class"
      />
    )

    expect(container.firstChild).toHaveClass('custom-layout-class')
  })

  it('handles localStorage errors gracefully', () => {
    const getItemSpy = vi.spyOn(localStorageMock, 'getItem').mockImplementation(() => {
      throw new Error('localStorage error')
    })

    // Should not throw
    expect(() => {
      render(
        <YamlEditorLayout
          header={<div>Header</div>}
          sidebar={<div>Sidebar</div>}
          content={<div>Content</div>}
          preview={<div>Preview</div>}
        />
      )
    }).not.toThrow()

    getItemSpy.mockRestore()
  })
})
