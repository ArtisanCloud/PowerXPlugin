// Package hosthttp defines the Core HTTP address boundary used by Host clients.
package hosthttp

import (
	"errors"
	"net/url"
	"strings"
)

// DefaultAPIPrefix is the declared v1 default, not a resource-specific prefix.
const DefaultAPIPrefix = "/api/v1"

// APIBaseURL joins a configured gateway base and Core API prefix once. It does
// not infer a version from the gateway URL or change the plugin's HTTP base.
func APIBaseURL(baseURL, apiPrefix string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || strings.ContainsAny(baseURL, "?#") || u.User != nil {
		return "", errors.New("FRAMEWORK_HOST_API_BASE_INVALID")
	}
	prefix := strings.TrimSpace(apiPrefix)
	if prefix == "" {
		prefix = DefaultAPIPrefix
	}
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return baseURL, nil
	}
	for _, part := range strings.Split(prefix, "/") {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("FRAMEWORK_HOST_API_PREFIX_INVALID")
		}
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == '~') {
				return "", errors.New("FRAMEWORK_HOST_API_PREFIX_INVALID")
			}
		}
	}
	return baseURL + "/" + prefix, nil
}
