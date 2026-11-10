package hostcontract

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// PluginRecovery contains only the recovery fields published by Core. Pointers
// distinguish an absent value from a meaningful false or zero. Retryable is
// information for the business caller, never an automatic retry instruction.
type PluginRecovery struct {
	NextActionCode   string `json:"next_action_code,omitempty"`
	Retryable        *bool  `json:"retryable,omitempty"`
	RetryAfterMS     *int64 `json:"retry_after_ms,omitempty"`
	RetryAfter       *int64 `json:"retry_after,omitempty"`
	OperationID      string `json:"operation_id,omitempty"`
	OperationUUID    string `json:"operation_uuid,omitempty"`
	RequestID        string `json:"request_id,omitempty"`
	CurrentRevision  *int64 `json:"current_revision,omitempty"`
	ExpectedRevision *int64 `json:"expected_revision,omitempty"`
}

type InvocationFailureDetails struct {
	PluginRecovery
	ReasonCode              string          `json:"reason_code,omitempty"`
	UpstreamErrorCode       string          `json:"upstream_error_code,omitempty"`
	UpstreamStatus          *int64          `json:"upstream_status,omitempty"`
	TraceID                 string          `json:"trace_id,omitempty"`
	UpstreamTraceID         string          `json:"upstream_trace_id,omitempty"`
	UpstreamRequestID       string          `json:"upstream_request_id,omitempty"`
	ExpectedProviderVersion string          `json:"expected_provider_version,omitempty"`
	ActualProviderVersion   string          `json:"actual_provider_version,omitempty"`
	Recovery                *PluginRecovery `json:"recovery,omitempty"`
}

func (d *InvocationFailureDetails) Map() map[string]any {
	if d == nil {
		return nil
	}
	raw, _ := json.Marshal(d)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}

type InvocationFailure struct {
	StatusCode int
	ErrorCode  string
	ReasonCode string
	TraceID    string
	RequestID  string
	Details    *InvocationFailureDetails
}

func (e *InvocationFailure) Error() string {
	return fmt.Sprintf("CORE_INVOCATION_FAILED reason=%s status=%d trace_id=%s", e.ReasonCode, e.StatusCode, e.TraceID)
}

// DecodeInvocationFailure reads Core's machine diagnostics, not upstream text.
// Core trace and plugin trace remain separate. Unknown details are discarded.
func DecodeInvocationFailure(status int, headers http.Header, raw []byte, defaultCode string) *InvocationFailure {
	e := &InvocationFailure{StatusCode: status, ErrorCode: defaultCode, ReasonCode: defaultCode, TraceID: safeDiagnostic(headers.Get("X-Trace-ID")), RequestID: safeDiagnostic(headers.Get("X-Request-ID"))}
	var body struct {
		Success    *bool                      `json:"success"`
		ErrorCode  string                     `json:"error_code"`
		ReasonCode string                     `json:"reason_code"`
		TraceID    string                     `json:"trace_id"`
		RequestID  string                     `json:"request_id"`
		Details    map[string]json.RawMessage `json:"details"`
		Error      json.RawMessage            `json:"error"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if dec.Decode(&body) != nil || dec.Decode(&struct{}{}) != io.EOF {
		return e
	}
	var nested struct {
		Code       string                     `json:"code"`
		ErrorCode  string                     `json:"error_code"`
		ReasonCode string                     `json:"reason_code"`
		Details    map[string]json.RawMessage `json:"details"`
	}
	_ = json.Unmarshal(body.Error, &nested)
	if body.Success == nil || *body.Success {
		nested.Code = ""
	}
	if body.Details == nil {
		body.Details = nested.Details
	}
	d := &InvocationFailureDetails{PluginRecovery: decodeRecovery(body.Details)}
	readString := func(key string) string { var v string; _ = json.Unmarshal(body.Details[key], &v); return v }
	d.ReasonCode = safeMachineCode(readString("reason_code"), 128)
	d.UpstreamErrorCode = safeMachineCode(readString("upstream_error_code"), 128)
	d.UpstreamStatus = boundedInteger(body.Details["upstream_status"], 599)
	if d.UpstreamStatus != nil && *d.UpstreamStatus < 400 {
		d.UpstreamStatus = nil
	}
	d.TraceID = safeDiagnostic(readString("trace_id"))
	d.UpstreamRequestID = safeUUID(readString("upstream_request_id"))
	if trace := readString("upstream_trace_id"); safeUUID(trace) != "" || len(trace) == 32 && validHex(trace) {
		d.UpstreamTraceID = trace
	}
	d.ExpectedProviderVersion = safeDiagnostic(readString("expected_provider_version"))
	d.ActualProviderVersion = safeDiagnostic(readString("actual_provider_version"))
	var recovery map[string]json.RawMessage
	if json.Unmarshal(body.Details["recovery"], &recovery) == nil && recovery != nil {
		r := decodeRecovery(recovery)
		d.Recovery = &r
	}
	e.Details = d
	firstCode := func(values ...string) string {
		for _, v := range values {
			if v = safeMachineCode(v, 128); v != "" {
				return v
			}
		}
		return defaultCode
	}
	e.ErrorCode = firstCode(body.ErrorCode, nested.ErrorCode, nested.Code, body.ReasonCode, d.ReasonCode, nested.ReasonCode)
	e.ReasonCode = firstCode(body.ReasonCode, d.ReasonCode, nested.ReasonCode, body.ErrorCode, nested.ErrorCode, nested.Code)
	if e.TraceID == "" {
		e.TraceID = firstDiagnostic(body.TraceID, d.TraceID)
	}
	if e.RequestID == "" {
		e.RequestID = safeDiagnostic(body.RequestID)
	}
	return e
}

func safeDiagnostic(v string) string {
	if len(v) > 256 || strings.ContainsAny(v, "\r\n\x00") {
		return ""
	}
	return strings.TrimSpace(v)
}
func firstDiagnostic(values ...string) string {
	for _, v := range values {
		if v = safeDiagnostic(v); v != "" {
			return v
		}
	}
	return ""
}
func safeMachineCode(v string, limit int) string {
	if len(v) < 3 || len(v) > limit {
		return ""
	}
	for i, c := range v {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			continue
		}
		if i > 0 && (c >= '0' && c <= '9' || c == '_' || limit == 128 && (c == '.' || c == '-')) {
			continue
		}
		return ""
	}
	return v
}
func safeUUID(v string) string {
	id, err := uuid.Parse(v)
	if err != nil || id == uuid.Nil || id.String() != v {
		return ""
	}
	return v
}
func validHex(v string) bool { _, err := hex.DecodeString(v); return err == nil }
func boundedInteger(raw json.RawMessage, max int64) *int64 {
	var n int64
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &n) != nil || n < 0 || n > max {
		return nil
	}
	return &n
}
func decodeRecovery(fields map[string]json.RawMessage) PluginRecovery {
	var r PluginRecovery
	read := func(key string) string { var v string; _ = json.Unmarshal(fields[key], &v); return v }
	r.NextActionCode = safeMachineCode(read("next_action_code"), 64)
	var retryable *bool
	if json.Unmarshal(fields["retryable"], &retryable) == nil {
		r.Retryable = retryable
	}
	r.RetryAfterMS = boundedInteger(fields["retry_after_ms"], 86400000)
	r.RetryAfter = boundedInteger(fields["retry_after"], 86400)
	r.CurrentRevision = boundedInteger(fields["current_revision"], 1<<53-1)
	r.ExpectedRevision = boundedInteger(fields["expected_revision"], 1<<53-1)
	r.OperationID = safeUUID(read("operation_id"))
	r.OperationUUID = safeUUID(read("operation_uuid"))
	r.RequestID = safeUUID(read("request_id"))
	return r
}
