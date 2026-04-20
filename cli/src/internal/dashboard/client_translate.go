// Translators that invert the rpc/services.go and rpc/logs.go forward
// mappings. The dashboard is a separate process so the CLI receives typed
// proto messages over the wire and has to reconstruct the serviceinfo /
// service structs the rest of the CLI still uses. Kept in its own file so
// the forward and reverse paths can evolve together without client.go
// ballooning past the 500-line cap.
package dashboard

import (
	"strings"
	"time"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"google.golang.org/protobuf/types/known/structpb"
)

// protoToServiceInfo inverts rpc.serviceInfoToProto. Any field that the
// forward translator routed through the metadata Struct is rehydrated here
// so downstream consumers (info command, MCP handlers) see the same shape
// they saw when the REST path served JSON directly.
func protoToServiceInfo(p *v1.ServiceInfo) *serviceinfo.ServiceInfo {
	if p == nil {
		return nil
	}

	meta := metaMap(p.GetMetadata())
	azureMeta := metaSubMap(meta, "azure")

	info := &serviceinfo.ServiceInfo{
		Name:            p.GetName(),
		Host:            p.GetKind(),
		Language:        p.GetLanguage(),
		Framework:       p.GetFramework(),
		Project:         metaString(meta, "azureYamlProject"),
		EnvironmentVars: cloneStringMap(p.GetEnvironment()),
	}

	if info.Project == "" {
		info.Project = p.GetProjectDir()
	}

	if local := buildLocalInfo(p, meta); local != nil {
		info.Local = local
	}

	if azure := buildAzureInfo(p, azureMeta); azure != nil {
		info.Azure = azure
	}

	return info
}

// buildLocalInfo reconstructs a LocalServiceInfo only when the proto carries
// evidence of local runtime state. Avoids manufacturing an empty shell for
// services that are Azure-only.
func buildLocalInfo(p *v1.ServiceInfo, meta map[string]*structpb.Value) *serviceinfo.LocalServiceInfo {
	status := statusFromProto(p.GetStatus())
	healthOverride := metaString(meta, "health")
	health := healthFromProto(p.GetHealth(), healthOverride)
	port := int(p.GetPort())
	pid := int(p.GetPid())
	url := p.GetUrl()
	autoURL := metaString(meta, "autoUrl")
	serviceType := metaString(meta, "serviceType")
	serviceMode := metaString(meta, "serviceMode")
	lastCheckedStr := metaString(meta, "lastChecked")

	hasState := status != "" || health != "" || port != 0 || pid != 0 ||
		url != "" || autoURL != "" || serviceType != "" ||
		serviceMode != "" || lastCheckedStr != "" || p.GetStartTime() != nil

	if !hasState {
		return nil
	}

	local := &serviceinfo.LocalServiceInfo{
		Status:      status,
		Health:      health,
		Port:        port,
		PID:         pid,
		ServiceType: serviceType,
		ServiceMode: serviceMode,
	}

	// serviceInfoToProto prefers CustomURL for the wire Url field and
	// stashes the auto-discovered URL under metadata["autoUrl"]. When
	// autoUrl is present we know the wire Url was an override.
	if autoURL != "" {
		local.CustomURL = url
		local.URL = autoURL
	} else {
		local.URL = url
	}

	if p.GetStartTime() != nil {
		t := p.GetStartTime().AsTime()
		local.StartTime = &t
	}
	if lastCheckedStr != "" {
		if t, err := time.Parse(time.RFC3339, lastCheckedStr); err == nil {
			local.LastChecked = &t
		}
	}
	return local
}

// buildAzureInfo reconstructs AzureServiceInfo from the typed AzureDeploymentInfo
// and the metadata["azure"] sub-struct. Returns nil when there's nothing to
// represent so callers don't have to null-check an empty struct.
func buildAzureInfo(p *v1.ServiceInfo, azureMeta map[string]*structpb.Value) *serviceinfo.AzureServiceInfo {
	a := p.GetAzure()
	resourceID := ""
	if a != nil {
		resourceID = a.GetResourceId()
	}

	url := metaString(azureMeta, "url")
	customURL := metaString(azureMeta, "customUrl")
	customDomain := metaString(azureMeta, "customDomain")
	customDomainSource := metaString(azureMeta, "customDomainSource")
	imageName := metaString(azureMeta, "imageName")

	if resourceID == "" && url == "" && customURL == "" &&
		customDomain == "" && customDomainSource == "" && imageName == "" {
		return nil
	}

	return &serviceinfo.AzureServiceInfo{
		URL:                url,
		CustomURL:          customURL,
		CustomDomain:       customDomain,
		CustomDomainSource: customDomainSource,
		ResourceName:       resourceID,
		ImageName:          imageName,
	}
}

// statusFromProto mirrors rpc.mapServiceStatus in reverse. READY maps to
// "running" rather than "ready" to match the registry's canonical string;
// rpc.mapServiceStatus accepts both on the way in so there is no data loss.
func statusFromProto(s v1.ServiceStatus) string {
	switch s {
	case v1.ServiceStatus_SERVICE_STATUS_READY:
		return constants.StatusRunning
	case v1.ServiceStatus_SERVICE_STATUS_STARTING:
		return constants.StatusStarting
	case v1.ServiceStatus_SERVICE_STATUS_STOPPED:
		return constants.StatusNotRunning
	case v1.ServiceStatus_SERVICE_STATUS_STOPPING:
		return constants.StatusStopping
	case v1.ServiceStatus_SERVICE_STATUS_ERROR:
		return constants.StatusError
	case v1.ServiceStatus_SERVICE_STATUS_DEGRADED:
		// No matching constant; pass the lowercase enum name so the UI
		// and tests that consult this field see something meaningful.
		return "degraded"
	default:
		return ""
	}
}

// healthFromProto inverts rpc.mapHealthState. mapHealthState collapses
// "degraded" into UNHEALTHY while preserving the original string in
// metadata["health"]; the override wins when present so the dashboard sees
// "degraded" round-trip cleanly.
func healthFromProto(h v1.HealthState, override string) string {
	if override != "" {
		return override
	}
	switch h {
	case v1.HealthState_HEALTH_STATE_HEALTHY:
		return "healthy"
	case v1.HealthState_HEALTH_STATE_UNHEALTHY:
		return "unhealthy"
	case v1.HealthState_HEALTH_STATE_STARTING:
		return "starting"
	case v1.HealthState_HEALTH_STATE_UNKNOWN:
		return "unknown"
	case v1.HealthState_HEALTH_STATE_DEGRADED:
		return "degraded"
	default:
		return ""
	}
}

// protoToLogEntry inverts rpc.toProtoLogEntry. Source is mapped back to the
// string constants consumed by the rest of the CLI; IsStderr is derived from
// the LogStream field.
func protoToLogEntry(p *v1.LogEntry) service.LogEntry {
	if p == nil {
		return service.LogEntry{}
	}
	entry := service.LogEntry{
		Service:  p.GetService(),
		Message:  p.GetMessage(),
		Level:    logLevelFromProto(p.GetLevel()),
		IsStderr: p.GetStream() == v1.LogStream_LOG_STREAM_STDERR,
		Source:   logSourceFromProto(p.GetSource()),
	}
	if ts := p.GetTimestamp(); ts != nil {
		entry.Timestamp = ts.AsTime()
	}
	return entry
}

// logLevelFromProto maps the proto LogLevel enum (0..6) to the service
// package's iota enum (0..3). TRACE is folded into DEBUG and FATAL into
// ERROR; UNSPECIFIED defaults to INFO which matches the forward mapping's
// "return INFO" branch for unknown service levels.
func logLevelFromProto(l v1.LogLevel) service.LogLevel {
	switch l {
	case v1.LogLevel_LOG_LEVEL_WARN:
		return service.LogLevelWarn
	case v1.LogLevel_LOG_LEVEL_ERROR, v1.LogLevel_LOG_LEVEL_FATAL:
		return service.LogLevelError
	case v1.LogLevel_LOG_LEVEL_DEBUG, v1.LogLevel_LOG_LEVEL_TRACE:
		return service.LogLevelDebug
	default:
		return service.LogLevelInfo
	}
}

func logSourceFromProto(s v1.LogSource) string {
	switch s {
	case v1.LogSource_LOG_SOURCE_AZURE:
		return service.LogSourceAzure
	case v1.LogSource_LOG_SOURCE_LOCAL:
		return service.LogSourceLocal
	default:
		return ""
	}
}

// metaMap returns the metadata Struct as a map[string]*structpb.Value for
// ergonomic field access. Returns nil (not an empty map) when the Struct is
// absent so callers can cheaply short-circuit.
func metaMap(s *structpb.Struct) map[string]*structpb.Value {
	if s == nil {
		return nil
	}
	return s.GetFields()
}

func metaString(meta map[string]*structpb.Value, key string) string {
	if meta == nil {
		return ""
	}
	v, ok := meta[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(v.GetStringValue())
}

func metaSubMap(meta map[string]*structpb.Value, key string) map[string]*structpb.Value {
	if meta == nil {
		return nil
	}
	v, ok := meta[key]
	if !ok || v == nil {
		return nil
	}
	sub := v.GetStructValue()
	if sub == nil {
		return nil
	}
	return sub.GetFields()
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
