package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/powerx-plugin/cli/internal/config"
	"github.com/powerx-plugin/cli/internal/manifest"
	packagepkg "github.com/powerx-plugin/cli/internal/package"
	"github.com/powerx-plugin/cli/internal/publish"
)

type publishFlags struct {
	Entry        string
	Artifact     string
	Metadata     string
	Manifest     string
	RBAC         string
	Channel      string
	Notes        string
	PublishAPI   string
	PublishToken string
	Timeout      int
}

func runPublish(args []string) error {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	fs.SetOutput(os.Stdout)

	var flags publishFlags
	fs.StringVar(&flags.Entry, "entry", "", "Path to the plugin directory (default: current)")
	fs.StringVar(&flags.Artifact, "artifact", "", "Path to package.tar.gz (default: latest .px-plugin/build)")
	fs.StringVar(&flags.Metadata, "metadata", "", "Path to metadata.json (default: latest .px-plugin/build)")
	fs.StringVar(&flags.Manifest, "manifest", "", "Path to manifest.json (default: latest .px-plugin/build/payload/manifest.json)")
	fs.StringVar(&flags.RBAC, "rbac", "", "Path to rbac.json (optional)")
	fs.StringVar(&flags.Channel, "channel", "", "Release channel (default: metadata.channel or dev)")
	fs.StringVar(&flags.Notes, "notes", "", "Release notes / description")
	fs.StringVar(&flags.PublishAPI, "publish-api", "", "Override Publish API base URL")
	fs.StringVar(&flags.PublishToken, "publish-token", "", "Override Publish API token")
	fs.IntVar(&flags.Timeout, "timeout", 60, "Publish HTTP timeout in seconds")

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
	if flags.Timeout <= 0 {
		flags.Timeout = 60
	}
	entryPath, err := filepath.Abs(flags.Entry)
	if err != nil {
		return fmt.Errorf("resolve entry path: %w", err)
	}
	if _, err := os.Stat(entryPath); err != nil {
		return fmt.Errorf("entry path invalid: %w", err)
	}

	cfg := loadUserConfig()
	baseURL := resolvePublishBase(flags.PublishAPI, cfg)
	if baseURL == "" {
		return fmt.Errorf("publish API base URL not configured; set --publish-api, PX_PUBLISH_API_BASE, or publishApi.baseUrl in ~/.px-plugin/config.json")
	}
	apiToken := resolvePublishToken(flags.PublishToken, cfg)
	if apiToken == "" {
		return fmt.Errorf("publish API token not configured; set --publish-token, PX_PUBLISH_API_TOKEN, or publishApi.apiKey in ~/.px-plugin/config.json")
	}

	buildDir := ""
	if flags.Artifact == "" || flags.Metadata == "" || flags.Manifest == "" {
		if dir, err := findLatestBuildDir(entryPath); err == nil {
			buildDir = dir
		} else if !haveAllArtifacts(flags) {
			return err
		}
	}

	if flags.Artifact == "" && buildDir != "" {
		flags.Artifact = filepath.Join(buildDir, "package.tar.gz")
	}
	if flags.Metadata == "" && buildDir != "" {
		flags.Metadata = filepath.Join(buildDir, "metadata.json")
	}
	if flags.Manifest == "" && buildDir != "" {
		flags.Manifest = filepath.Join(buildDir, "payload", "manifest.json")
	}
	if flags.RBAC == "" && buildDir != "" {
		rbacPath := filepath.Join(buildDir, "payload", "rbac.json")
		if _, err := os.Stat(rbacPath); err == nil {
			flags.RBAC = rbacPath
		}
	}

	if err := ensureFileExists(flags.Artifact, "package artifact", "run 'px-plugin package' first"); err != nil {
		return err
	}
	if err := ensureFileExists(flags.Metadata, "metadata.json", "run 'px-plugin package' first"); err != nil {
		return err
	}
	if err := ensureFileExists(flags.Manifest, "manifest.json", "run 'px-plugin package' first"); err != nil {
		return err
	}

	meta, err := readMetadata(flags.Metadata)
	if err != nil {
		return err
	}
	if flags.Channel == "" {
		flags.Channel = meta.Channel
	}
	if flags.Channel == "" {
		flags.Channel = "dev"
	}

	pluginManifest, err := manifest.Load(entryPath)
	if err != nil {
		return fmt.Errorf("load plugin manifest: %w", err)
	}

	fmt.Printf("Publishing package\n  Entry: %s\n  Plugin: %s@%s\n  Channel: %s\n  Registry: %s\n  Artifact: %s\n",
		entryPath, pluginManifest.ID, pluginManifest.Version, flags.Channel, baseURL, flags.Artifact)

	client, err := publish.NewClient(publish.Options{
		BaseURL:  baseURL,
		APIToken: apiToken,
		Timeout:  time.Duration(flags.Timeout) * time.Second,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flags.Timeout)*time.Second)
	defer cancel()

	resp, err := client.Submit(ctx, &publish.SubmitRequest{
		PluginID:     pluginManifest.ID,
		Version:      pluginManifest.Version,
		Channel:      flags.Channel,
		Notes:        flags.Notes,
		PackagePath:  flags.Artifact,
		MetadataPath: flags.Metadata,
		ManifestPath: flags.Manifest,
		RBACPath:     flags.RBAC,
		CLIVersion:   Version,
	})
	if err != nil {
		return err
	}

	fmt.Println("Publish request accepted:")
	fmt.Printf("  Publish ID: %s\n", resp.PublishID)
	if resp.Status != "" {
		fmt.Printf("  Status: %s\n", resp.Status)
	}
	if resp.ReviewURL != "" {
		fmt.Printf("  Review URL: %s\n", resp.ReviewURL)
	}
	fmt.Println("请前往 PowerX Marketplace/插件管理后台完成审核与安装。")
	return nil
}

func resolvePublishBase(flagVal string, cfg *config.Config) string {
	if flagVal != "" {
		return flagVal
	}
	if env := os.Getenv("PX_PUBLISH_API_BASE"); env != "" {
		return env
	}
	if cfg != nil && cfg.PublishAPI.BaseURL != "" {
		return cfg.PublishAPI.BaseURL
	}
	return ""
}

func resolvePublishToken(flagVal string, cfg *config.Config) string {
	if flagVal != "" {
		return flagVal
	}
	if env := os.Getenv("PX_PUBLISH_API_TOKEN"); env != "" {
		return env
	}
	if cfg != nil && cfg.PublishAPI.APIKey != "" {
		return cfg.PublishAPI.APIKey
	}
	return ""
}

func haveAllArtifacts(flags publishFlags) bool {
	return flags.Artifact != "" && flags.Metadata != "" && flags.Manifest != ""
}

func findLatestBuildDir(entry string) (string, error) {
	root := filepath.Join(entry, ".px-plugin", "build")
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("no build artifacts found in %s (run 'px-plugin package' first)", root)
		}
		return "", fmt.Errorf("read build directory: %w", err)
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	if len(dirs) == 0 {
		return "", fmt.Errorf("no build directories found in %s (run 'px-plugin package' first)", root)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	return filepath.Join(root, dirs[0]), nil
}

func readMetadata(path string) (*packagepkg.Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}
	var meta packagepkg.Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	return &meta, nil
}
