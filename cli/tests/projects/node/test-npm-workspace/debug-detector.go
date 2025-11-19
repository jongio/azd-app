package main

import (
	"fmt"
	"log"

	"github.com/jongio/azd-app/cli/src/internal/detector"
)

func main() {
	fmt.Println("Testing npm workspace detection...")
	fmt.Println()
	
	projects, err := detector.FindNodeProjects(".")
	if err != nil {
		log.Fatalf("Error finding projects: %v", err)
	}
	
	fmt.Printf("Found %d Node.js projects:\n\n", len(projects))
	
	for i, p := range projects {
		fmt.Printf("Project %d:\n", i+1)
		fmt.Printf("  Dir: %s\n", p.Dir)
		fmt.Printf("  PackageManager: %s\n", p.PackageManager)
		fmt.Printf("  IsWorkspaceRoot: %v\n", p.IsWorkspaceRoot)
		fmt.Printf("  WorkspaceRoot: %s\n", p.WorkspaceRoot)
		fmt.Println()
	}
}
