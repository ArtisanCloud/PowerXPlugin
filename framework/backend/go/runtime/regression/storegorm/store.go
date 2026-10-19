package storegorm

import (
	"context"
	"encoding/json"
	"errors"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	"gorm.io/gorm"
	"regexp"
	"sync"
)

type Tables struct{ Runs, Cases, Events string }
type TenantTransaction func(context.Context, string, func(*gorm.DB) error) error
type Store struct {
	tables        Tables
	transaction   TenantTransaction
	transactionMu sync.Mutex
}

func New(t Tables, tx TenantTransaction) (*Store, error) {
	valid := regexp.MustCompile(`^(?:(?:"[A-Za-z_][A-Za-z0-9_]*"|[A-Za-z_][A-Za-z0-9_]*)\.)?[a-z_][a-z0-9_]*$`)
	if tx == nil || !valid.MatchString(t.Runs) || !valid.MatchString(t.Cases) || !valid.MatchString(t.Events) {
		return nil, Regression.NewError(Regression.StoreUnavailable)
	}
	return &Store{tables: t, transaction: tx}, nil
}
func row[T any](value any) T {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var out T
	if err = json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}
func query(tx *gorm.DB, table string) *gorm.DB {
	return tx.Session(&gorm.Session{NewDB: true}).Table(table)
}
func dbError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Regression.NewError(Regression.NotFound)
	}
	if err == nil {
		return nil
	}
	var known *Regression.Error
	if errors.As(err, &known) {
		return err
	}
	return Regression.NewError(Regression.StoreUnavailable)
}
func (s *Store) within(ctx context.Context, tenant string, fn func(*gorm.DB) error) error {
	if !Regression.ValidUUID(tenant) {
		return Regression.NewError(Regression.InvalidRequest)
	}
	s.transactionMu.Lock()
	defer s.transactionMu.Unlock()
	return dbError(s.transaction(ctx, tenant, fn))
}
func validate(tenant string, d Regression.RunDetail, e Regression.RegressionEvent) error {
	if !Regression.ValidUUID(d.Run.UUID) || d.Run.TenantUUID != tenant || !Regression.ValidUUID(e.UUID) || e.RunUUID != d.Run.UUID || e.TenantUUID != tenant || e.Revision != d.Run.Revision || e.Sequence != e.Revision || e.Sequence == 0 || !json.Valid(e.Payload) {
		return Regression.NewError(Regression.InvalidRequest)
	}
	if err := Regression.ValidateMeta(e.Meta, tenant, e.Meta.SourcePlugin); err != nil {
		return err
	}
	for _, c := range d.CaseExecutions {
		if !Regression.ValidUUID(c.UUID) || c.TenantUUID != tenant || c.RunUUID != d.Run.UUID {
			return Regression.NewError(Regression.InvalidRequest)
		}
	}
	return nil
}
func (s *Store) get(tx *gorm.DB, tenant, id string) (Regression.RunDetail, error) {
	var r RunRecord
	if err := query(tx, s.tables.Runs).Where("tenant_uuid = ? AND uuid = ?", tenant, id).First(&r).Error; err != nil {
		return Regression.RunDetail{}, dbError(err)
	}
	var cases []CaseExecutionRecord
	if err := query(tx, s.tables.Cases).Where("tenant_uuid = ? AND run_uuid = ?", tenant, id).Order("ordinal").Find(&cases).Error; err != nil {
		return Regression.RunDetail{}, err
	}
	d := Regression.RunDetail{Run: row[Regression.Run](r), CaseExecutions: []Regression.CaseExecution{}, RequestHash: r.RequestHash}
	for _, c := range cases {
		d.CaseExecutions = append(d.CaseExecutions, row[Regression.CaseExecution](c))
	}
	var last []RegressionEventRecord
	if err := query(tx, s.tables.Events).Select("uuid").Where("tenant_uuid = ? AND run_uuid = ?", tenant, id).Order("sequence DESC").Limit(1).Find(&last).Error; err != nil {
		return d, err
	}
	if len(last) > 0 {
		d.LastEventUUID = &last[0].UUID
	}
	return d, nil
}
func (s *Store) Create(ctx context.Context, a Regression.Actor, d Regression.RunDetail, e Regression.RegressionEvent) (Regression.RunDetail, bool, error) {
	var out Regression.RunDetail
	duplicate := false
	if !Regression.ValidUUID(a.UUID) || d.Run.ActorUUID != a.UUID || !Regression.ValidUUID(d.Run.ClientRequestUUID) || d.RequestHash == "" {
		return out, false, Regression.NewError(Regression.InvalidRequest)
	}
	if err := validate(a.TenantUUID, d, e); err != nil {
		return out, false, err
	}
	if err := Regression.ValidateMeta(e.Meta, a.TenantUUID, a.PluginID); err != nil {
		return out, false, err
	}
	err := s.within(ctx, a.TenantUUID, func(tx *gorm.DB) error {
		var existing []RunRecord
		if err := query(tx, s.tables.Runs).Where("tenant_uuid = ? AND actor_uuid = ? AND client_request_uuid = ?", a.TenantUUID, a.UUID, d.Run.ClientRequestUUID).Find(&existing).Error; err != nil {
			return err
		}
		if len(existing) > 0 {
			if existing[0].RequestHash != d.RequestHash {
				return Regression.NewError(Regression.RequestConflict)
			}
			duplicate = true
			var err error
			out, err = s.get(tx, a.TenantUUID, existing[0].UUID)
			return err
		}
		r := row[RunRecord](d.Run)
		r.RequestHash = d.RequestHash
		if err := query(tx, s.tables.Runs).Create(&r).Error; err != nil {
			return err
		}
		for _, c := range d.CaseExecutions {
			v := row[CaseExecutionRecord](c)
			if err := query(tx, s.tables.Cases).Create(&v).Error; err != nil {
				return err
			}
		}
		v := row[RegressionEventRecord](e)
		if err := query(tx, s.tables.Events).Create(&v).Error; err != nil {
			return err
		}
		var err error
		out, err = s.get(tx, a.TenantUUID, d.Run.UUID)
		return err
	})
	return out, duplicate, err
}
func (s *Store) Get(ctx context.Context, tenant, id string) (Regression.RunDetail, error) {
	var out Regression.RunDetail
	if !Regression.ValidUUID(id) {
		return out, Regression.NewError(Regression.InvalidRequest)
	}
	err := s.within(ctx, tenant, func(tx *gorm.DB) error { var err error; out, err = s.get(tx, tenant, id); return err })
	return out, err
}
func (s *Store) Commit(ctx context.Context, tenant string, d Regression.RunDetail, e Regression.RegressionEvent) error {
	if err := validate(tenant, d, e); err != nil {
		return err
	}
	if e.Revision < 2 {
		return Regression.NewError(Regression.RequestConflict)
	}
	return s.within(ctx, tenant, func(tx *gorm.DB) error {
		r := row[RunRecord](d.Run)
		var previous RunRecord
		if err := query(tx, s.tables.Runs).Where("tenant_uuid = ? AND uuid = ?", tenant, d.Run.UUID).First(&previous).Error; err != nil {
			return err
		}
		var origin RegressionEventRecord
		if err := query(tx, s.tables.Events).Where("tenant_uuid = ? AND run_uuid = ? AND sequence = 1", tenant, d.Run.UUID).First(&origin).Error; err != nil {
			return err
		}
		if err := Regression.ValidateMeta(e.Meta, tenant, origin.Meta.SourcePlugin); err != nil {
			return err
		}
		if r.ActorUUID != previous.ActorUUID || r.ProfileUUID != previous.ProfileUUID || r.ClientRequestUUID != previous.ClientRequestUUID || !Regression.Equal(r.ProfileSnapshot, previous.ProfileSnapshot) || !Regression.Equal(r.SelectionSnapshot, previous.SelectionSnapshot) || !Regression.Equal(r.InputSnapshot, previous.InputSnapshot) || !Regression.Equal(r.SourceSnapshot, previous.SourceSnapshot) {
			return Regression.NewError(Regression.RequestConflict)
		}
		r.RequestHash = previous.RequestHash
		result := query(tx, s.tables.Runs).Where("tenant_uuid = ? AND uuid = ? AND revision = ?", tenant, d.Run.UUID, e.Revision-1).Select("*").Updates(&r)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return Regression.NewError(Regression.RequestConflict)
		}
		for _, c := range d.CaseExecutions {
			v := row[CaseExecutionRecord](c)
			result := query(tx, s.tables.Cases).Where("uuid = ? AND tenant_uuid = ? AND run_uuid = ?", c.UUID, tenant, d.Run.UUID).Select("*").Updates(&v)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return Regression.NewError(Regression.RequestConflict)
			}
		}
		v := row[RegressionEventRecord](e)
		return query(tx, s.tables.Events).Create(&v).Error
	})
}
func (s *Store) List(ctx context.Context, tenant string, page, size uint64, profile string) ([]Regression.Run, uint64, error) {
	out := []Regression.Run{}
	var count int64
	if page == 0 || size == 0 || size > 100 {
		return out, 0, Regression.NewError(Regression.InvalidRequest)
	}
	err := s.within(ctx, tenant, func(tx *gorm.DB) error {
		q := query(tx, s.tables.Runs).Where("tenant_uuid = ?", tenant)
		if profile != "" {
			if !Regression.ValidUUID(profile) {
				return Regression.NewError(Regression.InvalidRequest)
			}
			q = q.Where("profile_uuid = ?", profile)
		}
		if err := q.Count(&count).Error; err != nil {
			return err
		}
		var rows []RunRecord
		q = query(tx, s.tables.Runs).Where("tenant_uuid = ?", tenant)
		if profile != "" {
			q = q.Where("profile_uuid = ?", profile)
		}
		if err := q.Order("created_at DESC").Offset(int((page - 1) * size)).Limit(int(size)).Find(&rows).Error; err != nil {
			return err
		}
		for _, v := range rows {
			out = append(out, row[Regression.Run](v))
		}
		return nil
	})
	return out, uint64(count), err
}
func (s *Store) Events(ctx context.Context, tenant, id, cursor string) ([]Regression.RegressionEvent, error) {
	out := []Regression.RegressionEvent{}
	err := s.within(ctx, tenant, func(tx *gorm.DB) error {
		if _, err := s.get(tx, tenant, id); err != nil {
			return err
		}
		var seq uint64
		if cursor != "" {
			if !Regression.ValidUUID(cursor) {
				return Regression.NewError(Regression.InvalidRequest)
			}
			var e RegressionEventRecord
			if err := query(tx, s.tables.Events).Where("tenant_uuid = ? AND run_uuid = ? AND uuid = ?", tenant, id, cursor).First(&e).Error; err != nil {
				return err
			}
			seq = e.Sequence
		}
		var rows []RegressionEventRecord
		if err := query(tx, s.tables.Events).Where("tenant_uuid = ? AND run_uuid = ? AND sequence > ?", tenant, id, seq).Order("sequence").Find(&rows).Error; err != nil {
			return err
		}
		for _, v := range rows {
			out = append(out, row[Regression.RegressionEvent](v))
		}
		return nil
	})
	return out, err
}
func (s *Store) Active(ctx context.Context, tenant string) ([]Regression.RunDetail, error) {
	out := []Regression.RunDetail{}
	err := s.within(ctx, tenant, func(tx *gorm.DB) error {
		var rows []RunRecord
		if err := query(tx, s.tables.Runs).Where("tenant_uuid = ? AND (phase IN ? OR cleanup_state = ?)", tenant, []string{"queued", "running", "cancelling"}, "running").Find(&rows).Error; err != nil {
			return err
		}
		for _, r := range rows {
			d, err := s.get(tx, tenant, r.UUID)
			if err != nil {
				return err
			}
			out = append(out, d)
		}
		return nil
	})
	return out, err
}

func (s *Store) FindRequest(ctx context.Context, a Regression.Actor, client string) (Regression.RunDetail, error) {
	var out Regression.RunDetail
	if !Regression.ValidUUID(a.UUID) || !Regression.ValidUUID(client) {
		return out, Regression.NewError(Regression.InvalidRequest)
	}
	err := s.within(ctx, a.TenantUUID, func(tx *gorm.DB) error {
		var r RunRecord
		if err := query(tx, s.tables.Runs).Where("tenant_uuid = ? AND actor_uuid = ? AND client_request_uuid = ?", a.TenantUUID, a.UUID, client).First(&r).Error; err != nil {
			return err
		}
		var err error
		out, err = s.get(tx, a.TenantUUID, r.UUID)
		return err
	})
	return out, err
}
