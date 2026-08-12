// Package servicetarget provides custom service target implementations for azd-app.
package servicetarget

import (
	"context"
	"log/slog"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
)

// Ensure LocalServiceTargetProvider implements ServiceTargetProvider interface
var _ azdext.ServiceTargetProvider = &LocalServiceTargetProvider{}

// LocalServiceTargetProvider handles services with host: local
// These are local-only containers (like azurite, cosmos emulator, redis, postgres)
// that should be started during `azd app run` but skipped during `azd deploy`.
//
// # Why this does not embed azdext.BaseServiceTargetProvider
//
// The base provider answers every method with a nil result. None of the methods
// below are no-ops, despite reading like it: each returns a non-nil success
// sentinel so the azd host records the service as handled and carries on with
// the rest of the deployment. GetTargetResource in particular returns a
// synthetic "local" resource rather than nil. Swapping in the base defaults
// would turn "skipped cleanly" into "produced no result", which is exactly the
// failure this provider exists to avoid.
//
// Embedding purely for forward compatibility is also the wrong trade here. It
// would silently give any method added to ServiceTargetProvider in a future SDK
// the nil-returning default, which is the wrong answer for a local-only target.
// Leaving the interface satisfied explicitly means a new method breaks the
// build and forces a deliberate decision. See TestBaseProviderIsNotASubstitute.
type LocalServiceTargetProvider struct {
	azdClient     *azdext.AzdClient
	serviceConfig *azdext.ServiceConfig
}

// NewLocalServiceTargetProvider creates a new LocalServiceTargetProvider instance
func NewLocalServiceTargetProvider(azdClient *azdext.AzdClient) azdext.ServiceTargetProvider {
	return &LocalServiceTargetProvider{
		azdClient: azdClient,
	}
}

// Initialize initializes the service target provider with service configuration
func (p *LocalServiceTargetProvider) Initialize(ctx context.Context, serviceConfig *azdext.ServiceConfig) error {
	p.serviceConfig = serviceConfig
	slog.Info("local service target provider initialized", "service", serviceConfig.GetName())
	return nil
}

// Endpoints returns endpoints exposed by the service.
// Local services don't have Azure endpoints.
func (p *LocalServiceTargetProvider) Endpoints(
	ctx context.Context,
	serviceConfig *azdext.ServiceConfig,
	targetResource *azdext.TargetResource,
) ([]string, error) {
	// Local containers don't have Azure endpoints
	// Return empty list - the service is available locally via Docker
	slog.Info("local service has no Azure endpoints", "service", serviceConfig.GetName())
	return []string{}, nil
}

// GetTargetResource returns a target resource for the service.
// Local services don't have Azure target resources.
func (p *LocalServiceTargetProvider) GetTargetResource(
	ctx context.Context,
	subscriptionID string,
	serviceConfig *azdext.ServiceConfig,
	defaultResolver func() (*azdext.TargetResource, error),
) (*azdext.TargetResource, error) {
	slog.Info("local service has no Azure target resource", "service", serviceConfig.GetName())
	// Return a minimal target resource indicating this is local-only
	return &azdext.TargetResource{
		ResourceName: serviceConfig.GetName(),
		ResourceType: "local",
	}, nil
}

// Package performs packaging for the service.
// Local services don't need packaging for Azure deployment - skip silently.
func (p *LocalServiceTargetProvider) Package(
	ctx context.Context,
	serviceConfig *azdext.ServiceConfig,
	serviceContext *azdext.ServiceContext,
	progress azdext.ProgressReporter,
) (*azdext.ServicePackageResult, error) {
	slog.Info("skipping package for local-only service", "service", serviceConfig.GetName())
	// Return success with empty artifacts to allow deployment to continue for other services
	// Local services (azurite, postgres, redis, etc.) only run via 'azd app run'
	return &azdext.ServicePackageResult{
		Artifacts: []*azdext.Artifact{},
	}, nil
}

// Publish performs the publish operation for the service.
// Local services don't need publishing to Azure - skip silently.
func (p *LocalServiceTargetProvider) Publish(
	ctx context.Context,
	serviceConfig *azdext.ServiceConfig,
	serviceContext *azdext.ServiceContext,
	targetResource *azdext.TargetResource,
	publishOptions *azdext.PublishOptions,
	progress azdext.ProgressReporter,
) (*azdext.ServicePublishResult, error) {
	slog.Info("skipping publish for local-only service", "service", serviceConfig.GetName())
	// Return success to allow deployment to continue for other services
	return &azdext.ServicePublishResult{}, nil
}

// Deploy performs the deployment operation for the service.
// Local services don't deploy to Azure - skip silently.
func (p *LocalServiceTargetProvider) Deploy(
	ctx context.Context,
	serviceConfig *azdext.ServiceConfig,
	serviceContext *azdext.ServiceContext,
	targetResource *azdext.TargetResource,
	progress azdext.ProgressReporter,
) (*azdext.ServiceDeployResult, error) {
	slog.Info("skipping deploy for local-only service", "service", serviceConfig.GetName())
	// Return success to allow deployment to continue for other services
	// Local services run via 'azd app run', not 'azd deploy'
	return &azdext.ServiceDeployResult{
		Artifacts: []*azdext.Artifact{},
	}, nil
}
