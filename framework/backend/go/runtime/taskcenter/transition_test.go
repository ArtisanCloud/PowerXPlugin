package taskcenter

import (
	"errors"
	"testing"
)

func TestValidateTransition(t *testing.T) {
	scope := Scope{TenantUUID: id}
	for _, tc := range []struct {
		name   string
		mutate func(*Task, *UpdateInput)
		want   error
	}{
		{"progress", func(*Task, *UpdateInput) {}, nil},
		{"success", func(_ *Task, u *UpdateInput) { u.State = Succeeded; u.Progress = 100 }, nil},
		{"cancel", func(_ *Task, u *UpdateInput) { u.State = Cancelled }, nil},
		{"revision", func(_ *Task, u *UpdateInput) { u.ExpectedRevision = 2 }, ErrConflict},
		{"regression", func(_ *Task, u *UpdateInput) { u.Progress = 1 }, ErrConflict},
		{"restart", func(_ *Task, u *UpdateInput) { u.State = Queued }, ErrConflict},
		{"terminal", func(c *Task, _ *UpdateInput) { c.State = Failed }, ErrConflict},
		{"other_tenant", func(c *Task, _ *UpdateInput) { c.TenantUUID = "00000000-0000-4000-8000-000000000002" }, ErrNotFound},
		{"no_owner", func(c *Task, _ *UpdateInput) { c.TenantUUID = "" }, ErrNotFound},
		{"other_task", func(_ *Task, u *UpdateInput) { u.TaskUUID = "00000000-0000-4000-8000-000000000002" }, ErrNotFound},
		{"invalid_success", func(_ *Task, u *UpdateInput) { u.State = Succeeded }, ErrInvalidArgument},
		{"queued_success", func(c *Task, u *UpdateInput) { c.State = Queued; u.State = Succeeded; u.Progress = 100 }, ErrConflict},
		{"overflow", func(c *Task, u *UpdateInput) { c.Revision = MaxRevision; u.ExpectedRevision = c.Revision }, ErrConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Task{TaskUUID: id, TenantUUID: id, State: Running, Progress: 20, Revision: 1}
			u := UpdateInput{TaskUUID: id, ExpectedRevision: 1, State: Running, Progress: 30}
			tc.mutate(c, &u)
			if err := ValidateTransition(scope, c, u); !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want=%v", err, tc.want)
			}
		})
	}
}
