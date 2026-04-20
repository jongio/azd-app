package dashboard

import (
	"fmt"

	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

// BroadcastServiceUpdate fetches a fresh service snapshot and emits a
// TypeServicesChanged event to all Connect StreamBroadcast subscribers.
// Invoked after lifecycle or environment changes (cmd/app/commands/listen.go
// on env refresh; service_ops_rpc.go after bulk start/stop/restart).
func (s *Server) BroadcastServiceUpdate(projectDir string) error {
	services, err := serviceinfo.GetServiceInfo(projectDir)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}
	if s.broadcast != nil {
		s.broadcast.Emit(broadcast.Event{
			Type:    broadcast.TypeServicesChanged,
			Payload: serviceInfoPayload(services),
		})
	}
	return nil
}
