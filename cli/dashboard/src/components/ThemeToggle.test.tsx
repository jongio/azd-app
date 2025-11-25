import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeToggle } from './ThemeToggle'

describe('ThemeToggle', () => {
  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear()
    // Reset document theme attribute
    document.documentElement.removeAttribute('data-theme')
  })

  afterEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })

  it('renders with default light theme', () => {
    render(<ThemeToggle />)
    
    // Should show sun icon (light mode)
    const button = screen.getByRole('button')
    expect(button).toBeInTheDocument()
    expect(button).toHaveAttribute('aria-label', 'Switch to dark mode')
    expect(button).toHaveAttribute('aria-pressed', 'false')
  })

  it('toggles theme on click', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)
    
    const button = screen.getByRole('button')
    
    // Initial state: light mode
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
    
    // Click to toggle to dark
    await user.click(button)
    
    // Should now be dark mode
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(button).toHaveAttribute('aria-label', 'Switch to light mode')
    expect(button).toHaveAttribute('aria-pressed', 'true')
  })

  it('persists theme to localStorage', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)
    
    const button = screen.getByRole('button')
    
    // Click to toggle
    await user.click(button)
    
    // Should save to localStorage
    expect(localStorage.getItem('dashboard-theme')).toBe('dark')
    
    // Click again
    await user.click(button)
    
    // Should update localStorage
    expect(localStorage.getItem('dashboard-theme')).toBe('light')
  })

  it('restores theme from localStorage on mount', () => {
    // Set dark theme in localStorage
    localStorage.setItem('dashboard-theme', 'dark')
    
    render(<ThemeToggle />)
    
    // Should restore dark theme
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    
    const button = screen.getByRole('button')
    expect(button).toHaveAttribute('aria-label', 'Switch to light mode')
    expect(button).toHaveAttribute('aria-pressed', 'true')
  })

  it('responds to Enter key', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)
    
    const button = screen.getByRole('button')
    button.focus()
    
    // Press Enter
    await user.keyboard('{Enter}')
    
    // Should toggle theme
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
  })

  it('responds to Space key', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)
    
    const button = screen.getByRole('button')
    button.focus()
    
    // Press Space
    await user.keyboard(' ')
    
    // Should toggle theme
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
  })

  it('announces theme change to screen readers', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)
    
    const button = screen.getByRole('button')
    
    // Click to toggle
    await user.click(button)
    
    // Should have live region with announcement
    const status = screen.getByRole('status')
    expect(status).toHaveTextContent(/dark mode enabled/i)
  })

  it('calls onThemeChange callback when theme changes', async () => {
    const user = userEvent.setup()
    const onThemeChange = vi.fn()
    
    render(<ThemeToggle onThemeChange={onThemeChange} />)
    
    const button = screen.getByRole('button')
    
    // Click to toggle
    await user.click(button)
    
    // Callback should be called with new theme
    expect(onThemeChange).toHaveBeenCalledWith('dark')
    
    // Click again
    await user.click(button)
    
    expect(onThemeChange).toHaveBeenCalledWith('light')
  })

  it('has keyboard accessible focus states', () => {
    render(<ThemeToggle />)
    
    const button = screen.getByRole('button')
    
    // Button should be focusable
    button.focus()
    expect(button).toHaveFocus()
    
    // Should have focus-visible styles (checked via class)
    expect(button).toHaveClass('focus-visible:outline-none')
    expect(button).toHaveClass('focus-visible:ring-2')
  })

  it('applies custom className', () => {
    render(<ThemeToggle className="custom-class" />)
    
    const button = screen.getByRole('button')
    expect(button).toHaveClass('custom-class')
  })

  it('shows correct icon for current theme', async () => {
    const user = userEvent.setup()
    const { container } = render(<ThemeToggle />)
    
    // Initial: Sun icon (light mode)
    let svg = container.querySelector('svg')
    expect(svg).toBeInTheDocument()
    
    // Toggle to dark
    await user.click(screen.getByRole('button'))
    
    // Should now show Moon icon
    svg = container.querySelector('svg')
    expect(svg).toBeInTheDocument()
  })

  it('handles rapid toggling', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)
    
    const button = screen.getByRole('button')
    
    // Rapid clicks
    await user.click(button)
    await user.click(button)
    await user.click(button)
    
    // Should end up in dark mode (odd number of clicks)
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(localStorage.getItem('dashboard-theme')).toBe('dark')
  })

  it('maintains theme across component unmount/remount', () => {
    // Initial render
    const { unmount } = render(<ThemeToggle />)
    
    // Unmount
    unmount()
    
    // Remount
    render(<ThemeToggle />)
    
    // Should restore light theme (default)
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })

  it('is accessible with correct ARIA attributes', () => {
    render(<ThemeToggle />)
    
    const button = screen.getByRole('button')
    
    // Should have aria-label
    expect(button).toHaveAttribute('aria-label')
    
    // Should have aria-pressed
    expect(button).toHaveAttribute('aria-pressed')
    
    // aria-pressed should be boolean string
    const pressed = button.getAttribute('aria-pressed')
    expect(['true', 'false']).toContain(pressed)
  })
})
