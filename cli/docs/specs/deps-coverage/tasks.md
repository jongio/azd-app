# Deps Coverage Improvement Tasks

## Status Key
- TODO: Not started
- IN PROGRESS: Currently being worked on
- DONE: Completed

## Tasks

### Task 1: Test NewDepsCommand flag behavior [Developer]
**Status:** DONE
**Description:** Added tests for force flag, PreRunE with output format handling, all flag types.

### Task 2: Test filterProjectsByService with azure.yaml [Developer]
**Status:** DONE
**Description:** Added tests for filtering Node.js, Python, and .NET projects with azure.yaml present. Coverage: 100%.

### Task 3: Test showDryRunSummary [Developer]
**Status:** DONE
**Description:** Added tests for dry-run summary in both text and JSON output modes. Coverage: 100%.

### Task 4: Test handleNoProjectsCase [Developer]
**Status:** DONE
**Description:** Added tests for empty workspace, service filter with no matches, Logic Apps detection (both logic-apps-only and mixed function apps). Coverage: 100%.

### Task 5: Test parseAzureYaml [Developer]
**Status:** DONE
**Description:** Added tests for valid, invalid, empty, and complex azure.yaml files. Coverage: 100%.

### Task 6: Test cleanDependencies and cleanDirectory [Developer]
**Status:** DONE
**Description:** Added tests for cleaning Node.js, Python, and .NET dependency directories with success and error cases. Coverage: cleanDirectory 72.7%, cleanDependencies 75%.

### Task 7: Test getSearchRoot edge cases [Developer]
**Status:** DONE
**Description:** Added tests for search root with/without azure.yaml. Coverage: 77.8%.

### Task 8: Test detectAllProjects [Developer]
**Status:** DONE
**Description:** Added tests for detecting Node.js, Python, and .NET projects. Coverage: 70%.

### Task 9: Run coverage verification [Tester]
**Status:** DONE

## Coverage Summary

### deps.go functions:
- GetDepsOptions: 100%
- setDepsOptions: 100%
- ResetDepsOptions: 100%
- NewDepsCommand: 70.8%

### core.go deps-related functions:
- NewDependencyInstaller: 100%
- InstallAllFiltered: 100%
- installNodeProjectList: 100%
- installPythonProjectList: 100%
- installDotnetProjectList: 100%
- installProject: 92.3%
- handleDepsError: 100%
- handleNoProjectsCase: 100%
- filterProjectsByService: 100%
- parseAzureYaml: 100%
- showDryRunSummary: 100%
- checkAllSuccess: 100%
- getSearchRoot: 77.8%
- cleanDirectory: 72.7%
- cleanDependencies: 75%
- detectAllProjects: 70%

### Not unit testable (require integration tests):
- executeDeps: 0% (orchestrator function with global state)
- runParallelInstallation: 0% (calls actual package managers)
- runJSONInstallation: 0% (calls actual package managers)
- InstallAll: 0% (discovery-based function)
- installNodeProjects: 0% (discovery-based)
- installPythonProjects: 0% (discovery-based)
- installDotnetProjects: 0% (discovery-based)

## Analysis

The unit-testable deps functions have achieved weighted average coverage of approximately 88%. Functions at 100% coverage include all the core logic: option handling, filtering, dry-run, error handling, and YAML parsing.

Functions below 80% have uncovered error paths that require:
- Making os.Getwd() fail (getSearchRoot)
- Making os.RemoveAll fail (cleanDirectory)
- File system permission errors

The orchestrator function (executeDeps) and parallel installation functions require integration tests as they use global state and call external package managers.
