// Package ai provides the typed, delegated client for PowerX Core AI APIs.
// Plugin business code must use this package instead of constructing /ai/*
// HTTP requests or passing raw Gateway payloads.
package ai

import "encoding/json"

type ContentItem struct {
	Role    string `json:"role,omitempty"`
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	URL     string `json:"url,omitempty"`
}

type LLMInvokeInput struct {
	ModelKey string         `json:"model_key"`
	Inputs   []ContentItem  `json:"inputs"`
	Params   map[string]any `json:"params,omitempty"`
}

type LLMInvokeOutput struct {
	Type         string         `json:"type"`
	Text         string         `json:"text"`
	FinishReason string         `json:"finish_reason,omitempty"`
	Usage        map[string]any `json:"usage,omitempty"`
}

type LLMModel struct {
	ModelKey          string         `json:"model_key"`
	Provider          string         `json:"provider"`
	Model             string         `json:"model"`
	Label             string         `json:"label,omitempty"`
	Source            string         `json:"source"`
	Configured        bool           `json:"configured"`
	ProfileConfigured bool           `json:"profile_configured"`
	Defaults          map[string]any `json:"defaults,omitempty"`
	Tags              []string       `json:"tags,omitempty"`
}

type ListLLMModelsOutput struct {
	Environment string     `json:"env"`
	Items       []LLMModel `json:"items"`
}

type CreateLLMSessionInput struct {
	ModelKey string `json:"model_key"`
	Title    string `json:"title,omitempty"`
}

type LLMSession struct {
	SessionID string `json:"session_id"`
}

type AppendLLMSessionMessageInput struct {
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

type EmbeddingInvokeInput struct {
	ModelKey string         `json:"model_key"`
	Inputs   []string       `json:"inputs"`
	Params   map[string]any `json:"params,omitempty"`
}

type EmbeddingInvokeOutput struct {
	Vectors [][]float32       `json:"vectors"`
	Usage   map[string]any    `json:"usage,omitempty"`
	Extra   map[string]string `json:"extra,omitempty"`
}

// ModalInvokeInput is shared by VLM, image, video and TTS endpoints. The
// provider-specific result stays as JSON because Core intentionally exposes
// provider-defined result schemas for these modalities.
type ModalInvokeInput struct {
	ModelKey string         `json:"model_key"`
	Inputs   []ContentItem  `json:"inputs"`
	Params   map[string]any `json:"params,omitempty"`
}

type ModalInvokeOutput struct {
	Data json.RawMessage
}
