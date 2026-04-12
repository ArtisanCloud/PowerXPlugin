package security

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type eventFabricTopicsManifest struct {
	Topics []struct {
		Topic     string `yaml:"topic"`
		Namespace string `yaml:"namespace"`
		Name      string `yaml:"name"`
	} `yaml:"topics"`
}

func LoadEventFabricTopics(logger *logrus.Entry) ([]string, error) {
	paths := eventFabricPathCandidates()
	for _, candidate := range paths {
		content, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		var cfg eventFabricTopicsManifest
		if err := yaml.Unmarshal(content, &cfg); err != nil {
			return nil, fmt.Errorf("parse event fabric yaml failed: %w", err)
		}
		topics := make([]string, 0, len(cfg.Topics))
		seen := map[string]struct{}{}
		for _, item := range cfg.Topics {
			topic := strings.TrimSpace(item.Topic)
			if topic == "" && strings.TrimSpace(item.Namespace) != "" && strings.TrimSpace(item.Name) != "" {
				topic = strings.TrimRight(strings.TrimSpace(item.Namespace), ".") + "." + strings.TrimLeft(strings.TrimSpace(item.Name), ".")
			}
			if topic == "" {
				continue
			}
			if _, ok := seen[topic]; ok {
				continue
			}
			seen[topic] = struct{}{}
			topics = append(topics, topic)
		}
		if len(topics) > 0 {
			slices.Sort(topics)
			if logger != nil {
				logger.WithField("path", candidate).Info("loaded event fabric topics for bootstrap")
			}
			return topics, nil
		}
	}
	return nil, nil
}

func eventFabricPathCandidates() []string {
	if p := strings.TrimSpace(os.Getenv("POWERX_EVENT_FABRIC_PATH")); p != "" {
		return []string{p}
	}
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join("config", "event_fabric.yaml"),
		filepath.Join("platform_capabilities", "event_fabric.yaml"),
		"event_fabric.yaml",
		filepath.Join("skeleton", "config", "event_fabric.yaml"),
		filepath.Join("skeleton", "platform_capabilities", "event_fabric.yaml"),
		filepath.Join("skeleton", "event_fabric.yaml"),
	}
	if cwd != "" {
		candidates = append(candidates,
			filepath.Join(cwd, "config", "event_fabric.yaml"),
			filepath.Join(cwd, "platform_capabilities", "event_fabric.yaml"),
			filepath.Join(cwd, "event_fabric.yaml"),
			filepath.Join(cwd, "..", "config", "event_fabric.yaml"),
			filepath.Join(cwd, "..", "platform_capabilities", "event_fabric.yaml"),
			filepath.Join(cwd, "..", "event_fabric.yaml"),
		)
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(candidates))
	for _, cand := range candidates {
		cand = strings.TrimSpace(cand)
		if cand == "" {
			continue
		}
		if _, ok := seen[cand]; ok {
			continue
		}
		seen[cand] = struct{}{}
		out = append(out, cand)
	}
	return out
}
