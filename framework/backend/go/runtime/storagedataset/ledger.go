package storagedataset

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// SourceLedgerMigration is applied by the plugin's regular migration system
// with search_path pinned to the trusted plugin schema. It is never run by a worker.
//
//go:embed migrations/001_source_ledger.sql
var sourceLedgerMigration string

func SourceLedgerMigration() string { return sourceLedgerMigration }

type sourceLedger struct {
	binding Binding
	install Installation
	tenant  string
}

type sourceBatch struct {
	Grant       Grant
	Outcome     string
	ReceiptUUID string
	Affected    int
	Archive     []byte
	ArchiveSHA  string
	SealedAt    time.Time
	Delivered   bool
}

func (l sourceLedger) table() string {
	return `"` + l.binding.Schema + `"."powerx_dataset_source_batches"`
}

func (l sourceLedger) reserve(ctx context.Context, ds Dataset, grant Grant) error {
	tx, err := l.binding.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := sourceBudget(ctx, tx, grant.Deadline); err != nil {
		return err
	}
	if err := ds.verifySource(ctx, tx, grant.Scope); err != nil {
		return err
	}
	if err := verifyDBDeadline(ctx, tx, grant); err != nil {
		return err
	}
	if err := l.insertReservation(ctx, tx, grant); err != nil {
		return err
	}
	if _, err := l.lock(ctx, tx, grant.RequestUUID, grant.GrantUUID, &grant); err != nil {
		return err
	}
	return tx.Commit()
}

func (l sourceLedger) insertReservation(ctx context.Context, tx *sql.Tx, grant Grant) error {
	raw, err := json.Marshal(grant)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO "+l.table()+`
(deployment_id,request_uuid,grant_uuid,plugin_id,installation_revision,tenant_uuid,source_binding_uuid,deadline_at,grant_snapshot)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb) ON CONFLICT (deployment_id,request_uuid) DO NOTHING`,
		l.install.DeploymentID, grant.RequestUUID, grant.GrantUUID, l.install.PluginID, l.install.Revision, l.tenant, l.binding.UUID, grant.Deadline.UTC().Truncate(time.Microsecond), string(raw))
	return err
}

func (l sourceLedger) lookup(ctx context.Context, tx Tx, request string, lock bool) (sourceBatch, error) {
	var out sourceBatch
	var raw []byte
	var grant, plugin, installation, tenant, source string
	var deadline time.Time
	var sealed sql.NullTime
	query := "SELECT grant_snapshot,grant_uuid,plugin_id,installation_revision,tenant_uuid,source_binding_uuid,deadline_at," +
		" COALESCE(outcome,''),COALESCE(source_receipt_uuid::text,''),COALESCE(affected_count,0),archive_bytes,archive_sha256,sealed_at,delivered" +
		" FROM " + l.table() + " WHERE deployment_id=$1 AND request_uuid=$2"
	if lock {
		query += " FOR UPDATE"
	}
	err := tx.QueryRowContext(ctx, query, l.install.DeploymentID, request).
		Scan(&raw, &grant, &plugin, &installation, &tenant, &source, &deadline, &out.Outcome, &out.ReceiptUUID, &out.Affected, &out.Archive, &out.ArchiveSHA, &sealed, &out.Delivered)
	if err != nil {
		return out, err
	}
	if err := decodeContract(raw, ExecutionVersion, "Grant", &out.Grant); err != nil {
		return sourceBatch{}, ErrReceipt
	}
	if plugin != l.install.PluginID || installation != l.install.Revision || tenant != l.tenant || source != l.binding.UUID ||
		out.Grant.TenantUUID != tenant || out.Grant.SourceBindingUUID != source || out.Grant.GrantUUID != grant ||
		out.Grant.RequestUUID != request || !out.Grant.Deadline.Truncate(time.Microsecond).Equal(deadline) {
		return sourceBatch{}, ErrReceipt
	}
	if sealed.Valid {
		out.SealedAt = sealed.Time.UTC()
	}
	if out.Outcome != "" && (out.SealedAt.IsZero() || !canonicalUUID(out.ReceiptUUID)) {
		return sourceBatch{}, ErrReceipt
	}
	if out.Grant.Action == "archive" && out.Outcome == "committed" {
		if out.ArchiveSHA == "" || !out.Delivered && archiveDigest(out.Archive) != out.ArchiveSHA {
			return sourceBatch{}, ErrReceipt
		}
	} else if out.ArchiveSHA != "" {
		return sourceBatch{}, ErrReceipt
	}
	return out, nil
}

func (l sourceLedger) lock(ctx context.Context, tx *sql.Tx, request, grant string, expected *Grant) (sourceBatch, error) {
	out, err := l.lookup(ctx, tx, request, true)
	if err != nil {
		return out, err
	}
	if out.Grant.GrantUUID != grant || expected != nil && !sameGrant(*expected, out.Grant) {
		return sourceBatch{}, ErrReceipt
	}
	return out, nil
}

func sameGrant(a, b Grant) bool {
	a.Cutoff, a.Deadline, a.IssuedAt = a.Cutoff.UTC(), a.Deadline.UTC(), a.IssuedAt.UTC()
	b.Cutoff, b.Deadline, b.IssuedAt = b.Cutoff.UTC(), b.Deadline.UTC(), b.IssuedAt.UTC()
	return a == b
}

func verifyDBDeadline(ctx context.Context, tx *sql.Tx, g Grant) error {
	// Truncate toward the past, so timestamp storage precision cannot give
	// the worker a longer grant than the Core reply.
	var active bool
	err := tx.QueryRowContext(ctx, "SELECT clock_timestamp() >= $1::timestamptz AND clock_timestamp() < $2::timestamptz",
		g.IssuedAt.UTC().Round(time.Microsecond), g.Deadline.UTC().Truncate(time.Microsecond)).Scan(&active)
	if err != nil {
		return err
	}
	if !active || ctx.Err() != nil || !time.Now().Before(g.Deadline) {
		return ErrDeadline
	}
	return nil
}

func (l sourceLedger) execute(ctx context.Context, ds Dataset, grant Grant) (sourceBatch, error) {
	ctx, cancel := context.WithDeadline(ctx, grant.Deadline)
	defer cancel()
	if err := l.reserve(ctx, ds, grant); err != nil {
		return sourceBatch{}, err
	}
	tx, err := l.binding.DB.BeginTx(ctx, nil)
	if err != nil {
		return sourceBatch{}, err
	}
	defer tx.Rollback()
	if err := sourceBudget(ctx, tx, grant.Deadline); err != nil {
		return sourceBatch{}, err
	}
	current, err := l.lock(ctx, tx, grant.RequestUUID, grant.GrantUUID, &grant)
	if err != nil {
		return sourceBatch{}, err
	}
	if current.Outcome != "" {
		return current, nil
	}
	if err := ds.verifySource(ctx, tx, grant.Scope); err != nil {
		return sourceBatch{}, err
	}
	if err := verifyDBDeadline(ctx, tx, grant); err != nil {
		return sourceBatch{}, err
	}
	outcome := "committed"
	affected := 0
	var data []byte
	switch grant.Action {
	case "delete":
		actual, err := deleteRoots(ctx, tx, ds, grant)
		if err != nil {
			return sourceBatch{}, err
		}
		affected = len(actual)
	case "archive":
		data, affected, err = exportArchive(ctx, ds, grant)
		if err != nil {
			return sourceBatch{}, err
		}
		if affected == 0 {
			outcome = "aborted"
		}
	default:
		return sourceBatch{}, ErrContract
	}
	if err := verifyDBDeadline(ctx, tx, grant); err != nil {
		return sourceBatch{}, err
	}
	if err := l.seal(ctx, tx, current, outcome, affected, data); err != nil {
		return sourceBatch{}, err
	}
	if err := tx.Commit(); err != nil {
		// The connection may have dropped after commit. Recover from the
		// source ledger through the original request instead of guessing.
		return sourceBatch{}, errors.Join(ErrReconcile, err)
	}
	return l.lookup(ctx, l.binding.DB, grant.RequestUUID, false)
}

func deleteRoots(ctx context.Context, tx *sql.Tx, ds Dataset, grant Grant) ([]string, error) {
	scan, err := ds.Adapter.Scan(ctx, adapterTx{tx}, grant.Scope, DeleteScan)
	if err != nil {
		return nil, err
	}
	_, ids, err := aggregate(scan, grant.MaxExamined)
	if err != nil {
		return nil, err
	}
	ids = ids[:min(len(ids), grant.MaxAffected)]
	if len(ids) == 0 {
		return nil, nil
	}
	actual, err := ds.Adapter.Delete(ctx, adapterTx{tx}, grant, slices.Clone(ids))
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(actual))
	for _, id := range actual {
		if !slices.Contains(ids, id) || seen[id] {
			return nil, ErrContract
		}
		seen[id] = true
	}
	return actual, nil
}

func exportArchive(ctx context.Context, ds Dataset, grant Grant) ([]byte, int, error) {
	tx, err := ds.Binding.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()
	if err := sourceBudget(ctx, tx, grant.Deadline); err != nil {
		return nil, 0, err
	}
	if err := ds.verifySource(ctx, tx, grant.Scope); err != nil {
		return nil, 0, err
	}
	scan, err := ds.Adapter.Scan(ctx, adapterTx{tx}, grant.Scope, ArchiveScan)
	if err != nil {
		return nil, 0, err
	}
	_, ids, err := aggregate(scan, grant.MaxExamined)
	if err != nil {
		return nil, 0, err
	}
	ids = ids[:min(len(ids), grant.MaxAffected)]
	if len(ids) == 0 {
		return nil, 0, nil
	}
	rows, err := ds.Adapter.Archive(ctx, adapterTx{tx}, grant, slices.Clone(ids))
	if err != nil {
		return nil, 0, err
	}
	if len(rows) != len(ids) {
		return nil, 0, ErrContract
	}
	var out bytes.Buffer
	for _, row := range rows {
		if len(row)+1 > MaxArchiveBytes-out.Len() || !utf8.Valid(row) {
			return nil, 0, ErrContract
		}
		normalized := bytes.TrimSpace(row)
		if len(normalized) == 0 || normalized[0] != '{' || json.Compact(&out, normalized) != nil {
			return nil, 0, ErrContract
		}
		out.WriteByte('\n')
	}
	if err := validateArchive(out.Bytes(), len(rows)); err != nil {
		return nil, 0, err
	}
	return out.Bytes(), len(rows), tx.Commit()
}

func validateArchive(data []byte, n int) error {
	if n < 1 || n > MaxAffected || len(data) < 1 || len(data) > MaxArchiveBytes ||
		data[len(data)-1] != '\n' || !utf8.Valid(data) {
		return ErrContract
	}
	rows := bytes.Split(data[:len(data)-1], []byte{'\n'})
	if len(rows) != n {
		return ErrContract
	}
	for _, row := range rows {
		row = bytes.TrimSpace(row)
		if len(row) == 0 || row[0] != '{' || !json.Valid(row) {
			return ErrContract
		}
	}
	return nil
}

func (l sourceLedger) seal(ctx context.Context, tx *sql.Tx, current sourceBatch, outcome string, count int, data []byte) error {
	// Keep the deadline check at the transaction's commit boundary even if
	// the business adapter has made other deferred checks immediate.
	if _, err := tx.ExecContext(ctx, `SET CONSTRAINTS "`+l.binding.Schema+`"."powerx_dataset_seal_source_batch" DEFERRED`); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, "UPDATE "+l.table()+` SET outcome=$1,source_receipt_uuid=$2,affected_count=$3,archive_bytes=$4,archive_sha256=$5
WHERE deployment_id=$6 AND request_uuid=$7 AND grant_uuid=$8 AND outcome IS NULL`,
		outcome, uuid.NewString(), count, data, archiveDigest(data), l.install.DeploymentID, current.Grant.RequestUUID, current.Grant.GrantUUID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return ErrReceipt
	}
	return nil
}

// reconcile only reads a terminal result or writes a zero-change tombstone
// behind the same unique source reservation and lock as normal execution.
func (l sourceLedger) reconcile(ctx context.Context, ds Dataset, claim ExecutionClaim) (sourceBatch, error) {
	deadline := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	expired := claim.ExpiredGrant
	if expired != nil {
		if err := ds.verifyReadOnly(ctx, expired.Scope, true); err != nil {
			return sourceBatch{}, err
		}
	}
	tx, err := l.binding.DB.BeginTx(ctx, nil)
	if err != nil {
		return sourceBatch{}, err
	}
	defer tx.Rollback()
	if err := sourceBudget(ctx, tx, deadline); err != nil {
		return sourceBatch{}, err
	}
	if expired != nil {
		// Recheck the relation identities under the write transaction's locks.
		if err := ds.verifyFingerprint(ctx, tx, expired.Scope); err != nil {
			return sourceBatch{}, err
		}
		// Round upward to PostgreSQL precision so a faster Core clock or
		// timestamp truncation cannot prematurely seal a missing reservation.
		sourceDeadline := expired.Deadline.UTC().Truncate(time.Microsecond)
		if sourceDeadline.Before(expired.Deadline) {
			sourceDeadline = sourceDeadline.Add(time.Microsecond)
		}
		var elapsed bool
		if err := tx.QueryRowContext(ctx, "SELECT clock_timestamp() >= $1::timestamptz", sourceDeadline).Scan(&elapsed); err != nil {
			return sourceBatch{}, err
		}
		if !elapsed {
			return sourceBatch{}, ErrDeadline
		}
		if err := l.insertReservation(ctx, tx, *expired); err != nil {
			return sourceBatch{}, err
		}
	}
	current, err := l.lock(ctx, tx, claim.RequestUUID, claim.GrantUUID, expired)
	if err != nil {
		return sourceBatch{}, err
	}
	if expired == nil {
		if !ds.match(current.Grant.Scope, current.Grant.Action) {
			return sourceBatch{}, ErrReceipt
		}
		if err := ds.verifyFingerprint(ctx, tx, current.Grant.Scope); err != nil {
			return sourceBatch{}, err
		}
	}
	if current.Outcome != "" {
		return current, nil
	}
	if expired == nil {
		return sourceBatch{}, ErrReconcile
	}
	if err := l.seal(ctx, tx, current, "aborted", 0, nil); err != nil {
		return sourceBatch{}, err
	}
	if err := tx.Commit(); err != nil {
		return sourceBatch{}, errors.Join(ErrReconcile, err)
	}
	return l.lookup(ctx, l.binding.DB, claim.RequestUUID, false)
}

func (b sourceBatch) report() (Report, error) {
	if b.Outcome == "" || b.SealedAt.IsZero() {
		return Report{}, ErrReconcile
	}
	if b.SealedAt.Before(b.Grant.IssuedAt) || b.Outcome == "committed" && !b.SealedAt.Before(b.Grant.Deadline) {
		return Report{}, ErrReceipt
	}
	report := Report{RequestUUID: b.Grant.RequestUUID, GrantUUID: b.Grant.GrantUUID, ReceiptUUID: b.ReceiptUUID,
		Outcome: b.Outcome, Affected: b.Affected, SourceSealedAt: b.SealedAt}
	// The immutable representation binds the full grant and original archive
	// bytes by digest. It is stable across connection loss and worker restarts.
	raw, err := json.Marshal(struct {
		Grant         Grant  `json:"grant"`
		Report        Report `json:"report"`
		ArchiveSHA256 string `json:"archive_sha256,omitempty"`
	}{b.Grant, report, b.ArchiveSHA})
	if err != nil {
		return Report{}, err
	}
	report.Digest = digest(raw)
	return report, nil
}

func archiveDigest(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	return digest(data)
}

func (l sourceLedger) markDelivered(ctx context.Context, b sourceBatch) error {
	res, err := l.binding.DB.ExecContext(ctx, "UPDATE "+l.table()+` SET delivered=true,archive_bytes=NULL
WHERE deployment_id=$1 AND request_uuid=$2 AND grant_uuid=$3 AND source_receipt_uuid=$4`, l.install.DeploymentID, b.Grant.RequestUUID, b.Grant.GrantUUID, b.ReceiptUUID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return ErrReceipt
	}
	return nil
}

func (l sourceLedger) pending(ctx context.Context, limit int) ([]string, error) {
	rows, err := l.binding.DB.QueryContext(ctx, "SELECT request_uuid FROM "+l.table()+` WHERE deployment_id=$1 AND plugin_id=$2 AND installation_revision=$3
AND tenant_uuid=$4 AND source_binding_uuid=$5 AND NOT delivered
AND (outcome IS NOT NULL OR deadline_at < statement_timestamp()) ORDER BY (outcome IS NULL), request_uuid LIMIT $6`,
		l.install.DeploymentID, l.install.PluginID, l.install.Revision, l.tenant, l.binding.UUID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("DATASET_SOURCE_LEDGER_READ: %w", err)
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
