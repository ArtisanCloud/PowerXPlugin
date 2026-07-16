package customerfw

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

const CapabilityCustomerAccountsAdminManage = "com.corex.customer.accounts.admin_manage"

type AdminGatewayInvoker interface {
	Invoke(ctx context.Context, req gateway.InvokeRequest) (*gateway.Response, error)
}

type AdminClientConfig struct {
	Invoker    AdminGatewayInvoker
	TenantUUID string
	PageSize   int
}

type AdminClient struct {
	invoker    AdminGatewayInvoker
	tenantUUID string
	pageSize   int
}

type CustomerOverview map[string]any

type Account struct {
	ID            uint64         `json:"id,omitempty"`
	CustomerUUID  string         `json:"customer_uuid"`
	TenantUUID    string         `json:"tenant_uuid,omitempty"`
	PrimaryEmail  string         `json:"primary_email,omitempty"`
	PrimaryPhone  string         `json:"primary_phone,omitempty"`
	Email         string         `json:"email,omitempty"`
	Phone         string         `json:"phone,omitempty"`
	DisplayName   string         `json:"display_name,omitempty"`
	Nickname      string         `json:"nickname,omitempty"`
	GivenName     string         `json:"given_name,omitempty"`
	FamilyName    string         `json:"family_name,omitempty"`
	AvatarURL     string         `json:"avatar_url,omitempty"`
	Locale        string         `json:"locale,omitempty"`
	Timezone      string         `json:"timezone,omitempty"`
	Status        string         `json:"status,omitempty"`
	EmailVerified bool           `json:"email_verified,omitempty"`
	PhoneVerified bool           `json:"phone_verified,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     string         `json:"created_at,omitempty"`
	UpdatedAt     string         `json:"updated_at,omitempty"`
}

type AccountPage struct {
	Items      []Account `json:"items"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	Total      int64     `json:"total"`
	TotalPages int       `json:"total_pages,omitempty"`
}

type ListAccountsRequest struct {
	TenantUUID string `json:"tenant_uuid,omitempty"`
	Query      string `json:"q,omitempty"`
	Status     string `json:"status,omitempty"`
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
}

type CreateAccountRequest struct {
	TenantUUID  string         `json:"tenant_uuid,omitempty"`
	Email       string         `json:"email,omitempty"`
	Phone       string         `json:"phone,omitempty"`
	Password    string         `json:"password,omitempty"`
	DisplayName string         `json:"display_name,omitempty"`
	Nickname    string         `json:"nickname,omitempty"`
	GivenName   string         `json:"given_name,omitempty"`
	FamilyName  string         `json:"family_name,omitempty"`
	AvatarURL   string         `json:"avatar_url,omitempty"`
	Locale      string         `json:"locale,omitempty"`
	Timezone    string         `json:"timezone,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	RequestID   string         `json:"request_id,omitempty"`
}

type UpdateAccountRequest struct {
	CustomerUUID  string         `json:"customer_uuid,omitempty"`
	PrimaryEmail  *string        `json:"primary_email,omitempty"`
	PrimaryPhone  *string        `json:"primary_phone,omitempty"`
	Email         *string        `json:"email,omitempty"`
	Phone         *string        `json:"phone,omitempty"`
	DisplayName   *string        `json:"display_name,omitempty"`
	Nickname      *string        `json:"nickname,omitempty"`
	GivenName     *string        `json:"given_name,omitempty"`
	FamilyName    *string        `json:"family_name,omitempty"`
	AvatarURL     *string        `json:"avatar_url,omitempty"`
	Locale        *string        `json:"locale,omitempty"`
	Timezone      *string        `json:"timezone,omitempty"`
	Status        *string        `json:"status,omitempty"`
	EmailVerified *bool          `json:"email_verified,omitempty"`
	PhoneVerified *bool          `json:"phone_verified,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	RequestID     string         `json:"request_id,omitempty"`
}
