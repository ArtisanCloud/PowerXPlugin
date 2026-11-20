package storagedataset

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/taskqueue"
)

type Worker struct {
	client   *Client
	queue    *taskqueue.HostProvider
	install  Installation
	datasets []Dataset
}

func NewWorker(client *Client, queue *taskqueue.HostProvider, installation Installation, datasets []Dataset) (*Worker, error) {
	if client == nil || queue == nil || !installation.valid() || len(datasets) == 0 {
		return nil, ErrScope
	}
	var local []Dataset
	keys := map[string]bool{}
	bindings := map[string]Binding{}
	for _, ds := range datasets {
		if ds.Asset.digest == "" || !ds.Binding.valid() || nilAdapter(ds.Adapter) {
			return nil, ErrContract
		}
		key := ds.Asset.Key()
		if keys[key] {
			return nil, ErrContract
		}
		keys[key] = true
		if b, exists := bindings[ds.Binding.UUID]; exists && b != ds.Binding {
			return nil, ErrScope
		}
		bindings[ds.Binding.UUID] = ds.Binding
		local = append(local, ds)
	}
	return &Worker{client: client, queue: queue, install: installation, datasets: local}, nil
}

func (w *Worker) ledger(ds Dataset) sourceLedger {
	return sourceLedger{ds.Binding, w.install, w.client.tenant}
}

func (w *Worker) dataset(scope Scope, action string) (Dataset, error) {
	var result Dataset
	found := false
	for _, ds := range w.datasets {
		if ds.match(scope, action) {
			if found {
				return Dataset{}, ErrScope
			}
			found, result = true, ds
		}
	}
	if !found {
		return Dataset{}, ErrAdapter
	}
	return result, nil
}

func (w *Worker) runPreview(ctx context.Context, claim Claim, leaseExpiry time.Time) (string, error) {
	scope, err := w.client.ClaimPreview(ctx, claim)
	if err != nil {
		return "", err
	}
	if scope.Deadline.After(leaseExpiry) {
		return "", ErrDeadline
	}
	report := PreviewReport{Claim: claim, AcceptanceRevision: scope.AcceptanceRevision, Outcome: "completed"}
	ds, err := w.dataset(scope, "preview")
	var counts Counts
	if err == nil {
		counts, err = preview(ctx, ds, scope)
	}
	if err != nil {
		report.Outcome = "failed"
		report.Reason = previewReason(err)
	} else {
		report.Result = &counts
	}
	if err := w.client.ReportPreview(ctx, report); err != nil {
		// Only repeat the exact original preview report on transport or
		// upstream service failures. Core owns its idempotency decision.
		var httpErr *gateway.InvocationError
		if ctx.Err() != nil || errors.Is(err, ErrScope) || errors.Is(err, ErrContract) ||
			errors.As(err, &httpErr) && httpErr.StatusCode < 500 {
			return "", err
		}
		if retry := w.client.ReportPreview(ctx, report); retry != nil {
			return "", retry
		}
	}
	return report.Outcome, nil
}

func (w *Worker) runExecution(ctx context.Context, claim Claim, leaseExpiry time.Time) (string, error) {
	authorization, err := w.client.ClaimExecution(ctx, claim)
	if err != nil {
		return "", err
	}
	if authorization.Grant != nil {
		grant := *authorization.Grant
		if grant.Deadline.After(leaseExpiry) {
			return "", ErrDeadline
		}
		ds, err := w.dataset(grant.Scope, grant.Action)
		if err != nil {
			return "", err
		}
		ledger := w.ledger(ds)
		current, err := ledger.execute(ctx, ds, grant)
		if err != nil {
			return "", err
		}
		return w.deliver(ctx, ledger, current)
	}
	return w.reconcileExecution(ctx, authorization)
}

// ReconcileExecution recovers an already issued batch by its original request
// UUID, including when the TaskBus lease and local reservation are both absent.
func (w *Worker) ReconcileExecution(ctx context.Context, requestUUID string) error {
	authorization, err := w.client.ReconcileExecution(ctx, requestUUID)
	if err != nil {
		return err
	}
	_, err = w.reconcileExecution(ctx, authorization)
	return err
}

func (w *Worker) reconcileExecution(ctx context.Context, authorization ExecutionClaim) (string, error) {
	var ds Dataset
	var err error
	if expired := authorization.ExpiredGrant; expired != nil {
		ds, err = w.dataset(expired.Scope, expired.Action)
	} else {
		ds, err = w.findReservation(ctx, authorization.RequestUUID, authorization.GrantUUID)
	}
	if err != nil {
		return "", err
	}
	ledger := w.ledger(ds)
	current, err := ledger.reconcile(ctx, ds, authorization)
	if err != nil {
		return "", err
	}
	return w.deliver(ctx, ledger, current)
}

func (w *Worker) findReservation(ctx context.Context, request, grant string) (Dataset, error) {
	searched := map[string]bool{}
	var original *Grant
	for _, ds := range w.datasets {
		if !ds.Asset.allowsExecution() || searched[ds.Binding.UUID] {
			continue
		}
		searched[ds.Binding.UUID] = true
		ledger := w.ledger(ds)
		current, err := ledger.lookup(ctx, ds.Binding.DB, request, false)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return Dataset{}, errors.Join(ErrReconcile, err)
		}
		if original != nil || current.Grant.GrantUUID != grant {
			return Dataset{}, ErrReceipt
		}
		original = &current.Grant
	}
	if original == nil {
		return Dataset{}, ErrReconcile
	}
	return w.dataset(original.Scope, original.Action)
}

func (a Asset) allowsExecution() bool {
	for _, action := range a.declaration.Actions {
		if action == "archive" || action == "delete" && !a.declaration.ArchiveBeforeDelete {
			return true
		}
	}
	return false
}

func (w *Worker) deliver(ctx context.Context, ledger sourceLedger, batch sourceBatch) (string, error) {
	if batch.Delivered {
		return "completed", nil
	}
	report, err := batch.report()
	if err != nil {
		return "", err
	}
	if batch.Grant.Action == "archive" && report.Outcome == "committed" {
		err = w.client.ArchiveExecution(ctx, report, batch.Archive)
	} else {
		err = w.client.ReportExecution(ctx, report)
	}
	if err != nil {
		return "", err
	}
	if err := ledger.markDelivered(ctx, batch); err != nil {
		return "", err
	}
	return "completed", nil
}

type notice struct {
	Version     string `json:"contract_version"`
	RequestUUID string `json:"request_uuid"`
}

// Handle settles only a delivery supplied by the Host provider. Core separately
// checks the reserved task record, current lease, and credential.
func (w *Worker) Handle(ctx context.Context, msg taskqueue.Message) error {
	if msg.TenantKey != w.client.tenant || msg.Lease == nil || msg.ID == "" ||
		!canonicalUUID(msg.Lease.ReceiptUUID) || msg.Lease.Fence == 0 || !time.Now().Before(msg.Lease.ExpiresAt) {
		return ErrScope
	}
	var version, def string
	var run func(context.Context, Claim, time.Time) (string, error)
	switch {
	case msg.Topic == PreviewTopic && msg.SubscriberID == PreviewSubscriber:
		version, def, run = HostVersion, "PreviewTaskNotice", w.runPreview
	case msg.Topic == ExecutionTopic && msg.SubscriberID == ExecutionSubscriber:
		version, def, run = ExecutionVersion, "TaskNotice", w.runExecution
	default:
		return ErrScope
	}
	var item notice
	if err := decodeContract(msg.Payload, version, def, &item); err != nil {
		return err
	}
	outcome, err := run(ctx, Claim{item.RequestUUID, msg.Lease.ReceiptUUID, msg.Lease.Fence}, msg.Lease.ExpiresAt)
	if err != nil {
		return err
	}
	return w.queue.Ack(ctx, taskqueue.AckRequest{
		TenantKey: w.client.tenant, SubscriberID: msg.SubscriberID, MessageID: msg.ID,
		ReceiptUUID: msg.Lease.ReceiptUUID, Outcome: outcome,
	})
}

// ReceiveAndHandle performs a single bounded blocking receive, including the
// original TaskBus lease. A service can use this from its supervised worker loop.
func (w *Worker) ReceiveAndHandle(ctx context.Context, subscriber string, wait time.Duration) (bool, error) {
	if subscriber != PreviewSubscriber && subscriber != ExecutionSubscriber {
		return false, ErrScope
	}
	if wait < time.Millisecond || wait > 30*time.Second {
		return false, ErrContract
	}
	items, err := w.queue.Dequeue(ctx, taskqueue.DequeueRequest{
		TenantKey: w.client.tenant, SubscriberID: subscriber, MaxItems: 1, WaitTimeout: wait,
	})
	if err != nil {
		return false, err
	}
	if len(items) == 0 {
		return false, nil
	}
	if len(items) != 1 {
		return false, ErrContract
	}
	return true, w.Handle(ctx, items[0])
}

// RecoverPending replays terminal receipts and reconciles expired reservations
// after restart or transport recovery. It never requests new write authority.
func (w *Worker) RecoverPending(ctx context.Context, limit int) (int, error) {
	if limit < 1 || limit > 100 {
		return 0, ErrContract
	}
	count := 0
	seen := map[string]bool{}
	var failures error
	for _, ds := range w.datasets {
		if !ds.Asset.allowsExecution() || seen[ds.Binding.UUID] {
			continue
		}
		seen[ds.Binding.UUID] = true
		ledger := w.ledger(ds)
		ids, err := ledger.pending(ctx, limit-count)
		if err != nil {
			failures = errors.Join(failures, err)
			continue
		}
		for _, id := range ids {
			err := w.recoverBatch(ctx, ledger, id)
			if err != nil {
				failures = errors.Join(failures, fmt.Errorf("%w: %w", ErrReconcile, err))
				continue
			}
			count++
			if count >= limit {
				return count, failures
			}
		}
	}
	return count, failures
}

func (w *Worker) recoverBatch(ctx context.Context, ledger sourceLedger, requestUUID string) error {
	batch, err := ledger.lookup(ctx, ledger.binding.DB, requestUUID, false)
	if err != nil {
		return err
	}
	ds, err := w.dataset(batch.Grant.Scope, batch.Grant.Action)
	if err != nil {
		return err
	}
	if ds.Binding != ledger.binding {
		return ErrScope
	}
	if batch.Outcome == "" {
		authorization, err := w.client.ReconcileExecution(ctx, requestUUID)
		if err != nil {
			return err
		}
		if batch.Grant.GrantUUID != authorization.GrantUUID ||
			authorization.ExpiredGrant != nil && !sameGrant(batch.Grant, *authorization.ExpiredGrant) {
			return ErrReceipt
		}
		batch, err = ledger.reconcile(ctx, ds, authorization)
		if err != nil {
			return err
		}
	} else if err := ds.verifyReadOnly(ctx, batch.Grant.Scope, false); err != nil {
		return err
	}
	_, err = w.deliver(ctx, ledger, batch)
	return err
}

// Reject typed-nil adapters at assembly as well as nil interfaces.
func nilAdapter(a Adapter) bool {
	if a == nil {
		return true
	}
	v := reflect.ValueOf(a)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
