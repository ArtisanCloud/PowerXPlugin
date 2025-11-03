package privacy

import (
	"encoding/json"
	"time"

	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models"
	"gorm.io/datatypes"
)

// Consent token statuses.
const (
	ConsentStatusActive  = "ACTIVE"
	ConsentStatusRevoked = "REVOKED"
	ConsentStatusExpired = "EXPIRED"
)

// ConsentToken records a host-issued consent artifact authorising data use.
type ConsentToken struct {
	ID            string         `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantID      string         `gorm:"column:tenant_id;type:text;not null" json:"tenant_id"`
	Token         string         `gorm:"column:consent_token;type:text;not null" json:"consent_token"`
	Scope         datatypes.JSON `gorm:"column:scope;type:jsonb;not null" json:"scope"`
	ExpiresAt     *time.Time     `gorm:"column:expires_at;type:timestamptz" json:"expires_at,omitempty"`
	IssuedAt      time.Time      `gorm:"column:issued_at;type:timestamptz;not null;default:now()" json:"issued_at"`
	IssuedBy      string         `gorm:"column:issued_by;type:text;not null" json:"issued_by"`
	Status        string         `gorm:"column:status;type:text;not null;default:'ACTIVE'" json:"status"`
	RevokedAt     *time.Time     `gorm:"column:revoked_at;type:timestamptz" json:"revoked_at,omitempty"`
	RevokedReason string         `gorm:"column:revoked_reason;type:text" json:"revoked_reason,omitempty"`
	CreatedAt     time.Time      `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (*ConsentToken) TableName() string {
	return models.S(models.TablePrivacyConsentTokens)
}

// ScopeValues returns the parsed list of asset keys granted by this consent.
func (c *ConsentToken) ScopeValues() ([]string, error) {
	if len(c.Scope) == 0 {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(c.Scope, &values); err != nil {
		return nil, err
	}
	return values, nil
}
