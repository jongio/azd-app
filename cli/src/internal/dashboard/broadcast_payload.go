package dashboard

import (
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

// serviceInfoPayload returns the Event.Payload for a serviceinfo-sourced
// service list (BroadcastServiceUpdate path). The Connect
// LifecycleService.StreamBroadcast handler translates this to
// structpb.Struct at the wire boundary.
func serviceInfoPayload(services []*serviceinfo.ServiceInfo) map[string]any {
	if services == nil {
		services = []*serviceinfo.ServiceInfo{}
	}
	return map[string]any{
		"services": services,
	}
}
