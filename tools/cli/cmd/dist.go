package cmd

import (
	"flag"
	"os"
)

func runDist(args []string) error {
	fs := flag.NewFlagSet("dist", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	if err := fs.Parse(args); err != nil {
		return err
	}
	printExperimentalNotice("dist", "bundle package artifacts with plugin.yaml into dist/<plugin>-<version>.zip")
	return nil
}
