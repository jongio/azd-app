package commands

import (
	"os"
	"testing"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/azure/azure-dev/cli/azd/pkg/extensions"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// manifestCapabilities reads the capabilities list from cli/extension.yaml.
type manifestDocument struct {
	Capabilities []extensions.CapabilityType `yaml:"capabilities"`
	Providers    []struct {
		Name string                  `yaml:"name"`
		Type extensions.ProviderType `yaml:"type"`
	} `yaml:"providers"`
}

func loadManifest(t *testing.T) manifestDocument {
	t.Helper()

	data, err := os.ReadFile(manifestPath(t))
	require.NoError(t, err)

	var doc manifestDocument
	require.NoError(t, yaml.Unmarshal(data, &doc))
	require.NotEmpty(t, doc.Capabilities, "extension.yaml must declare capabilities")
	return doc
}

func hasCapability(doc manifestDocument, want extensions.CapabilityType) bool {
	for _, c := range doc.Capabilities {
		if c == want {
			return true
		}
	}
	return false
}

// configuredHost runs the real host configuration against a host with no azd
// connection. Registration is lazy (pinned by
// TestConfigureExtensionHostRegistrationIsLazy) so this is safe.
func configuredHost(t *testing.T) *azdext.ExtensionHost {
	t.Helper()
	host := azdext.NewExtensionHost(nil)
	ConfigureExtensionHost(host)
	return host
}

// TestCapabilitiesMatchRegisteredProviders closes a gap that
// azdext.VerifyProvidersMatchManifest leaves open.
//
// The verifier compares the manifest's providers list against what the
// extension registers, but azd gates on the separate capabilities list before
// it ever looks at providers. A provider can therefore be declared and
// registered correctly and still be unreachable because the matching
// capability is missing, which produces no error anywhere: azd simply never
// asks. This asserts the two lists agree in both directions.
func TestCapabilitiesMatchRegisteredProviders(t *testing.T) {
	doc := loadManifest(t)
	host := configuredHost(t)

	cases := []struct {
		capability extensions.CapabilityType
		registered int
	}{
		{extensions.ServiceTargetProviderCapability, len(host.ServiceTargets())},
		{extensions.ProvisioningProviderCapability, len(host.ProvisioningProviders())},
	}

	for _, tc := range cases {
		declared := hasCapability(doc, tc.capability)
		switch {
		case tc.registered > 0 && !declared:
			t.Errorf("the extension registers %d %s provider(s) but extension.yaml does not declare "+
				"the %q capability, so azd will never route to them",
				tc.registered, tc.capability, tc.capability)
		case tc.registered == 0 && declared:
			t.Errorf("extension.yaml declares the %q capability but the extension registers no such "+
				"provider, so azd will ask for one and get nothing", tc.capability)
		}
	}
}

// TestProvisioningProviderIsNotAdopted records a deliberate decision so it is
// not quietly reversed.
//
// azdext.ExtensionHost.WithProvisioningProvider registers an implementation of
// the full infrastructure lifecycle: Initialize, State, Deploy, Preview,
// Destroy, EnsureEnv, Parameters and PlannedOutputs. azd routes to it by the
// name a user puts in azure.yaml's infra provider field, which makes it the
// extension point for alternative IaC engines such as Terraform or Pulumi. It
// replaces Bicep for that project.
//
// azd app has no infrastructure. It starts local processes and containers for
// development. It has nothing to preview, no parameters to collect, no outputs
// to publish and nothing to destroy, so seven of the eight methods could only
// be stubs, and a user who selected it would get local processes instead of
// the Azure resources they asked azd to create.
//
// The correct integration for a local runtime is the service target provider
// already registered under the name "local", which lets azd deploy skip local
// services and keep going. That is in place.
//
// If this test fails, someone registered a provisioning provider. Read the
// above, decide whether it still applies, and delete this test as part of the
// same change rather than editing it to pass.
func TestProvisioningProviderIsNotAdopted(t *testing.T) {
	require.Empty(t, configuredHost(t).ProvisioningProviders(),
		"azd app registers no provisioning provider; see the comment on this test")
	require.False(t, hasCapability(loadManifest(t), extensions.ProvisioningProviderCapability),
		"extension.yaml must not claim the provisioning-provider capability")
}

// TestValidationProviderIsNotAdopted records the second half of the same
// decision.
//
// azdext.ExtensionHost.WithValidationCheck registers a rule that azd runs
// inside azd provision, receiving either ARM template context
// ("arm-provision") or ambient environment values ("provision"). azd collects
// the results and reports them as provisioning gates.
//
// Everything azd app knows how to check is local: whether node or docker is
// installed, whether a service command parses, whether a port is already
// bound, whether a local .env holds a secret. None of it affects whether Azure
// resources can be created, so surfacing it as a provisioning gate would block
// or warn on a cloud operation for a local development reason.
//
// The cost is not hypothetical. A registered check runs for every azd provision
// by every user who has this extension installed, including users who never
// run azd app at all. A bug in the check would sit directly in the path of
// their deployments.
//
// The extension already carries the lifecycle-events capability, so if a
// genuine pre-provision need appears it can be met with
// WithServiceEventHandler("preprovision", ...), the same lighter mechanism
// already used for postprovision. That keeps the failure surface owned by this
// extension instead of being aggregated into azd's provisioning verdict.
//
// If this test fails, someone registered a validation check. Read the above,
// decide whether it still applies, and delete this test as part of the same
// change rather than editing it to pass.
func TestValidationProviderIsNotAdopted(t *testing.T) {
	require.False(t, hasCapability(loadManifest(t), extensions.ValidationProviderCapability),
		"azd app registers no validation check; see the comment on this test")
}

// TestManifestCapabilitiesAreKnown catches a typo in extension.yaml. A
// capability azd does not recognize is silently ignored, so a misspelled entry
// disables the feature it was meant to enable with no error anywhere.
func TestManifestCapabilitiesAreKnown(t *testing.T) {
	known := map[extensions.CapabilityType]bool{
		extensions.CustomCommandCapability:            true,
		extensions.LifecycleEventsCapability:          true,
		extensions.McpServerCapability:                true,
		extensions.ServiceTargetProviderCapability:    true,
		extensions.FrameworkServiceProviderCapability: true,
		extensions.MetadataCapability:                 true,
		extensions.ProvisioningProviderCapability:     true,
		extensions.ValidationProviderCapability:       true,
	}

	for _, c := range loadManifest(t).Capabilities {
		require.True(t, known[c], "extension.yaml declares unknown capability %q", c)
	}
}

// TestManifestProviderTypesAreKnown does the same for the providers list.
func TestManifestProviderTypesAreKnown(t *testing.T) {
	known := map[extensions.ProviderType]bool{
		extensions.ServiceTargetProviderType: true,
		extensions.ProvisioningProviderType:  true,
	}

	doc := loadManifest(t)
	require.NotEmpty(t, doc.Providers, "extension.yaml must declare at least one provider")

	for _, p := range doc.Providers {
		require.True(t, known[p.Type],
			"provider %q declares unknown type %q", p.Name, p.Type)
	}
}
