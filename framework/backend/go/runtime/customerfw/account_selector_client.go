package customerfw

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/google/uuid"
)

const CapabilityCustomerAccountsServiceRead = "com.corex.customer.accounts.service_read"
const CapabilityCustomerAccountsServiceManage = "com.corex.customer.accounts.service_manage"
const customerAccountsCoreEndpoint = "core://customer/accounts"

// AccountSelectorClient uses the fixed, service-scoped Core account selector.
// Admin REST management remains on AdminClient.
type AccountSelectorClient struct{ invoker AdminGatewayInvoker }

type accountSelectorPageWire struct {
	Items []struct {
		UUID               string `json:"uuid"`
		Type               string `json:"type"`
		PrimaryContactUUID string `json:"primary_contact_uuid"`
		Status             string `json:"status"`
		PrimaryEmail       string `json:"primary_email"`
		PrimaryPhone       string `json:"primary_phone"`
		DisplayName        string `json:"display_name"`
		Nickname           string `json:"nickname"`
		GivenName          string `json:"given_name"`
		FamilyName         string `json:"family_name"`
		AvatarURL          string `json:"avatar_url"`
		Locale             string `json:"locale"`
		Timezone           string `json:"timezone"`
		CreatedAt          string `json:"created_at"`
		UpdatedAt          string `json:"updated_at"`
	} `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type CreateBasicAccountRequest struct {
	Type           string               `json:"type"`
	PrimaryContact *PrimaryContactInput `json:"primary_contact,omitempty"`
	Status         string               `json:"status,omitempty"`
	PrimaryEmail   string               `json:"primary_email,omitempty"`
	PrimaryPhone   string               `json:"primary_phone,omitempty"`
	DisplayName    string               `json:"display_name,omitempty"`
	Nickname       string               `json:"nickname,omitempty"`
	GivenName      string               `json:"given_name,omitempty"`
	FamilyName     string               `json:"family_name,omitempty"`
	AvatarURL      string               `json:"avatar_url,omitempty"`
	Locale         string               `json:"locale,omitempty"`
	Timezone       string               `json:"timezone,omitempty"`
	RequestID      string               `json:"-"`
}

type PrimaryContactInput struct {
	DisplayName string `json:"display_name"`
	GivenName   string `json:"given_name,omitempty"`
	FamilyName  string `json:"family_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
}

func (c *AccountSelectorClient) CreateBasicAccount(ctx context.Context, req CreateBasicAccountRequest) (*Account, error) {
	if c == nil || c.invoker == nil {
		return nil, errors.New("CUSTOMER_ACCOUNT_SELECTOR_GATEWAY_REQUIRED")
	}
	req = NormalizeBasicAccountLabels(req)
	req.Type = strings.TrimSpace(req.Type)
	if req.Type == "" {
		req.Type = "person"
	}
	if req.Type != "person" && req.Type != "company" {
		return nil, errors.New("CUSTOMER_TYPE_REQUIRED")
	}
	if req.Type == "company" && (req.PrimaryContact == nil || strings.TrimSpace(req.PrimaryContact.DisplayName) == "") {
		return nil, errors.New("CUSTOMER_PRIMARY_CONTACT_REQUIRED")
	}
	body := map[string]any{"operation": "create", "type": req.Type, "status": req.Status, "primary_email": req.PrimaryEmail, "primary_phone": req.PrimaryPhone, "display_name": req.DisplayName, "nickname": req.Nickname, "given_name": req.GivenName, "family_name": req.FamilyName, "avatar_url": req.AvatarURL, "locale": req.Locale, "timezone": req.Timezone}
	if req.PrimaryContact != nil {
		body["primary_contact"] = req.PrimaryContact
	}
	resp, err := c.invoker.Invoke(ctx, gateway.InvokeRequest{CapabilityID: CapabilityCustomerAccountsServiceManage, PreferredProtocol: "core_internal", RequestID: strings.TrimSpace(req.RequestID), Payload: map[string]any{"method": "INVOKE", "endpoint": customerAccountsCoreEndpoint, "body": body}})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Data == nil {
		return nil, errors.New("CUSTOMER_ACCOUNT_CREATE_EMPTY_RESPONSE")
	}
	payload, ok := resp.Data["payload"]
	if !ok {
		return nil, errors.New("CUSTOMER_ACCOUNT_CREATE_PAYLOAD_MISSING")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var wire struct {
		Item struct {
			UUID               string `json:"uuid"`
			Type               string `json:"type"`
			PrimaryContactUUID string `json:"primary_contact_uuid"`
			Status             string `json:"status"`
			PrimaryEmail       string `json:"primary_email"`
			PrimaryPhone       string `json:"primary_phone"`
			DisplayName        string `json:"display_name"`
			Nickname           string `json:"nickname"`
			GivenName          string `json:"given_name"`
			FamilyName         string `json:"family_name"`
			AvatarURL          string `json:"avatar_url"`
			Locale             string `json:"locale"`
			Timezone           string `json:"timezone"`
		} `json:"item"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(wire.Item.UUID)); err != nil {
		return nil, errors.New("CUSTOMER_ACCOUNT_CREATE_UUID_REQUIRED")
	}
	if wire.Item.Type != req.Type || strings.TrimSpace(wire.Item.PrimaryContactUUID) == "" {
		return nil, errors.New("CUSTOMER_ACCOUNT_CREATE_INCOMPLETE_RESPONSE")
	}
	return &Account{CustomerUUID: wire.Item.UUID, Type: wire.Item.Type, PrimaryContactUUID: wire.Item.PrimaryContactUUID, Status: wire.Item.Status, PrimaryEmail: wire.Item.PrimaryEmail, PrimaryPhone: wire.Item.PrimaryPhone, DisplayName: wire.Item.DisplayName, Nickname: wire.Item.Nickname, GivenName: wire.Item.GivenName, FamilyName: wire.Item.FamilyName, AvatarURL: wire.Item.AvatarURL, Locale: wire.Item.Locale, Timezone: wire.Item.Timezone}, nil
}

func NewAccountSelectorClient(invoker AdminGatewayInvoker) (*AccountSelectorClient, error) {
	if invoker == nil {
		return nil, errors.New("CUSTOMER_ACCOUNT_SELECTOR_GATEWAY_REQUIRED")
	}
	return &AccountSelectorClient{invoker: invoker}, nil
}

func (c *AccountSelectorClient) ListAccounts(ctx context.Context, req ListAccountsRequest) (*AccountPage, error) {
	if c == nil || c.invoker == nil {
		return nil, errors.New("CUSTOMER_ACCOUNT_SELECTOR_GATEWAY_REQUIRED")
	}
	page, size := req.Page, req.PageSize
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = defaultAdminPageSize
	}
	body := map[string]any{"operation": "list", "page": page, "page_size": size}
	if q := strings.TrimSpace(req.Query); q != "" {
		body["q"] = q
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		body["status"] = status
	}
	resp, err := c.invoker.Invoke(ctx, gateway.InvokeRequest{
		CapabilityID:      CapabilityCustomerAccountsServiceRead,
		PreferredProtocol: "core_internal",
		RequestID:         strings.TrimSpace(req.RequestID),
		Payload: map[string]any{
			"method": "INVOKE", "endpoint": customerAccountsCoreEndpoint, "body": body,
		},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Data == nil {
		return nil, errors.New("CUSTOMER_ACCOUNT_SELECTOR_EMPTY_RESPONSE")
	}
	payload, ok := resp.Data["payload"]
	if !ok {
		return nil, errors.New("CUSTOMER_ACCOUNT_SELECTOR_PAYLOAD_MISSING")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var wire accountSelectorPageWire
	if err := json.Unmarshal(raw, &wire); err != nil {
		return nil, err
	}
	out := &AccountPage{Items: make([]Account, 0, len(wire.Items)), Page: wire.Page, PageSize: wire.PageSize, Total: wire.Total}
	for _, item := range wire.Items {
		if _, err := uuid.Parse(strings.TrimSpace(item.UUID)); err != nil {
			return nil, errors.New("CUSTOMER_ACCOUNT_SELECTOR_UUID_REQUIRED")
		}
		out.Items = append(out.Items, Account{
			CustomerUUID: item.UUID, Type: item.Type, PrimaryContactUUID: item.PrimaryContactUUID, Status: item.Status,
			PrimaryEmail: item.PrimaryEmail, PrimaryPhone: item.PrimaryPhone,
			DisplayName: item.DisplayName, Nickname: item.Nickname,
			GivenName: item.GivenName, FamilyName: item.FamilyName,
			AvatarURL: item.AvatarURL, Locale: item.Locale, Timezone: item.Timezone,
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	if out.Page <= 0 {
		out.Page = page
	}
	if out.PageSize <= 0 {
		out.PageSize = size
	}
	if out.PageSize > 0 {
		out.TotalPages = int((out.Total + int64(out.PageSize) - 1) / int64(out.PageSize))
	}
	return out, nil
}

// NormalizeBasicAccountLabels sets display labels without inventing real names.
// Shared by local creation and delegated requests. It never changes given/family names.
func NormalizeBasicAccountLabels(req CreateBasicAccountRequest) CreateBasicAccountRequest {
	if strings.TrimSpace(req.Type) == "" {
		req.Type = "person"
	}
	label := func(name, nick, given, family, email string) string {
		for _, value := range []string{name, nick, strings.TrimSpace(given) + " " + strings.TrimSpace(family), email} {
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		}
		return ""
	}
	req.DisplayName = label(req.DisplayName, req.Nickname, req.GivenName, req.FamilyName, req.PrimaryEmail)
	if req.PrimaryContact != nil {
		p := *req.PrimaryContact
		p.DisplayName = label(p.DisplayName, "", p.GivenName, p.FamilyName, p.Email)
		req.PrimaryContact = &p
	}
	return req
}
