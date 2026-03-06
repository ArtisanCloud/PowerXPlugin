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
	defaultFrameworkVersion = "v0.0.4-alpha"
	schemaDependency        = "github.com/santhosh-tekuri/jsonschema/v5 v5.3.0"
	defaultAdminVersion     = "^0.0.1-alpha"
	defaultClientVersion    = "^0.0.1-alpha"
)

var pluginIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$`)

func runInit(args []string) error {
	// Go stdlib flag parsing stops at the first non-flag argument.
	// Many users naturally put plugin id first (e.g. `px-plugin init com.xxx --force`),
	// which would cause trailing flags to be ignored. Normalize that common shape so
	// flags work regardless of position.
	args = normalizeInitArgs(args)

	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	var (
		module              = fs.String("module", "", "override backend module import path (default derives from plugin id)")
		backend             = fs.String("backend", templates.BackendGoGin, fmt.Sprintf("backend framework (supported: %v)", templates.SupportedBackends()))
		adminFrontend       = fs.String("admin", templates.FrontendNuxt, fmt.Sprintf("admin frontend framework (supported: %v)", templates.SupportedFrontends()))
		appFrontend         = fs.String("app", "", "app frontend framework (optional)")
		backendPort         = fs.Int("backend-port", 8078, "backend service port")
		frontendPort        = fs.Int("frontend-port", 3131, "frontend dev server port")
		force               = fs.Bool("force", false, "overwrite existing files")
		directory           = fs.String("directory", "", "target directory (default: ./<plugin-id>)")
		version             = fs.String("version", defaultVersion, "plugin version")
		goVersion           = fs.String("go-version", defaultGoVersion, "Go version to use")
		installDeps         = fs.Bool("install-deps", false, "automatically install dependencies (go mod tidy & npm install)")
		initConfig          = fs.Bool("init-config", true, "create local config files from *.example templates")
		gitInit             = fs.Bool("git-init", true, "initialize git repository in target directory")
		sbomPath            = fs.String("sbom-path", "", "path to write SBOM file (default: <target>/reports/sbom.json)")
		publishManifestPath = fs.String("publish-manifest-path", "", "path to write publish manifest (default: <target>/publish.yml)")
	)
	fs.SetOutput(os.Stdout)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return errors.New("plugin id is required")
	}
	pluginID := fs.Arg(0)

	if shouldRunInitGuide(fs) {
		if err := runInitGuide(&pluginID, module, backend, adminFrontend, appFrontend, backendPort, frontendPort, directory, installDeps, initConfig, gitInit); err != nil {
			return err
		}
	}

	// Handle optional app frontend
	if *appFrontend == "" {
		*appFrontend = *adminFrontend // Default to same as admin if not specified
	}

	if !pluginIDPattern.MatchString(pluginID) {
		return fmt.Errorf("invalid plugin id %q: must match %s", pluginID, pluginIDPattern.String())
	}

	if err := templates.ValidateTemplateTypes(*backend, *adminFrontend); err != nil {
		return err
	}
	if err := templates.ValidateTemplateTypes(*backend, *appFrontend); err != nil {
		return err
	}
	if err := validatePort(*backendPort, "backend-port"); err != nil {
		return err
	}
	if err := validatePort(*frontendPort, "frontend-port"); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determine working dir: %w", err)
	}

	pluginName := derivePluginName(pluginID)
	pluginSlug := derivePluginSlug(pluginID)

	moduleRoot := normalizeModuleRoot(*module, pluginID)

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
		BackendPort:        *backendPort,
		FrontendPort:       *frontendPort,
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

	createdConfigFiles := make([]string, 0, 3)
	if *initConfig {
		configFiles, err := materializeLocalConfigs(targetRoot, *backend, *adminFrontend, *force)
		if err != nil {
			return err
		}
		createdConfigFiles = append(createdConfigFiles, configFiles...)
	}
	gitInitialized := false
	if *gitInit {
		ok, err := ensureGitRepository(targetRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to initialize git repository: %v\n", err)
		} else {
			gitInitialized = ok
		}
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
	for _, path := range createdConfigFiles {
		fmt.Fprintf(os.Stdout, "  created %s\n", strings.TrimPrefix(path, targetRoot+string(os.PathSeparator)))
	}
	if gitInitialized {
		fmt.Fprintln(os.Stdout, "  initialized .git repository")
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

func shouldRunInitGuide(fs *flag.FlagSet) bool {
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) == 0 {
		return false
	}
	hasConfigFlag := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "backend", "admin", "app", "module", "directory", "install-deps", "init-config", "git-init":
			hasConfigFlag = true
		case "backend-port", "frontend-port":
			hasConfigFlag = true
		}
	})
	return fs.NArg() >= 1 && !hasConfigFlag
}

func runInitGuide(
	pluginID *string,
	module *string,
	backend *string,
	adminFrontend *string,
	appFrontend *string,
	backendPort *int,
	frontendPort *int,
	directory *string,
	installDeps *bool,
	initConfig *bool,
	gitInit *bool,
) error {
	fmt.Fprintln(os.Stdout, "Entering init guide (interactive mode).")

	candidateID := *pluginID
	if !pluginIDPattern.MatchString(candidateID) {
		if suggested := sanitizePluginID(candidateID); suggested != "" && pluginIDPattern.MatchString(suggested) {
			fmt.Fprintf(os.Stdout, "Plugin ID %q is invalid, suggested: %q\n", candidateID, suggested)
			candidateID = suggested
		}
	}
	for {
		value, err := promptString("Plugin ID", candidateID)
		if err != nil {
			return err
		}
		if pluginIDPattern.MatchString(value) {
			*pluginID = value
			break
		}
		fmt.Fprintf(os.Stdout, "Invalid plugin id %q, expected pattern: %s\n", value, pluginIDPattern.String())
		if suggested := sanitizePluginID(value); suggested != "" && pluginIDPattern.MatchString(suggested) {
			fmt.Fprintf(os.Stdout, "Try: %s\n", suggested)
			candidateID = suggested
		}
	}

	selectedBackend, err := promptSelect("Backend", templates.SupportedBackends(), *backend)
	if err != nil {
		return err
	}
	*backend = selectedBackend

	selectedAdmin, err := promptSelect("Admin frontend", templates.SupportedFrontends(), *adminFrontend)
	if err != nil {
		return err
	}
	*adminFrontend = selectedAdmin
	*appFrontend = selectedAdmin

	defaultBackendPort := *backendPort
	if defaultBackendPort <= 0 {
		defaultBackendPort = 8078
	}
	selectedBackendPort, err := promptPort("Backend port", defaultBackendPort)
	if err != nil {
		return err
	}
	*backendPort = selectedBackendPort

	defaultFrontendPort := *frontendPort
	if defaultFrontendPort == 3131 && *adminFrontend == templates.FrontendNext {
		defaultFrontendPort = 3231
	}
	if defaultFrontendPort <= 0 {
		defaultFrontendPort = defaultFrontendPortByType(*adminFrontend)
	}
	selectedFrontendPort, err := promptPort("Frontend port", defaultFrontendPort)
	if err != nil {
		return err
	}
	*frontendPort = selectedFrontendPort

	defaultModule := *module
	if defaultModule == "" {
		orgOrUser, err := promptString("GitHub org/user", "your-org")
		if err != nil {
			return err
		}
		orgOrUser = strings.Trim(orgOrUser, "/ ")
		orgOrUser = strings.TrimPrefix(orgOrUser, "https://github.com/")
		orgOrUser = strings.TrimPrefix(orgOrUser, "http://github.com/")
		orgOrUser = strings.TrimPrefix(orgOrUser, "github.com/")
		if orgOrUser == "" {
			orgOrUser = "your-org"
		}
		defaultModule = fmt.Sprintf("github.com/%s/%s", orgOrUser, *pluginID)
	}
	selectedModule, err := promptString("Module root", defaultModule)
	if err != nil {
		return err
	}
	*module = normalizeModuleRoot(selectedModule, *pluginID)

	defaultDirectory := *directory
	if defaultDirectory == "" {
		defaultDirectory = *pluginID
	}
	selectedDirectory, err := promptString("Target directory", defaultDirectory)
	if err != nil {
		return err
	}
	*directory = strings.TrimSpace(selectedDirectory)

	selectedInstallDeps, err := promptBool("Install dependencies now (go mod tidy + npm install)", true)
	if err != nil {
		return err
	}
	*installDeps = selectedInstallDeps

	selectedInitConfig, err := promptBool("Create local config files from *.example", true)
	if err != nil {
		return err
	}
	*initConfig = selectedInitConfig

	selectedGitInit, err := promptBool("Initialize git repository (git init)", true)
	if err != nil {
		return err
	}
	*gitInit = selectedGitInit

	return nil
}

func promptString(label, defaultValue string) (string, error) {
	reader := io.Reader(os.Stdin)
	buf := make([]byte, 0, 128)
	fmt.Fprintf(os.Stdout, "%s [%s]: ", label, defaultValue)
	for {
		b := make([]byte, 1)
		n, err := reader.Read(b)
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(buf) == 0 {
					return defaultValue, nil
				}
				return strings.TrimSpace(string(buf)), nil
			}
			return "", err
		}
		if n == 0 {
			continue
		}
		ch := b[0]
		if ch == '\n' {
			value := strings.TrimSpace(string(buf))
			if value == "" {
				return defaultValue, nil
			}
			return value, nil
		}
		buf = append(buf, ch)
	}
}

func promptSelect(label string, options []string, defaultValue string) (string, error) {
	defaultIndex := 0
	for idx, option := range options {
		if option == defaultValue {
			defaultIndex = idx
			break
		}
	}
	fmt.Fprintf(os.Stdout, "%s:\n", label)
	for idx, option := range options {
		marker := ""
		if idx == defaultIndex {
			marker = " (default)"
		}
		fmt.Fprintf(os.Stdout, "  %d) %s%s\n", idx+1, option, marker)
	}
	for {
		raw, err := promptString("Choose number", fmt.Sprintf("%d", defaultIndex+1))
		if err != nil {
			return "", err
		}
		choice := strings.TrimSpace(raw)
		var selected int
		if _, err := fmt.Sscanf(choice, "%d", &selected); err != nil || selected < 1 || selected > len(options) {
			fmt.Fprintf(os.Stdout, "Invalid choice %q, please input 1-%d\n", choice, len(options))
			continue
		}
		return options[selected-1], nil
	}
}

func promptBool(label string, defaultValue bool) (bool, error) {
	defaultHint := "y"
	if !defaultValue {
		defaultHint = "n"
	}
	for {
		raw, err := promptString(label+" [y/n]", defaultHint)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Fprintf(os.Stdout, "Invalid choice %q, please input y or n\n", raw)
		}
	}
}

func promptPort(label string, defaultValue int) (int, error) {
	if defaultValue <= 0 {
		defaultValue = 1
	}
	for {
		raw, err := promptString(label, fmt.Sprintf("%d", defaultValue))
		if err != nil {
			return 0, err
		}
		value := strings.TrimSpace(raw)
		var port int
		if _, err := fmt.Sscanf(value, "%d", &port); err != nil || port < 1 || port > 65535 {
			fmt.Fprintf(os.Stdout, "Invalid port %q, please input 1-65535\n", value)
			continue
		}
		return port, nil
	}
}

func defaultFrontendPortByType(frontendType string) int {
	switch frontendType {
	case templates.FrontendNext:
		return 3231
	default:
		return 3131
	}
}

func validatePort(port int, label string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid %s %d: expected 1-65535", label, port)
	}
	return nil
}

func sanitizePluginID(input string) string {
	value := strings.ToLower(strings.TrimSpace(input))
	replacer := strings.NewReplacer("_", "-", " ", "-", "/", "-", ":", "-", "@", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, ".-")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	for strings.Contains(value, "..") {
		value = strings.ReplaceAll(value, "..", ".")
	}
	return value
}

func normalizeModuleRoot(input, pluginID string) string {
	value := strings.TrimSpace(input)
	if value == "" {
		return fmt.Sprintf("github.com/your-org/%s", pluginID)
	}
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "github.com/")
	value = strings.Trim(value, "/")
	if value == "" {
		return fmt.Sprintf("github.com/your-org/%s", pluginID)
	}
	if strings.Count(value, "/") == 0 {
		value = value + "/" + pluginID
	}
	return "github.com/" + value
}

func normalizeInitArgs(args []string) []string {
	if len(args) < 2 {
		return args
	}
	if strings.HasPrefix(args[0], "-") {
		return args
	}
	for _, token := range args[1:] {
		if strings.HasPrefix(token, "-") {
			normalized := make([]string, 0, len(args))
			normalized = append(normalized, args[1:]...)
			normalized = append(normalized, args[0])
			return normalized
		}
	}
	return args
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
		fmt.Fprintf(os.Stdout, "Installing backend dependencies: go mod tidy (dir=%s)\n", backendDir)
		cmd := exec.Command("go", "mod", "tidy")
		cmd.Dir = backendDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go mod tidy failed (dir=%s): %w", backendDir, err)
		}
	}

	// Install npm dependencies
	if info, err := os.Stat(filepath.Join(webDir, "package.json")); err == nil && !info.IsDir() {
		fmt.Fprintf(os.Stdout, "Installing frontend dependencies: npm install (dir=%s)\n", webDir)
		cmd := exec.Command("npm", "install")
		cmd.Dir = webDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("npm install failed (dir=%s): %w", webDir, err)
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

func materializeLocalConfigs(targetRoot, backendType, adminFrontendType string, force bool) ([]string, error) {
	type configPair struct {
		source string
		target string
	}
	pairs := []configPair{
		{source: "backend/etc/config.example.yaml", target: "backend/etc/config.yaml"},
	}
	switch backendType {
	case templates.BackendPythonFast:
		pairs = append(pairs, configPair{source: "backend/python-fastapi/.env.example", target: "backend/python-fastapi/.env.local"})
	default:
		pairs = append(pairs, configPair{source: "backend/.env.example", target: "backend/.env.local"})
	}
	switch adminFrontendType {
	case templates.FrontendNext:
		pairs = append(pairs, configPair{source: "web-admin/next/.env.example", target: "web-admin/next/.env.local"})
	default:
		pairs = append(pairs, configPair{source: "web-admin/.env.example", target: "web-admin/.env.local"})
	}
	created := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		src := filepath.Join(targetRoot, pair.source)
		dst := filepath.Join(targetRoot, pair.target)

		info, err := os.Stat(src)
		if err != nil || info.IsDir() {
			continue
		}
		if _, err := os.Stat(dst); err == nil && !force {
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", src, err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, fmt.Errorf("create dir for %s: %w", dst, err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", dst, err)
		}
		created = append(created, dst)
	}
	return created, nil
}

func ensureGitRepository(targetRoot string) (bool, error) {
	gitDir := filepath.Join(targetRoot, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		return false, nil
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = targetRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, err
	}
	return true, nil
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
