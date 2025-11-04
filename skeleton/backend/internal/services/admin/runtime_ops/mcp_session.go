package runtime_ops

import (
	"context"

	model "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/entity/models/runtime_ops"
	repository "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/entity/repository"
	runtimeRepo "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/entity/repository/runtime_ops"
	authx "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/middleware"
	"gorm.io/gorm"
)

// MCPSessionService coordinates REGISTER/ACK/CAPABILITY_SYNC flow.
type MCPSessionService struct {
	repo   *runtimeRepo.MCPSessionRepository
	audits *repository.BaseRepository[model.RuntimeAuditEvent]
}

// NewMCPSessionService constructs the MCP session service.
func NewMCPSessionService(db *gorm.DB) *MCPSessionService {
	service := &MCPSessionService{}
	if db != nil {
		service.repo = runtimeRepo.NewMCPSessionRepository(db)
		service.audits = repository.NewBaseRepository[model.RuntimeAuditEvent](db)
	}
	return service
}

// Register is a placeholder for MCP session registration logic.
func (s *MCPSessionService) Register(ctx context.Context, session *model.MCPSession) (*model.MCPSession, error) {
	if session == nil {
		return nil, gorm.ErrInvalidData
	}
	if _, err := authx.RequireTenantID(ctx); err != nil {
		return nil, err
	}
	if s.repo == nil {
		return nil, gorm.ErrInvalidDB
	}
	return s.repo.Create(ctx, session)
}

// RecordAudit writes an audit event for session lifecycle.
func (s *MCPSessionService) RecordAudit(ctx context.Context, evt *model.RuntimeAuditEvent) error {
	if evt == nil {
		return gorm.ErrInvalidData
	}
	if s.audits == nil {
		return gorm.ErrInvalidDB
	}
	_, err := s.audits.Create(ctx, evt)
	return err
}
