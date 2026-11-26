package storagedataset

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
)

// Tx exposes queries and statements on the worker-owned transaction.
type Tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type adapterTx struct{ Tx }

type ScanMode string

const (
	PreviewScan ScanMode = "preview"
	DeleteScan  ScanMode = "delete"
	ArchiveScan ScanMode = "archive"
)

type Root struct {
	UUID string
	// Protection is a closed reason from the Host contract. An empty reason is
	// a candidate only when the adapter has verified all business conditions.
	Protection string
}

type Scan struct {
	Roots    []Root
	Complete bool
}

// Adapter owns business eligibility. For DeleteScan it must acquire the same
// transaction protection used by live reference and retry writers. Delete
// repeats the guards under those locks and returns the actual deleted root UUIDs.
// All operations receive the fixed grant scope and the worker-owned transaction.
type Adapter interface {
	ValidateSchema(context.Context, Tx, int64) error
	Scan(context.Context, Tx, Scope, ScanMode) (Scan, error)
	Delete(context.Context, Tx, Grant, []string) ([]string, error)
	Archive(context.Context, Tx, Grant, []string) ([]json.RawMessage, error)
}

// Binding is configured by the deployment that owns the plugin installation.
// The DB is a pool for the restricted plugin role on the admitted source.
type Binding struct {
	UUID       string
	ClusterID  string
	Schema     string
	WorkerRole string
	DB         *sql.DB
}

type Installation struct {
	DeploymentID string
	PluginID     string
	Revision     string
}

type Dataset struct {
	Asset   Asset
	Binding Binding
	Adapter Adapter
}

func (b Binding) valid() bool {
	return b.DB != nil && canonicalUUID(b.UUID) && clusterName.MatchString(b.ClusterID) &&
		relationName.MatchString(b.Schema) && relationName.MatchString(b.WorkerRole)
}

func (i Installation) valid() bool {
	return clusterName.MatchString(i.DeploymentID) && manifest.ValidatePluginID(i.PluginID) == nil && i.Revision != "" && len(i.Revision) <= 128
}

func (d Dataset) match(scope Scope, action string) bool {
	a := d.Asset.declaration
	return d.Binding.UUID == scope.SourceBindingUUID && d.Asset.digest == scope.DefinitionDigest &&
		a.SchemaVersion == scope.SchemaVersion && a.AdapterKey == scope.AdapterKey && a.AdapterVersion == scope.AdapterVersion &&
		slices.Contains(a.Actions, action) && !(action == "delete" && a.ArchiveBeforeDelete)
}

func (d Dataset) verifySource(ctx context.Context, tx *sql.Tx, scope Scope) error {
	if err := d.verifyFingerprint(ctx, tx, scope); err != nil {
		return err
	}
	if err := d.Adapter.ValidateSchema(ctx, adapterTx{tx}, scope.SchemaVersion); err != nil {
		return ErrSchema
	}
	return nil
}

func (d Dataset) verifyReadOnly(ctx context.Context, scope Scope, checkSchema bool) error {
	deadline := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	tx, err := d.Binding.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := sourceBudget(ctx, tx, deadline); err != nil {
		return err
	}
	if checkSchema {
		return d.verifySource(ctx, tx, scope)
	}
	return d.verifyFingerprint(ctx, tx, scope)
}

func (d Dataset) verifyFingerprint(ctx context.Context, tx *sql.Tx, scope Scope) error {
	ids, err := d.Binding.relations(ctx, tx, d.Asset.tables)
	if err != nil {
		return err
	}
	slices.Sort(ids)
	raw, err := json.Marshal(ids)
	if err != nil || digest(raw) != scope.SourceFingerprint {
		return ErrScope
	}
	return nil
}

func (b Binding) relations(ctx context.Context, tx *sql.Tx, names []string) ([]string, error) {
	var role string
	var restricted bool
	if err := tx.QueryRowContext(ctx, `SELECT current_user, current_user = session_user
AND NOT (rolsuper OR rolcreaterole OR rolcreatedb OR rolreplication OR rolbypassrls)
FROM pg_catalog.pg_roles WHERE rolname=current_user`).Scan(&role, &restricted); err != nil {
		return nil, ErrSource
	}
	if role != b.WorkerRole || !restricted {
		return nil, ErrScope
	}
	ids := make([]string, 0, len(names))
	for _, name := range names {
		// Hold the relation against replacement from before the OID lookup
		// through the scan and any subsequent changes in this transaction.
		relation := `"` + b.Schema + `"."` + name + `"`
		if _, err := tx.ExecContext(ctx, "SELECT 1 FROM ONLY "+relation+" LIMIT 0"); err != nil {
			return nil, ErrSource
		}
		var databaseID, relationID int64
		err := tx.QueryRowContext(ctx, `SELECT d.oid::bigint,c.oid::bigint
FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
JOIN pg_catalog.pg_database d ON d.datname=current_database()
WHERE n.nspname=$1 AND c.relname=$2 AND c.relkind IN ('r','m') AND NOT c.relispartition
AND pg_catalog.has_schema_privilege(n.oid,'USAGE')
AND pg_catalog.has_table_privilege(c.oid,'SELECT')`, b.Schema, name).Scan(&databaseID, &relationID)
		if err != nil {
			return nil, ErrSource
		}
		ids = append(ids, fmt.Sprintf("%s:%d:%d", b.ClusterID, databaseID, relationID))
	}
	return ids, nil
}

func preview(ctx context.Context, ds Dataset, scope Scope) (Counts, error) {
	var out Counts
	ctx, cancel := context.WithDeadline(ctx, scope.Deadline)
	defer cancel()
	tx, err := ds.Binding.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return out, ErrSource
	}
	defer tx.Rollback()
	if err := sourceBudget(ctx, tx, scope.Deadline); err != nil {
		return out, err
	}
	if err := ds.verifySource(ctx, tx, scope); err != nil {
		return out, err
	}
	scan, err := ds.Adapter.Scan(ctx, adapterTx{tx}, scope, PreviewScan)
	if err != nil {
		return out, err
	}
	out, _, err = aggregate(scan, scope.MaxExamined)
	if err != nil {
		return out, err
	}
	if ctx.Err() != nil {
		return Counts{}, ErrDeadline
	}
	return out, tx.Commit()
}

func sourceBudget(ctx context.Context, tx *sql.Tx, deadline time.Time) error {
	remaining := min(time.Until(deadline), 5*time.Second)
	if remaining < time.Millisecond || ctx.Err() != nil {
		return ErrDeadline
	}
	// All values are computed locally. SET LOCAL via set_config also applies
	// when the caller provides a pooled connection.
	_, err := tx.ExecContext(ctx, "SELECT set_config('lock_timeout',$1,true),set_config('statement_timeout',$2,true)",
		fmt.Sprintf("%dms", min(500, remaining.Milliseconds())), fmt.Sprintf("%dms", remaining.Milliseconds()))
	return err
}

var reasons = map[string]bool{
	"referenced": true, "active": true, "retryable": true, "unknown_eligibility": true,
	"retained_evidence": true, "latest_effective": true, "incomplete_record": true,
}

func aggregate(scan Scan, max int) (Counts, []string, error) {
	var out Counts
	if max < 1 || max > MaxExamined || len(scan.Roots) > max {
		return out, nil, ErrContract
	}
	seen := make(map[string]bool, len(scan.Roots))
	protected := make(map[string]int)
	var candidates []string
	for _, root := range scan.Roots {
		if !canonicalUUID(root.UUID) || seen[root.UUID] {
			return out, nil, ErrContract
		}
		seen[root.UUID] = true
		if root.Protection == "" {
			candidates = append(candidates, root.UUID)
		} else {
			if !reasons[root.Protection] {
				return out, nil, ErrContract
			}
			protected[root.Protection]++
		}
	}
	out = Counts{Coverage: "bounded", Examined: len(scan.Roots), Candidates: len(candidates), Protected: make([]ProtectionCount, 0, len(protected))}
	if scan.Complete {
		out.Coverage = "complete"
	}
	for name, count := range protected {
		out.Protected = append(out.Protected, ProtectionCount{name, count})
	}
	sort.Slice(out.Protected, func(i, j int) bool { return out.Protected[i].Reason < out.Protected[j].Reason })
	return out, candidates, nil
}

func previewReason(err error) string {
	switch {
	case errors.Is(err, ErrDeadline), errors.Is(err, context.DeadlineExceeded):
		return "scan_timeout"
	case errors.Is(err, ErrSchema), errors.Is(err, ErrScope):
		return "schema_mismatch"
	case errors.Is(err, ErrAdapter):
		return "adapter_unavailable"
	case errors.Is(err, ErrContract):
		return "protection_unverifiable"
	default:
		return "source_unavailable"
	}
}
