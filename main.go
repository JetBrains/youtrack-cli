package main

import (
	"embed"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/JetBrains/youtrack-cli/cmd"
)

//go:embed all:.agents/skills
var skillsFS embed.FS

func main() {
	os.Exit(run())
}

func run() (exitCode int) {
	defer func() {
		if r := recover(); r != nil {
			_, _ = fmt.Fprintf(os.Stderr, "youtrack panic: %v\n\n%s\n", r, debug.Stack())
			_, _ = fmt.Fprintln(os.Stderr, "This is a bug. Please report it at https://jb.gg/yt/issues")
			exitCode = 1
		}
	}()

	if err := cmd.Execute(skillsFS); err != nil {
		return 1
	}
	return 0
}
