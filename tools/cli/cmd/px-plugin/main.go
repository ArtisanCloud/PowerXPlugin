package main

import (
	"errors"
	"fmt"
	fwmanifest "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
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
		var identity *fwmanifest.IdentityError
		if errors.As(err, &identity) {
			locale := os.Getenv("LC_ALL")
			if locale == "" {
				locale = os.Getenv("LANG")
			}
			fmt.Fprintf(os.Stderr, "%s: %s\n", identity.Code, fwmanifest.FormatPluginIDError(err, locale))
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		code := 1
		var coded interface{ ExitCode() int }
		if errors.As(err, &coded) {
			code = coded.ExitCode()
		}
		os.Exit(code)
	}
}
