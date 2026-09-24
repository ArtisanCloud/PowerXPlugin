package customer

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// Contact is tenant and Customer scoped natural-person master data. It is
// deliberately separate from CustomerAccount and CustomerAuthIdentity.
type Contact struct {
	models.BaseNoTenantModel

	ContactUUID  string         `gorm:"column:contact_uuid;type:uuid;not null;uniqueIndex" json:"contact_uuid"`
	TenantUUID   string         `gorm:"column:tenant_uuid;type:uuid;not null;uniqueIndex:uk_contact_scope,priority:1;index" json:"tenant_uuid"`
	CustomerUUID string         `gorm:"column:customer_uuid;type:uuid;not null;uniqueIndex:uk_contact_scope,priority:2;index" json:"customer_uuid"`
	DisplayName  string         `gorm:"column:display_name;type:varchar(128);not null;index" json:"display_name"`
	GivenName    string         `gorm:"column:given_name;type:varchar(128)" json:"given_name,omitempty"`
	FamilyName   string         `gorm:"column:family_name;type:varchar(128)" json:"family_name,omitempty"`
	Status       string         `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`
	Roles        datatypes.JSON `gorm:"column:roles;type:jsonb;not null;default:'[]'::jsonb" json:"roles"`
	Tags         datatypes.JSON `gorm:"column:tags;type:jsonb;not null;default:'[]'::jsonb" json:"tags"`
	Metadata     datatypes.JSON `gorm:"column:metadata;type:jsonb;not null;default:'{}'::jsonb" json:"metadata,omitempty"`
}

func (Contact) TableName() string { return models.S(models.TableCustomerContacts) }

type ContactIdentity struct {
	models.BaseNoTenantModel

	IdentityUUID              string         `gorm:"column:identity_uuid;type:uuid;not null;uniqueIndex" json:"identity_uuid"`
	TenantUUID                string         `gorm:"column:tenant_uuid;type:uuid;not null;uniqueIndex:uk_contact_identity_subject,priority:1;index" json:"tenant_uuid"`
	CustomerUUID              string         `gorm:"column:customer_uuid;type:uuid;not null;index" json:"customer_uuid"`
	ContactUUID               string         `gorm:"column:contact_uuid;type:uuid;not null;index" json:"contact_uuid"`
	ChannelDictionaryItemUUID *string        `gorm:"column:channel_dictionary_item_uuid;type:uuid;uniqueIndex:uk_contact_identity_subject,priority:2;index" json:"channel_dictionary_item_uuid,omitempty"`
	ExternalSubject           string         `gorm:"column:external_subject;type:varchar(255);not null;uniqueIndex:uk_contact_identity_subject,priority:3" json:"external_subject"`
	Status                    string         `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`
	VerifiedAt                *time.Time     `gorm:"column:verified_at" json:"verified_at,omitempty"`
	Metadata                  datatypes.JSON `gorm:"column:metadata;type:jsonb;not null;default:'{}'::jsonb" json:"metadata,omitempty"`
}

func (ContactIdentity) TableName() string { return models.S(models.TableCustomerContactIdentities) }
