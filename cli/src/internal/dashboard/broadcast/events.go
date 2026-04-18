package broadcast

// Event type identifiers used as Event.Type. These string values are the
// wire values surfaced via azdapp.v1.BroadcastEvent.type and must stay
// stable across releases because dashboard clients filter on them by name.
const (
	// TypeServicesChanged is emitted whenever the set of services or any
	// service's status / health / url changes. Payload shape mirrors the
	// legacy /api/ws "services" message: { "services": [ ... ] }.
	TypeServicesChanged = "services-changed"
)
