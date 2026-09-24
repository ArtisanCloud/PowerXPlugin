package contactfw

import "time"

const (
	CapabilityContactsAdminManage   = "com.corex.customer.contacts.admin_manage"
	CapabilityContactsServiceRead   = "com.corex.customer.contacts.service_read"
	CapabilityContactsServiceManage = "com.corex.customer.contacts.service_manage"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusInactive  Status = "inactive"
	StatusTemporary Status = "temporary"
)

type Role string

const (
	RolePrimary             Role = "primary"
	RoleLegalRepresentative Role = "legal_representative"
)

type IdentityStatus string

const (
	IdentityStatusActive   IdentityStatus = "active"
	IdentityStatusInactive IdentityStatus = "inactive"
)

type CreationIntent string

const (
	CreationIntentExplicitCreate    CreationIntent = "explicit_create"
	CreationIntentExplicitTemporary CreationIntent = "explicit_temporary"
)

// Contact is a tenant-scoped natural-person master record. TenantUUID is
// returned for diagnostics; implementations derive its request scope from a
// trusted context or credential and must not trust a caller-supplied tenant.
type Contact struct {
	ContactUUID  string         `json:"contact_uuid"`
	TenantUUID   string         `json:"tenant_uuid"`
	CustomerUUID string         `json:"customer_uuid"`
	DisplayName  string         `json:"display_name"`
	GivenName    string         `json:"given_name,omitempty"`
	FamilyName   string         `json:"family_name,omitempty"`
	Status       Status         `json:"status"`
	Roles        []Role         `json:"roles"`
	Tags         []string       `json:"tags"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type ContactIdentity struct {
	IdentityUUID              string         `json:"identity_uuid"`
	TenantUUID                string         `json:"tenant_uuid"`
	CustomerUUID              string         `json:"customer_uuid"`
	ContactUUID               string         `json:"contact_uuid"`
	ChannelDictionaryItemUUID string         `json:"channel_dictionary_item_uuid"`
	ExternalSubject           string         `json:"external_subject"`
	Status                    IdentityStatus `json:"status"`
	VerifiedAt                *time.Time     `json:"verified_at,omitempty"`
	Metadata                  map[string]any `json:"metadata,omitempty"`
}

// CreateContactInput carries only Contact business data. Tenant, actor,
// plugin, request, and trace dimensions are sourced by the typed Core
// invocation envelope or trusted local context, never by this DTO.
type CreateContactInput struct {
	CustomerUUID   string         `json:"customer_uuid"`
	DisplayName    string         `json:"display_name"`
	GivenName      string         `json:"given_name,omitempty"`
	FamilyName     string         `json:"family_name,omitempty"`
	Status         Status         `json:"status"`
	Roles          []Role         `json:"roles,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	CreationIntent CreationIntent `json:"creation_intent"`
}

type GetContactInput struct {
	CustomerUUID string `json:"customer_uuid"`
	ContactUUID  string `json:"contact_uuid"`
}

type UpdateContactInput struct {
	CustomerUUID string    `json:"customer_uuid"`
	ContactUUID  string    `json:"contact_uuid"`
	DisplayName  *string   `json:"display_name,omitempty"`
	GivenName    *string   `json:"given_name,omitempty"`
	FamilyName   *string   `json:"family_name,omitempty"`
	Status       *Status   `json:"status,omitempty"`
	Roles        *[]Role   `json:"roles,omitempty"`
	Tags         *[]string `json:"tags,omitempty"`
}

type ListByCustomerInput struct {
	CustomerUUID string  `json:"customer_uuid"`
	Query        string  `json:"q,omitempty"`
	Status       *Status `json:"status,omitempty"`
	Page         int     `json:"page,omitempty"`
	PageSize     int     `json:"page_size,omitempty"`
}

type ContactPage struct {
	Items    []Contact `json:"items"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	Total    int64     `json:"total"`
}

type ResolveContactIdentityInput struct {
	CustomerUUID              string `json:"customer_uuid"`
	ChannelDictionaryItemUUID string `json:"channel_dictionary_item_uuid"`
	ExternalSubject           string `json:"external_subject"`
}

type ContactIdentityResolution struct {
	Contact  Contact         `json:"contact"`
	Identity ContactIdentity `json:"identity"`
}

type BindContactIdentityInput struct {
	CustomerUUID              string `json:"customer_uuid"`
	ContactUUID               string `json:"contact_uuid"`
	ChannelDictionaryItemUUID string `json:"channel_dictionary_item_uuid"`
	ExternalSubject           string `json:"external_subject"`
}

// MigrateContactIdentityChannelInput is an explicit operator-only repair for
// legacy local identities. It never accepts a legacy channel code and cannot
// infer a dictionary item from historic data.
type MigrateContactIdentityChannelInput struct {
	CustomerUUID              string `json:"customer_uuid"`
	ContactUUID               string `json:"contact_uuid"`
	IdentityUUID              string `json:"identity_uuid"`
	ChannelDictionaryItemUUID string `json:"channel_dictionary_item_uuid"`
}
