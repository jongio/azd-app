package orchestrator_test

import (
	"fmt"

	"github.com/jongio/azd-app/cli/src/internal/orchestrator"
)

// Example demonstrates a simple command dependency chain.
func Example() {
	o := orchestrator.NewOrchestrator()

	// Register commands with dependencies
	_ = o.Register(&orchestrator.Command{
		Name: "install",
		Execute: func() error {
			fmt.Println("Installing packages...")
			return nil
		},
	})

	_ = o.Register(&orchestrator.Command{
		Name:         "build",
		Dependencies: []string{"install"},
		Execute: func() error {
			fmt.Println("Building project...")
			return nil
		},
	})

	_ = o.Register(&orchestrator.Command{
		Name:         "test",
		Dependencies: []string{"build"},
		Execute: func() error {
			fmt.Println("Running tests...")
			return nil
		},
	})

	// Run the test command - dependencies will run automatically
	if err := o.Run("test"); err != nil {
		fmt.Println("Error:", err)
	}

	// Output:
	// Installing packages...
	// Building project...
	// Running tests...
}

// Example_memoization demonstrates that commands run only once.
func Example_memoization() {
	o := orchestrator.NewOrchestrator()

	execCount := 0
	_ = o.Register(&orchestrator.Command{
		Name: "shared",
		Execute: func() error {
			execCount++
			fmt.Printf("Executing shared command (count: %d)\n", execCount)
			return nil
		},
	})

	_ = o.Register(&orchestrator.Command{
		Name:         "cmd1",
		Dependencies: []string{"shared"},
		Execute: func() error {
			fmt.Println("Executing cmd1")
			return nil
		},
	})

	_ = o.Register(&orchestrator.Command{
		Name:         "cmd2",
		Dependencies: []string{"shared"},
		Execute: func() error {
			fmt.Println("Executing cmd2")
			return nil
		},
	})

	// Run both commands - shared runs only once
	_ = o.Run("cmd1")
	_ = o.Run("cmd2")

	// Output:
	// Executing shared command (count: 1)
	// Executing cmd1
	// Executing cmd2
}

// Example_diamondDependency demonstrates handling of diamond dependencies.
func Example_diamondDependency() {
	o := orchestrator.NewOrchestrator()

	// Diamond dependency:
	//       top
	//      /   \
	//   left   right
	//      \   /
	//      base

	_ = o.Register(&orchestrator.Command{
		Name: "base",
		Execute: func() error {
			fmt.Println("base")
			return nil
		},
	})

	_ = o.Register(&orchestrator.Command{
		Name:         "left",
		Dependencies: []string{"base"},
		Execute: func() error {
			fmt.Println("left")
			return nil
		},
	})

	_ = o.Register(&orchestrator.Command{
		Name:         "right",
		Dependencies: []string{"base"},
		Execute: func() error {
			fmt.Println("right")
			return nil
		},
	})

	_ = o.Register(&orchestrator.Command{
		Name:         "top",
		Dependencies: []string{"left", "right"},
		Execute: func() error {
			fmt.Println("top")
			return nil
		},
	})

	_ = o.Run("top")

	// Output:
	// base
	// left
	// right
	// top
}
