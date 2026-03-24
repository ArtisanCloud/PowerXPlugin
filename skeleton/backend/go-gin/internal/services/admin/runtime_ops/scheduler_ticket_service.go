package runtime_ops

import "time"

// RecoveryTicket captures minimal ticket fields for pause/resume flow.
type RecoveryTicket struct {
	TicketID           string    `json:"ticket_id"`
	DispatchID         string    `json:"dispatch_id"`
	TicketStatus       string    `json:"ticket_status"`
	PausedJobID        string    `json:"paused_job_id"`
	ResumeRoleRequired string    `json:"resume_role_required"`
	CreatedAt          time.Time `json:"created_at"`
}

// SchedulerTicketService manages recovery ticket scaffolding.
type SchedulerTicketService struct{}

// NewSchedulerTicketService constructs ticket service.
func NewSchedulerTicketService() *SchedulerTicketService {
	return &SchedulerTicketService{}
}

// CreatePausedTicket builds a minimal paused ticket payload.
func (s *SchedulerTicketService) CreatePausedTicket(dispatchID, pausedJobID string) RecoveryTicket {
	return RecoveryTicket{
		TicketID:           "",
		DispatchID:         dispatchID,
		TicketStatus:       "open",
		PausedJobID:        pausedJobID,
		ResumeRoleRequired: "ops_admin_only",
		CreatedAt:          time.Now().UTC(),
	}
}

// RecordResumeAudit is a placeholder for resume audit persistence.
func (s *SchedulerTicketService) RecordResumeAudit(ticketID, operatorID, operatorRole string) error {
	_ = ticketID
	_ = operatorID
	_ = operatorRole
	return nil
}
