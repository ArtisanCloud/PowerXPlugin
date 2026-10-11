package customerfw

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/google/uuid"
)

// UpdateBasicAccountRequest preserves omission separately from explicit clearing.
// Tenant and actor are derived by the Host from the outbound credential.
type UpdateBasicAccountRequest struct {
	CustomerUUID string  `json:"-"`
	DisplayName  *string `json:"display_name,omitempty"`
	Nickname     *string `json:"nickname,omitempty"`
	GivenName    *string `json:"given_name,omitempty"`
	FamilyName   *string `json:"family_name,omitempty"`
	PrimaryEmail *string `json:"primary_email,omitempty"`
	PrimaryPhone *string `json:"primary_phone,omitempty"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	Locale       *string `json:"locale,omitempty"`
	Timezone     *string `json:"timezone,omitempty"`
	Status       *string `json:"status,omitempty"`
	RequestID    string  `json:"-"`
}

// UnmarshalJSON rejects unknown fields, aliases and null instead of losing presence.
func (r *UpdateBasicAccountRequest) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	allowed := map[string]bool{"display_name": true, "nickname": true, "given_name": true, "family_name": true, "primary_email": true, "primary_phone": true, "avatar_url": true, "locale": true, "timezone": true, "status": true}
	for key, value := range fields {
		if !allowed[key] || string(value) == "null" {
			return errors.New("CUSTOMER_ACCOUNT_INVALID_ARGUMENT")
		}
		var text string
		if err := json.Unmarshal(value, &text); err != nil {
			return errors.New("CUSTOMER_ACCOUNT_INVALID_ARGUMENT")
		}
	}
	type wire UpdateBasicAccountRequest
	return json.Unmarshal(data, (*wire)(r))
}

func (c *AccountSelectorClient) UpdateBasicAccount(ctx context.Context, req UpdateBasicAccountRequest) (*Account, error) {
	if c == nil || c.invoker == nil {
		return nil, errors.New("CUSTOMER_ACCOUNT_SELECTOR_GATEWAY_REQUIRED")
	}
	id, err := uuid.Parse(strings.TrimSpace(req.CustomerUUID))
	if err != nil || id == uuid.Nil {
		return nil, errors.New("CUSTOMER_ACCOUNT_INVALID_ARGUMENT")
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	body := map[string]any{}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, errors.New("CUSTOMER_ACCOUNT_INVALID_ARGUMENT")
	}
	body["operation"], body["customer_uuid"] = "update", id.String()
	resp, err := c.invoker.Invoke(ctx, gateway.InvokeRequest{CapabilityID: CapabilityCustomerAccountsServiceManage, PreferredProtocol: "core_internal", RequestID: strings.TrimSpace(req.RequestID), Payload: map[string]any{"method": "INVOKE", "endpoint": customerAccountsCoreEndpoint, "body": body}})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Data == nil {
		return nil, errors.New("CUSTOMER_ACCOUNT_UPDATE_EMPTY_RESPONSE")
	}
	payload, ok := resp.Data["payload"]
	if !ok {
		return nil, errors.New("CUSTOMER_ACCOUNT_UPDATE_PAYLOAD_MISSING")
	}
	raw, err = json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var wire struct {
		Item struct {
			Account
			UUID string `json:"uuid"`
		} `json:"item"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return nil, err
	}
	if wire.Item.UUID != id.String() || (wire.Item.Type != "person" && wire.Item.Type != "company") {
		return nil, errors.New("CUSTOMER_ACCOUNT_UPDATE_INCOMPLETE_RESPONSE")
	}
	primary, err := uuid.Parse(wire.Item.PrimaryContactUUID)
	if err != nil || primary == uuid.Nil {
		return nil, errors.New("CUSTOMER_ACCOUNT_UPDATE_INCOMPLETE_RESPONSE")
	}
	wire.Item.Account.CustomerUUID = wire.Item.UUID
	return &wire.Item.Account, nil
}
