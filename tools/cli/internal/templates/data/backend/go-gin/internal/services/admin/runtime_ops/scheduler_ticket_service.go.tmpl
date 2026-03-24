package runtime_ops

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RecoveryTicket captures minimal ticket fields for pause/resume flow.
type RecoveryTicket struct {
	TicketID           string    `json:"ticket_id"`
	DispatchID         string    `json:"dispatch_id"`
	TicketStatus       string    `json:"ticket_status"`
	PausedJobID        string    `json:"paused_job_id"`
	ResumeRoleRequired string    `json:"resume_role_required"`
	CreatedAt          time.Time `json:"created_at"`
	ResolvedBy         string    `json:"resolved_by,omitempty"`
	ResolvedAt         time.Time `json:"resolved_at,omitempty"`
}

// ResumeAuditRecord captures resume operator audit fields.
type ResumeAuditRecord struct {
	TicketID     string    `json:"ticket_id"`
	DispatchID   string    `json:"dispatch_id"`
	OperatorID   string    `json:"operator_id"`
	OperatorRole string    `json:"operator_role"`
	RecordedAt   time.Time `json:"recorded_at"`
}

var (
	ErrTicketIDRequired = errors.New("ticket_id is required")
	ErrTicketNotFound   = errors.New("ticket not found")
	ErrOperatorRequired = errors.New("operator_id is required")
	ErrRoleRequired     = errors.New("operator_role is required")
	ErrRoleForbidden    = errors.New("resume requires ops/admin role")
)

// SchedulerTicketService manages recovery ticket scaffolding.
type SchedulerTicketService struct {
	mu             sync.RWMutex
	tickets        map[string]RecoveryTicket
	dispatchTicket map[string]string
	resumeAudits   []ResumeAuditRecord
}

// NewSchedulerTicketService constructs ticket service.
func NewSchedulerTicketService() *SchedulerTicketService {
	return &SchedulerTicketService{
		tickets:        make(map[string]RecoveryTicket),
		dispatchTicket: make(map[string]string),
	}
}

// CreatePausedTicket builds a minimal paused ticket payload.
func (s *SchedulerTicketService) CreatePausedTicket(dispatchID, pausedJobID string) RecoveryTicket {
	dispatchID = strings.TrimSpace(dispatchID)
	pausedJobID = strings.TrimSpace(pausedJobID)
	if pausedJobID == "" {
		pausedJobID = dispatchID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existingID := strings.TrimSpace(s.dispatchTicket[dispatchID]); existingID != "" {
		if ticket, ok := s.tickets[existingID]; ok && ticket.TicketStatus != "resolved" {
			return ticket
		}
	}

	ticket := RecoveryTicket{
		TicketID:           "rtk-" + uuid.NewString(),
		DispatchID:         dispatchID,
		TicketStatus:       "open",
		PausedJobID:        pausedJobID,
		ResumeRoleRequired: "ops_admin_only",
		CreatedAt:          time.Now().UTC(),
	}
	s.tickets[ticket.TicketID] = ticket
	s.dispatchTicket[dispatchID] = ticket.TicketID
	return ticket
}

// GetTicket returns ticket by id.
func (s *SchedulerTicketService) GetTicket(ticketID string) (RecoveryTicket, bool) {
	ticketID = strings.TrimSpace(ticketID)
	if ticketID == "" {
		return RecoveryTicket{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ticket, ok := s.tickets[ticketID]
	return ticket, ok
}

// ResumeTicket resolves a paused ticket when caller role is allowed.
func (s *SchedulerTicketService) ResumeTicket(ticketID, operatorID, operatorRole string) (RecoveryTicket, error) {
	ticketID = strings.TrimSpace(ticketID)
	if ticketID == "" {
		return RecoveryTicket{}, ErrTicketIDRequired
	}
	operatorID = strings.TrimSpace(operatorID)
	if operatorID == "" {
		return RecoveryTicket{}, ErrOperatorRequired
	}
	operatorRole = normalizeOperatorRole(operatorRole)
	if operatorRole == "" {
		return RecoveryTicket{}, ErrRoleRequired
	}
	if !isOpsAdminRole(operatorRole) {
		return RecoveryTicket{}, ErrRoleForbidden
	}

	s.mu.Lock()
	ticket, ok := s.tickets[ticketID]
	if !ok {
		s.mu.Unlock()
		return RecoveryTicket{}, ErrTicketNotFound
	}
	if ticket.TicketStatus == "resolved" {
		s.mu.Unlock()
		return ticket, nil
	}
	now := time.Now().UTC()
	ticket.TicketStatus = "resolved"
	ticket.ResolvedBy = operatorID
	ticket.ResolvedAt = now
	s.tickets[ticketID] = ticket
	s.mu.Unlock()

	if err := s.RecordResumeAudit(ticketID, operatorID, operatorRole); err != nil {
		return RecoveryTicket{}, err
	}
	return ticket, nil
}

// ListResumeAudits returns a snapshot of resume audit records.
func (s *SchedulerTicketService) ListResumeAudits() []ResumeAuditRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ResumeAuditRecord, len(s.resumeAudits))
	copy(out, s.resumeAudits)
	return out
}

// RecordResumeAudit stores an audit record for resume operation.
func (s *SchedulerTicketService) RecordResumeAudit(ticketID, operatorID, operatorRole string) error {
	ticketID = strings.TrimSpace(ticketID)
	if ticketID == "" {
		return ErrTicketIDRequired
	}
	operatorID = strings.TrimSpace(operatorID)
	if operatorID == "" {
		return ErrOperatorRequired
	}
	operatorRole = normalizeOperatorRole(operatorRole)
	if operatorRole == "" {
		return ErrRoleRequired
	}
	if !isOpsAdminRole(operatorRole) {
		return ErrRoleForbidden
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ticket, ok := s.tickets[ticketID]
	if !ok {
		return ErrTicketNotFound
	}
	s.resumeAudits = append(s.resumeAudits, ResumeAuditRecord{
		TicketID:     ticketID,
		DispatchID:   ticket.DispatchID,
		OperatorID:   operatorID,
		OperatorRole: operatorRole,
		RecordedAt:   time.Now().UTC(),
	})
	return nil
}

func normalizeOperatorRole(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func isOpsAdminRole(role string) bool {
	switch normalizeOperatorRole(role) {
	case "ops", "admin":
		return true
	default:
		return false
	}
}
