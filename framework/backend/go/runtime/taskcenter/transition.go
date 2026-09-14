package taskcenter

import "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"

// ValidateTransition validates a proposed mutation against the stored record.
// Local adapters must call it inside their transaction/CAS critical section;
// validation alone is not an atomic update. The runtime deliberately does not
// perform a separate Get before Update (which would introduce a race).
// Adapters own revision increments, UpdatedAt and terminal CompletedAt.
func ValidateTransition(scope Scope, current *Task, in UpdateInput) error {
	if in.ExpectedRevision > MaxRevision || (in.MessageKey != "" && !hostapi.ValidKey(in.MessageKey, 256)) {
		return ErrInvalidArgument
	}
	if !validUUID(scope.TenantUUID) || !validUUID(in.TaskUUID) || in.ExpectedRevision == 0 || !validState(in.State) || in.Progress < 0 || in.Progress > 100 || (in.State == Succeeded && in.Progress != 100) || !validJSON(in.Result) {
		return ErrInvalidArgument
	}
	if current == nil || current.TenantUUID != scope.TenantUUID || current.TaskUUID != in.TaskUUID {
		return ErrNotFound
	}
	if !validState(current.State) || current.Revision == 0 || current.Progress < 0 || current.Progress > 100 {
		return ErrInvalidResponse
	}
	if current.Revision != in.ExpectedRevision || current.Revision >= MaxRevision || terminal(current.State) || in.Progress < current.Progress {
		return ErrConflict
	}
	// Queued tasks may be cancelled/failed without starting. Success requires
	// running first; an active worker cannot put its task back into the queue.
	if current.State == Queued && in.State == Succeeded || current.State == Running && in.State == Queued {
		return ErrConflict
	}
	return nil
}
