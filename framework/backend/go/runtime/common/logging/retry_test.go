package logging

import (
	"context"
	"errors"
	"testing"
)

func TestExecuteWithRetryEventuallySuccess(t *testing.T) {
	calls := 0
	outcomes := executeWithRetry(context.Background(), SinkStdout, RetryPolicy{Enabled: true, MaxAttempts: 3, BackoffMS: 1}, func() error {
		calls++
		if calls < 2 {
			return errors.New("temporary")
		}
		return nil
	})
	if len(outcomes) != 2 {
		t.Fatalf("expected 2 outcomes, got %+v", outcomes)
	}
	if outcomes[0].Status != OutcomeRetrying || outcomes[1].Status != OutcomeSuccess {
		t.Fatalf("unexpected outcomes: %+v", outcomes)
	}
}

func TestExecuteWithRetryFails(t *testing.T) {
	outcomes := executeWithRetry(context.Background(), SinkStdout, RetryPolicy{Enabled: true, MaxAttempts: 2, BackoffMS: 1}, func() error {
		return errors.New("boom")
	})
	if len(outcomes) != 2 {
		t.Fatalf("expected 2 outcomes, got %+v", outcomes)
	}
	if outcomes[1].Status != OutcomeFailed {
		t.Fatalf("expected failed final outcome, got %+v", outcomes)
	}
}

func TestExecuteWithRetryContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	outcomes := executeWithRetry(ctx, SinkStdout, RetryPolicy{Enabled: true, MaxAttempts: 3, BackoffMS: 10}, func() error {
		return errors.New("boom")
	})
	if len(outcomes) != 2 {
		t.Fatalf("expected 2 outcomes, got %+v", outcomes)
	}
	if outcomes[0].Status != OutcomeRetrying {
		t.Fatalf("expected first outcome retrying, got %+v", outcomes[0])
	}
	if outcomes[1].Status != OutcomeDropped || outcomes[1].ErrorCode != "retry_interrupted" {
		t.Fatalf("expected dropped with retry_interrupted, got %+v", outcomes[1])
	}
}
