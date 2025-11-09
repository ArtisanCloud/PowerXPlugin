package cmd

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/powerx-plugin/cli/internal/templates"
)

// Version holds the build version string. Overridden at build time via ldflags.
var Version = "dev"

// Execute routes CLI invocations.
func Execute(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp()
		return nil
	case "version", "--version", "-v":
		printVersion()
		return nil
	case "init":
		return runInit(args[1:])
	case "package":
		return runPackage(args[1:])
	case "dist":
		return runDist(args[1:])
	case "publish":
		return runPublish(args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printHelp() {
	backends := templates.SupportedBackends()
	frontends := templates.SupportedFrontends()
	backendList := strings.Join(backends, ", ")
	frontendList := strings.Join(frontends, ", ")

	fmt.Fprintf(os.Stdout, `px-plugin CLI

Usage:
  px-plugin <command> [flags]

Commands:
  init       Generate a new plugin project from scaffold templates
  package    Experimental packaging workflow
  dist       Experimental distribution workflow
  publish    Experimental publish workflow
  version    Print CLI version information
  help       Show this help message

Init command flags:
  --backend <type>                 Backend framework (default: %s)
  --admin <type>                   Admin frontend framework (default: %s)
  --app <type>                     App frontend framework (optional, default: same as --admin)
  --module <path>                  Override backend module import path
  --directory <path>               Target directory (default: ./<plugin-id>)
  --version <version>              Plugin version (default: 0.1.0)
  --go-version <version>           Go version (default: 1.24)
  --force                          Overwrite existing files
  --install-deps                   Automatically install dependencies
  --sbom-path <path>               Path to write SBOM file
  --publish-manifest-path <path>   Path to write publish manifest

Available frameworks:
  --backend: %s
  --admin/--app: %s

Examples:
  px-plugin init com.example.myplugin
  px-plugin init --backend %s --admin %s com.example.myplugin
  px-plugin init --backend %s --admin %s --app %s com.example.myplugin
  px-plugin init --directory ./my-plugin --version 1.0.0 com.example.myplugin
  px-plugin init --install-deps --force com.example.myplugin
`, templates.BackendGoGin, templates.FrontendNuxt, backendList, frontendList,
		templates.BackendGoGin, templates.FrontendNuxt,
		templates.BackendGoGin, templates.FrontendNuxt, templates.FrontendNuxt)
}

func printVersion() {
	version := Version
	commit := ""
	dirty := ""

	if info, ok := debug.ReadBuildInfo(); ok {
		if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if len(setting.Value) >= 7 {
					commit = setting.Value[:7]
				} else {
					commit = setting.Value
				}
			case "vcs.modified":
				if setting.Value == "true" {
					dirty = "-dirty"
				}
			}
		}
	}

	if commit != "" {
		fmt.Fprintf(os.Stdout, "px-plugin version %s (commit %s%s)\n", version, commit, dirty)
	} else {
		fmt.Fprintf(os.Stdout, "px-plugin version %s\n", version)
	}
}
