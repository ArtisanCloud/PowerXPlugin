package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
)

// RefreshToolToken 调用 PowerX Auth 接口刷新 Tool Token。
func RefreshToolToken(ctx context.Context, cfg *config.Config) (string, string, error) {
	if cfg == nil || cfg.Gateway == nil {
		return "", "", fmt.Errorf("gateway config missing")
	}
	refreshToken := strings.TrimSpace(cfg.Gateway.RefreshToken)
	if refreshToken == "" {
		return "", "", fmt.Errorf("PX_TOOL_REFRESH_TOKEN 未配置")
	}

	base := strings.TrimSpace(cfg.Gateway.AuthBaseURL)
	if base == "" {
		base = cfg.Gateway.BaseURL
	}
	if base == "" {
		return "", "", fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	endpoints := buildRefreshEndpoints(base, cfg.Gateway.APIPrefix)
	if len(endpoints) == 0 {
		return "", "", fmt.Errorf("refresh endpoint 未配置")
	}

	payload := map[string]string{"refresh_token": refreshToken}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	attemptErrors := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(bodyBytes)))
		if err != nil {
			attemptErrors = append(attemptErrors, fmt.Sprintf("endpoint=%s build_request_err=%v", endpoint, err))
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			attemptErrors = append(attemptErrors, fmt.Sprintf("endpoint=%s request_err=%v", endpoint, err))
			continue
		}

		responseBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			attemptErrors = append(attemptErrors, fmt.Sprintf("endpoint=%s read_body_err=%v", endpoint, readErr))
			continue
		}

		newToken, newRefresh, parseErr := parseRefreshResponse(endpoint, resp.StatusCode, responseBody, refreshToken)
		if parseErr != nil {
			attemptErrors = append(attemptErrors, parseErr.Error())
			continue
		}

		cfg.Gateway.ToolToken = newToken
		cfg.Gateway.RefreshToken = newRefresh
		os.Setenv("PX_TOOL_TOKEN", newToken)
		os.Setenv("PX_TOOL_REFRESH_TOKEN", newRefresh)

		return newToken, newRefresh, nil
	}

	return "", "", fmt.Errorf("token refresh failed on all endpoints (%d): %s", len(endpoints), strings.Join(attemptErrors, " | "))
}

func parseRefreshResponse(endpoint string, statusCode int, bodyBytes []byte, fallbackRefreshToken string) (string, string, error) {
	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			TokenType    string `json:"token_type"`
			ExpiresIn    int64  `json:"expires_in"`
		} `json:"data"`
		Error *struct {
			Code    any    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	var legacy struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			TokenType    string `json:"token_type"`
			ExpiresIn    int64  `json:"expires_in"`
		} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &envelope); err != nil || (envelope.Data.AccessToken == "" && envelope.Error == nil && envelope.Message == "") {
		if err := json.Unmarshal(bodyBytes, &legacy); err != nil {
			return "", "", fmt.Errorf("endpoint=%s status=%d non_json_response=%q parse_err=%w", endpoint, statusCode, previewBody(bodyBytes), err)
		}
		if statusCode >= 400 || legacy.Code >= 400 {
			return "", "", fmt.Errorf("endpoint=%s refresh_failed status=%d code=%d message=%s", endpoint, statusCode, legacy.Code, legacy.Message)
		}
		envelope.Data = legacy.Data
		envelope.Message = legacy.Message
		envelope.Success = statusCode < 400
	}
	if statusCode >= 400 || (envelope.Error != nil && envelope.Error.Message != "") {
		errMessage := ""
		if envelope.Error != nil {
			errMessage = strings.TrimSpace(envelope.Error.Message)
		}
		if errMessage == "" {
			errMessage = strings.TrimSpace(envelope.Message)
		}
		if errMessage == "" {
			errMessage = previewBody(bodyBytes)
		}
		return "", "", fmt.Errorf("endpoint=%s refresh_failed status=%d message=%s", endpoint, statusCode, errMessage)
	}
	newToken := strings.TrimSpace(envelope.Data.AccessToken)
	if newToken == "" {
		return "", "", fmt.Errorf("endpoint=%s refresh response missing access_token", endpoint)
	}
	newRefresh := strings.TrimSpace(envelope.Data.RefreshToken)
	if newRefresh == "" {
		newRefresh = fallbackRefreshToken
	}

	return newToken, newRefresh, nil
}

func buildRefreshEndpoints(base, apiPrefix string) []string {
	trimmedBase := strings.TrimRight(strings.TrimSpace(base), "/")
	if trimmedBase == "" {
		return nil
	}
	normalizedPrefix := normalizeAPIPrefix(apiPrefix)
	baseWithoutPrefix := strings.TrimSuffix(trimmedBase, normalizedPrefix)
	if baseWithoutPrefix == "" {
		baseWithoutPrefix = trimmedBase
	}
	legacyBaseWithoutAPIV1 := strings.TrimSuffix(trimmedBase, "/api/v1")
	if legacyBaseWithoutAPIV1 == "" {
		legacyBaseWithoutAPIV1 = trimmedBase
	}

	prefixPath := normalizedPrefix
	if prefixPath == "" {
		prefixPath = "/api/v1"
	}
	candidates := []string{
		baseWithoutPrefix + prefixPath + "/admin/user/auth/refresh",
		trimmedBase + "/admin/user/auth/refresh",
		baseWithoutPrefix + "/admin/user/auth/refresh",
		legacyBaseWithoutAPIV1 + "/api/v1/admin/user/auth/refresh",
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(candidates))
	for _, endpoint := range candidates {
		clean := strings.TrimSpace(endpoint)
		if clean == "" {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	return result
}

func previewBody(body []byte) string {
	const limit = 160
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}
	if utf8.RuneCountInString(trimmed) <= limit {
		return trimmed
	}
	runes := []rune(trimmed)
	return string(runes[:limit]) + "..."
}
