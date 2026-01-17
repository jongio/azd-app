# Tasks: azd app init with Copilot SDK

## In Progress

- None

## TODO

1. Research Copilot SDK Go implementation, API methods, and authentication requirements
2. Design command structure and flag definitions for `azd app init`
3. Implement Copilot SDK client wrapper (`internal/ai/copilot.go`)
4. Implement project context detection integration (`internal/ai/context.go`)
5. Create prompt templates for azure.yaml generation (`internal/ai/prompts.go`)
6. Implement YAML generation logic (`internal/ai/yaml_generator.go`)
7. Define Copilot SDK tools (validate_service_name, suggest_ports, etc.) (`internal/ai/tools.go`)
8. Implement main init command (`commands/init.go`)
9. Implement interactive mode handler (`commands/init_interactive.go`)
10. Implement schema validation for generated YAML
11. Implement file operations (backup, write, atomic operations)
12. Integrate with existing project detector (`internal/detector`)
13. Add well-known services support
14. Implement requirements detection and generation
15. Add error handling and recovery mechanisms
16. Create unit tests for AI integration components
17. Create integration tests for init command
18. Add documentation and usage examples
19. Update CLI help text and command reference

## Done

- None
