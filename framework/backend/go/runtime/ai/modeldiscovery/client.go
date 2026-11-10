// Package modeldiscovery 仅查询供应商模型目录，不提交生成或下载请求。
package modeldiscovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
)

const maxResponseBytes = 8 << 20
const maxModels = 10000
const maxPages = 100

type Spec struct {
	Protocol, Path        string
	RootPath, OptionalKey bool
}

// ProviderSpec 显式声明协议；不能把所有供应商都当成 OpenAI。
func ProviderSpec(provider string) (Spec, bool) {
	switch provider {
	case "openai", "openrouter", "moonshot", "huggingface":
		return Spec{Protocol: "openai", Path: "models"}, true
	case "openai_compatible":
		return Spec{Protocol: "openai", Path: "models", OptionalKey: true}, true
	case "deepseek":
		return Spec{Protocol: "openai", Path: "models"}, true
	case "qwen-cn", "qwen-intl":
		return Spec{Protocol: "dashscope", Path: "/api/v1/models", RootPath: true}, true
	case "anthropic":
		return Spec{Protocol: "anthropic", Path: "models"}, true
	case "google":
		return Spec{Protocol: "gemini", Path: "models"}, true
	case "ollama":
		return Spec{Protocol: "ollama", Path: "api/tags", OptionalKey: true}, true
	}
	return Spec{}, false
}

type Input struct {
	Provider, Endpoint             string
	APIKey                         string `json:"-"`
	Organization, AnthropicVersion string `json:"-"`
	Timeout                        time.Duration
}
type Model struct {
	ID         string   `json:"id"`
	Label      string   `json:"label"`
	Modalities []string `json:"modalities"`
}
type Result struct {
	Protocol string
	Source   string
	Models   []Model
}

func Error(status int, code string, details map[string]any) error {
	return &powerxai.HTTPError{StatusCode: status, ErrorCode: code, ReasonCode: code, Details: details}
}

// Fetch 使用保存的连接地址和固定协议路径；分页完整结束后才返回成功。
func Fetch(ctx context.Context, in Input) (resultValue *Result, failure error) {
	spec, ok := ProviderSpec(in.Provider)
	if !ok {
		return nil, Error(422, "AI_MODEL_SYNC_UNSUPPORTED", nil)
	}
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(in.Endpoint), "/"))
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, Error(422, "AI_MODEL_DISCOVERY_CONFIG_INVALID", nil)
	}
	if strings.TrimSpace(in.APIKey) == "" && !spec.OptionalKey {
		return nil, Error(409, "AI_SETTINGS_PROVIDER_NOT_CONFIGURED", nil)
	}
	if (in.APIKey != "" && strings.Contains(in.Endpoint, in.APIKey)) || strings.ContainsAny(in.APIKey+in.Organization+in.AnthropicVersion, "\r\n") {
		return nil, Error(422, "AI_MODEL_DISCOVERY_CONFIG_INVALID", nil)
	}
	if spec.RootPath {
		// DashScope 原生目录与兼容推理接口共用已配置 origin，保留网关路径前缀。
		base := strings.TrimSuffix(strings.TrimSuffix(u.Path, "/compatible-mode/v1"), "/api/v1")
		u.Path = strings.TrimRight(base, "/") + spec.Path
	} else {
		switch spec.Protocol {
		case "ollama":
			u.Path = strings.TrimSuffix(strings.TrimSuffix(u.Path, "/v1"), "/api")
		case "anthropic":
			if !strings.HasSuffix(u.Path, "/v1") {
				u.Path += "/v1"
			}
		case "openai":
			if (in.Provider == "openai" || in.Provider == "openai_compatible") && u.Path == "" {
				u.Path = "/v1"
			}
		}
		u.Path = strings.TrimRight(u.Path, "/") + "/" + spec.Path
	}
	u.RawPath = ""
	source := u.String()
	failureSource := source
	failureModel := ""
	defer func() {
		var appErr *powerxai.HTTPError
		if errors.As(failure, &appErr) {
			if appErr.Details == nil {
				appErr.Details = map[string]any{}
			}
			appErr.Details["provider"] = in.Provider
			appErr.Details["protocol"] = spec.Protocol
			appErr.Details["source"] = failureSource
			if failureModel != "" {
				appErr.Details["model"] = failureModel
			}
			appErr.Details["generation_submitted"] = false
		}
	}()
	timeout := in.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	read := func(method string, target *url.URL, body []byte, out any) error {
		location := *target
		location.RawQuery = ""
		failureSource = location.String()
		req, e := http.NewRequestWithContext(ctx, method, target.String(), bytes.NewReader(body))
		if e != nil {
			return Error(422, "AI_MODEL_DISCOVERY_CONFIG_INVALID", nil)
		}
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		switch spec.Protocol {
		case "gemini":
			req.Header.Set("x-goog-api-key", in.APIKey)
		case "anthropic":
			req.Header.Set("x-api-key", in.APIKey)
			version := in.AnthropicVersion
			if version == "" {
				version = "2023-06-01"
			}
			req.Header.Set("anthropic-version", version)
		default:
			if in.APIKey != "" {
				req.Header.Set("Authorization", "Bearer "+in.APIKey)
			}
		}
		if in.Organization != "" && in.Provider == "openai" {
			req.Header.Set("OpenAI-Organization", in.Organization)
		}
		resp, e := client.Do(req)
		if e != nil {
			if ctx.Err() != nil {
				return Error(504, "AI_MODEL_DISCOVERY_TIMEOUT", nil)
			}
			return Error(502, "AI_MODEL_DISCOVERY_UNREACHABLE", nil)
		}
		defer resp.Body.Close()
		trace, requestID := resp.Header.Get("X-Trace-ID"), resp.Header.Get("X-Request-ID")
		if in.APIKey != "" {
			if strings.Contains(trace, in.APIKey) {
				trace = ""
			}
			if strings.Contains(requestID, in.APIKey) {
				requestID = ""
			}
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			code := "AI_MODEL_DISCOVERY_UPSTREAM_FAILED"
			status := 502
			if resp.StatusCode == 401 || resp.StatusCode == 403 {
				code = "AI_MODEL_SYNC_AUTH_FAILED"
				status = 422
			}
			return &powerxai.HTTPError{StatusCode: status, ErrorCode: code, ReasonCode: code, TraceID: trace, RequestID: requestID, Details: map[string]any{"upstream_status": resp.StatusCode}}
		}
		raw, e := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
		if ctx.Err() != nil {
			return Error(504, "AI_MODEL_DISCOVERY_TIMEOUT", nil)
		}
		if e != nil || len(raw) > maxResponseBytes || json.Unmarshal(raw, out) != nil {
			return Error(502, "AI_MODEL_DISCOVERY_RESPONSE_INVALID", nil)
		}
		return nil
	}
	result := &Result{Protocol: spec.Protocol, Source: source, Models: []Model{}}
	seen := map[string]bool{}
	tokens := map[string]bool{}
	for page := 1; page <= maxPages; page++ {
		q := u.Query()
		if spec.Protocol == "dashscope" {
			q.Set("page_no", strconv.Itoa(page))
			q.Set("page_size", "100")
		}
		if spec.Protocol == "gemini" {
			q.Set("pageSize", "1000")
		}
		if spec.Protocol == "anthropic" {
			q.Set("limit", "1000")
		}
		if in.Provider == "openrouter" {
			q.Set("output_modalities", "all")
		}
		u.RawQuery = q.Encode()
		var response struct {
			Data          []remoteModel `json:"data"`
			Models        []remoteModel `json:"models"`
			HasMore       bool          `json:"has_more"`
			LastID        string        `json:"last_id"`
			NextPageToken string        `json:"nextPageToken"`
			Success       *bool         `json:"success"`
			Output        *struct {
				Total  int           `json:"total"`
				Models []remoteModel `json:"models"`
			} `json:"output"`
		}
		if e := read(http.MethodGet, u, nil, &response); e != nil {
			return nil, e
		}
		items := response.Data
		switch spec.Protocol {
		case "gemini", "ollama":
			items = response.Models
		case "dashscope":
			if response.Output == nil || (response.Success != nil && !*response.Success) {
				return nil, Error(502, "AI_MODEL_DISCOVERY_RESPONSE_INVALID", nil)
			}
			items = response.Output.Models
		}
		if items == nil {
			return nil, Error(502, "AI_MODEL_DISCOVERY_RESPONSE_INVALID", nil)
		}
		for _, item := range items {
			id := item.ID
			if spec.Protocol == "gemini" {
				id = strings.TrimPrefix(item.Name, "models/")
			}
			if spec.Protocol == "ollama" {
				id = item.Name
				if id == "" {
					id = item.Model
				}
			}
			if spec.Protocol == "dashscope" {
				id = item.Model
			}
			if id == "" || id != strings.TrimSpace(id) || len(id) > 512 || strings.ContainsAny(id, "\r\n\x00") {
				return nil, Error(502, "AI_MODEL_DISCOVERY_RESPONSE_INVALID", nil)
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			mods := []string{}
			add := func(mod string) {
				for _, v := range mods {
					if v == mod {
						return
					}
				}
				mods = append(mods, mod)
			}
			if spec.Protocol == "ollama" {
				failureModel = id
				show := *u
				show.RawQuery = ""
				show.Path = strings.TrimSuffix(show.Path, "tags") + "show"
				body, _ := json.Marshal(map[string]string{"model": id})
				var detail struct {
					Capabilities []string `json:"capabilities"`
				}
				if e := read(http.MethodPost, &show, body, &detail); e != nil {
					return nil, e
				}
				if len(detail.Capabilities) == 0 {
					return nil, Error(502, "AI_MODEL_DISCOVERY_CAPABILITIES_UNAVAILABLE", nil)
				}
				failureModel = ""
				for _, c := range detail.Capabilities {
					switch c {
					case "completion":
						add("llm")
					case "embedding":
						add("embedding")
					case "vision":
						add("vlm")
					}
				}
			}
			if spec.Protocol == "anthropic" {
				add("llm")
			}
			for _, m := range item.SupportedGenerationMethods {
				switch m {
				case "generateContent":
					add("llm")
				case "embedContent", "batchEmbedContents":
					add("embedding")
				}
			}
			outputs := append(item.OutputModalities, item.Architecture.OutputModalities...)
			for _, m := range outputs {
				switch strings.ToLower(m) {
				case "text":
					add("llm")
				case "embeddings", "embedding":
					add("embedding")
				}
			}
			for _, c := range item.Capabilities {
				switch c {
				case "TG", "Reasoning":
					add("llm")
				case "TR", "ME":
					add("embedding")
				}
			}
			label := item.DisplayName
			if label == "" {
				label = item.DisplayNameCamel
			}
			if label == "" {
				label = item.Name
			}
			if label == "" || spec.Protocol == "ollama" {
				label = id
			}
			result.Models = append(result.Models, Model{ID: id, Label: label, Modalities: mods})
			if len(result.Models) > maxModels {
				return nil, Error(502, "AI_MODEL_SYNC_LIMIT_EXCEEDED", nil)
			}
		}
		next := ""
		param := ""
		switch spec.Protocol {
		case "gemini":
			next = response.NextPageToken
			param = "pageToken"
		case "anthropic", "openai":
			if response.HasMore {
				next = response.LastID
				param = "after_id"
				if spec.Protocol == "openai" {
					param = "after"
				}
				if next == "" {
					return nil, Error(502, "AI_MODEL_DISCOVERY_RESPONSE_INVALID", nil)
				}
			}
		case "dashscope":
			if response.Output.Total > len(result.Models) {
				if len(items) == 0 {
					return nil, Error(502, "AI_MODEL_DISCOVERY_RESPONSE_INVALID", nil)
				}
				continue
			}
		}
		if next == "" {
			return result, nil
		}
		if tokens[next] {
			return nil, Error(502, "AI_MODEL_DISCOVERY_RESPONSE_INVALID", nil)
		}
		tokens[next] = true
		q = u.Query()
		q.Set(param, next)
		u.RawQuery = q.Encode()
	}
	return nil, Error(502, "AI_MODEL_SYNC_LIMIT_EXCEEDED", nil)
}

type remoteModel struct {
	ID                         string   `json:"id"`
	Name                       string   `json:"name"`
	Model                      string   `json:"model"`
	DisplayName                string   `json:"display_name"`
	DisplayNameCamel           string   `json:"displayName"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	OutputModalities           []string `json:"output_modalities"`
	Capabilities               []string `json:"capabilities"`
	Architecture               struct {
		OutputModalities []string `json:"output_modalities"`
	} `json:"architecture"`
}
