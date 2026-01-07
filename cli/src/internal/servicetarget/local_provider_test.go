package servicetarget

import (
	"context"
	"testing"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
)

// mockProgress is a simple mock for testing
func mockProgress(message string) {
	// No-op for testing
}

// TestNewLocalServiceTargetProvider tests provider creation
func TestNewLocalServiceTargetProvider(t *testing.T) {
	provider := NewLocalServiceTargetProvider(nil)
	if provider == nil {
		t.Fatal("NewLocalServiceTargetProvider() returned nil")
	}

	// NewLocalServiceTargetProvider returns a ServiceTargetProvider interface; ensure it's non-nil
}

// TestLocalServiceTargetProvider_Initialize tests provider initialization
func TestLocalServiceTargetProvider_Initialize(t *testing.T) {
	provider := &LocalServiceTargetProvider{}
	ctx := context.Background()

	serviceConfig := &azdext.ServiceConfig{
		Name: "test-service",
	}

	err := provider.Initialize(ctx, serviceConfig)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if provider.serviceConfig == nil {
		t.Error("Initialize() did not set serviceConfig")
	}

	if provider.serviceConfig.GetName() != "test-service" {
		t.Errorf("Initialize() serviceConfig.Name = %v, want 'test-service'", provider.serviceConfig.GetName())
	}
}

// TestLocalServiceTargetProvider_Endpoints tests endpoint retrieval
func TestLocalServiceTargetProvider_Endpoints(t *testing.T) {
	provider := &LocalServiceTargetProvider{}
	ctx := context.Background()

	serviceConfig := &azdext.ServiceConfig{
		Name: "test-service",
	}

	endpoints, err := provider.Endpoints(ctx, serviceConfig, nil)
	if err != nil {
		t.Fatalf("Endpoints() error = %v", err)
	}

	// Local services should return empty endpoints (no Azure endpoints)
	if len(endpoints) != 0 {
		t.Errorf("Endpoints() = %v, want empty slice", endpoints)
	}
}

// TestLocalServiceTargetProvider_GetTargetResource tests target resource retrieval
func TestLocalServiceTargetProvider_GetTargetResource(t *testing.T) {
	provider := &LocalServiceTargetProvider{}
	ctx := context.Background()

	serviceConfig := &azdext.ServiceConfig{
		Name: "azurite",
	}

	targetResource, err := provider.GetTargetResource(ctx, "subscription-123", serviceConfig, nil)
	if err != nil {
		t.Fatalf("GetTargetResource() error = %v", err)
	}

	if targetResource == nil {
		t.Fatal("GetTargetResource() returned nil")
	}

	if targetResource.ResourceName != "azurite" {
		t.Errorf("GetTargetResource().ResourceName = %v, want 'azurite'", targetResource.ResourceName)
	}

	if targetResource.ResourceType != "local" {
		t.Errorf("GetTargetResource().ResourceType = %v, want 'local'", targetResource.ResourceType)
	}
}

// TestLocalServiceTargetProvider_Package tests packaging operation
func TestLocalServiceTargetProvider_Package(t *testing.T) {
	provider := &LocalServiceTargetProvider{}
	ctx := context.Background()

	serviceConfig := &azdext.ServiceConfig{
		Name: "postgres",
	}

	serviceContext := &azdext.ServiceContext{}

	result, err := provider.Package(ctx, serviceConfig, serviceContext, mockProgress)
	if err != nil {
		t.Fatalf("Package() error = %v", err)
	}

	if result == nil {
		t.Fatal("Package() returned nil result")
	}

	// Local services should return empty artifacts (not packaged for deployment)
	if len(result.Artifacts) != 0 {
		t.Errorf("Package() returned %d artifacts, want 0", len(result.Artifacts))
	}
}

// TestLocalServiceTargetProvider_Publish tests publishing operation
func TestLocalServiceTargetProvider_Publish(t *testing.T) {
	provider := &LocalServiceTargetProvider{}
	ctx := context.Background()

	serviceConfig := &azdext.ServiceConfig{
		Name: "redis",
	}

	serviceContext := &azdext.ServiceContext{}
	targetResource := &azdext.TargetResource{
		ResourceName: "redis",
		ResourceType: "local",
	}
	publishOptions := &azdext.PublishOptions{}

	result, err := provider.Publish(ctx, serviceConfig, serviceContext, targetResource, publishOptions, mockProgress)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if result == nil {
		t.Fatal("Publish() returned nil result")
	}

	// Local services don't publish to Azure - should succeed with empty result
}

// TestLocalServiceTargetProvider_Deploy tests deployment operation
func TestLocalServiceTargetProvider_Deploy(t *testing.T) {
	provider := &LocalServiceTargetProvider{}
	ctx := context.Background()

	serviceConfig := &azdext.ServiceConfig{
		Name: "cosmos-emulator",
	}

	serviceContext := &azdext.ServiceContext{}
	targetResource := &azdext.TargetResource{
		ResourceName: "cosmos-emulator",
		ResourceType: "local",
	}

	result, err := provider.Deploy(ctx, serviceConfig, serviceContext, targetResource, mockProgress)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	if result == nil {
		t.Fatal("Deploy() returned nil result")
	}

	// Local services should return empty artifacts (not deployed to Azure)
	if len(result.Artifacts) != 0 {
		t.Errorf("Deploy() returned %d artifacts, want 0", len(result.Artifacts))
	}
}

// TestLocalServiceTargetProvider_FullWorkflow tests a complete workflow
func TestLocalServiceTargetProvider_FullWorkflow(t *testing.T) {
	provider := &LocalServiceTargetProvider{}
	ctx := context.Background()

	serviceConfig := &azdext.ServiceConfig{
		Name: "azurite",
	}

	// Initialize
	err := provider.Initialize(ctx, serviceConfig)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Get target resource
	targetResource, err := provider.GetTargetResource(ctx, "sub-123", serviceConfig, nil)
	if err != nil {
		t.Fatalf("GetTargetResource() error = %v", err)
	}

	// Package
	serviceContext := &azdext.ServiceContext{}
	_, err = provider.Package(ctx, serviceConfig, serviceContext, mockProgress)
	if err != nil {
		t.Fatalf("Package() error = %v", err)
	}

	// Publish
	publishOptions := &azdext.PublishOptions{}
	_, err = provider.Publish(ctx, serviceConfig, serviceContext, targetResource, publishOptions, mockProgress)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	// Deploy
	_, err = provider.Deploy(ctx, serviceConfig, serviceContext, targetResource, mockProgress)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	// Get endpoints
	endpoints, err := provider.Endpoints(ctx, serviceConfig, targetResource)
	if err != nil {
		t.Fatalf("Endpoints() error = %v", err)
	}

	if len(endpoints) != 0 {
		t.Errorf("Endpoints() = %v, want empty slice for local service", endpoints)
	}
}
