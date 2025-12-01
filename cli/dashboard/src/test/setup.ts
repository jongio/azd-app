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
  value: vi.fn().mockImplementation((query: string) => ({
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

// Mock navigator.clipboard
Object.defineProperty(navigator, 'clipboard', {
  value: {
    writeText: vi.fn().mockResolvedValue(undefined),
    readText: vi.fn().mockResolvedValue(''),
  },
  writable: true,
  configurable: true,
})

// Mock localStorage with proper implementation
const localStorageStore = new Map<string, string>()
const localStorageMock = {
  getItem: vi.fn((key: string) => localStorageStore.get(key) ?? null),
  setItem: vi.fn((key: string, value: string) => {
    localStorageStore.set(key, value)
  }),
  removeItem: vi.fn((key: string) => {
    localStorageStore.delete(key)
  }),
  clear: vi.fn(() => {
    localStorageStore.clear()
  }),
  get length() {
    return localStorageStore.size
  },
  key: vi.fn((index: number) => {
    const keys = Array.from(localStorageStore.keys())
    return keys[index] ?? null
  }),
}
globalThis.localStorage = localStorageMock as Storage

// Mock WebSocket
class WebSocketMock {
  url: string
  onopen: ((this: WebSocket, ev: Event) => unknown) | null = null
  onmessage: ((this: WebSocket, ev: MessageEvent) => unknown) | null = null
  onerror: ((this: WebSocket, ev: Event) => unknown) | null = null
  onclose: ((this: WebSocket, ev: CloseEvent) => unknown) | null = null
  readyState = 1 // OPEN

  constructor(url: string) {
    this.url = url
    setTimeout(() => {
      if (this.onopen) {
        this.onopen.call(this as unknown as WebSocket, {} as Event)
      }
    }, 0)
  }

  send(_data: string) {
    // Mock send
  }

  close() {
    this.readyState = 3 // CLOSED
    if (this.onclose) {
      this.onclose.call(this as unknown as WebSocket, {} as CloseEvent)
    }
  }
}

globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket

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
document.createElement = vi.fn((tagName: string, options?: ElementCreationOptions) => {
  const element = originalCreateElement(tagName, options)
  if (tagName === 'a') {
    (element as HTMLAnchorElement).click = vi.fn()
  }
  return element
}) as typeof document.createElement
