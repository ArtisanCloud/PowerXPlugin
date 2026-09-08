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
		ReasonCode string `json:"reason_code"`
		ErrorCode  string `json:"error_code"`
		Error      struct {
			ReasonCode string `json:"reason_code"`
			ErrorCode  string `json:"error_code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return strings.TrimSpace(fallback)
	}
	for _, value := range []string{envelope.Error.ReasonCode, envelope.ReasonCode, envelope.Error.ErrorCode, envelope.ErrorCode, fallback} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
