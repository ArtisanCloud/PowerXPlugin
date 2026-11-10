package delegated

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

// AuthorizationRequestError preserves machine diagnostics without retaining
// upstream free text or bodies which could echo a service credential.
type AuthorizationRequestError struct{ IdentityRequestError }

func (e *AuthorizationRequestError) HTTPStatus() int {
	if e.Status >= 200 && e.Status < 300 {
		return http.StatusBadGateway
	}
	if e.Status == 0 {
		return http.StatusServiceUnavailable
	}
	return e.Status
}

func authorizationError(resp *http.Response, raw []byte, invalid bool, credential string) *AuthorizationRequestError {
	e := identityError(resp, raw)
	for _, value := range []*string{&e.Code, &e.ReasonCode} {
		if !authorizationMachineCode(*value) || (credential != "" && strings.Contains(*value, credential)) {
			*value = ""
		}
	}
	for _, value := range []*string{&e.TraceID, &e.RequestID, &e.SourceTraceID, &e.SourceRequestID} {
		if len(*value) > 256 || strings.ContainsAny(*value, "\r\n\x00") || (credential != "" && strings.Contains(*value, credential)) {
			*value = ""
		}
	}
	if invalid {
		e.Code = iamerrors.CodeAuthorizationResponseInvalid
		e.ReasonCode = ""
	} else if e.Code == "" && e.ReasonCode == "" || e.Code == iamerrors.CodeContextUnavailable && e.ReasonCode == "" {
		e.Code = iamerrors.CodeUpstreamDependency
	}
	return &AuthorizationRequestError{IdentityRequestError: *e}
}

func authorizationMachineCode(value string) bool {
	if len(value) < 3 || len(value) > 128 {
		return false
	}
	for i, ch := range value {
		if ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' {
			continue
		}
		if i > 0 && (ch >= '0' && ch <= '9' || ch == '_' || ch == '.' || ch == '-') {
			continue
		}
		return false
	}
	return true
}

func (c *CoreClient) Authorize(ctx context.Context, in contracts.AuthorizationRequest) (*contracts.AuthorizationDecision, error) {
	query, err := in.NormalizeForCore()
	if err != nil {
		return nil, err
	}
	if c == nil || c.tokens == nil || c.client == nil {
		return nil, iamerrors.New(iamerrors.CodeAdapterNotBound, iamerrors.CodeAdapterNotBound)
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	credential, err := c.tokens.Token(ctx)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil || strings.TrimSpace(credential) == "" {
		return nil, iamerrors.New(iamerrors.CodeServiceCredentialInvalid, iamerrors.CodeServiceCredentialInvalid)
	}
	credential = strings.TrimSpace(credential)
	body := struct {
		MemberUUID     string `json:"member_uuid"`
		UserUUID       string `json:"user_uuid"`
		PluginID       string `json:"plugin_id,omitempty"`
		PermissionCode string `json:"permission_code,omitempty"`
		Resource       string `json:"resource,omitempty"`
		Action         string `json:"action,omitempty"`
	}{query.MemberUUID, query.UserUUID, query.PluginID, query.PermissionCode, query.Resource, query.Action}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, iamerrors.New(iamerrors.CodeInvalidArgument, iamerrors.CodeInvalidArgument)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/tenant/iam/authorization:check", bytes.NewReader(raw))
	if err != nil {
		return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, iamerrors.CodeUpstreamDependency)
	}
	request.Header.Set("Authorization", c.authScheme+" "+strings.TrimSpace(credential))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	metadata, _ := ctx.Value(identityMetadataKey{}).(RequestMetadata)
	request.Header.Set("X-Request-ID", metadata.RequestID)
	trace := metadata.TraceID
	if trace == "" {
		trace = query.TraceID
	}
	request.Header.Set("X-Trace-ID", trace)
	client := *c.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, &AuthorizationRequestError{IdentityRequestError: IdentityRequestError{Code: iamerrors.CodeUpstreamDependency}}
	}
	defer response.Body.Close()
	raw, err = io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil || len(raw) > 1<<20 {
		return nil, authorizationError(response, nil, true, credential)
	}
	if response.StatusCode != http.StatusOK {
		return nil, authorizationError(response, raw, response.StatusCode >= 200 && response.StatusCode < 300, credential)
	}
	var envelope struct {
		Code *int            `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Code == nil || *envelope.Code != http.StatusOK {
		return nil, authorizationError(response, raw, true, credential)
	}
	out, err := contracts.DecodeCoreAuthorizationDecision(envelope.Data, query)
	if err != nil {
		return nil, authorizationError(response, raw, true, credential)
	}
	if out.TraceID == "" {
		out.TraceID = response.Header.Get("X-Trace-ID")
	}
	if len(out.TraceID) > 256 || strings.ContainsAny(out.TraceID, "\r\n\x00") || strings.Contains(out.TraceID, credential) {
		return nil, authorizationError(response, raw, true, credential)
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return out, nil
}
