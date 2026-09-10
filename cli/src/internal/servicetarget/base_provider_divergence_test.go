package servicetarget

import (
	"context"
	"testing"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBaseProviderIsNotASubstitute states, executably, why
// LocalServiceTargetProvider does not embed azdext.BaseServiceTargetProvider.
//
// The base answers every method with a nil result. This provider answers with a
// non-nil success sentinel so the azd host records the service as handled and
// continues deploying the rest of the project. Deleting a method here in favour
// of the embedded default would silently change "skipped cleanly" into
// "produced no result".
//
// If a future SDK changes the base to return non-nil sentinels, this test fails
// and embedding becomes worth reconsidering.
func TestBaseProviderIsNotASubstitute(t *testing.T) {
	ctx := context.Background()
	base := &azdext.BaseServiceTargetProvider{}
	local := &LocalServiceTargetProvider{}
	cfg := &azdext.ServiceConfig{Name: "azurite"}
	svcCtx := &azdext.ServiceContext{}
	target := &azdext.TargetResource{ResourceName: "azurite", ResourceType: "local"}

	t.Run("GetTargetResource", func(t *testing.T) {
		baseRes, err := base.GetTargetResource(ctx, "sub", cfg, nil)
		require.NoError(t, err)
		assert.Nil(t, baseRes, "base returns no target resource")

		localRes, err := local.GetTargetResource(ctx, "sub", cfg, nil)
		require.NoError(t, err)
		require.NotNil(t, localRes, "local must synthesize a target resource")
		assert.Equal(t, "local", localRes.ResourceType)
		assert.Equal(t, "azurite", localRes.ResourceName)
	})

	t.Run("Package", func(t *testing.T) {
		baseRes, err := base.Package(ctx, cfg, svcCtx, mockProgress)
		require.NoError(t, err)
		assert.Nil(t, baseRes, "base returns no package result")

		localRes, err := local.Package(ctx, cfg, svcCtx, mockProgress)
		require.NoError(t, err)
		assert.NotNil(t, localRes, "local must report a successful package")
	})

	t.Run("Publish", func(t *testing.T) {
		opts := &azdext.PublishOptions{}
		baseRes, err := base.Publish(ctx, cfg, svcCtx, target, opts, mockProgress)
		require.NoError(t, err)
		assert.Nil(t, baseRes, "base returns no publish result")

		localRes, err := local.Publish(ctx, cfg, svcCtx, target, opts, mockProgress)
		require.NoError(t, err)
		assert.NotNil(t, localRes, "local must report a successful publish")
	})

	t.Run("Deploy", func(t *testing.T) {
		baseRes, err := base.Deploy(ctx, cfg, svcCtx, target, mockProgress)
		require.NoError(t, err)
		assert.Nil(t, baseRes, "base returns no deploy result")

		localRes, err := local.Deploy(ctx, cfg, svcCtx, target, mockProgress)
		require.NoError(t, err)
		assert.NotNil(t, localRes, "local must report a successful deploy")
	})

	t.Run("Endpoints", func(t *testing.T) {
		baseRes, err := base.Endpoints(ctx, cfg, target)
		require.NoError(t, err)
		assert.Nil(t, baseRes, "base returns a nil slice")

		localRes, err := local.Endpoints(ctx, cfg, target)
		require.NoError(t, err)
		assert.NotNil(t, localRes, "local returns an allocated empty slice, which marshals as [] not null")
		assert.Empty(t, localRes)
	})

	t.Run("Initialize", func(t *testing.T) {
		require.NoError(t, base.Initialize(ctx, cfg))

		fresh := &LocalServiceTargetProvider{}
		require.NoError(t, fresh.Initialize(ctx, cfg))
		assert.NotNil(t, fresh.serviceConfig,
			"local retains the service config; the base discards it")
	})
}
