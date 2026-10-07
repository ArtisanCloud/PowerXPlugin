package hostcontract

import (
	"encoding/json"
	"strings"
)

// ParseReasonCode extracts the machine-readable Core reason from either
// supported error envelope shape. fallback is used only for malformed or
// intermediary responses which do not contain a Core reason.
func ParseReasonCode(raw []byte, fallback string) string {
	var envelope struct {
		ReasonCode string          `json:"reason_code"`
		ErrorCode  string          `json:"error_code"`
		Error      json.RawMessage `json:"error"`
		Details    json.RawMessage `json:"details"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return strings.TrimSpace(fallback)
	}
	var nested struct {
		ReasonCode string `json:"reason_code"`
		ErrorCode  string `json:"error_code"`
		Code       string `json:"code"`
	}
	// Core can return a diagnostic string in error alongside top-level reason_code.
	// Parsing the optional object separately preserves the stable top-level fields.
	_ = json.Unmarshal(envelope.Error, &nested)
	var details struct {
		ReasonCode string `json:"reason_code"`
		ErrorCode  string `json:"error_code"`
		Code       string `json:"code"`
	}
	_ = json.Unmarshal(envelope.Details, &details)
	for _, value := range []string{nested.ReasonCode, envelope.ReasonCode, details.ReasonCode, nested.ErrorCode, envelope.ErrorCode, details.ErrorCode, nested.Code, details.Code, fallback} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
