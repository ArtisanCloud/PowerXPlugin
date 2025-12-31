package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type ContractDoc struct {
	Version   string `yaml:"version"`
	Namespace string `yaml:"namespace"`
	Meta      struct {
		Required []string `yaml:"required"`
	} `yaml:"meta"`
	Topics []TopicContract `yaml:"topics"`
}

type TopicContract struct {
	Name string `yaml:"name"`
}

var requiredMetaKeys = []string{
	"tenant_uuid",
	"request_id",
	"source_plugin",
	"trace_id",
	"occurred_at",
	"payload_version",
}

var forbiddenPayloadFieldSubstrings = []string{
	"password",
	"secret",
	"token",
	"access_key",
	"private_key",
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: validate-taskbus-contracts <path-to-contracts-yaml>")
		os.Exit(2)
	}

	path := strings.TrimSpace(os.Args[1])
	if path == "" {
		fmt.Fprintln(os.Stderr, "invalid contracts path")
		os.Exit(2)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read contracts failed: %v\n", err)
		os.Exit(1)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		fmt.Fprintf(os.Stderr, "parse contracts yaml (node) failed: %v\n", err)
		os.Exit(1)
	}

	var doc ContractDoc
	if err := yaml.Unmarshal(content, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "parse contracts yaml failed: %v\n", err)
		os.Exit(1)
	}

	var problems []string
	if err := validateMetaRequired(doc.Meta.Required); err != nil {
		problems = append(problems, err.Error())
	}
	if err := validateTopicUnique(doc.Topics); err != nil {
		problems = append(problems, err.Error())
	}
	if err := validateNoSensitivePayloadFields(&root); err != nil {
		problems = append(problems, err.Error())
	}

	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, p)
		}
		os.Exit(1)
	}
}

func validateMetaRequired(keys []string) error {
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		seen[k] = struct{}{}
	}

	var missing []string
	for _, required := range requiredMetaKeys {
		if _, ok := seen[required]; !ok {
			missing = append(missing, required)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("contracts.meta.required missing keys: %s", strings.Join(missing, ", "))
}

func validateTopicUnique(topics []TopicContract) error {
	seen := map[string]int{}
	var dups []string
	for _, t := range topics {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			continue
		}
		seen[name]++
		if seen[name] == 2 {
			dups = append(dups, name)
		}
	}
	if len(dups) == 0 {
		return nil
	}
	sort.Strings(dups)
	return errors.New("duplicate topic names: " + strings.Join(dups, ", "))
}

func validateNoSensitivePayloadFields(root *yaml.Node) error {
	if root == nil {
		return nil
	}

	var badKeys []string
	walkYAML(root, func(key string) {
		lower := strings.ToLower(strings.TrimSpace(key))
		for _, sub := range forbiddenPayloadFieldSubstrings {
			if strings.Contains(lower, sub) {
				badKeys = append(badKeys, key)
				return
			}
		}
	})

	if len(badKeys) == 0 {
		return nil
	}

	sort.Strings(badKeys)
	dedup := make([]string, 0, len(badKeys))
	seen := map[string]struct{}{}
	for _, k := range badKeys {
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		dedup = append(dedup, k)
	}

	return fmt.Errorf("contracts contain forbidden payload field names (possible secrets): %s", strings.Join(dedup, ", "))
}

func walkYAML(node *yaml.Node, onKey func(key string)) {
	if node == nil {
		return
	}

	// mapping node: Content is [keyNode, valueNode, ...]
	if node.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			k := node.Content[i]
			v := node.Content[i+1]
			if k != nil && k.Kind == yaml.ScalarNode {
				onKey(k.Value)
			}
			walkYAML(v, onKey)
		}
		return
	}

	// sequence node: Content is [item...]
	for _, c := range node.Content {
		walkYAML(c, onKey)
	}
}
