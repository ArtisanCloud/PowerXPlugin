package delegated

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

// AuthorizationBaselineRequestError keeps the upstream status and diagnostic
// identifiers, including for an HTTP-200 response with an invalid contract.
type AuthorizationBaselineRequestError struct{ IdentityRequestError }

func (e *AuthorizationBaselineRequestError) HTTPStatus() int {
	if e.Status == http.StatusOK {
		return http.StatusBadGateway
	}
	return e.Status
}

func baselineResponseError(response *http.Response, raw []byte, invalid bool) error {
	diagnostic := identityError(response, raw)
	if invalid {
		diagnostic.Code = iamerrors.CodeBaselineResponseInvalid
		diagnostic.ReasonCode = ""
	} else if diagnostic.Code == iamerrors.CodeContextUnavailable {
		diagnostic.Code = iamerrors.CodeBaselineUnavailable
	}
	return &AuthorizationBaselineRequestError{IdentityRequestError: *diagnostic}
}

func (c *CoreClient) ReadMemberBaselines(ctx context.Context, query contracts.AuthorizationBaselineQuery) (*contracts.AuthorizationBaselineResult, error) {
	q, err := query.Normalize()
	if err != nil {
		return nil, err
	}
	if c == nil || c.tokens == nil || c.client == nil || c.pluginID == "" {
		return nil, iamerrors.New(iamerrors.CodeAdapterNotBound, iamerrors.CodeAdapterNotBound)
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	credential, err := c.tokens.Token(ctx)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil || credential == "" {
		return nil, iamerrors.New(iamerrors.CodeServiceCredentialInvalid, iamerrors.CodeServiceCredentialInvalid)
	}
	requestBody := struct {
		Schema          string   `json:"schema"`
		MemberUUIDs     []string `json:"member_uuids"`
		PermissionCodes []string `json:"permission_codes"`
	}{contracts.AuthorizationBaselineQuerySchema, q.MemberUUIDs, q.PermissionCodes}
	raw, err := json.Marshal(struct {
		CapabilityID      string `json:"capability_id"`
		PreferredProtocol string `json:"preferred_protocol"`
		Payload           any    `json:"payload"`
	}{contracts.AuthorizationBaselineCapability, "core_internal", map[string]any{"body": map[string]any{"operation": contracts.AuthorizationBaselineOperation, "request": requestBody}}})
	if err != nil {
		return nil, iamerrors.New(iamerrors.CodeBaselineInvalidArgument, iamerrors.CodeBaselineInvalidArgument)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/tenant/invocations", bytes.NewReader(raw))
	if err != nil {
		return nil, iamerrors.New(iamerrors.CodeBaselineUnavailable, iamerrors.CodeBaselineUnavailable)
	}
	request.Header.Set("Authorization", c.authScheme+" "+credential)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if metadata, ok := ctx.Value(identityMetadataKey{}).(RequestMetadata); ok {
		request.Header.Set("X-Request-ID", metadata.RequestID)
		request.Header.Set("X-Trace-ID", metadata.TraceID)
	}
	client := *c.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, iamerrors.New(iamerrors.CodeBaselineUnavailable, iamerrors.CodeBaselineUnavailable)
	}
	defer response.Body.Close()
	raw, err = io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil || len(raw) > 1<<20 {
		return nil, baselineResponseError(response, nil, true)
	}
	if response.StatusCode != http.StatusOK {
		return nil, baselineResponseError(response, raw, false)
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Result struct {
				Result json.RawMessage `json:"result"`
			} `json:"result"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Code != http.StatusOK {
		return nil, baselineResponseError(response, raw, true)
	}
	result, err := contracts.DecodeAuthorizationBaselineResult(envelope.Data.Result.Result, q, c.pluginID)
	if err != nil {
		return nil, baselineResponseError(response, raw, true)
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return result, nil
}

var _ contracts.AuthorizationBaselineService = (*CoreClient)(nil)
