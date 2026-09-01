package security

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	"gopkg.in/yaml.v3"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
)

var ErrEventPermissionDenied = errors.New("event permission denied")

type EventPermissions struct {
	enforced  bool
	publish   map[string]struct{}
	subscribe map[string]struct{}
}

func (p EventPermissions) Enforced() bool { return p.enforced }

func (p EventPermissions) CanPublish(topic string) bool {
	if !p.enforced {
		return true
	}
	_, ok := p.publish[strings.TrimSpace(topic)]
	return ok
}

func (p EventPermissions) CanSubscribe(topic string) bool {
	if !p.enforced {
		return true
	}
	_, ok := p.subscribe[strings.TrimSpace(topic)]
	return ok
}

func (p EventPermissions) Topics() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(p.publish)+len(p.subscribe))
	for topic := range p.publish {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			continue
		}
		if _, ok := seen[topic]; ok {
			continue
		}
		seen[topic] = struct{}{}
		out = append(out, topic)
	}
	for topic := range p.subscribe {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			continue
		}
		if _, ok := seen[topic]; ok {
			continue
		}
		seen[topic] = struct{}{}
		out = append(out, topic)
	}
	slices.Sort(out)
	return out
}

type pluginManifest struct {
	Events *struct {
		Publish   []string `yaml:"publish"`
		Subscribe []string `yaml:"subscribe"`
		Topics    []struct {
			Key         string   `yaml:"key"`
			Topic       string   `yaml:"topic"`
			Actions     []string `yaml:"actions"`
			Description string   `yaml:"description"`
		} `yaml:"topics"`
	} `yaml:"events"`
}

func LoadEventPermissionsFromManifest(manifestPath string, logger *pxlog.Entry) (EventPermissions, error) {
	manifestPath = strings.TrimSpace(manifestPath)
	if manifestPath == "" {
		manifestPath = defaultManifestPath()
	}

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		if logger != nil {
			pxlog.WarnCtx(pxlog.WithLogFields(context.Background(), map[string]interface{}{
				"module":     "security",
				"biz_scene":  "event_permissions_load",
				"biz_domain": "security",
				"component":  "security.event_permissions",
				"error":      err.Error(),
			}), "event permissions manifest not found; permissions enforcement disabled")
		}
		return EventPermissions{enforced: false}, nil
	}

	var root map[string]any
	if err := yaml.Unmarshal(content, &root); err != nil {
		return EventPermissions{}, fmt.Errorf("parse manifest yaml failed: %w", err)
	}
	if err := mergeCatalogReferences(manifestPath, root); err != nil {
		return EventPermissions{}, fmt.Errorf("merge manifest catalogs failed: %w", err)
	}
	merged, err := yaml.Marshal(root)
	if err != nil {
		return EventPermissions{}, fmt.Errorf("marshal merged manifest failed: %w", err)
	}
	var m pluginManifest
	if err := yaml.Unmarshal(merged, &m); err != nil {
		return EventPermissions{}, fmt.Errorf("decode merged manifest failed: %w", err)
	}

	if m.Events == nil {
		if logger != nil {
			pxlog.WarnCtx(pxlog.WithLogFields(context.Background(), map[string]interface{}{
				"module":     "security",
				"biz_scene":  "event_permissions_load",
				"biz_domain": "security",
				"component":  "security.event_permissions",
			}), "manifest.events not found; permissions enforcement disabled")
		}
		return EventPermissions{enforced: false}, nil
	}

	perms := EventPermissions{
		enforced:  true,
		publish:   map[string]struct{}{},
		subscribe: map[string]struct{}{},
	}

	for _, t := range m.Events.Publish {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		perms.publish[t] = struct{}{}
	}
	for _, t := range m.Events.Subscribe {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		perms.subscribe[t] = struct{}{}
	}

	for _, item := range m.Events.Topics {
		topic := strings.TrimSpace(item.Key)
		if topic == "" {
			topic = strings.TrimSpace(item.Topic)
		}
		if topic == "" {
			continue
		}
		for _, action := range item.Actions {
			switch strings.ToLower(strings.TrimSpace(action)) {
			case "publish":
				perms.publish[topic] = struct{}{}
			case "subscribe":
				perms.subscribe[topic] = struct{}{}
			}
		}
	}

	return perms, nil
}

func mergeCatalogReferences(manifestPath string, root map[string]any) error {
	catalogsValue, ok := root["catalogs"]
	if !ok || catalogsValue == nil {
		return nil
	}
	catalogs, ok := catalogsValue.(map[string]any)
	if !ok {
		return errors.New("catalogs must be an object")
	}

	manifestDir := filepath.Dir(manifestPath)
	loadCatalog := func(name string) (map[string]any, error) {
		rawPath, _ := catalogs[name].(string)
		rawPath = strings.TrimSpace(rawPath)
		if rawPath == "" {
			return nil, nil
		}
		filePath := rawPath
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(manifestDir, filepath.FromSlash(rawPath))
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read catalogs.%s (%s): %w", name, rawPath, err)
		}
		var doc map[string]any
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse catalogs.%s (%s): %w", name, rawPath, err)
		}
		return doc, nil
	}

	if doc, err := loadCatalog("events"); err != nil {
		return err
	} else if doc != nil {
		if section, ok := doc["events"]; ok {
			root["events"] = section
		}
	}
	return nil
}

func defaultManifestPath() string {
	for _, key := range []string{"POWERX_PLUGIN_MANIFEST_PATH", "POWERX_PLUGIN_MANIFEST"} {
		p := strings.TrimSpace(os.Getenv(key))
		if p == "" {
			continue
		}
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
		return p
	}
	cwd, _ := os.Getwd()
	candidates := []string{
		"plugin.yaml",
		filepath.Join("skeleton", "plugin.yaml"),
	}
	if cwd != "" {
		candidates = append(candidates,
			filepath.Join(cwd, "plugin.yaml"),
			filepath.Join(cwd, "skeleton", "plugin.yaml"),
			filepath.Join(cwd, "..", "plugin.yaml"),
			filepath.Join(cwd, "..", "..", "plugin.yaml"),
			filepath.Join(cwd, "..", "..", "..", "plugin.yaml"),
		)
	}
	for _, cand := range candidates {
		if info, err := os.Stat(cand); err == nil && !info.IsDir() {
			return cand
		}
	}
	if exe, err := os.Executable(); err == nil {
		if found := findManifestFrom(filepath.Dir(exe)); found != "" {
			return found
		}
	}
	return filepath.Join("skeleton", "plugin.yaml")
}

func DefaultManifestPath() string { return defaultManifestPath() }

func findManifestFrom(start string) string {
	current := filepath.Clean(start)
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(current, "plugin.yaml")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

type PermissionedEmitter struct {
	inner  fweventbridge.Emitter
	perms  EventPermissions
	logger *pxlog.Entry
}

func NewPermissionedEmitter(inner fweventbridge.Emitter, perms EventPermissions, logger *pxlog.Entry) fweventbridge.Emitter {
	if inner == nil {
		return nil
	}
	if logger == nil {
		logger = pxlog.NewEntry(pxlog.StandardLogger())
	}
	return &PermissionedEmitter{
		inner:  inner,
		perms:  perms,
		logger: logger,
	}
}

func (e *PermissionedEmitter) Emit(ctx context.Context, ev event.Event) error {
	if e == nil || e.inner == nil {
		return errors.New("permissioned emitter not configured")
	}
	topic := strings.TrimSpace(string(ev.Topic))
	if !e.perms.CanPublish(topic) {
		pxlog.WarnCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
			"module":      "security",
			"biz_scene":   "event_publish_permission",
			"biz_domain":  "security",
			"component":   "security.event_permissions",
			"topic":       topic,
			"tenant_uuid": ev.Meta.TenantUUID,
			"trace_id":    ev.Meta.TraceID,
		}), "event publish denied by manifest permissions")
		return ErrEventPermissionDenied
	}
	return e.inner.Emit(ctx, ev)
}
