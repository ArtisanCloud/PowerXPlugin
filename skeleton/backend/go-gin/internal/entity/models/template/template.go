package template

// internal/entity/models/template/template.go

import (
	"encoding/json"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Template represents a reusable snippet that can be shared across the Base plugin.
type Template struct {
	models.BaseModel
	UUID           string     `gorm:"type:uuid;not null;uniqueIndex:idx_template_uuid" json:"uuid"`
	Name           string     `gorm:"type:varchar(255);not null;comment:模板名称" json:"name"`
	Description    string     `gorm:"type:text;comment:模板描述" json:"description"`
	Content        string     `gorm:"type:text;comment:模板内容" json:"content"`
	Status         string     `gorm:"type:varchar(50);not null;default:'draft';comment:发布状态(draft/published/archived) " json:"status"`
	ReviewStatus   string     `gorm:"type:varchar(50);not null;default:'pending';comment:审核状态(pending/approved/rejected)" json:"review_status"`
	ReviewComment  string     `gorm:"type:text;comment:审核备注" json:"review_comment"`
	ReviewedBy     string     `gorm:"type:varchar(100);comment:审核人" json:"reviewed_by"`
	ReviewedAt     *time.Time `gorm:"comment:审核时间" json:"reviewed_at"`
	PublishChannel string     `gorm:"type:varchar(120);comment:发布渠道" json:"publish_channel"`
	PublishedAt    *time.Time `gorm:"comment:发布时间" json:"published_at"`
	CleanupReason  string     `gorm:"type:varchar(255);comment:清理原因" json:"cleanup_reason"`
	CleanedAt      *time.Time `gorm:"comment:清理完成时间" json:"cleaned_at"`
}

func (t *Template) TableName() string {
	return models.S(models.TableTemplate)
}

func (t *Template) BeforeCreate(*gorm.DB) error {
	if t.UUID == "" {
		t.UUID = uuid.NewString()
	}
	parsed, err := uuid.Parse(t.UUID)
	if err != nil || parsed == uuid.Nil || parsed.String() != t.UUID {
		return gorm.ErrInvalidData
	}
	return nil
}

// Raw model responses (mini-app and capability consumers) must not expose the
// inherited storage key. HTTP/gRPC DTOs use the same UUID identity.
func (t Template) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"uuid": t.UUID, "tenant_uuid": t.TenantUuid, "name": t.Name, "description": t.Description, "content": t.Content, "status": t.Status, "review_status": t.ReviewStatus, "review_comment": t.ReviewComment, "reviewed_by": t.ReviewedBy, "reviewed_at": t.ReviewedAt, "publish_channel": t.PublishChannel, "published_at": t.PublishedAt, "cleanup_reason": t.CleanupReason, "cleaned_at": t.CleanedAt, "created_at": t.CreatedAt, "updated_at": t.UpdatedAt})
}
