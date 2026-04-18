// service_ops_rpc.go bridges the Connect-RPC ServicesService.{Start,Stop,Restart}
// handlers to the existing serviceOperationHandler bulk machinery without
// dragging the HTTP-coupled single-service path along for the ride.
//
// Why this file exists separately:
//   - The original serviceOperationHandler.handleSingleOperation writes
//     directly to http.ResponseWriter (status code, JSON body, headers).
//     The Connect handler can't supply one; faking a recorder would be a
//     hack and would bypass Connect's status/code translation.
//   - The bulk machinery (handleBulkOperation, executeBulkServiceOperation,
//     performStartBulk, performStopBulk) is already HTTP-decoupled because
//     it returns a structured BulkOperationResult that the legacy handler
//     then JSON-encodes. We reuse that path for both bulk (empty name) and
//     single-service (one-element list) calls.
//   - For single-service calls we still need the precondition checks
//     (exists? state valid? op already in flight?) so the Connect client
//     gets NOT_FOUND / FAILED_PRECONDITION / ALREADY_EXISTS instead of a
//     generic INTERNAL with a stuffed message. Those checks live on
//     serviceOperationHandler too; we just call them ahead of execution.

package dashboard

import (
	"context"
	"fmt"
	"log"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/rpc"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/registry"
	"github.com/jongio/azd-core/security"
)

// servicesLifecycleAdapter satisfies rpc.ServiceLifecycle by forwarding to
// the dashboard server's RunServiceOperation. It exists as a separate type
// (rather than methods directly on *Server) for two reasons:
//  1. *Server currently has too many responsibilities to want to advertise
//     a Connect-facing interface as part of its public surface.
//  2. A dedicated adapter type keeps the Connect contract documented in
//     one place near the wiring code, so reviewers don't have to chase
//     methods across server.go variants.
type servicesLifecycleAdapter struct {
	server *Server
}

// newServicesLifecycleAdapter constructs the adapter. The server pointer is
// required; passing nil indicates a wiring bug and we'd rather fail at
// construction than NPE on the first request.
func newServicesLifecycleAdapter(s *Server) *servicesLifecycleAdapter {
	if s == nil {
		panic("dashboard: newServicesLifecycleAdapter called with nil *Server")
	}
	return &servicesLifecycleAdapter{server: s}
}

// Compile-time conformance check.
var _ rpc.ServiceLifecycle = (*servicesLifecycleAdapter)(nil)

// StartService implements rpc.ServiceLifecycle. noWait is currently advisory:
// the underlying StartService call already fires the process and returns,
// without waiting for readiness probes, so honoring noWait would be a no-op
// at this layer. The flag is preserved on the wire so a future
// "synchronous start" mode can land without a proto change.
func (a *servicesLifecycleAdapter) StartService(ctx context.Context, name string, _ bool) (string, error) {
	return a.server.RunServiceOperation(ctx, opStart, name)
}

// StopService implements rpc.ServiceLifecycle. force is currently advisory:
// stopService already escalates from graceful → port-kill on its own, so
// there's no separate "graceful" mode to suppress. Surfacing the flag now
// avoids a proto change later if/when we add a true SIGKILL path.
func (a *servicesLifecycleAdapter) StopService(ctx context.Context, name string, _ bool) (string, error) {
	return a.server.RunServiceOperation(ctx, opStop, name)
}

// RestartService implements rpc.ServiceLifecycle.
func (a *servicesLifecycleAdapter) RestartService(ctx context.Context, name string) (string, error) {
	return a.server.RunServiceOperation(ctx, opRestart, name)
}

// RunServiceOperation drives a start/stop/restart against either a single
// named service (preconditions enforced; sentinel errors returned for
// NOT_FOUND / FAILED_PRECONDITION / ALREADY_EXISTS) or every applicable
// service when name == "".
//
// Returns a human-readable summary string suitable for an OperationResult
// message; partial failures in the bulk path are summarized rather than
// surfaced as Connect errors because some-succeed-some-fail is a real
// operational outcome the dashboard wants to render.
func (s *Server) RunServiceOperation(ctx context.Context, op serviceOperation, name string) (string, error) {
	handler := newServiceOperationHandler(s, op)
	reg := registry.GetRegistry(s.projectDir)

	if name != "" {
		// Single-service path: validate up front so the Connect client
		// gets a typed error code, then run through the same bulk
		// primitive used by the empty-name path. Reusing the bulk
		// runner for a one-element list keeps the broadcast/registry
		// bookkeeping in one place rather than duplicating it.
		if err := security.ValidateServiceName(name, false); err != nil {
			return "", fmt.Errorf("%w: %s", rpc.ErrServiceInvalidState, err.Error())
		}
		entry, exists := reg.GetService(name)
		if !exists {
			return "", fmt.Errorf("%w: %s", rpc.ErrServiceNotFound, name)
		}
		opMgr := service.GetOperationManager()
		if opMgr.IsOperationInProgress(name) {
			return "", fmt.Errorf("%w: %s", rpc.ErrServiceOpInProgress, name)
		}
		if err := handler.validateState(entry, name); err != nil {
			return "", fmt.Errorf("%w: %s", rpc.ErrServiceInvalidState, err.Error())
		}
		return s.runBulk(ctx, handler, reg, []string{name})
	}

	// Bulk path: filter the registry by what's actually applicable for
	// the requested operation (running services for stop, etc.) so we
	// don't fire no-op operations and pollute the result summary with
	// "0 services started" when the user clicked Restart All.
	applicable := applicableServices(reg.ListAll(), op)
	if len(applicable) == 0 {
		return fmt.Sprintf("No services to %s", handler.getOperationVerb()), nil
	}
	return s.runBulk(ctx, handler, reg, applicable)
}

// runBulk executes the operation across the given service names and emits
// one broadcast update afterward (matching legacy behavior). Per-service
// failures are folded into the return string; the Go error return is
// reserved for catastrophic failures that prevented the run from happening
// at all (currently none — the bulk runner cannot itself fail).
func (s *Server) runBulk(
	ctx context.Context,
	handler *serviceOperationHandler,
	reg *registry.ServiceRegistry,
	names []string,
) (string, error) {
	opMgr := service.GetOperationManager()
	opType := handler.toServiceOperationType()

	operationFactory := func(svcName string) func(ctx context.Context) error {
		return func(_ context.Context) error {
			entry, exists := reg.GetService(svcName)
			if !exists {
				return fmt.Errorf("service '%s' not found", svcName)
			}
			return handler.executeBulkServiceOperation(entry, svcName, reg)
		}
	}

	result := opMgr.ExecuteBulkOperation(ctx, names, opType, operationFactory)

	// Broadcast registry state to subscribed clients. We log-and-continue
	// on failure: a missed broadcast is a UI-refresh nuisance, not a
	// reason to fail the operation itself.
	if err := s.BroadcastServiceUpdate(s.projectDir); err != nil {
		log.Printf("Warning: failed to broadcast update: %v", err)
	}

	verb := handler.getOperationPastTense()
	msg := fmt.Sprintf("%d service(s) %s, %d failed", result.SuccessCount, verb, result.FailureCount)

	// Single-service convenience: when the caller asked about exactly
	// one service and it failed, surface that service's error directly
	// rather than a "0 succeeded, 1 failed" summary that hides the
	// real cause from CLI/UI consumers.
	if len(names) == 1 && result.FailureCount == 1 && len(result.Results) == 1 {
		return "", fmt.Errorf("%s: %w", msg, result.Results[0].Error)
	}

	return msg, nil
}

// applicableServices filters a registry snapshot down to the services that
// can meaningfully participate in op. Mirrors the filter previously buried
// in serviceOperationHandler.handleBulkOperation; extracted so both the
// HTTP and Connect paths share one definition of "applicable".
func applicableServices(all []*registry.ServiceRegistryEntry, op serviceOperation) []string {
	out := make([]string, 0, len(all))
	for _, entry := range all {
		switch op {
		case opStart:
			if entry.Status == constants.StatusStopped ||
				entry.Status == constants.StatusNotRunning ||
				entry.Status == constants.StatusError {
				out = append(out, entry.Name)
			}
		case opStop:
			if entry.Status == constants.StatusRunning ||
				entry.Status == constants.StatusReady ||
				entry.Status == constants.StatusStarting {
				out = append(out, entry.Name)
			}
		case opRestart:
			out = append(out, entry.Name)
		}
	}
	return out
}
