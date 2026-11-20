package storagedataset

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/google/uuid"
)

type invoker interface {
	Invoke(context.Context, gateway.InvokeRequest) (*gateway.Response, error)
}

// Client uses the plugin's Host Gateway with a tenant-scoped STS token provider.
type Client struct {
	gateway invoker
	tenant  string
}

func NewClient(host *gateway.Client, tenant string) (*Client, error) {
	if host == nil || !canonicalUUID(tenant) {
		return nil, ErrScope
	}
	return &Client{gateway: host, tenant: tenant}, nil
}

func canonicalUUID(s string) bool {
	id, err := uuid.Parse(s)
	return err == nil && id != uuid.Nil && id.String() == s
}

func (c *Client) invoke(ctx context.Context, capability, version, operation string, input any, def string, out any) error {
	body := struct {
		Version   string `json:"contract_version"`
		Operation string `json:"operation"`
		Input     any    `json:"input"`
	}{version, operation, input}
	raw, err := json.Marshal(body)
	if err != nil || decodeContract(raw, version, "", nil) != nil {
		return ErrContract
	}
	res, err := c.gateway.Invoke(ctx, gateway.InvokeRequest{
		CapabilityID: capability, PreferredProtocol: "core_internal",
		Payload: map[string]any{"body": body},
	})
	if err != nil {
		return err
	}
	if res == nil {
		return ErrContract
	}
	var data struct {
		Protocol string          `json:"protocol_used"`
		Fallback *bool           `json:"fallback_used"`
		Status   string          `json:"status"`
		Result   json.RawMessage `json:"result"`
	}
	if json.Unmarshal(res.RawData, &data) != nil || data.Protocol != "core_internal" || data.Fallback == nil || *data.Fallback || data.Status != "completed" || bytes.Equal(data.Result, []byte("null")) {
		return ErrContract
	}
	if capability == RegisterCapability {
		// The registration invoker wraps its receipt in result before the
		// invocation transport adds the common envelope.
		var receipt struct {
			Result json.RawMessage `json:"result"`
		}
		if json.Unmarshal(data.Result, &receipt) != nil {
			return ErrContract
		}
		data.Result = receipt.Result
	}
	return decodeContract(data.Result, version, def, out)
}

func (c *Client) Register(ctx context.Context, requestUUID string, asset Asset) (Registration, error) {
	var result Registration
	input := struct {
		Request    string `json:"request_uuid"`
		Key        string `json:"dataset_key"`
		Version    int64  `json:"definition_version"`
		Digest     string `json:"definition_digest"`
		BindingRef string `json:"source_binding_ref"`
	}{requestUUID, asset.Key(), asset.declaration.Version, asset.digest, asset.BindingRef()}
	err := c.invoke(ctx, RegisterCapability, HostVersion, "register", input, "RegisterResult", &result)
	if err == nil && requestUUID != result.RequestUUID {
		err = ErrScope
	}
	return result, err
}

func (c *Client) ClaimPreview(ctx context.Context, claim Claim) (Scope, error) {
	var result Scope
	err := c.invoke(ctx, PreviewCapability, HostVersion, "claim_preview", claim, "ClaimPreviewResult", &result)
	if err == nil {
		err = c.checkScope(claim.RequestUUID, result)
	}
	return result, err
}

func (c *Client) ClaimExecution(ctx context.Context, claim Claim) (ExecutionClaim, error) {
	var result ExecutionClaim
	err := c.invoke(ctx, ExecutionCapability, ExecutionVersion, "claim_execution", claim, "ClaimResult", &result)
	if err != nil {
		return ExecutionClaim{}, err
	}
	if result.Grant == nil {
		err = c.checkReconciliation(claim.RequestUUID, result)
	} else {
		err = c.checkGrant(claim.RequestUUID, *result.Grant)
	}
	if err != nil {
		return ExecutionClaim{}, err
	}
	return result, nil
}

// ReconcileExecution looks up the original expired authorization without
// acquiring a TaskBus lease or opening a new source operation.
func (c *Client) ReconcileExecution(ctx context.Context, requestUUID string) (ExecutionClaim, error) {
	var result ExecutionClaim
	input := struct {
		RequestUUID string `json:"request_uuid"`
	}{requestUUID}
	err := c.invoke(ctx, ExecutionCapability, ExecutionVersion, "reconcile_execution", input, "ReconcileResult", &result)
	if err == nil {
		err = c.checkReconciliation(requestUUID, result)
	}
	if err != nil {
		return ExecutionClaim{}, err
	}
	return result, nil
}

func (c *Client) checkReconciliation(requestUUID string, result ExecutionClaim) error {
	if result.RequestUUID != requestUUID {
		return ErrScope
	}
	if result.ExpiredGrant != nil {
		if result.ExpiredGrant.GrantUUID != result.GrantUUID {
			return ErrScope
		}
		return c.checkGrant(requestUUID, *result.ExpiredGrant)
	}
	return nil
}

func (c *Client) checkGrant(requestUUID string, grant Grant) error {
	if err := c.checkScope(requestUUID, grant.Scope); err != nil {
		return err
	}
	if !grant.IssuedAt.Before(grant.Deadline) || grant.Deadline.Sub(grant.IssuedAt) > 15*time.Second {
		return ErrContract
	}
	return nil
}

func (c *Client) checkScope(requestUUID string, scope Scope) error {
	if scope.TenantUUID != c.tenant || scope.RequestUUID != requestUUID {
		return ErrScope
	}
	if !scope.Cutoff.Before(scope.Deadline) {
		return ErrContract
	}
	return nil
}

func (c *Client) ReportPreview(ctx context.Context, report PreviewReport) error {
	return c.report(ctx, PreviewCapability, HostVersion, "report_preview", report, "ReportPreviewResult", report.RequestUUID, report.Outcome)
}

func (c *Client) ReportExecution(ctx context.Context, report Report) error {
	return c.report(ctx, ExecutionCapability, ExecutionVersion, "report_execution", report, "ReportResult", report.RequestUUID, report.Outcome)
}

func (c *Client) ArchiveExecution(ctx context.Context, report Report, data []byte) error {
	if err := validateArchive(data, report.Affected); err != nil {
		return err
	}
	input := struct {
		Report Report `json:"report"`
		Data   []byte `json:"archive_base64"`
	}{report, data}
	return c.report(ctx, ExecutionCapability, ExecutionVersion, "archive_execution", input, "ReportResult", report.RequestUUID, report.Outcome)
}

func (c *Client) report(ctx context.Context, capability, version, operation string, in any, def, request, outcome string) error {
	var result Result
	if err := c.invoke(ctx, capability, version, operation, in, def, &result); err != nil {
		return err
	}
	if result.RequestUUID != request || result.State != outcome {
		return ErrScope
	}
	return nil
}
