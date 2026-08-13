/// <reference types="node" />
/**
 * Playwright-level Connect-RPC mock helpers.
 *
 * Why this exists:
 * - The dashboard now talks to the backend over Connect-RPC
 *   (`POST /azdapp.v1.{Service}/{Method}`) instead of REST. The e2e
 *   suite's fixtures pre-date that migration, so without these helpers
 *   every test falls through to the in-app `MOCK_SERVICES` dev-mode
 *   fallback and asserts against stale port/shape data.
 * - We don't mock the Connect transport globally. Mocks live at the
 *   HTTP route layer (`page.route`) so they survive a real client
 *   instantiation and exercise the same serialization path production
 *   does.
 *
 * Wire format:
 * - Unary calls ride `application/json` with a raw message body: easy.
 * - Server-streaming rides `application/connect+json` with length-prefixed
 *   envelopes: one 5-byte header (1 flag byte + 4-byte big-endian length)
 *   per frame, terminated by an end-stream envelope (flag 0x02). Request
 *   bodies for streams are a single data envelope, which is why we parse
 *   with `postDataBuffer()` rather than `postData()`: the 5-byte prefix
 *   is not UTF-8-safe.
 *
 * Enum encoding:
 * - protobuf-es's Connect JSON codec accepts enums as either their string
 *   name or their integer value. We send integers to sidestep any
 *   prefix/short-name confusion (`SERVICE_STATUS_READY` vs `READY`).
 */
import type { Page, Route } from '@playwright/test'

// =============================================================================
// Envelope encoding (Connect server-streaming JSON)
// =============================================================================

const FLAG_DATA = 0x00
const FLAG_END = 0x02

function encodeEnvelope(flag: number, payload: Uint8Array): Uint8Array {
  const header = new Uint8Array(5)
  header[0] = flag
  // Big-endian 4-byte length.
  header[1] = (payload.length >>> 24) & 0xff
  header[2] = (payload.length >>> 16) & 0xff
  header[3] = (payload.length >>> 8) & 0xff
  header[4] = payload.length & 0xff
  const out = new Uint8Array(5 + payload.length)
  out.set(header, 0)
  out.set(payload, 5)
  return out
}

function encodeStreamBody(messages: unknown[], endPayload: unknown = {}): Buffer {
  const enc = new TextEncoder()
  const frames: Uint8Array[] = []
  for (const msg of messages) {
    frames.push(encodeEnvelope(FLAG_DATA, enc.encode(JSON.stringify(msg))))
  }
  frames.push(encodeEnvelope(FLAG_END, enc.encode(JSON.stringify(endPayload))))
  let total = 0
  for (const f of frames) total += f.length
  const out = new Uint8Array(total)
  let offset = 0
  for (const f of frames) {
    out.set(f, offset)
    offset += f.length
  }
  return Buffer.from(out)
}

/**
 * Build a single data-envelope (flag 0x00) for a JSON message. Used by
 * callers that construct a never-closing ReadableStream inside the page
 * (via addInitScript) and need raw bytes to enqueue. Returns a plain
 * Uint8Array: Node's Buffer doesn't survive the structured-clone serde
 * boundary into the page context.
 */
export function encodeStreamEnvelopeNoEnd(message: unknown): Uint8Array {
  const enc = new TextEncoder()
  return encodeEnvelope(FLAG_DATA, enc.encode(JSON.stringify(message)))
}

function decodeStreamRequest(buf: Buffer | null): unknown {
  // Request body is a single data envelope. If it's missing (no body) we
  // return undefined so handlers can choose a default.
  if (!buf || buf.length < 5) return undefined
  const len = (buf[1] << 24) | (buf[2] << 16) | (buf[3] << 8) | buf[4]
  if (buf.length < 5 + len) return undefined
  try {
    return JSON.parse(buf.subarray(5, 5 + len).toString('utf8'))
  } catch {
    return undefined
  }
}

// =============================================================================
// Helpers
// =============================================================================

function rpcPath(service: string, method: string): string {
  // Connect routes are absolute paths under the dashboard origin.
  return `**/azdapp.v1.${service}/${method}`
}

export type UnaryHandler<Req = unknown, Resp = unknown> = (
  req: Req,
  route: Route,
) => Resp | Promise<Resp>

export type StreamHandler<Req = unknown, Msg = unknown> = (
  req: Req,
  route: Route,
) => Msg[] | Promise<Msg[]>

/**
 * Register a unary RPC mock. The handler receives the JSON-decoded
 * request body and returns the JSON response message.
 */
export async function mockConnectUnary<Req = unknown, Resp = unknown>(
  page: Page,
  service: string,
  method: string,
  handler: UnaryHandler<Req, Resp>,
): Promise<void> {
  await page.route(rpcPath(service, method), async (route) => {
    let req: unknown = {}
    try {
      const raw = route.request().postData()
      if (raw) req = JSON.parse(raw)
    } catch {
      // Keep {} on parse failure; server would reject but tests don't need
      // us to emulate that path.
    }
    const resp = await handler(req as Req, route)
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(resp),
    })
  })
}

/**
 * Register a server-streaming RPC mock. The handler returns the full
 * sequence of messages to emit; the helper wraps them in Connect
 * envelopes and appends the end-stream trailer.
 *
 * Streams that the dashboard opens on page load (health, broadcast,
 * state transitions, local logs) should emit zero or one data messages
 * and return; hanging a stream keeps `useHealthStream` in its
 * "waiting for first message" state forever.
 */
export async function mockConnectServerStream<Req = unknown, Msg = unknown>(
  page: Page,
  service: string,
  method: string,
  handler: StreamHandler<Req, Msg>,
): Promise<void> {
  await page.route(rpcPath(service, method), async (route) => {
    const req = decodeStreamRequest(route.request().postDataBuffer())
    const messages = await handler(req as Req, route)
    await route.fulfill({
      status: 200,
      contentType: 'application/connect+json',
      body: encodeStreamBody(messages),
    })
  })
}
