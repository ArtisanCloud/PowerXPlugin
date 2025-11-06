package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/powerx-plugin/cli/internal/contracts"
	"github.com/powerx-plugin/cli/internal/templates"
)

const (
	defaultVersion          = "0.1.0"
	defaultGoVersion        = "1.24"
	defaultFrameworkVersion = "v0.0.1-alpha"
	schemaDependency        = "github.com/santhosh-tekuri/jsonschema/v5 v5.3.0"
	defaultAdminVersion     = "latest"
	defaultClientVersion    = "latest"
)

var pluginIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$`)

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	var (
		module   = fs.String("module", "", "override backend module import path (default derives from plugin id)")
		backend  = fs.String("backend", "go-gin", "backend template (only go-gin supported)")
		frontend = fs.String("frontend", "nuxt", "frontend template (only nuxt supported)")
		force    = fs.Bool("force", false, "overwrite existing files")
	)
	fs.SetOutput(os.Stdout)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return errors.New("plugin id is required")
	}
	pluginID := fs.Arg(0)
	if !pluginIDPattern.MatchString(pluginID) {
		return fmt.Errorf("invalid plugin id %q: must match %s", pluginID, pluginIDPattern.String())
	}

	if *backend != "go-gin" {
		fmt.Fprintf(os.Stdout, "warning: only backend=go-gin is supported, ignoring %q\n", *backend)
	}
	if *frontend != "nuxt" {
		fmt.Fprintf(os.Stdout, "warning: only frontend=nuxt is supported, ignoring %q\n", *frontend)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determine working dir: %w", err)
	}

	targetRoot := filepath.Join(cwd, pluginID)
	if err := ensureTargetDir(targetRoot, *force); err != nil {
		return err
	}

	pluginName := derivePluginName(pluginID)
	pluginSlug := derivePluginSlug(pluginID)

	moduleRoot := *module
	if moduleRoot == "" {
		moduleRoot = fmt.Sprintf("github.com/ArtisanCloud/PowerXPlugin/plugins/%s", pluginSlug)
	}

	backendModule := moduleRoot + "/backend"
	backendDir := filepath.Join(targetRoot, "backend")
	webDir := filepath.Join(targetRoot, "web-admin")

	frameworkReplace := detectFrameworkReplace(backendDir)
	adminRef, clientRef := detectWorkspaceRefs(webDir)

	data := templates.Data{
		PluginID:           pluginID,
		PluginName:         pluginName,
		PluginSlug:         pluginSlug,
		Version:            defaultVersion,
		GoVersion:          defaultGoVersion,
		BackendModulePath:  backendModule,
		FrameworkVersion:   defaultFrameworkVersion,
		FrameworkReplace:   frameworkReplace,
		SchemaDependency:   schemaDependency,
		FrameworkAdminRef:  adminRef,
		FrameworkClientRef: clientRef,
	}

	renderResult, err := templates.RenderAll(targetRoot, data, templates.Options{Force: *force})
	if err != nil {
		return err
	}

	contractFiles, err := writeContracts(targetRoot, *force)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Initialized plugin %q at %s\n", pluginID, targetRoot)
	for _, path := range renderResult.Files {
		if strings.HasPrefix(path, targetRoot) {
			fmt.Fprintf(os.Stdout, "  created %s\n", strings.TrimPrefix(path, targetRoot+string(os.PathSeparator)))
		}
	}
	for _, path := range contractFiles {
		fmt.Fprintf(os.Stdout, "  created %s\n", strings.TrimPrefix(path, targetRoot+string(os.PathSeparator)))
	}
	fmt.Fprintln(os.Stdout, "Next steps:")
	fmt.Fprintln(os.Stdout, "  - Update go.mod module path if necessary.")
	fmt.Fprintln(os.Stdout, "  - Run go mod tidy && npm install in the generated project.")
	fmt.Fprintln(os.Stdout, "  - Review plugin.yaml and README for TODO items.")

	return nil
}

func ensureTargetDir(path string, force bool) error {
	info, err := os.Stat(path)
	switch {
	case err == nil && !info.IsDir():
		return fmt.Errorf("target %s exists and is not a directory", path)
	case err == nil:
		if force {
			return nil
		}
		empty, err := isDirEmpty(path)
		if err != nil {
			return err
		}
		if !empty {
			return fmt.Errorf("target directory %s is not empty (use --force to overwrite)", path)
		}
		return nil
	case errors.Is(err, os.ErrNotExist):
		return os.MkdirAll(path, 0o755)
	default:
		return fmt.Errorf("stat %s: %w", path, err)
	}
}

func isDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	entries, err := f.Readdirnames(1)
	if err == nil && len(entries) > 0 {
		return false, nil
	}
	if errors.Is(err, io.EOF) {
		return true, nil
	}
	return false, err
}

func derivePluginName(id string) string {
	parts := strings.FieldsFunc(id, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})
	if len(parts) == 0 {
		return id
	}
	if parts[0] == "com" && len(parts) > 1 {
		parts = parts[1:]
	}
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, " ") + " Plugin"
}

func derivePluginSlug(id string) string {
	slug := strings.ToLower(id)
	replacer := strings.NewReplacer(".", "-", "_", "-", " ", "-")
	slug = replacer.Replace(slug)
	return slug
}

func detectFrameworkReplace(backendDir string) string {
	root := filepath.Join(backendDir, "..", "..", "framework")
	if info, err := os.Stat(root); err == nil && info.IsDir() {
		rel, err := filepath.Rel(backendDir, root)
		if err == nil {
			return toSlash(rel)
		}
	}
	return ""
}

func detectWorkspaceRefs(webDir string) (string, string) {
	workspace := filepath.Join(webDir, "..", "..", "framework", "frontend", "nuxt")
	admin := filepath.Join(workspace, "framework-admin")
	client := filepath.Join(workspace, "framework-client")

	if info, err := os.Stat(admin); err == nil && info.IsDir() {
		relAdmin, errAdmin := filepath.Rel(webDir, admin)
		relClient, errClient := filepath.Rel(webDir, client)
		if errAdmin == nil && errClient == nil {
			adminRef := "file:" + toSlash(relAdmin)
			clientRef := "file:" + toSlash(relClient)
			return adminRef, clientRef
		}
	}

	return defaultAdminVersion, defaultClientVersion
}

func toSlash(path string) string {
	return filepath.ToSlash(path)
}

func writeContracts(root string, force bool) ([]string, error) {
	files, err := contracts.Files()
	if err != nil {
		return nil, err
	}
	var written []string
	for _, file := range files {
		target := filepath.Join(root, filepath.FromSlash(file.Name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, fmt.Errorf("create dir for %s: %w", target, err)
		}
		if !force {
			if _, err := os.Stat(target); err == nil {
				return nil, fmt.Errorf("contract file exists: %s", target)
			}
		}
		if err := os.WriteFile(target, file.Data, 0o644); err != nil {
			return nil, fmt.Errorf("write contract %s: %w", target, err)
		}
		written = append(written, target)
	}
	return written, nil
}
