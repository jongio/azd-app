import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, vi } from 'vitest'

// Cleanup after each test
afterEach(() => {
  cleanup()
})

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
}
globalThis.localStorage = localStorageMock as any

// Mock WebSocket
class WebSocketMock {
  url: string
  onopen: ((this: WebSocket, ev: Event) => any) | null = null
  onmessage: ((this: WebSocket, ev: MessageEvent) => any) | null = null
  onerror: ((this: WebSocket, ev: Event) => any) | null = null
  onclose: ((this: WebSocket, ev: CloseEvent) => any) | null = null
  readyState = 1 // OPEN

  constructor(url: string) {
    this.url = url
    setTimeout(() => {
      if (this.onopen) {
        this.onopen.call(this as any, {} as Event)
      }
    }, 0)
  }

  send(_data: string) {
    // Mock send
  }

  close() {
    this.readyState = 3 // CLOSED
    if (this.onclose) {
      this.onclose.call(this as any, {} as CloseEvent)
    }
  }
}

globalThis.WebSocket = WebSocketMock as any

// Mock fetch
globalThis.fetch = vi.fn()

// Mock scrollIntoView
Element.prototype.scrollIntoView = vi.fn()

// Mock scrollTo
Element.prototype.scrollTo = vi.fn()

// Mock URL.createObjectURL and revokeObjectURL
globalThis.URL.createObjectURL = vi.fn(() => 'mock-url')
globalThis.URL.revokeObjectURL = vi.fn()

// Mock HTMLAnchorElement click to prevent jsdom navigation errors
const originalCreateElement = document.createElement.bind(document)
document.createElement = vi.fn((tagName: string, options?: any) => {
  const element = originalCreateElement(tagName, options)
  if (tagName === 'a') {
    element.click = vi.fn()
  }
  return element
}) as any
