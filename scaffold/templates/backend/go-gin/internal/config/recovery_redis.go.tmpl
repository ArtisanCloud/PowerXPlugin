package config

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
)

// Recovery Redis bindings are explicit and only supplied by the owned host.
// Other endpoints fail closed rather than silently dropping their persistence.
func applyRecoveryRedisBindings(c *Config) error {
	if _, ok := os.LookupEnv("POWERX_RECOVERY_READ_ONLY"); !ok {
		return nil
	}
	source := os.Getenv("POWERX_RECOVERY_REDIS_SOURCE_ADDR")
	target := os.Getenv("POWERX_REDIS_ADDR")
	fields := []*string{}
	if c.WSBus != nil && c.WSBus.Provider == "redis" {
		fields = append(fields, &c.WSBus.RedisURL)
	}
	if c.EventBridge != nil && c.EventBridge.TaskBusProvider == "redis" {
		fields = append(fields, &c.EventBridge.RedisURL)
	}
	if c.Integration != nil && c.Integration.Idempotency.Provider == "redis" {
		fields = append(fields, &c.Integration.Idempotency.RedisURL)
	}
	if c.Marketplace != nil && c.Marketplace.License.Cache.Provider == "redis" {
		fields = append(fields, &c.Marketplace.License.Cache.RedisURL)
	}
	if c.Runtime != nil && c.Runtime.Cache != nil && c.Runtime.Cache.Provider == "redis" {
		fields = append(fields, &c.Runtime.Cache.RedisURL)
	}
	for _, field := range fields {
		rebound, err := recoveryRedisURL(*field, source, target, os.Getenv("POWERX_REDIS_PASSWORD"))
		if err != nil {
			return err
		}
		*field = rebound
	}
	return nil
}

func recoveryRedisURL(raw, source, target, password string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "redis" || source == "" || target == "" {
		return "", errors.New("RECOVERY_PLUGIN_REDIS_BINDING_INVALID")
	}
	endpoint := func(addr string) string {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return ""
		}
		host = strings.ToLower(host)
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			host = "loopback"
		}
		return net.JoinHostPort(host, port)
	}
	port := u.Port()
	if port == "" {
		port = "6379"
	}
	if endpoint(source) == "" || endpoint(source) != endpoint(net.JoinHostPort(u.Hostname(), port)) || endpoint(target) == "" {
		return "", errors.New("RECOVERY_PLUGIN_REDIS_EXTERNAL_UNSUPPORTED")
	}
	u.Host = target
	u.User = nil
	if password != "" {
		u.User = url.UserPassword("", password)
	}
	// Preserve the component's original Redis DB and query options.
	return u.String(), nil
}
