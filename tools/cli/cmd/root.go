package cmd

import (
	"fmt"
	"os"
	"runtime/debug"
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
`)
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
