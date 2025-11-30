import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useTheme } from './useTheme'

describe('useTheme', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })

  afterEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })

  it('initializes with light theme by default', () => {
    const { result } = renderHook(() => useTheme())
    
    expect(result.current.theme).toBe('light')
  })

  it('applies theme to document root on initialization', async () => {
    renderHook(() => useTheme())
    
    // Wait for effect
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0))
    })
    
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })

  it('restores theme from localStorage', () => {
    localStorage.setItem('dashboard-theme', 'dark')
    
    const { result } = renderHook(() => useTheme())
    
    // Theme should be restored synchronously from localStorage
    expect(result.current.theme).toBe('dark')
    
    // After effect runs, document should also have the theme
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
  })

  it('changes theme with setTheme', () => {
    const { result } = renderHook(() => useTheme())
    
    act(() => {
      result.current.setTheme('dark')
    })
    
    expect(result.current.theme).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(localStorage.getItem('dashboard-theme')).toBe('dark')
  })

  it('toggles theme with toggleTheme', () => {
    const { result } = renderHook(() => useTheme())
    
    // Initially light
    expect(result.current.theme).toBe('light')
    
    // Toggle to dark
    act(() => {
      result.current.toggleTheme()
    })
    
    expect(result.current.theme).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(localStorage.getItem('dashboard-theme')).toBe('dark')
    
    // Toggle back to light
    act(() => {
      result.current.toggleTheme()
    })
    
    expect(result.current.theme).toBe('light')
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
    expect(localStorage.getItem('dashboard-theme')).toBe('light')
  })

  it('persists theme changes to localStorage', () => {
    const { result } = renderHook(() => useTheme())
    
    act(() => {
      result.current.setTheme('dark')
    })
    
    expect(localStorage.getItem('dashboard-theme')).toBe('dark')
    
    act(() => {
      result.current.setTheme('light')
    })
    
    expect(localStorage.getItem('dashboard-theme')).toBe('light')
  })

  it('handles invalid localStorage values gracefully', () => {
    localStorage.setItem('dashboard-theme', 'invalid-theme')
    
    const { result } = renderHook(() => useTheme())
    
    // Should fall back to light theme
    expect(result.current.theme).toBe('light')
  })

  it('updates document attribute when theme changes', () => {
    const { result } = renderHook(() => useTheme())
    
    act(() => {
      result.current.setTheme('dark')
    })
    
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    
    act(() => {
      result.current.setTheme('light')
    })
    
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })
})
