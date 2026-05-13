/**
 * Helpers for the Azure Setup wizard. Centralises the
 * `GetAzureSetupState` RPC so both AuthSetupStep and WorkspaceSetupStep
 * share one code path - the legacy REST endpoint
 * (`/api/azure/logs/setup-state`) returned a single JSON document and
 * each step picked the keys it cared about.
 *
 * The proto carries the JSON document inside a `google.protobuf.Struct`
 * (`state`) so the schema can evolve without proto churn. We toJson()
 * it here and surface the same plain-object shape the legacy fetch
 * returned, then each step casts to its own narrower interface.
 */
import type { Transport } from '@connectrpc/connect'
import { create } from '@bufbuild/protobuf'

import { createAzureClient } from '@/lib/connectClient'
import { GetAzureSetupStateRequestSchema } from '@/gen/proto/azdapp/v1/azure_pb.js'

/**
 * Fetch the full setup-state document via Connect. Returns a plain JSON
 * object (not a proto message) so consumers can treat it as a freeform
 * record. Throws on transport / Connect errors so callers can keep
 * their existing try/catch.
 */
export async function fetchAzureSetupState(
  options: { signal?: AbortSignal; transport?: Transport } = {},
): Promise<Record<string, unknown>> {
  const { signal, transport } = options
  const client = createAzureClient(transport)
  const resp = await client.getAzureSetupState(
    create(GetAzureSetupStateRequestSchema),
    signal ? { signal } : undefined,
  )
  // `state` is required-on-the-wire but defensive code lets us tolerate
  // an unset field by returning {} - matches what the legacy `data`
  // would have looked like for an empty 200.
  if (!resp.state) return {}
  const json = resp.state
  if (json && typeof json === 'object' && !Array.isArray(json)) {
    return json as Record<string, unknown>
  }
  return {}
}
