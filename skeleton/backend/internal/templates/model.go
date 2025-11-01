package templates

import "time"

// Template represents an in-memory template entity used by the skeleton backend.
type Template struct {
	ID          uint64    `json:"id"`
	TenantID    uint64    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
