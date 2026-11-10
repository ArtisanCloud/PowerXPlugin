package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"

	fwmanifest "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
	climanifest "github.com/powerx-plugin/cli/internal/manifest"
)

func pluginIDDiagnostic(err error) error {
	locale := os.Getenv("LC_ALL")
	if locale == "" {
		locale = os.Getenv("LANG")
	}
	return fmt.Errorf("%w: %s", err, fwmanifest.FormatPluginIDError(err, locale))
}

// Machine-shaped flags keep this usable from Go, Python and Make alike.
func runPluginID(args []string) error {
	if len(args) == 0 || args[0] != "check" {
		return errors.New("PLUGIN_ID_CHECK_REQUIRED")
	}
	flags := flag.NewFlagSet("plugin-id check", flag.ContinueOnError)
	id := flags.String("id", "", "plugin_id")
	file := flags.String("manifest", "", "manifest_path")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || (*id == "") == (*file == "") {
		return errors.New("PLUGIN_ID_CHECK_INPUT_REQUIRED")
	}
	var err error
	if *file != "" {
		var document *climanifest.PluginManifest
		document, err = climanifest.Load(*file)
		if err == nil {
			err = fwmanifest.ValidatePluginEnvironment(document.ID, os.Getenv)
		}
	} else {
		err = fwmanifest.ValidatePluginID(*id)
	}
	if err != nil {
		return pluginIDDiagnostic(err)
	}
	return nil
}
