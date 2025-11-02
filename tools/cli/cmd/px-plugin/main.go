package main

import (
	"fmt"
	"os"

	"github.com/powerx-plugin/cli/cmd"
)

// version is injected at build time via ldflags (e.g. -X main.version=v0.3.0).
// Leave empty to fallback to cmd.Version's default ("dev").
var version = ""

func main() {
	if version != "" {
		cmd.Version = version
	}
	if err := cmd.Execute(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
