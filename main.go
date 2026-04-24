package main

import (
	"embed"
	"os"

	"github.com/JetBrains/youtrack-cli/cmd"
)

//go:embed all:.agents/skills
var skillsFS embed.FS

func main() {
	if err := cmd.Execute(skillsFS); err != nil {
		os.Exit(1)
	}
}
