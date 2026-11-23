// Package pluginpublisher is the portable P01 contract reference. It does not
// register HTTP handlers, grant credentials, upload files or modify databases.
package wire

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gowebpki/jcs"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Bundle contains public schemas, policy and synthetic interoperability cases.
//
//go:embed *.json
var Bundle embed.FS

const schemaID = "urn:powerx:plugin-publisher:v1"
const fingerprintDomain = "powerx.plugin-publisher/v1/request\n"

var initOnce sync.Once
var compiler *jsonschema.Compiler
var initErr error
var compileMu sync.Mutex
var recoveryPolicy struct {
	MaxElapsed int64 `json:"max_recovery_elapsed_seconds"`
	MaxPoll    int64 `json:"max_operation_poll_seconds"`
	Poll       int64 `json:"operation_poll_seconds"`
	MaxRetry   int64 `json:"max_retry_after_seconds"`
	Attempts   int64 `json:"max_attempts"`
}
var errorPolicy map[string]struct {
	Status    int  `json:"http_status"`
	Retryable bool `json:"retryable"`
}

func initialize() {
	compiler = jsonschema.NewCompiler()
	compiler.AssertFormat = true
	raw, err := Bundle.ReadFile("protocol.schema.json")
	if err == nil {
		err = compiler.AddResource(schemaID, bytes.NewReader(raw))
	}
	if err == nil {
		raw, err = Bundle.ReadFile("errors.json")
	}
	if err == nil {
		err = json.Unmarshal(raw, &errorPolicy)
	}
	if err == nil {
		raw, err = Bundle.ReadFile("constants.json")
	}
	if err == nil {
		var p struct {
			Recovery json.RawMessage `json:"recovery"`
		}
		if err = json.Unmarshal(raw, &p); err == nil {
			err = json.Unmarshal(p.Recovery, &recoveryPolicy)
		}
	}
	initErr = err
}

// Validate enforces schema, strict JSON and semantic/UTF-8 byte constraints.
// It never substitutes for runtime ownership, token or package-byte checks.
func Validate(definition string, raw []byte) (any, error) {
	limit := 65536
	if responseDefinition(definition) {
		limit = 8388608
	}
	value, err := strictJSONWithLimit(raw, limit)
	if err != nil {
		return nil, err
	}
	initOnce.Do(initialize)
	if initErr != nil {
		return nil, initErr
	}
	compileMu.Lock()
	schema, err := compiler.Compile(schemaID + "#/$defs/" + definition)
	compileMu.Unlock()
	if err != nil {
		return nil, err
	}
	if err = schema.Validate(value); err != nil {
		return nil, fmt.Errorf("%s schema rejected value", definition)
	}
	if err = semantic(value); err != nil {
		return nil, err
	}
	return value, nil
}

func strictJSON(raw []byte) (any, error) {
	return strictJSONWithLimit(raw, 65536)
}

func responseDefinition(definition string) bool {
	if strings.HasSuffix(definition, "Response") {
		return true
	}
	switch definition {
	case "Discovery", "Me", "Plugin", "MarketVersion", "Artifact", "Artifacts", "Publication", "PublicationOptions", "PublishToken", "CreatedToken", "UploadReceipt", "SubmissionReceipt", "Operation", "OperationProcessing", "OperationSucceeded", "OperationFailed", "VersionsPage", "TokensPage", "PublicationsPage", "Error":
		return true
	}
	return false
}

func strictJSONWithLimit(raw []byte, limit int) (any, error) {
	if len(raw) > limit || !utf8.Valid(raw) {
		return nil, fmt.Errorf("JSON byte or Unicode limit")
	}
	if bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) {
		return nil, fmt.Errorf("UTF-8 BOM is not allowed")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("invalid JSON")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return nil, fmt.Errorf("trailing JSON")
	}
	if err := safeNumbers(value, 0); err != nil {
		return nil, err
	}
	// Validate the original bytes before using the decoded value: JCS rejects
	// duplicate members and lone surrogates that encoding/json would replace.
	if _, err := jcs.Transform(raw); err != nil {
		return nil, fmt.Errorf("invalid strict JSON")
	}
	return value, nil
}

func safeNumbers(value any, depth int) error {
	if depth > 64 {
		return fmt.Errorf("JSON nesting limit")
	}
	switch v := value.(type) {
	case string:
		if strings.IndexByte(v, 0) >= 0 {
			return fmt.Errorf("NUL is not supported in protocol text")
		}
	case json.Number:
		n, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil || strings.ContainsAny(string(v), ".eE") || n > 9007199254740991 || n < -9007199254740991 {
			return fmt.Errorf("only safe JSON integers are supported")
		}
	case map[string]any:
		for key, child := range v {
			if strings.IndexByte(key, 0) >= 0 {
				return fmt.Errorf("NUL is not supported in JSON member names")
			}
			if err := safeNumbers(child, depth+1); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range v {
			if err := safeNumbers(child, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func integer(v any) int64 {
	if n, ok := v.(json.Number); ok {
		f, _ := strconv.ParseFloat(string(n), 64)
		return int64(f)
	}
	return 0
}

func timestamp(v any) time.Time {
	s, _ := v.(string)
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}

func trimASCII(s string) string { return strings.Trim(s, " \t\r\n") }

func semantic(value any) error {
	switch v := value.(type) {
	case []any:
		for _, child := range v {
			if err := semantic(child); err != nil {
				return err
			}
		}
	case map[string]any:
		for key, child := range v {
			if s, ok := child.(string); ok {
				limit := 0
				switch key {
				case "release_notes", "description", "security_statement", "privacy_statement":
					limit = 10000
				case "support_url", "icon_url", "source", "console_base_url", "market_version_url", "market_publication_url":
					limit = 2000
					if s != "" {
						if _, err := webURL(s); err != nil {
							return err
						}
					}
				case "min_core":
					limit = 20
				case "name":
					limit = 200
				case "summary":
					limit = 2000
				case "reason", "message":
					limit = 1000
				case "review_reason", "publish_reason":
					limit = 4000
				}
				if limit > 0 && len(s) > limit {
					return fmt.Errorf("%s exceeds UTF-8 byte budget", key)
				}
				if key == "description" || key == "security_statement" || key == "privacy_statement" {
					if utf8.RuneCountInString(trimASCII(s)) < 10 {
						return fmt.Errorf("metadata text is too short after normalization")
					}
				}
			}
			if err := semantic(child); err != nil {
				return err
			}
		}
		for _, key := range []string{"artifacts", "expected_artifacts"} {
			if list, ok := v[key].([]any); ok {
				ids, targets := map[string]bool{}, map[string]bool{}
				for _, child := range list {
					a := child.(map[string]any)
					id := a["artifact_id"].(string)
					target := a["os"].(string) + "/" + a["arch"].(string)
					if ids[id] || targets[target] {
						return fmt.Errorf("duplicate artifact UUID or target")
					}
					ids[id], targets[target] = true, true
				}
			}
		}
		if list, ok := v["scopes"].([]any); ok {
			scopes := map[string]bool{}
			for _, s := range list {
				scopes[s.(string)] = true
			}
			if !scopes["plugins:read"] || scopes["publications:submit"] && !scopes["publications:read"] {
				return fmt.Errorf("missing scope dependency")
			}
			if name, ok := v["name"].(string); ok && utf8.RuneCountInString(trimASCII(name)) < 2 {
				return fmt.Errorf("credential name is too short after trimming")
			}
		}
		_, fieldError := v["field"]
		if reason, ok := v["reason"].(string); ok && !fieldError && utf8.RuneCountInString(trimASCII(reason)) < 2 {
			return fmt.Errorf("reason is too short after trimming")
		}
		if name, ok := v["error"].(string); ok {
			p := errorPolicy[name]
			if integer(v["code"]) != int64(p.Status) || v["retryable"] != p.Retryable {
				return fmt.Errorf("error status or retry policy mismatch")
			}
		}
		if items, ok := v["items"].([]any); ok {
			if int64(len(items)) > integer(v["size"]) || int64(len(items)) > integer(v["total"]) {
				return fmt.Errorf("inconsistent page counts")
			}
		}
		if _, ok := v["token_id"]; ok {
			created, expires := timestamp(v["created_at"]), timestamp(v["expires_at"])
			if !created.IsZero() && (!expires.After(created) || expires.Sub(created) > 30*24*time.Hour) {
				return fmt.Errorf("invalid credential lifetime")
			}
			if status, ok := v["status"].(string); ok && (status == "revoked") != (v["revoked_at"] != nil) {
				return fmt.Errorf("inconsistent credential revocation")
			}
		}
		if _, ok := v["request_sha256"]; ok {
			created, updated := timestamp(v["created_at"]), timestamp(v["updated_at"])
			if updated.Before(created) {
				return fmt.Errorf("operation time moved backwards")
			}
			if v["status"] != "processing" && timestamp(v["retain_until"]).Before(updated.Add(7*24*time.Hour)) {
				return fmt.Errorf("operation retention is too short")
			}
			if result, ok := v["result"].(map[string]any); ok {
				_, upload := result["artifact"]
				if upload != (v["type"] == "package-upload") || result["operation_id"] != v["operation_id"] || upload && result["plugin_id"] != v["plugin_id"] {
					return fmt.Errorf("operation receipt mismatch")
				}
			}
		}
		if features, ok := v["features"].([]any); ok {
			set := map[string]bool{}
			for _, feature := range features {
				set[feature.(string)] = true
			}
			if set["package-upload"] && !set["idempotent-writes"] || set["publication-submit"] && !set["package-upload"] {
				return fmt.Errorf("missing feature dependency")
			}
		}
	}
	return nil
}

// Fingerprint returns the JCS payload and domain-separated SHA256. The input is
// UploadFingerprintInput or SubmitFingerprintInput, never multipart file bytes.
func Fingerprint(kind string, raw []byte) (string, string, error) {
	value, err := strictJSON(raw)
	if err != nil {
		return "", "", err
	}
	v, ok := value.(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("fingerprint input must be an object")
	}
	definition := "UploadFingerprintInput"
	if kind == "publication-submit" {
		definition = "SubmitFingerprintInput"
		if meta, ok := v["metadata"].(map[string]any); ok {
			for key, value := range meta {
				if text, ok := value.(string); ok {
					meta[key] = trimASCII(text)
				}
			}
			if _, exists := meta["icon_url"]; !exists {
				meta["icon_url"] = ""
			}
		}
	} else if kind != "package-upload" {
		return "", "", fmt.Errorf("unknown operation type")
	}
	encoded, err := json.Marshal(v)
	if err != nil {
		return "", "", err
	}
	if _, err = Validate(definition, encoded); err != nil {
		return "", "", err
	}
	if kind == "publication-submit" {
		artifacts := v["expected_artifacts"].([]any)
		sort.Slice(artifacts, func(i, j int) bool {
			a, b := artifacts[i].(map[string]any), artifacts[j].(map[string]any)
			for _, key := range []string{"os", "arch", "artifact_id"} {
				if a[key] != b[key] {
					return a[key].(string) < b[key].(string)
				}
			}
			return false
		})
	}
	v["protocol"], v["type"] = "powerx.plugin-publisher/v1", kind
	encoded, err = json.Marshal(v)
	if err != nil {
		return "", "", err
	}
	canonical, err := jcs.Transform(encoded)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(append([]byte(fingerprintDomain), canonical...))
	return string(canonical), hex.EncodeToString(sum[:]), nil
}

func webURL(raw string) (*url.URL, error) {
	if strings.ContainsAny(raw, "\\\x00\r\n\t ") || !strings.HasPrefix(raw, "https://") && !strings.HasPrefix(raw, "http://") {
		return nil, fmt.Errorf("invalid web URL")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.Opaque != "" {
		return nil, fmt.Errorf("invalid web URL")
	}
	for _, r := range u.Hostname() {
		if r > 127 {
			return nil, fmt.Errorf("web URL hosts must use ASCII/IDNA form")
		}
	}
	host := strings.ToLower(u.Hostname())
	if ip := net.ParseIP(host); ip != nil {
		if ip.String() != host {
			return nil, fmt.Errorf("IP URL host must be canonical")
		}
	} else if strings.Trim(host, "0123456789.") == "" || strings.HasPrefix(host, "0x") {
		return nil, fmt.Errorf("ambiguous IP URL host")
	}
	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid web URL port")
		}
	}
	return u, nil
}

func origin(u *url.URL) string {
	host, port := strings.ToLower(u.Hostname()), u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	} else {
		n, _ := strconv.Atoi(port)
		port = strconv.Itoa(n)
	}
	return u.Scheme + "://" + net.JoinHostPort(host, port)
}

// TrustedMarketLink validates a receipt link against an explicitly trusted
// console base. It does not authorize trusting a new origin from discovery.
func TrustedMarketLink(base, target string) bool {
	b, err := webURL(base)
	if err != nil || b.RawQuery != "" || b.Fragment != "" {
		return false
	}
	if b.Scheme == "http" && !strings.EqualFold(b.Hostname(), "localhost") {
		ip := net.ParseIP(b.Hostname())
		if ip == nil || !ip.IsLoopback() {
			return false
		}
	}
	t, err := webURL(target)
	if err != nil || t.Fragment != "" || origin(b) != origin(t) {
		return false
	}
	for _, u := range []*url.URL{b, t} {
		p := u.Path
		if p == "" {
			p = "/"
		}
		if u.EscapedPath() != u.Path || path.Clean(p) != strings.TrimSuffix(p, "/") && p != "/" {
			return false
		}
	}
	root := strings.TrimSuffix(b.Path, "/")
	return t.Path == root || strings.HasPrefix(t.Path, root+"/")
}

// RecoveryAction is a reference client decision, not a network retry or an
// authorization check. The caller supplies the actual observed binding/state.
func RecoveryAction(raw []byte) (string, error) {
	value, err := Validate("RecoveryContext", raw)
	if err != nil {
		return "", err
	}
	v := value.(map[string]any)
	if v["credential_status"] != "active" {
		return "reauthorize", nil
	}
	if v["identity_matches"] != true || v["plugin_allowed"] != true || v["scope_allowed"] != true {
		return "deny", nil
	}
	if v["request_matches"] != true {
		return "conflict", nil
	}
	switch v["operation_status"] {
	case "succeeded":
		return "read_current_state", nil
	case "failed_permanent":
		return "stop_permanent", nil
	}
	remaining := recoveryPolicy.MaxElapsed - integer(v["elapsed_seconds"])
	wait := integer(v["retry_after_seconds"])
	if v["operation_status"] == "processing" && wait < recoveryPolicy.Poll {
		wait = recoveryPolicy.Poll
	}
	if remaining <= 0 || integer(v["retry_after_seconds"]) > recoveryPolicy.MaxRetry || wait >= remaining {
		return "manual_review", nil
	}
	if v["operation_status"] == "processing" {
		if wait >= recoveryPolicy.MaxPoll-integer(v["poll_elapsed_seconds"]) {
			return "keep_pending", nil
		}
		return "poll_operation", nil
	}
	if v["operation_status"] == "unknown" {
		return "query_operation", nil
	}
	if v["retention_expired"] == true {
		return "reconcile_version", nil
	}
	if v["feature_available"] != true {
		return "wait_capability", nil
	}
	if integer(v["attempts"]) >= recoveryPolicy.Attempts {
		return "manual_review", nil
	}
	return "retry_same_operation", nil
}
