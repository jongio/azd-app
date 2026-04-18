package rpc

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/azure"
)

// =============================================================================
// Generator: produces the consolidated Bicep template
// =============================================================================

// BicepGenerator is the narrow contract BicepHandler depends on. It mirrors
// the relevant subset of *azure.BicepGenerator so tests can inject a stub
// without touching the discovery + credential pipeline that the production
// implementation needs.
//
// Returning the rich *azure.BicepTemplateResponse (rather than re-encoding
// to a proto type here) keeps response shaping centralized in this package
// and means future RPCs can surface additional fields without touching the
// generator.
type BicepGenerator interface {
	GenerateTemplate(ctx context.Context) (*azure.BicepTemplateResponse, error)
}

// BicepGeneratorFactory builds a BicepGenerator for a single request. We use
// a factory (not a long-lived instance) because the production generator
// needs an Azure credential that is acquired per-request and a discovery
// helper bound to the project directory. Tests can return a canned generator
// without paying that setup cost.
//
// projectDir is passed explicitly (rather than captured at construction) so
// the same factory can serve multiple project roots in the future without
// requiring a re-wire; today the dashboard uses a single project, so the
// distinction is structural rather than functional.
type BicepGeneratorFactory func(ctx context.Context, projectDir string) (BicepGenerator, error)

// =============================================================================
// Sentinel errors
// =============================================================================
//
// A factory or generator implementation should return one of these (possibly
// wrapped via fmt.Errorf("...: %w", ...)) so the handler can map to a
// specific Connect status code without parsing error strings. Anything not
// classified maps to CodeInternal.

var (
	// ErrAzureCredentialsUnavailable: the Azure credential chain (env,
	// managed identity, CLI, etc.) yielded no usable credential. Maps to
	// connect.CodeFailedPrecondition because the operation cannot proceed
	// until the user signs in or configures credentials.
	ErrAzureCredentialsUnavailable = errors.New("rpc: azure credentials unavailable")

	// ErrNoAzureResources: discovery succeeded but found no resources to
	// emit diagnostic settings for. Maps to connect.CodeNotFound mirroring
	// the legacy 404 response.
	ErrNoAzureResources = errors.New("rpc: no azure resources found in environment")

	// ErrAzureDiscoveryFailed: the discovery call itself failed (network,
	// permissions, transient ARM error). Maps to connect.CodeUnavailable
	// because retrying is generally appropriate. Distinct from
	// ErrAzureCredentialsUnavailable, which is a hard prerequisite failure.
	ErrAzureDiscoveryFailed = errors.New("rpc: azure resource discovery failed")
)

// =============================================================================
// Handler
// =============================================================================

// BicepHandler implements the BicepService Connect handler. It owns no state
// beyond the factory + project root; each RPC builds a fresh generator so
// per-request credentials and discovery state stay scoped correctly.
type BicepHandler struct {
	factory    BicepGeneratorFactory
	projectDir string
	// timeout caps the total time spent building the template. The legacy
	// REST handler used 30s; we preserve that to avoid surprising changes
	// in observable latency caps. Configurable in case tests want a tighter
	// bound, but production wires the constant.
	timeout time.Duration
}

// NewBicepHandler constructs a handler. Both factory and projectDir are
// required (panics if either is missing) -- a handler with no generator is a
// programming error caught at startup, not at first request.
func NewBicepHandler(factory BicepGeneratorFactory, projectDir string) *BicepHandler {
	if factory == nil {
		panic("rpc: NewBicepHandler: factory must not be nil")
	}
	if projectDir == "" {
		panic("rpc: NewBicepHandler: projectDir must not be empty")
	}
	return &BicepHandler{
		factory:    factory,
		projectDir: projectDir,
		timeout:    30 * time.Second,
	}
}

// Compile-time assertion that BicepHandler satisfies the generated interface.
var _ azdappv1connect.BicepServiceHandler = (*BicepHandler)(nil)

// GetBicepTemplate returns the consolidated Bicep template for all detected
// services in the project's Azure environment. Mirrors the legacy
// GET /api/azure/bicep-template handler.
//
// req.Msg.ServiceNames is currently advisory: the underlying generator emits
// modules for every discovered resource type and does not support per-service
// trimming. We surface every service the generator returns so dashboards can
// render an accurate header. A future generator change can honor the filter
// without breaking the wire contract.
func (h *BicepHandler) GetBicepTemplate(
	ctx context.Context,
	_ *connect.Request[v1.GetBicepTemplateRequest],
) (*connect.Response[v1.GetBicepTemplateResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	gen, err := h.factory(ctx, h.projectDir)
	if err != nil {
		return nil, classifyBicepError(ctx, err)
	}

	res, err := gen.GenerateTemplate(ctx)
	if err != nil {
		return nil, classifyBicepError(ctx, err)
	}

	return connect.NewResponse(&v1.GetBicepTemplateResponse{
		Template:         res.Template,
		IncludedServices: res.Services,
		// WorkspaceId is intentionally empty: the legacy REST response did
		// not surface it (the value is baked into the template as a
		// parameter the user supplies at deploy time). When the generator
		// learns to resolve a concrete workspace ID we can populate this.
		WorkspaceId: "",
		GeneratedAt: timestamppb.New(time.Now().UTC()),
	}), nil
}

// classifyBicepError maps an internal error to a Connect status code. We
// prefer errors.Is over string matching so wrapped sentinel errors remain
// classifiable, but we keep substring fallbacks for the existing
// fmt.Errorf("no Azure resources found...") call sites in azure/bicep.go --
// rewriting those to wrap sentinels is a separate cleanup.
func classifyBicepError(ctx context.Context, err error) error {
	// A cancelled/expired ctx wins over the underlying error: callers care
	// that they hit a deadline, not that discovery returned ECONNRESET.
	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			return connect.NewError(connect.CodeDeadlineExceeded, err)
		}
		return connect.NewError(connect.CodeCanceled, err)
	}

	switch {
	case errors.Is(err, ErrAzureCredentialsUnavailable):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrNoAzureResources):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrAzureDiscoveryFailed):
		return connect.NewError(connect.CodeUnavailable, err)
	}

	// String-fallback for the existing azure package errors. These match
	// the substrings the legacy REST handler tested against.
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no Azure resources found"):
		return connect.NewError(connect.CodeNotFound, err)
	case strings.Contains(msg, "failed to discover resources"):
		return connect.NewError(connect.CodeUnavailable, err)
	}

	return connect.NewError(connect.CodeInternal, err)
}
