package cmd

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/powerx-plugin/cli/internal/contracts"
	"github.com/powerx-plugin/cli/internal/templates"
)

const (
	defaultVersion          = "0.1.0"
	defaultGoVersion        = "1.24"
	defaultFrameworkVersion = "v0.0.1-alpha"
	schemaDependency        = "github.com/santhosh-tekuri/jsonschema/v5 v5.3.0"
	defaultAdminVersion     = "^0.0.1-alpha"
	defaultClientVersion    = "^0.0.1-alpha"
)

var pluginIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$`)

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	var (
		module              = fs.String("module", "", "override backend module import path (default derives from plugin id)")
		backend             = fs.String("backend", templates.BackendGoGin, fmt.Sprintf("backend framework (supported: %v)", templates.SupportedBackends()))
		adminFrontend       = fs.String("admin", templates.FrontendNuxt, fmt.Sprintf("admin frontend framework (supported: %v)", templates.SupportedFrontends()))
		appFrontend         = fs.String("app", "", "app frontend framework (optional)")
		force               = fs.Bool("force", false, "overwrite existing files")
		directory           = fs.String("directory", "", "target directory (default: ./<plugin-id>)")
		version             = fs.String("version", defaultVersion, "plugin version")
		goVersion           = fs.String("go-version", defaultGoVersion, "Go version to use")
		installDeps         = fs.Bool("install-deps", false, "automatically install dependencies (go mod tidy & npm install)")
		sbomPath            = fs.String("sbom-path", "", "path to write SBOM file (default: <target>/reports/sbom.json)")
		publishManifestPath = fs.String("publish-manifest-path", "", "path to write publish manifest (default: <target>/publish.yml)")
	)
	fs.SetOutput(os.Stdout)
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Handle optional app frontend
	if *appFrontend == "" {
		*appFrontend = *adminFrontend // Default to same as admin if not specified
	}

	if fs.NArg() < 1 {
		return errors.New("plugin id is required")
	}
	pluginID := fs.Arg(0)
	if !pluginIDPattern.MatchString(pluginID) {
		return fmt.Errorf("invalid plugin id %q: must match %s", pluginID, pluginIDPattern.String())
	}

	if err := templates.ValidateTemplateTypes(*backend, *adminFrontend); err != nil {
		return err
	}
	if err := templates.ValidateTemplateTypes(*backend, *appFrontend); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determine working dir: %w", err)
	}

	pluginName := derivePluginName(pluginID)
	pluginSlug := derivePluginSlug(pluginID)

	moduleRoot := *module
	if moduleRoot == "" {
		moduleRoot = fmt.Sprintf("github.com/ArtisanCloud/PowerXPlugin/plugins/%s", pluginSlug)
	}

	backendModule := moduleRoot + "/backend"

	// Determine target root directory
	targetRoot := filepath.Join(cwd, pluginID)
	if *directory != "" {
		if filepath.IsAbs(*directory) {
			targetRoot = *directory
		} else {
			targetRoot = filepath.Join(cwd, *directory)
		}
	}
	if err := ensureTargetDir(targetRoot, *force); err != nil {
		return err
	}

	backendDir := filepath.Join(targetRoot, "backend")
	webDir := filepath.Join(targetRoot, "web-admin")

	frameworkReplace := detectFrameworkReplace(backendDir)
	adminRef, clientRef := detectWorkspaceRefs(webDir)

	data := templates.Data{
		PluginID:           pluginID,
		PluginName:         pluginName,
		PluginSlug:         pluginSlug,
		Version:            *version,
		GoVersion:          *goVersion,
		BackendModulePath:  backendModule,
		BackendType:        *backend,
		FrontendType:       *adminFrontend,
		AppFrontendType:    *appFrontend,
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

	// Generate SBOM if requested or use default path
	if *sbomPath == "" {
		*sbomPath = filepath.Join(targetRoot, "reports", "sbom.json")
	}
	preset := struct{ Backend, Frontend string }{
		Backend:  *backend,
		Frontend: *adminFrontend,
	}
	if err := writeSbom(*sbomPath, pluginID, *version, moduleRoot, renderResult.Files, preset); err != nil {
		return err
	}

	// Generate publish manifest if requested or use default path
	if *publishManifestPath == "" {
		*publishManifestPath = filepath.Join(targetRoot, "publish.yml")
	}
	if err := writePublishManifest(*publishManifestPath, pluginID); err != nil {
		return err
	}

	if err := copyGovernanceDirs(targetRoot, *force); err != nil {
		return err
	}

	// Install dependencies if requested
	if *installDeps {
		if err := installDependencies(backendDir, webDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to install dependencies: %v\n", err)
		}
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
	if *sbomPath != "" {
		fmt.Fprintf(os.Stdout, "  created %s\n", strings.TrimPrefix(*sbomPath, targetRoot+string(os.PathSeparator)))
	}
	if *publishManifestPath != "" {
		fmt.Fprintf(os.Stdout, "  created %s\n", strings.TrimPrefix(*publishManifestPath, targetRoot+string(os.PathSeparator)))
	}
	fmt.Fprintln(os.Stdout, "  governance .specify/ & .codex copied from CLI repo for Speckit/Codex automation (safe to remove or relocate if not needed).")
	fmt.Fprintln(os.Stdout, "Next steps:")
	fmt.Fprintln(os.Stdout, "  - Update go.mod module path if necessary.")
	if !*installDeps {
		fmt.Fprintln(os.Stdout, "  - Run go mod tidy && npm install in the generated project.")
	}
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
	if os.Getenv("POWERXPLUGIN_USE_LOCAL_FRONTEND") != "" {
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

func writeSbom(sbomPath, pluginID, version, modulePath string, files []string, preset struct{ Backend, Frontend string }) error {
	// Convert files to relative paths
	relFiles := make([]string, 0, len(files))
	for _, file := range files {
		rel := filepath.Base(file)
		relFiles = append(relFiles, rel)
	}

	// Determine template preset
	presetID := ""
	if preset.Backend != "" || preset.Frontend != "" {
		parts := []string{}
		if preset.Backend != "" {
			parts = append(parts, preset.Backend)
		}
		if preset.Frontend != "" {
			parts = append(parts, preset.Frontend)
		}
		presetID = strings.Join(parts, "-")
	}

	type sbomData struct {
		Schema      string `json:"schema"`
		GeneratedAt string `json:"generatedAt"`
		Plugin      struct {
			ID      string `json:"id"`
			Version string `json:"version"`
			Module  string `json:"module"`
		} `json:"plugin"`
		TemplatePreset string   `json:"templatePreset"`
		Files          []string `json:"files"`
	}

	data := sbomData{
		Schema:      "powerx.plugin.sbom@v1",
		GeneratedAt: time.Now().Format(time.RFC3339),
		Plugin: struct {
			ID      string `json:"id"`
			Version string `json:"version"`
			Module  string `json:"module"`
		}{
			ID:      pluginID,
			Version: version,
			Module:  modulePath,
		},
		TemplatePreset: presetID,
		Files:          relFiles,
	}

	if err := os.MkdirAll(filepath.Dir(sbomPath), 0o755); err != nil {
		return fmt.Errorf("create sbom dir: %w", err)
	}

	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sbom: %w", err)
	}

	if err := os.WriteFile(sbomPath, payload, 0o644); err != nil {
		return fmt.Errorf("write sbom: %w", err)
	}

	return nil
}

func writePublishManifest(publishPath, pluginID string) error {
	content := []string{
		"# Generated by px-plugin init",
		fmt.Sprintf("plugin: %s", pluginID),
		"channels:",
		"  - stable",
		"  - beta",
		"rollout:",
		"  strategy: canary",
		"  batches:",
		"    - percentage: 20",
		"      wait: 10m",
		"    - percentage: 80",
		"      wait: 20m",
		"rollback:",
		"  strategy: automatic",
		"  maxFailingTenants: 5",
		"",
	}

	if err := os.MkdirAll(filepath.Dir(publishPath), 0o755); err != nil {
		return fmt.Errorf("create publish dir: %w", err)
	}

	if err := os.WriteFile(publishPath, []byte(strings.Join(content, "\n")), 0o644); err != nil {
		return fmt.Errorf("write publish manifest: %w", err)
	}

	return nil
}

func installDependencies(backendDir, webDir string) error {
	// Install Go dependencies
	if info, err := os.Stat(filepath.Join(backendDir, "go.mod")); err == nil && !info.IsDir() {
		cmd := exec.Command("go", "mod", "tidy")
		cmd.Dir = backendDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go mod tidy failed: %w", err)
		}
	}

	// Install npm dependencies
	if info, err := os.Stat(filepath.Join(webDir, "package.json")); err == nil && !info.IsDir() {
		cmd := exec.Command("npm", "install")
		cmd.Dir = webDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("npm install failed: %w", err)
		}
	}

	return nil
}

func copyGovernanceDirs(targetRoot string, force bool) error {
	sourceDirs := []string{".specify", ".codex"}
	projectRoot := findRepoRoot()
	if projectRoot == "" {
		return nil
	}
	for _, dir := range sourceDirs {
		src := filepath.Join(projectRoot, dir)
		info, err := os.Stat(src)
		if err != nil || !info.IsDir() {
			continue
		}
		dst := filepath.Join(targetRoot, dir)
		if _, err := os.Stat(dst); err == nil && !force {
			continue
		}
		if err := copyDirFiltered(src, dst); err != nil {
			return fmt.Errorf("copy %s: %w", dir, err)
		}
	}
	return nil
}

func copyDirFiltered(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git/") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFileContents(path, target, info.Mode())
	})
}

func copyFileContents(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, mode)
}

func findRepoRoot() string {
	starts := []string{}
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		starts = append(starts, cwd)
	}
	for _, start := range starts {
		if root := ascendUntilGit(start); root != "" {
			return root
		}
	}
	return ""
}

func ascendUntilGit(start string) string {
	current := start
	for {
		gitPath := filepath.Join(current, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return current
		}
		next := filepath.Dir(current)
		if next == current {
			return ""
		}
		current = next
	}
}
