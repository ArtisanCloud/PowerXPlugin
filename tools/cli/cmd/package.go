package cmd

import (
	"flag"
	"fmt"
	"os"
)

func runPackage(args []string) error {
	fs := flag.NewFlagSet("package", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	if err := fs.Parse(args); err != nil {
		return err
	}
	printExperimentalNotice("package", "go build ./backend/cmd/plugin && npm run build")
	return nil
}

func printExperimentalNotice(name, planned string) {
	fmt.Fprintf(os.Stdout, "px-plugin %s is currently experimental.\n", name)
	fmt.Fprintf(os.Stdout, "Planned workflow: %s\n", planned)
	fmt.Fprintln(os.Stdout, "Track progress in docs/init-project.md before relying on this command.")
}
