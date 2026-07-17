package knowledge

import (
	"fmt"
	"regexp"
	"strings"
)

var sensitiveKeyPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|access[_-]?key|api[_-]?key|credential|authorization)`)

func RedactString(value string) string {
	if strings.TrimSpace(value) == "" {
		return value
	}
	out := value
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]+`),
		regexp.MustCompile(`(?i)(token|secret|password|access_key|api_key)=([^&\s]+)`),
	}
	for _, pattern := range patterns {
		out = pattern.ReplaceAllString(out, `${1}[redacted]`)
	}
	return out
}

func RedactMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		if sensitiveKeyPattern.MatchString(key) {
			out[key] = "[redacted]"
			continue
		}
		switch typed := value.(type) {
		case string:
			out[key] = RedactString(typed)
		case map[string]any:
			out[key] = RedactMap(typed)
		default:
			out[key] = typed
		}
	}
	return out
}

func SafeDetails(values map[string]any) map[string]any {
	return RedactMap(values)
}

func containsSensitiveValue(values map[string]any) bool {
	for key, value := range values {
		if sensitiveKeyPattern.MatchString(key) {
			return true
		}
		if strings.Contains(strings.ToLower(fmt.Sprint(value)), "bearer ") {
			return true
		}
	}
	return false
}
