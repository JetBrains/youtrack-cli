package main

import (
	"os"

	"github.com/JetBrains/youtrack-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
