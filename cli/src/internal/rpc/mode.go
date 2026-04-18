package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// ModeStore is the narrow interface ModeHandler needs for read/write
// access to the current log source mode. dashboard.Server satisfies it
// via its modeMu/currentMode pair; tests inject an in-memory stub.
//
// Get/Set semantics are intentionally synchronous — callers must hold
// no other locks. The store implementation is responsible for its own
// concurrency control (Server uses a sync.RWMutex).
type ModeStore interface {
	GetMode() service.LogMode
	SetMode(service.LogMode)
}

// ModeStoreFuncs adapts a pair of plain functions to ModeStore. Lets
// dashboard.Server wire its modeMu-guarded accessors as method values
// without exposing the mutex or introducing a wrapper struct.
type ModeStoreFuncs struct {
	Get func() service.LogMode
	Set func(service.LogMode)
}

// GetMode implements ModeStore.
func (f ModeStoreFuncs) GetMode() service.LogMode { return f.Get() }

// SetMode implements ModeStore.
func (f ModeStoreFuncs) SetMode(m service.LogMode) { f.Set(m) }

// ModeHandler implements azdappv1connect.ModeServiceHandler.
//
// It owns no state of its own; reads and writes are delegated to the
// injected ModeStore, and azure.yaml introspection is delegated to the
// shared ProjectSource. This deliberate symmetry with ProjectHandler
// keeps the per-service recipe boring: narrow interfaces in, generated
// types out, no business logic.
type ModeHandler struct {
	store      ModeStore
	source     ProjectSource
	projectDir string
}

// Compile-time interface conformance.
var _ azdappv1connect.ModeServiceHandler = (*ModeHandler)(nil)

// NewModeHandler constructs a ModeHandler. All deps are required; nil
// or empty values are programming errors so we panic at construction
// time rather than surface NPEs on the first request.
func NewModeHandler(store ModeStore, source ProjectSource, projectDir string) *ModeHandler {
	if store == nil {
		panic("rpc: NewModeHandler called with nil ModeStore")
	}
	if source == nil {
		panic("rpc: NewModeHandler called with nil ProjectSource")
	}
	if projectDir == "" {
		panic("rpc: NewModeHandler called with empty projectDir")
	}
	return &ModeHandler{store: store, source: source, projectDir: projectDir}
}

// GetMode returns the current LogMode plus an azure.yaml-derived
// snapshot of Azure logging availability. Mirrors GET /api/mode.
//
// The connection_message field is populated only when Azure logging is
// unavailable (load error or missing config); the legacy REST handler
// did the same and the React consumer surfaces it verbatim.
func (h *ModeHandler) GetMode(
	_ context.Context,
	_ *connect.Request[v1.GetModeRequest],
) (*connect.Response[v1.GetModeResponse], error) {
	azureCfg := h.probeAzureConfig()

	return connect.NewResponse(&v1.GetModeResponse{
		Mode:              logModeToProto(h.store.GetMode()),
		AzureEnabled:      azureCfg.enabled,
		AzureStatus:       azureCfg.status,
		AzureRealtime:     azureCfg.realtime,
		ConnectionMessage: azureCfg.message,
	}), nil
}

// SetMode switches the current LogMode. Mirrors PUT /api/mode.
//
// Switching to AZURE without azure.yaml `logs.analytics` configured
// returns FailedPrecondition (more specific than the legacy 400 but
// semantically equivalent: the system is not in the required state for
// the operation). The React consumer treats any non-OK as a failure
// and rolls back the toggle, so the code change is transparent.
func (h *ModeHandler) SetMode(
	_ context.Context,
	req *connect.Request[v1.SetModeRequest],
) (*connect.Response[v1.SetModeResponse], error) {
	mode, err := protoToLogMode(req.Msg.GetMode())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if mode == service.LogModeAzure {
		// Verify the project is actually configured for Azure logging
		// before flipping state. We do NOT mutate currentMode on a
		// failed precondition so the dashboard reflects reality.
		azureYaml, parseErr := h.source.ParseAzureYaml(h.projectDir)
		if parseErr != nil || azureYaml.Logs == nil || azureYaml.Logs.Analytics == nil {
			return nil, connect.NewError(
				connect.CodeFailedPrecondition,
				errors.New("azure logging not configured: add logs.analytics section to azure.yaml"),
			)
		}
	}

	h.store.SetMode(mode)

	azureCfg := h.probeAzureConfig()
	return connect.NewResponse(&v1.SetModeResponse{
		Mode:              logModeToProto(mode),
		AzureEnabled:      azureCfg.enabled,
		AzureStatus:       azureCfg.status,
		AzureRealtime:     azureCfg.realtime,
		ConnectionMessage: azureCfg.message,
	}), nil
}

// azureConfigSnapshot captures the four azure.yaml-derived fields the
// Mode RPCs expose. Bundling them keeps GetMode/SetMode response
// construction symmetric and prevents drift between the two paths
// (the legacy SetMode handler diverged from GetMode by skipping
// connection_message; that drift is now gone).
type azureConfigSnapshot struct {
	enabled  bool
	status   string
	realtime bool
	message  string
}

// probeAzureConfig parses azure.yaml and projects the result into the
// snapshot the wire types want. Errors are encoded into the snapshot
// (not returned) because both RPCs need to surface a partial answer
// even when the manifest is unreadable — telling the user "I can't
// read azure.yaml" is more useful than a generic Internal error.
func (h *ModeHandler) probeAzureConfig() azureConfigSnapshot {
	snap := azureConfigSnapshot{
		// Default to the "not configured" baseline so every code path
		// below only has to set the fields that change.
		status: "disabled",
	}

	azureYaml, err := h.source.ParseAzureYaml(h.projectDir)
	switch {
	case err != nil:
		snap.message = "Could not load azure.yaml: " + err.Error()
	case azureYaml.Logs == nil || azureYaml.Logs.Analytics == nil:
		snap.message = "Azure logging not configured in azure.yaml"
	default:
		snap.enabled = true
		snap.status = "connected"
		snap.realtime = azureYaml.Logs.Analytics.Realtime
	}

	return snap
}

// logModeToProto maps the internal string-based LogMode to its proto
// enum counterpart. Unknown values map to UNSPECIFIED so the wire is
// always well-formed.
func logModeToProto(m service.LogMode) v1.LogMode {
	switch m {
	case service.LogModeLocal:
		return v1.LogMode_LOG_MODE_LOCAL
	case service.LogModeAzure:
		return v1.LogMode_LOG_MODE_AZURE
	default:
		return v1.LogMode_LOG_MODE_UNSPECIFIED
	}
}

// protoToLogMode converts a proto LogMode to the internal string-based
// LogMode. UNSPECIFIED is rejected (returns an error) because every
// SetMode caller must declare a concrete intent — silently treating
// UNSPECIFIED as a default would mask client bugs.
func protoToLogMode(m v1.LogMode) (service.LogMode, error) {
	switch m {
	case v1.LogMode_LOG_MODE_LOCAL:
		return service.LogModeLocal, nil
	case v1.LogMode_LOG_MODE_AZURE:
		return service.LogModeAzure, nil
	default:
		return "", errors.New("mode must be LOG_MODE_LOCAL or LOG_MODE_AZURE")
	}
}
