package dashboard

import (
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/jongio/azd-core/registry"
)

// servicesPayload returns the broadcast Event.Payload for a registry-sourced
// service list (BroadcastUpdate path). The payload mirrors the JSON shape
// the legacy /api/ws "services" message emits today: a top-level "services"
// array. The Connect rpc handler translates this to structpb.Struct at the
// wire boundary.
func servicesPayload(services []*registry.ServiceRegistryEntry) map[string]any {
	// Always emit a non-nil slice so consumers don't have to special-case
	// "field absent" vs. "empty list".
	if services == nil {
		services = []*registry.ServiceRegistryEntry{}
	}
	return map[string]any{
		"services": services,
	}
}

// serviceInfoPayload returns the Event.Payload for a serviceinfo-sourced
// service list (BroadcastServiceUpdate path). Same wire shape as
// servicesPayload; the difference is only the upstream Go type.
func serviceInfoPayload(services []*serviceinfo.ServiceInfo) map[string]any {
	if services == nil {
		services = []*serviceinfo.ServiceInfo{}
	}
	return map[string]any{
		"services": services,
	}
}
