package commands

import (
	"os"
	"sort"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/service"

	"github.com/spf13/cobra"
)

// serviceNamesFromAzureYaml returns the service names defined in the project's
// azure.yaml, sorted alphabetically. It returns nil when azure.yaml cannot be
// found or parsed so that shell completion degrades to no suggestions rather
// than surfacing an error to the user's terminal.
func serviceNamesFromAzureYaml() []string {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	azureYaml, err := service.ParseAzureYaml(cwd)
	if err != nil || azureYaml == nil {
		return nil
	}

	names := make([]string, 0, len(azureYaml.Services))
	for name := range azureYaml.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// completeServiceNames is a cobra flag completion function for the --service
// flag. It suggests service names read from azure.yaml.
//
// The --service flag accepts a comma-separated list, so completion only fills
// in the segment after the last comma and skips names already chosen earlier in
// the same value. It always returns ShellCompDirectiveNoFileComp so the shell
// does not fall back to file-name completion.
func completeServiceNames(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	all := serviceNamesFromAzureYaml()
	if len(all) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	prefix, last, chosen := splitServiceCompletion(toComplete)

	suggestions := make([]string, 0, len(all))
	for _, name := range all {
		if chosen[name] {
			continue
		}
		if strings.HasPrefix(name, last) {
			suggestions = append(suggestions, prefix+name)
		}
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// splitServiceCompletion breaks an in-progress --service value on commas. It
// returns the already-typed prefix (everything up to and including the last
// comma), the final segment still being typed, and the set of names already
// present before the final segment.
func splitServiceCompletion(toComplete string) (prefix, last string, chosen map[string]bool) {
	chosen = make(map[string]bool)

	idx := strings.LastIndex(toComplete, ",")
	if idx < 0 {
		return "", toComplete, chosen
	}

	prefix = toComplete[:idx+1]
	last = toComplete[idx+1:]
	for _, part := range strings.Split(toComplete[:idx], ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			chosen[part] = true
		}
	}
	return prefix, last, chosen
}

// registerServiceFlagCompletion wires completeServiceNames to the given flag on
// cmd. It is a no-op when the flag does not exist so callers can register it
// unconditionally after defining flags.
func registerServiceFlagCompletion(cmd *cobra.Command, flagName string) {
	_ = cmd.RegisterFlagCompletionFunc(flagName, completeServiceNames)
}

// completeServiceArgs is a cobra ValidArgsFunction for commands that accept one
// or more service names as positional arguments (for example env, info, and
// logs). It suggests service names read from azure.yaml, skips names already
// given earlier on the command line, and always returns
// ShellCompDirectiveNoFileComp so the shell does not fall back to file-name
// completion.
func completeServiceArgs(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	all := serviceNamesFromAzureYaml()
	if len(all) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	chosen := make(map[string]bool, len(args))
	for _, a := range args {
		chosen[a] = true
	}

	suggestions := make([]string, 0, len(all))
	for _, name := range all {
		if chosen[name] {
			continue
		}
		if strings.HasPrefix(name, toComplete) {
			suggestions = append(suggestions, name)
		}
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}
