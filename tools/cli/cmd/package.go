package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	packagepkg "github.com/powerx-plugin/cli/internal/package"
)

type packageFlags struct {
	Entry        string
	FrontendDir  string
	BackendDir   string
	OutputDir    string
	Channel      string
	Version      string
	SkipFrontend bool
	SkipBackend  bool
}

func runPackage(args []string) error {
	fs := flag.NewFlagSet("package", flag.ExitOnError)
	fs.SetOutput(os.Stdout)

	var flags packageFlags
	fs.StringVar(&flags.Entry, "entry", "", "Path to the plugin directory (default: current)")
	fs.StringVar(&flags.FrontendDir, "frontend-dir", "", "Path to the frontend workspace (default: <entry>/web-admin)")
	fs.StringVar(&flags.BackendDir, "backend-dir", "", "Path to the backend workspace (default: <entry>/backend)")
	fs.StringVar(&flags.OutputDir, "output-dir", "", "Override .px-plugin/build directory")
	fs.StringVar(&flags.Channel, "channel", "dev", "Release channel to annotate in metadata")
	fs.StringVar(&flags.Version, "version", "", "Override manifest version in metadata")
	fs.BoolVar(&flags.SkipFrontend, "skip-frontend", false, "Skip building and packaging the frontend dist")
	fs.BoolVar(&flags.SkipBackend, "skip-backend", false, "Skip building and packaging the backend binary")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if flags.Entry == "" {
		if cwd, err := os.Getwd(); err == nil {
			flags.Entry = cwd
		} else {
			return fmt.Errorf("determine current directory: %w", err)
		}
	}
	flags.Entry = filepath.Clean(flags.Entry)

	opts := &packagepkg.Options{
		EntryPath:       flags.Entry,
		FrontendDir:     flags.FrontendDir,
		BackendDir:      flags.BackendDir,
		OutputDir:       flags.OutputDir,
		SkipFrontend:    flags.SkipFrontend,
		SkipBackend:     flags.SkipBackend,
		Channel:         flags.Channel,
		VersionOverride: flags.Version,
		CLIVersion:      Version,
	}

	builder := packagepkg.NewBuilder()
	result, err := builder.Build(context.Background(), opts)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Package complete: %s\n", result.PackagePath)
	fmt.Fprintf(os.Stdout, "Metadata: %s\n", result.MetadataPath)
	if result.FrontendPath != "" {
		fmt.Fprintf(os.Stdout, "Frontend dist: %s\n", result.FrontendPath)
	}
	if result.BackendBinaryPath != "" {
		fmt.Fprintf(os.Stdout, "Backend binary: %s\n", result.BackendBinaryPath)
	}
	if result.DistHash != "" {
		fmt.Fprintf(os.Stdout, "Frontend hash: %s\n", result.DistHash)
	}

	if len(result.Artifacts) > 0 {
		fmt.Fprintln(os.Stdout, "\nArtifacts:")
		for _, artifact := range result.Artifacts {
			fmt.Fprintf(os.Stdout, "  - %-18s %s (size=%d hash=%s)\n",
				artifact.Name, artifact.Path, artifact.Size, shortHash(artifact.Hash))
		}
	}

	return nil
}

func shortHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12] + "..."
}
