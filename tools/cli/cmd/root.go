package cmd

import (
	"fmt"
	"os"
)

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
  help       Show this help message
`)
}
