package templates

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const templateRoot = "data/"

// Data 提供渲染模板所需的上下文。
type Data struct {
	PluginID           string
	PluginName         string
	PluginSlug         string
	Version            string
	GoVersion          string
	BackendModulePath  string
	FrameworkVersion   string
	FrameworkReplace   string
	SchemaDependency   string
	FrameworkAdminRef  string
	FrameworkClientRef string
}

// Options 控制渲染行为。
type Options struct {
	Force bool
}

// Result 描述渲染输出。
type Result struct {
	Files []string
}

// RenderAll 将模板渲染到目标目录。
func RenderAll(baseDir string, data Data, opts Options) (Result, error) {
	var result Result

	err := fs.WalkDir(content, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		rel, err := relativeTemplatePath(path)
		if err != nil {
			return err
		}
		if !strings.HasSuffix(rel, ".tmpl") {
			return nil
		}

		targetRel := strings.TrimSuffix(rel, ".tmpl")
		targetRel = strings.ReplaceAll(targetRel, "com.powerx.plugin.base", data.PluginID)
		targetRel = strings.ReplaceAll(targetRel, "__plugin__", data.PluginID)
		targetRel = normalizeTargetPath(targetRel)

		targetPath := filepath.Join(baseDir, filepath.FromSlash(targetRel))
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", filepath.Dir(targetPath), err)
		}

		if !opts.Force {
			if _, err := os.Stat(targetPath); err == nil {
				return fmt.Errorf("file already exists: %s", targetPath)
			}
		}

		raw, err := fs.ReadFile(content, path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", path, err)
		}

		rendered, err := executeTemplate(rel, raw, data)
		if err != nil {
			return err
		}

		if err := os.WriteFile(targetPath, rendered, 0o644); err != nil {
			return fmt.Errorf("write file %s: %w", targetPath, err)
		}

		result.Files = append(result.Files, targetPath)
		return nil
	})

	return result, err
}

func relativeTemplatePath(path string) (string, error) {
	idx := strings.Index(path, templateRoot)
	if idx == -1 {
		return "", fmt.Errorf("unexpected template path: %s", path)
	}
	return path[idx+len(templateRoot):], nil
}

func executeTemplate(name string, raw []byte, data Data) ([]byte, error) {
	tpl, err := template.New(name).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

func normalizeTargetPath(rel string) string {
	switch {
	case strings.HasPrefix(rel, "backend/go-gin/"):
		return "backend/" + strings.TrimPrefix(rel, "backend/go-gin/")
	case strings.HasPrefix(rel, "web/nuxt/"):
		return "web-admin/" + strings.TrimPrefix(rel, "web/nuxt/")
	default:
		return rel
	}
}
