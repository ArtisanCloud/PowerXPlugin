package cmd

import (
	"fmt"
	"os"
)

func printExperimentalNotice(name, planned string) {
	fmt.Fprintf(os.Stdout, "px-plugin %s is currently experimental.\n", name)
	fmt.Fprintf(os.Stdout, "Planned workflow: %s\n", planned)
	fmt.Fprintln(os.Stdout, "Track progress in docs/init-project.md before relying on this command.")
}
