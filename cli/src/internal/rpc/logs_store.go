package rpc

import "github.com/jongio/azd-app/cli/src/internal/service"

// LogSource is the narrow read/subscribe slice of service.LogManager that
// LogsHandler needs. Production wires it to a per-call LogManager
// (matching how dashboard/server_handlers.go and dashboard/server_websocket.go
// resolve the manager); tests inject an in-memory stub.
//
// Subscribe returns the buffered channel that *service.LogBuffer.Subscribe
// produces (capacity 100, broadcast drop-newest with timeout). Unsubscribe
// MUST close the channel; the implementation's existing semantics
// (logbuffer.go) already do this.
type LogSource interface {
	// GetRecent returns the last `tail` entries for one service. The
	// bool reports whether the service exists in the manager (false ->
	// NotFound on the wire).
	GetRecent(serviceName string, tail int) ([]service.LogEntry, bool)
	// GetAll returns the merged tail across all services (limit per
	// service applied internally, matching LogManager.GetAllLogs).
	GetAll(tail int) []service.LogEntry
	// ServiceNames lists the buffer keys currently registered. Used
	// for the all-services subscribe path so the handler can fan in
	// across every buffer without poking LogManager internals.
	ServiceNames() []string
	// Subscribe creates a per-service subscription channel. Returns
	// false if the service does not exist (NotFound on the wire).
	Subscribe(serviceName string) (chan service.LogEntry, bool)
	// Unsubscribe releases the channel and frees server-side resources.
	// MUST be safe to call with a channel from a now-removed service
	// (idempotent / no panic on missing buffer).
	Unsubscribe(serviceName string, ch chan service.LogEntry)
}

// ClassificationStore is the read/write slice of azure.yaml-stored log
// classifications LogsHandler needs. Production wires it to dashboard's
// loadAzureYaml/saveAzureYaml pair (closing over the package-level
// classificationsMu so REST and Connect handlers share the same lock);
// tests inject an in-memory stub.
//
// SaveClassifications receives the FULL replacement slice; the handler
// performs read-modify-write under the same mutex the legacy REST
// handler holds, so concurrent Add/Delete operations do not race.
type ClassificationStore interface {
	// LoadClassifications returns the current rule list from azure.yaml.
	// An empty slice is valid (no rules). Error means the YAML could not
	// be read or parsed; the handler maps that to Internal.
	LoadClassifications() ([]service.LogClassification, error)
	// SaveClassifications persists the full replacement slice, atomically
	// from the dashboard side (fileutil.AtomicWriteFile).
	SaveClassifications(classifications []service.LogClassification) error
}

// PreferenceStore is the read/write slice of the user-preferences blob.
// Production wires it to azdconfig.ConfigClient.{Get,Set}PreferenceSection
// keyed by "logs"; tests inject an in-memory stub.
//
// The blob is opaque JSON; the handler decodes/encodes via protojson with
// DiscardUnknown=true so older or future-schema keys round-trip without
// breaking GetPreferences. See the proto comment on Preferences for why
// theme + ui.grid_auto_fit live here despite being absent from the Go
// dashboard.UserPreferences struct.
type PreferenceStore interface {
	// LoadPreferences returns the raw JSON blob, or nil/empty if no
	// preferences have been stored yet. nil and zero-length both signal
	// "use defaults".
	LoadPreferences() ([]byte, error)
	// SavePreferences persists the raw JSON blob.
	SavePreferences(data []byte) error
}

// LogsStore bundles the three narrow interfaces LogsHandler needs so the
// Mount() conditional stays a single nil-check. Production wires it via
// LogsStoreFuncs (below) closing over dashboard.Server methods; tests
// satisfy it with in-memory stubs that implement all three slices.
type LogsStore interface {
	LogSource
	ClassificationStore
	PreferenceStore
}

// LogsStoreFuncs adapts plain function values to LogsStore. Lets the
// dashboard wire its private loadAzureYaml/saveAzureYaml/getOrCreateConfigClient
// helpers as method values without exporting them or smuggling the
// classifications mutex through a wrapper struct.
//
// Field semantics mirror the LogSource/ClassificationStore/PreferenceStore
// interface methods one-for-one. Every field MUST be set; nil function
// values cause LogsHandler RPCs to panic at call time, which is the
// fail-loud signal we want for a misconfigured wiring.
type LogsStoreFuncs struct {
	GetRecentFn           func(serviceName string, tail int) ([]service.LogEntry, bool)
	GetAllFn              func(tail int) []service.LogEntry
	ServiceNamesFn        func() []string
	SubscribeFn           func(serviceName string) (chan service.LogEntry, bool)
	UnsubscribeFn         func(serviceName string, ch chan service.LogEntry)
	LoadClassificationsFn func() ([]service.LogClassification, error)
	SaveClassificationsFn func(classifications []service.LogClassification) error
	LoadPreferencesFn     func() ([]byte, error)
	SavePreferencesFn     func(data []byte) error
}

// GetRecent implements LogSource.
func (f LogsStoreFuncs) GetRecent(serviceName string, tail int) ([]service.LogEntry, bool) {
	return f.GetRecentFn(serviceName, tail)
}

// GetAll implements LogSource.
func (f LogsStoreFuncs) GetAll(tail int) []service.LogEntry { return f.GetAllFn(tail) }

// ServiceNames implements LogSource.
func (f LogsStoreFuncs) ServiceNames() []string { return f.ServiceNamesFn() }

// Subscribe implements LogSource.
func (f LogsStoreFuncs) Subscribe(serviceName string) (chan service.LogEntry, bool) {
	return f.SubscribeFn(serviceName)
}

// Unsubscribe implements LogSource.
func (f LogsStoreFuncs) Unsubscribe(serviceName string, ch chan service.LogEntry) {
	f.UnsubscribeFn(serviceName, ch)
}

// LoadClassifications implements ClassificationStore.
func (f LogsStoreFuncs) LoadClassifications() ([]service.LogClassification, error) {
	return f.LoadClassificationsFn()
}

// SaveClassifications implements ClassificationStore.
func (f LogsStoreFuncs) SaveClassifications(classifications []service.LogClassification) error {
	return f.SaveClassificationsFn(classifications)
}

// LoadPreferences implements PreferenceStore.
func (f LogsStoreFuncs) LoadPreferences() ([]byte, error) { return f.LoadPreferencesFn() }

// SavePreferences implements PreferenceStore.
func (f LogsStoreFuncs) SavePreferences(data []byte) error { return f.SavePreferencesFn(data) }
