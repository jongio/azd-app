package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

// =============================================================================
// Lister: read-only service discovery
// =============================================================================

// ServiceLister is the narrow read-only dependency ServicesHandler needs to
// satisfy GetServices. The dashboard wires it to serviceinfo.GetServiceInfo;
// tests inject a stub that returns canned data without touching disk.
//
// Returning the rich serviceinfo.ServiceInfo (rather than a pre-translated
// proto type) keeps proto translation centralized in this package and means
// future RPCs that surface additional fields don't ripple back through the
// dashboard's information-gathering code.
type ServiceLister interface {
	ListServices(ctx context.Context, projectDir string) ([]*serviceinfo.ServiceInfo, error)
}

// ServiceListerFunc adapts a plain function to the ServiceLister interface so
// production code can wire serviceinfo.GetServiceInfo (which has no context
// param today) without an extra struct literal at the call site.
type ServiceListerFunc func(projectDir string) ([]*serviceinfo.ServiceInfo, error)

// ListServices implements ServiceLister by delegating to the underlying func.
// Context is intentionally dropped: the underlying GetServiceInfo is currently
// synchronous and CPU-bound (parses azure.yaml, walks the registry). When a
// future implementation needs cancellation it can satisfy the interface with
// a richer struct that honors ctx.
func (f ServiceListerFunc) ListServices(_ context.Context, projectDir string) ([]*serviceinfo.ServiceInfo, error) {
	return f(projectDir)
}

// =============================================================================
// Lifecycle: mutating operations
// =============================================================================

// Sentinel errors a ServiceLifecycle implementation should return so the
// Connect handler can map them to specific gRPC/Connect status codes without
// inspecting error strings. Anything not in this list maps to CodeInternal.
//
// Using errors.Is rather than equality so adapters can wrap with context
// (e.g. fmt.Errorf("%w: %s", ErrServiceNotFound, name)) and still be
// classified correctly.
var (
	// ErrServiceNotFound: the named service does not exist in the registry.
	// Maps to connect.CodeNotFound.
	ErrServiceNotFound = errors.New("rpc: service not found")

	// ErrServiceInvalidState: the named service is in a state where the
	// requested operation is not valid (e.g. start an already-running
	// service, stop an already-stopped one). Maps to
	// connect.CodeFailedPrecondition.
	ErrServiceInvalidState = errors.New("rpc: service in invalid state for operation")

	// ErrServiceOpInProgress: another mutating operation against this
	// service is in flight. Maps to connect.CodeAlreadyExists, mirroring
	// the legacy 409 Conflict response used by the REST handlers.
	ErrServiceOpInProgress = errors.New("rpc: operation already in progress for service")
)

// ServiceLifecycle drives start/stop/restart RPCs.
//
// Contract: an empty serviceName means "all applicable services" (bulk
// operation). The bulk variant aggregates per-service results into a single
// message; the wire shape (OperationResult) intentionally does not surface
// per-service detail because clients receive that via
// LifecycleService.StreamBroadcast. If a future need arises for typed
// per-service status the proto layer should grow a new field rather than
// overload OperationResult.message.
//
// For named-service calls the implementation must return one of the
// sentinel errors above when applicable so the handler can produce a
// meaningful Connect code; bulk calls should return nil + a summary
// message even when individual services fail.
type ServiceLifecycle interface {
	// StartService starts a single service or all stopped services when
	// serviceName == "". noWait is propagated for protocol completeness;
	// adapters may treat it as advisory.
	StartService(ctx context.Context, serviceName string, noWait bool) (message string, err error)

	// StopService stops a single service or all running services when
	// serviceName == "". force is advisory: implementations that do not
	// distinguish forced from graceful stops may ignore it.
	StopService(ctx context.Context, serviceName string, force bool) (message string, err error)

	// RestartService restarts a single service or all services when
	// serviceName == "".
	RestartService(ctx context.Context, serviceName string) (message string, err error)
}

// =============================================================================
// Handler
// =============================================================================

// ServicesHandler implements azdappv1connect.ServicesServiceHandler.
//
// The handler holds only the narrow per-RPC dependencies (lister, lifecycle,
// projectDir) so its tests can be driven by trivial stubs without spinning up
// the dashboard registry, broadcast manager, or filesystem.
type ServicesHandler struct {
	lister     ServiceLister
	lifecycle  ServiceLifecycle
	projectDir string
}

// Compile-time interface conformance check: a generated-stub regression
// (e.g. a new RPC added to ServicesService) will fail to build here before it
// fails at registration time.
var _ azdappv1connect.ServicesServiceHandler = (*ServicesHandler)(nil)

// NewServicesHandler constructs a ServicesHandler. All deps are required: a
// nil lister or lifecycle would surface as an NPE on the first request, and
// an empty projectDir means the registry walks would target the wrong tree.
// Programming errors are panics (not errors) so they fail loudly at process
// start rather than degrading silently in production.
func NewServicesHandler(lister ServiceLister, lifecycle ServiceLifecycle, projectDir string) *ServicesHandler {
	if lister == nil {
		panic("rpc: NewServicesHandler called with nil ServiceLister")
	}
	if lifecycle == nil {
		panic("rpc: NewServicesHandler called with nil ServiceLifecycle")
	}
	if projectDir == "" {
		panic("rpc: NewServicesHandler called with empty projectDir")
	}
	return &ServicesHandler{
		lister:     lister,
		lifecycle:  lifecycle,
		projectDir: projectDir,
	}
}

// GetServices returns every service known to the project (azure.yaml
// definitions merged with runtime registry state). Mirrors the legacy
// GET /api/services payload structurally; field-by-field translation lives
// in serviceInfoToProto.
func (h *ServicesHandler) GetServices(
	ctx context.Context,
	_ *connect.Request[v1.GetServicesRequest],
) (*connect.Response[v1.GetServicesResponse], error) {
	infos, err := h.lister.ListServices(ctx, h.projectDir)
	if err != nil {
		// Surface as Internal: a list failure here means the dashboard's
		// view of the project is broken (azure.yaml unreadable, registry
		// IO error). Granular codes can come later if a concrete consumer
		// needs to distinguish them.
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := make([]*v1.ServiceInfo, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		out = append(out, serviceInfoToProto(info, h.projectDir))
	}

	return connect.NewResponse(&v1.GetServicesResponse{Services: out}), nil
}

// StartService starts a single service or all applicable services when
// service_name is empty.
func (h *ServicesHandler) StartService(
	ctx context.Context,
	req *connect.Request[v1.StartServiceRequest],
) (*connect.Response[v1.StartServiceResponse], error) {
	msg, err := h.lifecycle.StartService(ctx, req.Msg.GetServiceName(), req.Msg.GetNoWait())
	if err != nil {
		return nil, lifecycleError(err)
	}
	return connect.NewResponse(&v1.StartServiceResponse{
		Result: newOperationResult(msg),
	}), nil
}

// StopService stops a single service or all applicable services when
// service_name is empty.
func (h *ServicesHandler) StopService(
	ctx context.Context,
	req *connect.Request[v1.StopServiceRequest],
) (*connect.Response[v1.StopServiceResponse], error) {
	msg, err := h.lifecycle.StopService(ctx, req.Msg.GetServiceName(), req.Msg.GetForce())
	if err != nil {
		return nil, lifecycleError(err)
	}
	return connect.NewResponse(&v1.StopServiceResponse{
		Result: newOperationResult(msg),
	}), nil
}

// RestartService restarts a single service or all services when service_name
// is empty.
func (h *ServicesHandler) RestartService(
	ctx context.Context,
	req *connect.Request[v1.RestartServiceRequest],
) (*connect.Response[v1.RestartServiceResponse], error) {
	msg, err := h.lifecycle.RestartService(ctx, req.Msg.GetServiceName())
	if err != nil {
		return nil, lifecycleError(err)
	}
	return connect.NewResponse(&v1.RestartServiceResponse{
		Result: newOperationResult(msg),
	}), nil
}

// =============================================================================
// Helpers
// =============================================================================

// lifecycleError maps a ServiceLifecycle error to a Connect status code.
// Wrapped sentinels are honored via errors.Is.
func lifecycleError(err error) error {
	switch {
	case errors.Is(err, ErrServiceNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrServiceInvalidState):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrServiceOpInProgress):
		return connect.NewError(connect.CodeAlreadyExists, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

// newOperationResult builds the standard mutating-RPC response envelope:
// success=true (failures travel as Connect errors, not in-band booleans),
// human-readable message, freshly minted operation_id for correlation with
// broadcast events, and a server-side completion timestamp.
//
// operation_id is intentionally a UUID rather than a monotonic counter so
// log correlation across server restarts is unambiguous.
func newOperationResult(message string) *v1.OperationResult {
	return &v1.OperationResult{
		Success:     true,
		Message:     message,
		OperationId: uuid.NewString(),
		CompletedAt: timestamppb.Now(),
	}
}

// serviceInfoToProto translates serviceinfo.ServiceInfo to the wire type. It
// flattens the Local/Azure nested structs and routes fields with no typed
// proto home (CustomURL, ImageName, ServiceMode, etc.) into the metadata
// google.protobuf.Struct so the dashboard can keep rendering them without
// proto schema churn during the migration.
func serviceInfoToProto(info *serviceinfo.ServiceInfo, projectDir string) *v1.ServiceInfo {
	out := &v1.ServiceInfo{
		Name:        info.Name,
		Framework:   info.Framework,
		Language:    info.Language,
		Kind:        info.Host,
		ProjectDir:  projectDir,
		Status:      v1.ServiceStatus_SERVICE_STATUS_UNSPECIFIED,
		Environment: cloneStringMap(info.EnvironmentVars),
	}

	metadata := map[string]any{}
	if info.Project != "" && info.Project != projectDir {
		// Defensive: the dashboard mostly ignores per-service project paths
		// but we surface a divergence rather than silently overwriting.
		metadata["azureYamlProject"] = info.Project
	}

	if local := info.Local; local != nil {
		out.Status = mapServiceStatus(local.Status)
		out.Health = mapHealthState(local.Health)
		out.Port = int32(local.Port)
		out.Pid = int32(local.PID)
		// CustomURL takes precedence: it's how dashboard users override
		// auto-discovered URLs (ngrok, custom domains in dev).
		if local.CustomURL != "" {
			out.Url = local.CustomURL
		} else {
			out.Url = local.URL
		}
		if local.StartTime != nil {
			out.StartTime = timestamppb.New(*local.StartTime)
		}
		if local.LastChecked != nil {
			metadata["lastChecked"] = local.LastChecked.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		if local.ServiceType != "" {
			metadata["serviceType"] = local.ServiceType
		}
		if local.ServiceMode != "" {
			metadata["serviceMode"] = local.ServiceMode
		}
		if local.URL != "" && local.CustomURL != "" {
			// Preserve the auto-discovered URL alongside the override so
			// the UI can show both ("ngrok URL active; localhost:5000").
			metadata["autoUrl"] = local.URL
		}
		// Preserve the original health string when the proto enum can't
		// represent it precisely (e.g., "degraded" collapses to UNHEALTHY).
		if local.Health == "degraded" {
			metadata["health"] = local.Health
		}
	}

	if azure := info.Azure; azure != nil {
		out.Azure = &v1.AzureDeploymentInfo{
			ResourceId:   azure.ResourceName,
			ResourceType: info.Host,
		}
		azureMeta := map[string]any{}
		if azure.URL != "" {
			azureMeta["url"] = azure.URL
		}
		if azure.CustomURL != "" {
			azureMeta["customUrl"] = azure.CustomURL
		}
		if azure.CustomDomain != "" {
			azureMeta["customDomain"] = azure.CustomDomain
		}
		if azure.CustomDomainSource != "" {
			azureMeta["customDomainSource"] = azure.CustomDomainSource
		}
		if azure.ImageName != "" {
			azureMeta["imageName"] = azure.ImageName
		}
		if len(azureMeta) > 0 {
			metadata["azure"] = azureMeta
		}
	}

	if len(metadata) > 0 {
		s, err := structpb.NewStruct(metadata)
		if err == nil {
			out.Metadata = s
		}
		// structpb.NewStruct only fails on unsupported value types; all
		// values we put in are plain strings/maps so an error here is a
		// programming bug. Drop metadata silently rather than failing the
		// whole list call -- losing metadata is strictly less bad than a
		// 500 on /api/services.
	}

	return out
}

// mapServiceStatus translates the legacy string-typed status (running,
// not-running, ...) to the proto enum. Unknown / empty statuses map to
// UNSPECIFIED so future statuses don't cause hard failures.
func mapServiceStatus(s string) v1.ServiceStatus {
	switch s {
	case constants.StatusRunning, constants.StatusReady:
		return v1.ServiceStatus_SERVICE_STATUS_READY
	case constants.StatusStarting:
		return v1.ServiceStatus_SERVICE_STATUS_STARTING
	case constants.StatusStopped, constants.StatusNotRunning:
		return v1.ServiceStatus_SERVICE_STATUS_STOPPED
	case constants.StatusStopping:
		return v1.ServiceStatus_SERVICE_STATUS_STOPPING
	case constants.StatusError:
		return v1.ServiceStatus_SERVICE_STATUS_ERROR
	default:
		return v1.ServiceStatus_SERVICE_STATUS_UNSPECIFIED
	}
}

// mapHealthState translates the legacy string-typed health to the proto enum.
// "degraded" has no dedicated enum value yet, so it maps to UNHEALTHY (the
// closest non-OK semantic) and the original string is preserved in the
// metadata Struct for finer-grained UI rendering.
func mapHealthState(h string) v1.HealthState {
	switch h {
	case "healthy":
		return v1.HealthState_HEALTH_STATE_HEALTHY
	case "unhealthy", "degraded":
		return v1.HealthState_HEALTH_STATE_UNHEALTHY
	case "starting":
		return v1.HealthState_HEALTH_STATE_STARTING
	case "unknown":
		return v1.HealthState_HEALTH_STATE_UNKNOWN
	default:
		return v1.HealthState_HEALTH_STATE_UNSPECIFIED
	}
}

// cloneStringMap returns a shallow copy or nil when src is empty. Avoids
// aliasing the caller's map into the proto message (which would let later
// mutations leak across goroutines).
func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
