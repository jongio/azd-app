package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Go Tool - Process service example")

	if len(os.Args) > 1 {
		fmt.Printf("Command: %s\n", os.Args[1])
	}

	fmt.Println("Tool executed successfully")
}
