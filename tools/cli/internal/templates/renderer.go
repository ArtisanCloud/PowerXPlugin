package templates

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode/utf8"
)

const templateRoot = "data/"

var binaryExtensions = []string{
	".png", ".jpg", ".jpeg", ".gif", ".svg",
	".webp", ".avif", ".ico", ".ttf", ".woff", ".woff2",
}

// Data 提供渲染模板所需的上下文。
type Data struct {
	PluginID           string
	PluginName         string
	PluginSlug         string
	Version            string
	GoVersion          string
	BackendModulePath  string
	BackendType        string
	FrontendType       string
	BackendPort        int
	FrontendPort       int
	FrameworkVersion   string
	FrameworkReplace   string
	SchemaDependency   string
	FrameworkAdminRef  string
	FrameworkClientRef string
	AppFrontendType    string // 可选的第二个前端框架
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
		targetRel := rel
		isTemplate := strings.HasSuffix(rel, ".tmpl")
		shouldRender := isTemplate && !isBinaryTemplate(rel)

		if isTemplate {
			targetRel = strings.TrimSuffix(rel, ".tmpl")
		}
		targetRel = strings.ReplaceAll(targetRel, "com.powerx.plugin.base", data.PluginID)
		targetRel = strings.ReplaceAll(targetRel, "com.powerx.plugins.base", data.PluginID)
		targetRel = strings.ReplaceAll(targetRel, "__plugin__", data.PluginID)
		targetRel = normalizeTargetPath(targetRel, data.BackendType, data.FrontendType)

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

		var rendered []byte
		if shouldRender {
			rendered, err = executeTemplate(rel, raw, data)
			if err != nil {
				return err
			}
		} else {
			rendered = raw
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
	if !utf8.Valid(raw) {
		// Binary files may still carry .tmpl suffix for sync convenience.
		// In that case, skip template parsing and return bytes as-is.
		return raw, nil
	}
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

func normalizeTargetPath(rel, backendType, frontendType string) string {
	// Dynamic mapping based on backend and frontend types
	switch {
	case strings.HasPrefix(rel, "backend/"+backendType+"/"):
		return "backend/" + strings.TrimPrefix(rel, "backend/"+backendType+"/")
	case strings.HasPrefix(rel, "web-admin/"+frontendType+"/"):
		return "web-admin/" + strings.TrimPrefix(rel, "web-admin/"+frontendType+"/")
	default:
		return rel
	}
}

func isBinaryTemplate(rel string) bool {
	for _, ext := range binaryExtensions {
		if strings.HasSuffix(rel, ext+".tmpl") {
			return true
		}
	}
	return false
}

// ValidateTemplateTypes checks if the given backend and frontend types are supported.
func ValidateTemplateTypes(backendType, frontendType string) error {
	if !IsValidBackend(backendType) {
		return fmt.Errorf("unsupported backend type: %q (supported: %v)", backendType, SupportedBackends())
	}
	if !IsValidFrontend(frontendType) {
		return fmt.Errorf("unsupported frontend type: %q (supported: %v)", frontendType, SupportedFrontends())
	}
	return nil
}
