package delegated

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

// IdentityRequestError retains diagnostic identifiers without keeping an
// upstream message/body which could echo the proof or service credential.
type IdentityRequestError struct {
	Status          int
	Code            string
	ReasonCode      string
	TraceID         string
	RequestID       string
	SourceTraceID   string
	SourceRequestID string
}

func (e *IdentityRequestError) Error() string { return e.IAMCode() }
func (e *IdentityRequestError) IAMCode() string {
	if e.ReasonCode != "" {
		return e.ReasonCode
	}
	return e.Code
}
func (e *IdentityRequestError) HTTPStatus() int { return e.Status }

type identityMetadataKey struct{}
type RequestMetadata struct{ RequestID, TraceID string }

func WithRequestMetadata(ctx context.Context, m RequestMetadata) context.Context {
	return context.WithValue(ctx, identityMetadataKey{}, m)
}

func (c *CoreClient) ResolveDelegatedIdentity(ctx context.Context, in contracts.DelegatedIdentityQuery) (*contracts.DelegatedIdentity, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	raw, err := c.invokeIdentity(ctx, "resolve_request_identity", map[string]string{"schema": contracts.IdentityQuerySchema, "context_ticket": in.ContextTicket})
	if err != nil {
		return nil, err
	}
	out, err := contracts.DecodeDelegatedIdentity(raw)
	if err != nil {
		return nil, err
	}
	if err = out.Validate(c.pluginID, time.Now()); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *CoreClient) RevalidateWSIdentity(ctx context.Context, in contracts.WSIdentityQuery) (*contracts.WSIdentity, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	raw, err := c.invokeIdentity(ctx, "revalidate_ws_identity", map[string]string{"schema": contracts.WSIdentityQuerySchema, "ws_revalidation_handle": in.Handle})
	if err != nil {
		return nil, err
	}
	out, err := contracts.DecodeWSIdentity(raw)
	if err != nil {
		return nil, err
	}
	if err = out.Validate(c.pluginID, time.Now()); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *CoreClient) invokeIdentity(ctx context.Context, operation string, input map[string]string) (json.RawMessage, error) {
	if c == nil || c.tokens == nil || c.client == nil || c.pluginID == "" {
		return nil, iamerrors.New(iamerrors.CodeAdapterNotBound, iamerrors.CodeAdapterNotBound)
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	credential, err := c.tokens.Token(ctx)
	if err != nil || credential == "" {
		return nil, iamerrors.New(iamerrors.CodeServiceCredentialInvalid, iamerrors.CodeServiceCredentialInvalid)
	}
	body := struct {
		CapabilityID      string `json:"capability_id"`
		PreferredProtocol string `json:"preferred_protocol"`
		Payload           any    `json:"payload"`
	}{contracts.IdentityContextCapability, "core_internal", map[string]any{"body": map[string]any{"operation": operation, "request": input}}}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, iamerrors.New(iamerrors.CodeContextInvalidArgument, iamerrors.CodeContextInvalidArgument)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/tenant/invocations", bytes.NewReader(raw))
	if err != nil {
		return nil, iamerrors.New(iamerrors.CodeContextUnavailable, iamerrors.CodeContextUnavailable)
	}
	req.Header.Set("Authorization", c.authScheme+" "+credential)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if m, ok := ctx.Value(identityMetadataKey{}).(RequestMetadata); ok {
		req.Header.Set("X-Request-ID", m.RequestID)
		req.Header.Set("X-Trace-ID", m.TraceID)
	}
	// Do not route this payload through a transport which logs request bodies.
	client := *c.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, iamerrors.New(iamerrors.CodeContextUnavailable, iamerrors.CodeContextUnavailable)
	}
	defer response.Body.Close()
	raw, err = io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return nil, iamerrors.New(iamerrors.CodeIdentityResponseInvalid, iamerrors.CodeIdentityResponseInvalid)
	}
	if response.StatusCode != http.StatusOK {
		return nil, identityError(response, raw)
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Result struct {
				Result json.RawMessage `json:"result"`
			} `json:"result"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Code != 200 {
		return nil, iamerrors.New(iamerrors.CodeIdentityResponseInvalid, iamerrors.CodeIdentityResponseInvalid)
	}
	return envelope.Data.Result.Result, nil
}

func identityError(response *http.Response, raw []byte) *IdentityRequestError {
	var e struct {
		Code            json.RawMessage `json:"code"`
		ErrorCode       string          `json:"error_code"`
		ReasonCode      string          `json:"reason_code"`
		TraceID         string          `json:"trace_id"`
		RequestID       string          `json:"request_id"`
		SourceTraceID   string          `json:"source_trace_id"`
		SourceRequestID string          `json:"source_request_id"`
		Details         struct {
			Code       string `json:"code"`
			ReasonCode string `json:"reason_code"`
		} `json:"details"`
		Error struct {
			Code       string `json:"code"`
			ReasonCode string `json:"reason_code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &e)
	var code string
	_ = json.Unmarshal(e.Code, &code)
	if e.ErrorCode != "" {
		code = e.ErrorCode
	}
	if e.Error.Code != "" {
		code = e.Error.Code
	}
	if e.Details.Code != "" {
		code = e.Details.Code
	}
	reason := e.ReasonCode
	if e.Error.ReasonCode != "" {
		reason = e.Error.ReasonCode
	}
	if e.Details.ReasonCode != "" {
		reason = e.Details.ReasonCode
	}
	if code == "" && reason == "" {
		code = iamerrors.CodeContextUnavailable
	}
	trace := response.Header.Get("X-Trace-ID")
	if trace == "" {
		trace = e.TraceID
	}
	request := response.Header.Get("X-Request-ID")
	if request == "" {
		request = e.RequestID
	}
	return &IdentityRequestError{Status: response.StatusCode, Code: code, ReasonCode: reason, TraceID: trace, RequestID: request, SourceTraceID: e.SourceTraceID, SourceRequestID: e.SourceRequestID}
}
