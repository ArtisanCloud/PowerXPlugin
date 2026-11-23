package config

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var ErrEventResourceInvalid = errors.New("EVENT_FABRIC_TOPIC_MANIFEST_INVALID")

// Core owns these delivery policies. The startup reader only extracts business
// keys/directions, while accepting the policy fields of the v1 declaration.
type eventTopicPolicy struct {
	PayloadFormat   string         `yaml:"payload_format"`
	VersioningMode  string         `yaml:"versioning_mode"`
	MaxRetry        *int           `yaml:"max_retry"`
	AckTimeoutSec   *int           `yaml:"ack_timeout_seconds"`
	RetentionPolicy map[string]any `yaml:"retention_policy"`
	Metadata        map[string]any `yaml:"metadata"`
}

// Business keys come from the package declaration; physical mappings only come from Core.
func (c *Config) loadEventBridgeResources(configDir string) error {
	if c.EventBridge == nil || strings.TrimSpace(c.EventBridge.TopicManifest) == "" {
		return nil
	}
	rel := c.EventBridge.TopicManifest
	if filepath.IsAbs(rel) || filepath.Clean(rel) != rel || strings.HasPrefix(rel, "..") {
		return ErrEventResourceInvalid
	}
	root := strings.TrimSpace(os.Getenv("POWERX_PLUGIN_ROOT"))
	if root == "" {
		root = configDir
		for i := 0; i < 8; i++ {
			if _, err := os.Stat(filepath.Join(root, "plugin.yaml")); err == nil {
				break
			}
			parent := filepath.Dir(root)
			if parent == root {
				return ErrEventResourceInvalid
			}
			root = parent
		}
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return ErrEventResourceInvalid
	}
	var plugin struct {
		ID string `yaml:"id"`
	}
	rawPlugin, err := os.ReadFile(filepath.Join(root, "plugin.yaml"))
	if err != nil || len(rawPlugin) > 256*1024 || yaml.Unmarshal(rawPlugin, &plugin) != nil || plugin.ID == "" {
		return ErrEventResourceInvalid
	}
	if c.EventBridge.SourcePlugin != "" && c.EventBridge.SourcePlugin != plugin.ID {
		return ErrEventResourceInvalid
	}
	c.EventBridge.SourcePlugin = plugin.ID
	path, err := filepath.EvalSymlinks(filepath.Join(root, rel))
	if err != nil {
		return ErrEventResourceInvalid
	}
	inside, err := filepath.Rel(root, path)
	if err != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return ErrEventResourceInvalid
	}
	f, err := os.Open(path)
	if err != nil {
		return ErrEventResourceInvalid
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 256*1024+1))
	if err != nil || len(raw) > 256*1024 {
		return ErrEventResourceInvalid
	}
	var declaration struct {
		Version  string           `yaml:"version"`
		Defaults eventTopicPolicy `yaml:"defaults"`
		Topics   []struct {
			eventTopicPolicy `yaml:",inline"`
			Key              string `yaml:"key"`
			Description      string `yaml:"description"`
			Namespace        string `yaml:"namespace"`
			Name             string `yaml:"name"`
			ACL              []struct {
				PrincipalType string   `yaml:"principal_type"`
				PrincipalID   string   `yaml:"principal_id"`
				Actions       []string `yaml:"actions"`
			} `yaml:"acl"`
		} `yaml:"topics"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if decoder.Decode(&declaration) != nil || (declaration.Version != "v1" && declaration.Version != "1") || len(declaration.Topics) == 0 || len(declaration.Topics) > 1000 {
		return ErrEventResourceInvalid
	}
	var tail any
	if decoder.Decode(&tail) != io.EOF {
		return ErrEventResourceInvalid
	}
	seen := map[string]bool{}
	publish, consume := []string{}, []string{}
	for _, topic := range declaration.Topics {
		if strings.TrimSpace(topic.Key) == "" || seen[topic.Key] || topic.Namespace == "" || topic.Name == "" || len(topic.ACL) == 0 || strings.Contains(topic.Key, "{{") {
			return ErrEventResourceInvalid
		}
		seen[topic.Key] = true
		if !eventNamespaceOwned(c.EventBridge.SourcePlugin, topic.Namespace) {
			return ErrEventResourceInvalid
		}
		pub, sub := false, false
		for _, acl := range topic.ACL {
			if acl.PrincipalType != "plugin" || (acl.PrincipalID != c.EventBridge.SourcePlugin && acl.PrincipalID != "{{plugin_id}}") || len(acl.Actions) == 0 {
				return ErrEventResourceInvalid
			}
			for _, action := range acl.Actions {
				switch action {
				case "publish":
					pub = true
				case "subscribe":
					sub = true
				default:
					return ErrEventResourceInvalid
				}
			}
		}
		if pub {
			publish = append(publish, topic.Key)
		}
		if sub {
			consume = append(consume, topic.Key)
		}
	}
	// Installed static mappings and resolved business keys are separate formal profiles.
	if len(c.EventBridge.TopicBindings) > 0 {
		if len(c.EventBridge.PublishTopicKeys) > 0 || len(c.EventBridge.ConsumeTopicKeys) > 0 {
			return ErrEventResourceInvalid
		}
		return nil
	}
	if len(c.EventBridge.PublishTopicKeys) > 0 || len(c.EventBridge.ConsumeTopicKeys) > 0 {
		return ErrEventResourceInvalid
	}
	c.EventBridge.PublishTopicKeys = publish
	c.EventBridge.ConsumeTopicKeys = consume
	return nil
}

func eventNamespaceOwned(plugin, namespace string) bool {
	roots := []string{plugin}
	if alias := strings.TrimPrefix(plugin, "com.powerx.plugins."); alias != plugin && alias != "" && !strings.Contains(alias, ".") {
		alias = strings.ReplaceAll(alias, "-", "_")
		roots = append(roots, alias, "powerx."+alias, "_topic."+alias)
	}
	for _, root := range roots {
		if namespace == root || strings.HasPrefix(namespace, root+".") {
			return true
		}
	}
	return false
}
