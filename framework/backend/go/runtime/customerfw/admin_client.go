package customerfw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

const defaultAdminPageSize = 20

type adminRESTPayload struct {
	Method   string         `json:"method"`
	Endpoint string         `json:"endpoint"`
	Query    map[string]any `json:"query,omitempty"`
	Body     any            `json:"body,omitempty"`
}

func NewAdminClient(cfg AdminClientConfig) (*AdminClient, error) {
	if cfg.Invoker == nil {
		return nil, errors.New("customer admin: gateway invoker is required")
	}
	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = defaultAdminPageSize
	}
	return &AdminClient{invoker: cfg.Invoker, tenantUUID: strings.TrimSpace(cfg.TenantUUID), pageSize: pageSize}, nil
}

func (c *AdminClient) Overview(ctx context.Context, req ListAccountsRequest) (CustomerOverview, error) {
	query := c.pageQuery(req)
	var out CustomerOverview
	if err := c.invoke(ctx, http.MethodGet, "/api/v1/admin/customers/overview", query, nil, req.RequestID, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = CustomerOverview{}
	}
	return out, nil
}

func (c *AdminClient) ListAccounts(ctx context.Context, req ListAccountsRequest) (*AccountPage, error) {
	query := c.pageQuery(req)
	var out AccountPage
	if err := c.invoke(ctx, http.MethodGet, "/api/v1/admin/customers/accounts", query, nil, req.RequestID, &out); err != nil {
		return nil, err
	}
	if out.Items == nil {
		out.Items = []Account{}
	}
	if out.Page <= 0 {
		out.Page = 1
	}
	if out.PageSize <= 0 {
		out.PageSize = c.pageSize
	}
	return &out, nil
}

func (c *AdminClient) CreateAccount(ctx context.Context, req CreateAccountRequest) (*Account, error) {
	body, requestID := createAccountBody(req)
	var out Account
	if err := c.invoke(ctx, http.MethodPost, "/api/v1/admin/customers/accounts", nil, body, requestID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AdminClient) UpdateAccount(ctx context.Context, req UpdateAccountRequest) (*Account, error) {
	customerUUID := strings.TrimSpace(req.CustomerUUID)
	if customerUUID == "" {
		return nil, errors.New("customer admin: customer_uuid is required")
	}
	body, requestID := updateAccountBody(req)
	var out Account
	if err := c.invoke(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/admin/customers/accounts/%s", customerUUID), nil, body, requestID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AdminClient) UpdateStatus(ctx context.Context, customerUUID, status, requestID string) (*Account, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		return nil, errors.New("customer admin: status is required")
	}
	return c.UpdateAccount(ctx, UpdateAccountRequest{CustomerUUID: customerUUID, Status: &status, RequestID: requestID})
}

func (c *AdminClient) invoke(ctx context.Context, method, endpoint string, query map[string]any, body any, requestID string, out any) error {
	if c == nil || c.invoker == nil {
		return errors.New("customer admin: gateway invoker is required")
	}
	resp, err := c.invoker.Invoke(ctx, gateway.InvokeRequest{
		CapabilityID:      CapabilityCustomerAccountsAdminManage,
		PreferredProtocol: "rest",
		RequestID:         strings.TrimSpace(requestID),
		TenantUUID:        c.tenantUUID,
		Payload: adminRESTPayload{
			Method:   method,
			Endpoint: endpoint,
			Query:    cleanAdminQuery(query),
			Body:     body,
		},
	})
	if err != nil {
		return err
	}
	if resp == nil || resp.Data == nil {
		return errors.New("customer admin: empty gateway response")
	}
	payload, ok := resp.Data["payload"]
	if !ok {
		return errors.New("customer admin: response payload is missing")
	}
	if out == nil {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func (c *AdminClient) pageQuery(req ListAccountsRequest) map[string]any {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	size := req.PageSize
	if size <= 0 {
		size = c.pageSize
	}
	query := map[string]any{"page": page, "page_size": size}
	setAdminQuery(query, "tenant_uuid", req.TenantUUID)
	setAdminQuery(query, "q", req.Query)
	setAdminQuery(query, "status", req.Status)
	return query
}

func createAccountBody(req CreateAccountRequest) (map[string]any, string) {
	return map[string]any{
		"tenant_uuid":  strings.TrimSpace(req.TenantUUID),
		"email":        strings.TrimSpace(req.Email),
		"phone":        strings.TrimSpace(req.Phone),
		"password":     req.Password,
		"display_name": strings.TrimSpace(req.DisplayName),
		"nickname":     strings.TrimSpace(req.Nickname),
		"given_name":   strings.TrimSpace(req.GivenName),
		"family_name":  strings.TrimSpace(req.FamilyName),
		"avatar_url":   strings.TrimSpace(req.AvatarURL),
		"locale":       strings.TrimSpace(req.Locale),
		"timezone":     strings.TrimSpace(req.Timezone),
		"metadata":     req.Metadata,
	}, req.RequestID
}

func updateAccountBody(req UpdateAccountRequest) (map[string]any, string) {
	return map[string]any{
		"primary_email":  req.PrimaryEmail,
		"primary_phone":  req.PrimaryPhone,
		"email":          req.Email,
		"phone":          req.Phone,
		"display_name":   req.DisplayName,
		"nickname":       req.Nickname,
		"given_name":     req.GivenName,
		"family_name":    req.FamilyName,
		"avatar_url":     req.AvatarURL,
		"locale":         req.Locale,
		"timezone":       req.Timezone,
		"status":         req.Status,
		"email_verified": req.EmailVerified,
		"phone_verified": req.PhoneVerified,
		"metadata":       req.Metadata,
	}, req.RequestID
}

func setAdminQuery(query map[string]any, key, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		query[key] = value
	}
}

func cleanAdminQuery(query map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range query {
		if value != nil {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
