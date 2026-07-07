package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/cliout"
)

// envDiffValues holds the masked value of a single variable in each of the two
// compared services. Only the side that has the variable is populated for
// "only in" entries; both are populated for changed entries.
type envDiffValues struct {
	A string `json:"a"`
	B string `json:"b"`
}

// envDiffResult is the JSON shape for "env --diff". OnlyInA and OnlyInB list the
// variables unique to each service, Changed lists the variables present in both
// with different values, and Same is the count of variables that match.
type envDiffResult struct {
	ServiceA string                   `json:"serviceA"`
	ServiceB string                   `json:"serviceB"`
	OnlyInA  map[string]string        `json:"onlyInA"`
	OnlyInB  map[string]string        `json:"onlyInB"`
	Changed  map[string]envDiffValues `json:"changed"`
	Same     int                      `json:"same"`
}

// runEnvDiff resolves the environment for two services and reports the
// differences: variables only in the first, only in the second, and those
// present in both with different values. Both services are resolved before any
// output so a resolution failure is reported without a partial diff. Secret
// shaped values are masked unless --no-mask is set.
func runEnvDiff(azureYaml *service.AzureYaml, names []string, args []string) error {
	if envAll {
		return fmt.Errorf("cannot combine --diff with --all")
	}
	if envExplain {
		return fmt.Errorf("cannot combine --diff with --explain")
	}
	if len(args) != 2 {
		return fmt.Errorf("--diff needs exactly two service names, for example: azd app env --diff api web")
	}

	nameA, nameB := args[0], args[1]
	if nameA == nameB {
		return fmt.Errorf("--diff needs two different service names")
	}
	for _, name := range []string{nameA, nameB} {
		if _, ok := azureYaml.Services[name]; !ok {
			if len(names) == 0 {
				return fmt.Errorf("service %q not found. No services are defined in azure.yaml", name)
			}
			return fmt.Errorf("service %q not found. Available services: %s",
				name, strings.Join(names, ", "))
		}
	}

	mask := !envNoMask
	resolvedA, err := resolveServiceEnv(nameA, azureYaml.Services[nameA])
	if err != nil {
		return err
	}
	resolvedB, err := resolveServiceEnv(nameB, azureYaml.Services[nameB])
	if err != nil {
		return err
	}

	maskedA := maskEnv(resolvedA, mask)
	maskedB := maskEnv(resolvedB, mask)
	result := buildEnvDiff(nameA, nameB, maskedA, maskedB)

	if envFormatJSONSelected() {
		return cliout.PrintJSON(result)
	}
	printEnvDiff(result)
	return nil
}

// resolveServiceEnv resolves the effective environment for a single service,
// wrapping any failure with the service name for context.
func resolveServiceEnv(name string, svc service.Service) (map[string]string, error) {
	resolved, err := service.ResolveEnvironment(context.Background(), svc, getAzureEnvironmentValues(), envFile, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve environment for %q: %w", name, err)
	}
	return resolved, nil
}

// envFormatJSONSelected reports whether JSON output was requested, either through
// --format json or the global --json flag.
func envFormatJSONSelected() bool {
	if cliout.IsJSON() {
		return true
	}
	format, err := resolveEnvFormat(envFormat)
	return err == nil && format == envFormatJSON
}

// buildEnvDiff computes the difference between two already-masked environments.
func buildEnvDiff(nameA, nameB string, a, b map[string]string) envDiffResult {
	result := envDiffResult{
		ServiceA: nameA,
		ServiceB: nameB,
		OnlyInA:  map[string]string{},
		OnlyInB:  map[string]string{},
		Changed:  map[string]envDiffValues{},
	}

	for k, va := range a {
		vb, ok := b[k]
		switch {
		case !ok:
			result.OnlyInA[k] = va
		case va != vb:
			result.Changed[k] = envDiffValues{A: va, B: vb}
		default:
			result.Same++
		}
	}
	for k, vb := range b {
		if _, ok := a[k]; !ok {
			result.OnlyInB[k] = vb
		}
	}
	return result
}

// printEnvDiff writes the diff in a readable text form, grouping variables that
// are unique to each service and those whose values differ.
func printEnvDiff(r envDiffResult) {
	if len(r.OnlyInA) == 0 && len(r.OnlyInB) == 0 && len(r.Changed) == 0 {
		cliout.Info("%s and %s have the same environment (%d variables)", r.ServiceA, r.ServiceB, r.Same)
		return
	}

	fmt.Printf("Comparing environment: %s vs %s\n", r.ServiceA, r.ServiceB)

	if len(r.OnlyInA) > 0 {
		fmt.Printf("\nOnly in %s:\n", r.ServiceA)
		for _, k := range sortedKeys(r.OnlyInA) {
			fmt.Printf("  %s=%s\n", k, r.OnlyInA[k])
		}
	}

	if len(r.OnlyInB) > 0 {
		fmt.Printf("\nOnly in %s:\n", r.ServiceB)
		for _, k := range sortedKeys(r.OnlyInB) {
			fmt.Printf("  %s=%s\n", k, r.OnlyInB[k])
		}
	}

	if len(r.Changed) > 0 {
		fmt.Print("\nDifferent values:\n")
		for _, k := range sortedChangedKeys(r.Changed) {
			v := r.Changed[k]
			fmt.Printf("  %s:\n", k)
			fmt.Printf("    %s: %s\n", r.ServiceA, v.A)
			fmt.Printf("    %s: %s\n", r.ServiceB, v.B)
		}
	}
}

// sortedKeys returns the keys of a string map in sorted order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// sortedChangedKeys returns the keys of a changed-values map in sorted order.
func sortedChangedKeys(m map[string]envDiffValues) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
