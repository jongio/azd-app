/**
 * Tests for accessibility utilities
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { getFocusableElements, createFocusTrap } from './focus-trap'
import { getAnnouncer } from './announcer'
import { getShortcutsRegistry } from './keyboard-shortcuts'
import { SkipLinks, DEFAULT_SKIP_LINKS } from './skip-links'

describe('Focus Trap', () => {
  describe('getFocusableElements', () => {
    it('should find all focusable elements', () => {
      const container = document.createElement('div')
      container.innerHTML = `
        <button>Button 1</button>
        <a href="#">Link</a>
        <input type="text" />
        <button disabled>Disabled</button>
        <div tabindex="0">Focusable div</div>
        <div tabindex="-1">Not focusable</div>
      `

      const focusable = getFocusableElements(container)
      expect(focusable).toHaveLength(4) // button, link, input, div with tabindex=0
    })

    it('should exclude elements with aria-hidden', () => {
      const container = document.createElement('div')
      container.innerHTML = `
        <button>Visible</button>
        <button aria-hidden="true">Hidden</button>
      `

      const focusable = getFocusableElements(container)
      expect(focusable).toHaveLength(1)
    })
  })

  describe('createFocusTrap', () => {
    let container: HTMLElement

    beforeEach(() => {
      container = document.createElement('div')
      container.innerHTML = `
        <button id="first">First</button>
        <button id="second">Second</button>
        <button id="last">Last</button>
      `
      document.body.appendChild(container)
    })

    afterEach(() => {
      document.body.removeChild(container)
    })

    it('should focus first element on creation', () => {
      createFocusTrap(container)
      expect(document.activeElement?.id).toBe('first')
    })

    it('should focus custom initial element', () => {
      const second = container.querySelector<HTMLElement>('#second')!
      createFocusTrap(container, { initialFocus: second })
      expect(document.activeElement?.id).toBe('second')
    })

    it('should trap tab navigation', () => {
      createFocusTrap(container)

      const last = container.querySelector<HTMLElement>('#last')!
      last.focus()

      // Tab from last should go to first
      const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true })
      container.dispatchEvent(event)
    })

    it('should call onEscape when Escape pressed', () => {
      const onEscape = vi.fn()
      createFocusTrap(container, { onEscape })

      const event = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })
      container.dispatchEvent(event)

      expect(onEscape).toHaveBeenCalled()
    })

    it('should cleanup and restore focus', () => {
      const outside = document.createElement('button')
      document.body.appendChild(outside)
      outside.focus()

      const cleanup = createFocusTrap(container)
      expect(document.activeElement?.id).toBe('first')

      cleanup()
      expect(document.activeElement).toBe(outside)

      document.body.removeChild(outside)
    })
  })
})

describe('Screen Reader Announcer', () => {
  afterEach(() => {
    getAnnouncer().cleanup()
  })

  it('should create live regions', () => {
    getAnnouncer()

    const polite = document.querySelector('[aria-live="polite"]')
    const assertive = document.querySelector('[aria-live="assertive"]')

    expect(polite).toBeTruthy()
    expect(assertive).toBeTruthy()
  })

  it('should announce message', async () => {
    const announcer = getAnnouncer()

    announcer.announce('Test message', { priority: 'polite', delay: 0 })

    await new Promise((resolve) => setTimeout(resolve, 150))

    const polite = document.querySelector('[aria-live="polite"]')
    expect(polite?.textContent).toBe('Test message')
  })

  it('should announce success', async () => {
    const announcer = getAnnouncer()

    announcer.announceSuccess('Operation completed')

    await new Promise((resolve) => setTimeout(resolve, 150))

    const polite = document.querySelector('[aria-live="polite"]')
    expect(polite?.textContent).toBe('Success: Operation completed')
  })

  it('should announce error with assertive priority', async () => {
    const announcer = getAnnouncer()

    announcer.announceError('Something failed')

    await new Promise((resolve) => setTimeout(resolve, 150))

    const assertive = document.querySelector('[aria-live="assertive"]')
    expect(assertive?.textContent).toBe('Error: Something failed')
  })

  it('should clear message after timeout', async () => {
    const announcer = getAnnouncer()

    announcer.announce('Test message', { priority: 'polite', delay: 0 })

    await new Promise((resolve) => setTimeout(resolve, 150))
    const polite = document.querySelector('[aria-live="polite"]')
    expect(polite?.textContent).toBe('Test message')

    await new Promise((resolve) => setTimeout(resolve, 3100))
    expect(polite?.textContent).toBe('')
  })
})

describe('Keyboard Shortcuts', () => {
  it('should register shortcut', () => {
    const registry = getShortcutsRegistry()
    const action = vi.fn()

    registry.register({
      id: 'test',
      key: 's',
      modifiers: { ctrl: true },
      description: 'Save',
      action,
      category: 'editor',
    })

    const shortcuts = registry.getAll()
    expect(shortcuts).toHaveLength(1)
    expect(shortcuts[0].id).toBe('test')
  })

  it('should handle keyboard event', () => {
    const registry = getShortcutsRegistry()
    const action = vi.fn()

    registry.register({
      id: 'test',
      key: 's',
      modifiers: { ctrl: true },
      description: 'Save',
      action,
      category: 'editor',
    })

    const event = new KeyboardEvent('keydown', { key: 's', ctrlKey: true })
    const handled = registry.handle(event)

    expect(handled).toBe(true)
    expect(action).toHaveBeenCalled()
  })

  it('should format shortcut for display', () => {
    const registry = getShortcutsRegistry()

    const formatted = registry.formatShortcut({
      id: 'test',
      key: 's',
      modifiers: { ctrl: true, shift: true },
      description: 'Save',
      action: () => {},
      category: 'editor',
    })

    expect(formatted).toContain('S')
    expect(formatted).toContain('Shift')
  })

  it('should get shortcuts by category', () => {
    const registry = getShortcutsRegistry()

    registry.register({
      id: 'save',
      key: 's',
      modifiers: { ctrl: true },
      description: 'Save',
      action: () => {},
      category: 'editor',
    })

    registry.register({
      id: 'close',
      key: 'Escape',
      modifiers: {},
      description: 'Close',
      action: () => {},
      category: 'modal',
    })

    const editorShortcuts = registry.getByCategory('editor')
    expect(editorShortcuts).toHaveLength(1)
    expect(editorShortcuts[0].id).toBe('save')
  })
})

describe('Skip Links', () => {
  it('should render skip links', () => {
    render(<SkipLinks links={DEFAULT_SKIP_LINKS} />)

    const links = screen.getAllByRole('link')
    expect(links.length).toBeGreaterThan(0)
  })

  it('should have correct href attributes', () => {
    render(<SkipLinks links={DEFAULT_SKIP_LINKS} />)

    const mainLink = screen.getByText('Skip to main content')
    expect(mainLink).toHaveAttribute('href', '#main-content')
  })

  it('should focus target on click', () => {
    const target = document.createElement('div')
    target.id = 'main-content'
    target.tabIndex = -1
    document.body.appendChild(target)

    render(<SkipLinks links={[DEFAULT_SKIP_LINKS[0]]} />)

    const link = screen.getByText('Skip to main content')
    link.click()

    expect(document.activeElement).toBe(target)

    document.body.removeChild(target)
  })
})
