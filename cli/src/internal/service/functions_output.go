// Package service provides Azure Functions output parsing and display.
package service

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/jongio/azd-app/cli/src/internal/output"
)

// FunctionEndpoint represents a discovered function endpoint.
type FunctionEndpoint struct {
	Name        string
	TriggerType string
	Methods     []string
	Route       string
}

// FunctionsOutputParser parses Azure Functions Core Tools output to extract endpoints.
type FunctionsOutputParser struct {
	mu        sync.Mutex
	endpoints map[string][]FunctionEndpoint // serviceName -> endpoints
	verbose   bool
}

// NewFunctionsOutputParser creates a new parser for Functions output.
func NewFunctionsOutputParser(verbose bool) *FunctionsOutputParser {
	return &FunctionsOutputParser{
		endpoints: make(map[string][]FunctionEndpoint),
		verbose:   verbose,
	}
}

// Patterns to match func output
var (
	// HTTP trigger patterns from func start output
	httpTriggerPattern = regexp.MustCompile(`^\s*(\w+):\s+\[(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS|.*?)\]\s+http://[^/]+/(.*)$`)

	// Alternative pattern for simpler output
	simpleHttpPattern = regexp.MustCompile(`^\s*(\w+):\s+http://[^/]+/(.*)$`)

	// Non-HTTP trigger patterns
	timerTriggerPattern      = regexp.MustCompile(`^\s*(\w+):\s+\[timerTrigger\]`)
	queueTriggerPattern      = regexp.MustCompile(`^\s*(\w+):\s+\[queueTrigger\]`)
	serviceBusTriggerPattern = regexp.MustCompile(`^\s*(\w+):\s+\[serviceBusTrigger\]`)
)

// ParseLine processes a single line of func output.
func (p *FunctionsOutputParser) ParseLine(serviceName, line string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// HTTP trigger with methods
	if matches := httpTriggerPattern.FindStringSubmatch(line); matches != nil {
		endpoint := FunctionEndpoint{
			Name:        matches[1],
			TriggerType: "HTTP",
			Methods:     parseHTTPMethods(matches[2]),
			Route:       matches[3],
		}
		p.addEndpoint(serviceName, endpoint)
		return
	}

	// Simple HTTP trigger
	if matches := simpleHttpPattern.FindStringSubmatch(line); matches != nil {
		endpoint := FunctionEndpoint{
			Name:        matches[1],
			TriggerType: "HTTP",
			Methods:     []string{"GET", "POST"},
			Route:       matches[2],
		}
		p.addEndpoint(serviceName, endpoint)
		return
	}

	// Timer trigger
	if matches := timerTriggerPattern.FindStringSubmatch(line); matches != nil {
		endpoint := FunctionEndpoint{
			Name:        matches[1],
			TriggerType: "Timer",
		}
		p.addEndpoint(serviceName, endpoint)
		return
	}

	// Queue trigger
	if matches := queueTriggerPattern.FindStringSubmatch(line); matches != nil {
		endpoint := FunctionEndpoint{
			Name:        matches[1],
			TriggerType: "Queue",
		}
		p.addEndpoint(serviceName, endpoint)
		return
	}

	// Service Bus trigger
	if matches := serviceBusTriggerPattern.FindStringSubmatch(line); matches != nil {
		endpoint := FunctionEndpoint{
			Name:        matches[1],
			TriggerType: "ServiceBus",
		}
		p.addEndpoint(serviceName, endpoint)
		return
	}
}

// ParseStream processes a stream of func output.
func (p *FunctionsOutputParser) ParseStream(serviceName string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		p.ParseLine(serviceName, line)
	}
}

// addEndpoint adds an endpoint if it doesn't already exist.
func (p *FunctionsOutputParser) addEndpoint(serviceName string, endpoint FunctionEndpoint) {
	if p.endpoints[serviceName] == nil {
		p.endpoints[serviceName] = []FunctionEndpoint{}
	}

	// Check for duplicates
	for _, existing := range p.endpoints[serviceName] {
		if existing.Name == endpoint.Name {
			return // Already exists
		}
	}

	p.endpoints[serviceName] = append(p.endpoints[serviceName], endpoint)
}

// GetEndpoints returns all endpoints for a service.
func (p *FunctionsOutputParser) GetEndpoints(serviceName string) []FunctionEndpoint {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.endpoints[serviceName]
}

// HasEndpoints checks if any endpoints were discovered for a service.
func (p *FunctionsOutputParser) HasEndpoints(serviceName string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.endpoints[serviceName]) > 0
}

// DisplayEndpoints shows discovered endpoints for a service.
func (p *FunctionsOutputParser) DisplayEndpoints(serviceName string, port int) {
	p.mu.Lock()
	endpoints := p.endpoints[serviceName]
	p.mu.Unlock()

	if len(endpoints) == 0 {
		return
	}

	baseURL := fmt.Sprintf("http://localhost:%d", port)

	fmt.Printf("\n")
	output.Item("%sFunctions:%s", output.Cyan, output.Reset)

	for _, endpoint := range endpoints {
		switch endpoint.TriggerType {
		case "HTTP":
			methods := strings.Join(endpoint.Methods, ", ")
			url := baseURL + "/" + endpoint.Route
			fmt.Printf("  %s%-20s%s [%s%-8s%s] %s%s%s\n",
				output.Green, endpoint.Name, output.Reset,
				output.Yellow, methods, output.Reset,
				output.Blue, url, output.Reset)

		case "Timer":
			fmt.Printf("  %s%-20s%s [%s%-8s%s] (triggered on schedule)\n",
				output.Green, endpoint.Name, output.Reset,
				output.Yellow, "Timer", output.Reset)

		case "Queue":
			fmt.Printf("  %s%-20s%s [%s%-8s%s] (triggered by queue messages)\n",
				output.Green, endpoint.Name, output.Reset,
				output.Yellow, "Queue", output.Reset)

		case "ServiceBus":
			fmt.Printf("  %s%-20s%s [%s%-8s%s] (triggered by Service Bus)\n",
				output.Green, endpoint.Name, output.Reset,
				output.Yellow, "ServiceBus", output.Reset)
		}
	}

	fmt.Printf("\n")
}

// parseHTTPMethods extracts HTTP methods from the pattern match.
func parseHTTPMethods(methodsStr string) []string {
	methods := strings.Split(methodsStr, ",")
	result := make([]string, 0, len(methods))
	for _, method := range methods {
		method = strings.TrimSpace(method)
		if method != "" {
			result = append(result, method)
		}
	}
	return result
}
