package runtime_ops

import "testing"

func TestSchedulerRetryServiceExhaustionAndPauseTicket(t *testing.T) {
	retrySvc := NewSchedulerRetryService()
	ticketSvc := NewSchedulerTicketService()
	dispatchID := "dispatch-us3-001"

	decision1, err := retrySvc.Retry(dispatchID, 3, "AUTH_FORBIDDEN", "topic not allowed")
	if err != nil {
		t.Fatalf("retry attempt1 failed: %v", err)
	}
	if decision1.Exhausted {
		t.Fatalf("attempt1 should not be exhausted")
	}
	if decision1.CurrentAttempt != 1 {
		t.Fatalf("attempt1 should be 1, got %d", decision1.CurrentAttempt)
	}

	decision2, err := retrySvc.Retry(dispatchID, 3, "AUTH_FORBIDDEN", "topic not allowed")
	if err != nil {
		t.Fatalf("retry attempt2 failed: %v", err)
	}
	if decision2.Exhausted {
		t.Fatalf("attempt2 should not be exhausted")
	}

	decision3, err := retrySvc.Retry(dispatchID, 3, "AUTH_FORBIDDEN", "topic not allowed")
	if err != nil {
		t.Fatalf("retry attempt3 failed: %v", err)
	}
	if !decision3.Exhausted {
		t.Fatalf("attempt3 should be exhausted")
	}
	if decision3.Action != RetryActionCreateTicket {
		t.Fatalf("attempt3 action mismatch, got=%s", decision3.Action)
	}

	if err := retrySvc.Pause(dispatchID); err != nil {
		t.Fatalf("pause should succeed after exhaustion: %v", err)
	}

	ticket := ticketSvc.CreatePausedTicket(dispatchID, "job-closed-loop")
	if ticket.TicketID == "" {
		t.Fatal("ticket id should not be empty")
	}
	if ticket.DispatchID != dispatchID {
		t.Fatalf("ticket dispatch mismatch, got=%s", ticket.DispatchID)
	}
	if ticket.PausedJobID != "job-closed-loop" {
		t.Fatalf("ticket paused_job_id mismatch, got=%s", ticket.PausedJobID)
	}

	state, ok := retrySvc.GetState(dispatchID)
	if !ok {
		t.Fatalf("state should exist")
	}
	if state.Status != DispatchStatusPaused {
		t.Fatalf("state status should be paused, got=%s", state.Status)
	}
}
