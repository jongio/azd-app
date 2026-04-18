// Package rpc hosts Connect-RPC handlers for the azdapp.v1.* services.
//
// Handlers in this package are intentionally thin: they translate between
// the generated proto request/response types and the existing dashboard
// internals (broadcast.Manager, environment detection helpers, project
// state). Business logic stays in the dashboard package; this package owns
// only the wire/transport concerns.
//
// Mounting: dashboard.Server.setupRoutes calls Mount(s.mux, deps...) once
// after the legacy REST routes are registered. Connect handlers and REST
// endpoints coexist during the transport migration (see ADR-0001) so the
// dashboard can move one consumer at a time.
package rpc

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
)

// Mount registers every Connect handler from this package onto mux. It is
// the single entry point dashboard code uses; adding a new service is a
// one-line edit here, not a search across the dashboard package.
//
// All handlers share the same []connect.HandlerOption set so observability
// and policy interceptors apply uniformly.
func Mount(mux *http.ServeMux, deps Dependencies) {
	opts := []connect.HandlerOption{
		connect.WithInterceptors(NewObservabilityInterceptor()),
	}

	{
		path, handler := azdappv1connect.NewLifecycleServiceHandler(
			NewLifecycleHandler(deps),
			opts...,
		)
		mux.Handle(path, handler)
	}

	if deps.Project != nil && deps.ProjectDir != "" {
		path, handler := azdappv1connect.NewProjectServiceHandler(
			NewProjectHandler(deps.Project, deps.ProjectDir),
			opts...,
		)
		mux.Handle(path, handler)
	}

	// ModeService rides the same Project + ProjectDir plumbing because
	// it also probes azure.yaml. Mounting it requires a ModeStore on
	// top; tests that don't exercise mode can leave Mode unset and the
	// service simply isn't registered (clients get 404).
	if deps.Mode != nil && deps.Project != nil && deps.ProjectDir != "" {
		path, handler := azdappv1connect.NewModeServiceHandler(
			NewModeHandler(deps.Mode, deps.Project, deps.ProjectDir),
			opts...,
		)
		mux.Handle(path, handler)
	}

	// ServicesService needs both the read-only Lister (for GetServices)
	// and the mutating Lifecycle (for Start/Stop/Restart). We mount it
	// only when both are present so Connect tests focused on other
	// services don't have to thread a stub through Dependencies. A
	// production wiring that supplies one without the other is a
	// programming error, so the conditional intentionally ANDs both.
	if deps.ServicesLister != nil && deps.ServicesLifecycle != nil && deps.ProjectDir != "" {
		path, handler := azdappv1connect.NewServicesServiceHandler(
			NewServicesHandler(deps.ServicesLister, deps.ServicesLifecycle, deps.ProjectDir),
			opts...,
		)
		mux.Handle(path, handler)
	}

	// BicepService is mounted only when the caller wires a generator
	// factory. Keeping the factory optional means rpc-only tests don't
	// need to plumb Azure credentials, and dashboards built without the
	// Bicep feature can omit it without failing startup.
	if deps.BicepFactory != nil && deps.ProjectDir != "" {
		path, handler := azdappv1connect.NewBicepServiceHandler(
			NewBicepHandler(deps.BicepFactory, deps.ProjectDir),
			opts...,
		)
		mux.Handle(path, handler)
	}

	// HealthService is mounted whenever a HealthSource is wired. The
	// secondary StateTransitionSource is optional - if absent,
	// StreamStateTransitions returns Unimplemented so clients can detect
	// "feature off" cleanly. GetHealth + StreamHealth depend only on
	// HealthSource and remain functional in that mode.
	if deps.Health != nil {
		path, handler := azdappv1connect.NewHealthServiceHandler(
			NewHealthHandler(deps.Health, deps.StateTransitions),
			opts...,
		)
		mux.Handle(path, handler)
	}
}

// Dependencies bundles the dashboard internals every service handler may
// need. Adding a new dependency is a single-line struct field; no handler
// constructor signatures change. Each handler picks the subset it actually
// uses, keeping its surface narrow without forcing call sites to thread N
// args through Mount.
type Dependencies struct {
	// Broadcast is the in-process pub/sub manager that backs
	// LifecycleService.StreamBroadcast. Required (non-nil).
	Broadcast BroadcastSource

	// Version is the server build version reported by Ping. Empty string
	// is allowed (e.g., in unit tests) and surfaces as "" on the wire.
	Version string

	// Project parses azure.yaml for ProjectService.GetProject. Optional:
	// if either Project or ProjectDir is unset, the ProjectService handler
	// is not mounted, leaving the legacy /api/project endpoint as the sole
	// provider. This lets unit tests that only exercise Lifecycle skip the
	// project plumbing entirely.
	Project ProjectSource

	// ProjectDir is the absolute working directory for ProjectService.
	// Paired with Project (see above).
	ProjectDir string

	// Mode is the read/write store backing ModeService.GetMode/SetMode.
	// Optional: ModeService is mounted only when Mode, Project, and
	// ProjectDir are all set (Mode RPCs probe azure.yaml via Project).
	Mode ModeStore

	// ServicesLister enumerates services for ServicesService.GetServices.
	// Production wires it to serviceinfo.GetServiceInfo via
	// ServiceListerFunc; tests inject a stub.
	//
	// Optional: ServicesService is mounted only when both ServicesLister
	// and ServicesLifecycle are set, alongside a non-empty ProjectDir.
	ServicesLister ServiceLister

	// ServicesLifecycle drives Start/Stop/Restart RPCs. See ServicesLister
	// for mounting rules.
	ServicesLifecycle ServiceLifecycle

	// BicepFactory builds a per-request BicepGenerator for
	// BicepService.GetBicepTemplate. Optional: BicepService is mounted
	// only when this is non-nil and ProjectDir is set. Tests that focus
	// on other services can leave it unset.
	BicepFactory BicepGeneratorFactory

	// Health probes services for HealthService.GetHealth and
	// HealthService.StreamHealth. Optional: HealthService is mounted
	// only when this is non-nil. Production wires it to a per-call
	// HealthStreamManager via HealthSourceFunc.
	Health HealthSource

	// StateTransitions feeds HealthService.StreamStateTransitions.
	// Optional even when Health is set: dashboards without a wired
	// monitor.StateMonitor leave this nil and the handler returns
	// Unimplemented to clients invoking that one RPC.
	StateTransitions StateTransitionSource
}
