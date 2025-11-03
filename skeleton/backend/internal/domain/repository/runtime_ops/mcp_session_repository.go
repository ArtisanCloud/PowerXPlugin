package runtime_ops

import (
	"context"
	"strconv"

	model "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models/runtime_ops"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/repository"
	authx "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/middleware"
	"gorm.io/gorm"
)

// MCPSessionRepository manages MCP session persistence with tenant isolation.
type MCPSessionRepository struct {
	*repository.BaseRepository[model.MCPSession]
}

// NewMCPSessionRepository constructs a repository backed by BaseRepository.
func NewMCPSessionRepository(db *gorm.DB) *MCPSessionRepository {
	return &MCPSessionRepository{
		BaseRepository: repository.NewBaseRepository[model.MCPSession](db),
	}
}

// Create inserts a session record, ensuring tenant scope consistency.
func (r *MCPSessionRepository) Create(ctx context.Context, session *model.MCPSession) (*model.MCPSession, error) {
	tenantID, err := authx.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	tid := strconv.FormatUint(tenantID, 10)
	if session.TenantID == "" {
		session.TenantID = tid
	} else if session.TenantID != tid {
		return nil, gorm.ErrInvalidData
	}

	return r.BaseRepository.Create(ctx, session)
}

// UpdateFields updates session fields while enforcing tenant filter.
func (r *MCPSessionRepository) UpdateFields(ctx context.Context, id string, fields map[string]interface{}) (*model.MCPSession, error) {
	tenantID, err := authx.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	updated, err := r.BaseRepository.Patch(ctx, map[string]interface{}{
		"id":        id,
		"tenant_id": strconv.FormatUint(tenantID, 10),
	}, fields)
	if err != nil {
		return nil, err
	}
	return updated, nil
}
