// Type-safe mock types for testing
import type { Mock } from 'vitest'

export type MockWebSocket = {
  url: string
  onopen: ((this: WebSocket, ev: Event) => unknown) | null
  onmessage: ((this: WebSocket, ev: MessageEvent) => unknown) | null
  onerror: ((this: WebSocket, ev: Event) => unknown) | null
  onclose: ((this: WebSocket, ev: CloseEvent) => unknown) | null
  readyState: number
  send: (data: string) => void
  close: () => void
}

export type MockFetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>

export type MockLocalStorage = Storage & {
  getItem: Mock<(key: string) => string | null>
  setItem: Mock<(key: string, value: string) => void>
  removeItem: Mock<(key: string) => void>
  clear: Mock<() => void>
  key: Mock<(index: number) => string | null>
}
