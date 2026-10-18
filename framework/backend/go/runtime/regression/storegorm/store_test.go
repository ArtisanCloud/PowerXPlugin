package storegorm

import (
	"context"
	"encoding/json"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Store, Regression.Actor, Regression.RunDetail, Regression.RegressionEvent) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	t.Cleanup(func() { _ = conn.Close() })
	for _, table := range []struct {
		name  string
		model any
	}{{"regression_runs", &RunRecord{}}, {"regression_case_executions", &CaseExecutionRecord{}}, {"regression_events", &RegressionEventRecord{}}} {
		if err = db.Table(table.name).AutoMigrate(table.model); err != nil {
			t.Fatal(err)
		}
	}
	s, err := New(Tables{Runs: "regression_runs", Cases: "regression_case_executions", Events: "regression_events"}, func(ctx context.Context, tenant string, fn func(*gorm.DB) error) error {
		return db.WithContext(ctx).Transaction(fn)
	})
	if err != nil {
		t.Fatal(err)
	}
	a := Regression.Actor{TenantUUID: uuid.NewString(), UUID: uuid.NewString(), PluginID: "com.powerx.fixture", RequestID: uuid.NewString(), TraceID: uuid.NewString()}
	now := time.Now().UTC()
	r := Regression.Run{UUID: uuid.NewString(), TenantUUID: a.TenantUUID, ActorUUID: a.UUID, ProfileUUID: uuid.NewString(), ClientRequestUUID: uuid.NewString(), Phase: "queued", Verdict: "pending", CleanupState: "pending", Revision: 1, CreatedAt: now, UpdatedAt: now}
	d := Regression.RunDetail{Run: r, CaseExecutions: []Regression.CaseExecution{}, RequestHash: "a"}
	b, _ := json.Marshal(map[string]any{"run": r})
	e := Regression.RegressionEvent{UUID: uuid.NewString(), TenantUUID: a.TenantUUID, RunUUID: r.UUID, Sequence: 1, Revision: 1, Kind: "run.created", Payload: b, Meta: Regression.EventMeta{TenantUUID: a.TenantUUID, RequestID: a.RequestID, TraceID: a.TraceID, SourcePlugin: a.PluginID, OccurredAt: now, PayloadVersion: "v1"}, CreatedAt: now, UpdatedAt: now}
	return s, a, d, e
}
func TestStoreSubmissionIsolationAndAtomicEvent(t *testing.T) {
	s, a, d, e := fixture(t)
	ctx := context.Background()
	created, duplicate, err := s.Create(ctx, a, d, e)
	if err != nil || duplicate {
		t.Fatal(err)
	}
	if created.LastEventUUID == nil || *created.LastEventUUID != e.UUID {
		t.Fatal(Regression.InvalidRequest)
	}
	_, duplicate, err = s.Create(ctx, a, d, e)
	if err != nil || !duplicate {
		t.Fatal(err)
	}
	changed := d
	changed.RequestHash = "b"
	if _, _, err = s.Create(ctx, a, changed, e); Regression.CodeOf(err) != Regression.RequestConflict {
		t.Fatal(err)
	}
	if _, err = s.Get(ctx, uuid.NewString(), d.Run.UUID); Regression.CodeOf(err) != Regression.NotFound {
		t.Fatal(err)
	}
	d.Run.Revision = 2
	d.Run.Phase = "running"
	e.UUID = uuid.NewString()
	e.Sequence = 2
	e.Revision = 2
	e.Kind = "run.updated"
	e.Meta.PayloadVersion = "v2"
	if err = s.Commit(ctx, a.TenantUUID, d, e); err == nil {
		t.Fatal(Regression.EventMetaInvalid)
	}
	current, _ := s.Get(ctx, a.TenantUUID, d.Run.UUID)
	if current.Run.Revision != 1 {
		t.Fatal(Regression.RequestConflict)
	}
	e.Meta.PayloadVersion = "v1"
	if err = s.Commit(ctx, a.TenantUUID, d, e); err != nil {
		t.Fatal(err)
	}
	events, err := s.Events(ctx, a.TenantUUID, d.Run.UUID, "")
	if err != nil || len(events) != 2 || events[0].Meta.RequestID != a.RequestID {
		t.Fatal(err)
	}
	if _, err = s.Events(ctx, a.TenantUUID, d.Run.UUID, uuid.NewString()); Regression.CodeOf(err) != Regression.NotFound {
		t.Fatal(err)
	}
}

func TestStoreEventFailureRollsBackSnapshotAndRejectsChangedOrigin(t *testing.T) {
	s, a, d, e := fixture(t)
	ctx := context.Background()
	if _, _, err := s.Create(ctx, a, d, e); err != nil {
		t.Fatal(err)
	}
	d.Run.Revision = 2
	d.Run.Phase = "running"
	e.Sequence = 2
	e.Revision = 2
	e.Kind = "run.updated"
	// A duplicate event UUID fails after the snapshot update inside the same transaction.
	if err := s.Commit(ctx, a.TenantUUID, d, e); err == nil {
		t.Fatal(Regression.StoreUnavailable)
	}
	got, err := s.Get(ctx, a.TenantUUID, d.Run.UUID)
	if err != nil || got.Run.Revision != 1 || got.Run.Phase != "queued" {
		t.Fatal(got, err)
	}
	e.UUID = uuid.NewString()
	e.Meta.SourcePlugin = "com.powerx.other"
	if err := s.Commit(ctx, a.TenantUUID, d, e); Regression.CodeOf(err) != Regression.EventMetaInvalid {
		t.Fatal(err)
	}
	for _, edit := range []func(*Regression.EventMeta){func(m *Regression.EventMeta) { m.RequestID = "" }, func(m *Regression.EventMeta) { m.TraceID = "" }, func(m *Regression.EventMeta) { m.OccurredAt = time.Time{} }, func(m *Regression.EventMeta) { m.TenantUUID = uuid.NewString() }} {
		e.Meta.SourcePlugin = a.PluginID
		changed := e
		edit(&changed.Meta)
		if err := s.Commit(ctx, a.TenantUUID, d, changed); err == nil {
			t.Fatal(Regression.EventMetaInvalid)
		}
	}
}
