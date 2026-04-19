package federated

import (
	"context"
	"testing"
	"time"

	iamrepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/repository/iam"
	EntityModels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	iammodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
)

func TestBindingServiceCRUDAndTenantIsolation(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })

	db, err := openTestSQLite("file:binding-service-a?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&iammodel.User{}, &iammodel.Member{}, &iammodel.FederatedBinding{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	seedUser := &iammodel.User{Email: "user@example.com", PasswordHash: "x", Status: iammodel.StatusActive}
	if err := db.Create(seedUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	memberA := &iammodel.Member{BaseModel: EntityModels.BaseModel{TenantUuid: "tenant-a"}, UserID: seedUser.ID, Username: "u1", Status: iammodel.StatusActive}
	memberB := &iammodel.Member{BaseModel: EntityModels.BaseModel{TenantUuid: "tenant-b"}, UserID: seedUser.ID, Username: "u2", Status: iammodel.StatusActive}
	if err := db.Create(memberA).Error; err != nil {
		t.Fatalf("seed memberA: %v", err)
	}
	if err := db.Create(memberB).Error; err != nil {
		t.Fatalf("seed memberB: %v", err)
	}

	repo := iamrepo.NewFederatedBindingRepository(db)
	sessionSvc := NewSessionService()
	svc := NewBindingService(repo, db, sessionSvc)

	created, err := svc.Bind(context.Background(), BindInput{TenantUUID: "tenant-a", Provider: "wecom", ExternalUserID: "wx-1", MemberID: memberA.ID, Source: "admin"})
	if err != nil {
		t.Fatalf("bind err: %v", err)
	}
	if created.MemberID != memberA.ID || created.Status != "active" {
		t.Fatalf("created = %+v", created)
	}

	if _, err := svc.Bind(context.Background(), BindInput{TenantUUID: "tenant-a", Provider: "wecom", ExternalUserID: "wx-2", MemberID: memberB.ID, Source: "admin"}); err == nil {
		t.Fatalf("expected tenant isolation error")
	}

	listed, err := svc.List(context.Background(), "tenant-a", "wecom")
	if err != nil {
		t.Fatalf("list err: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("len(listed) = %d, want 1", len(listed))
	}

	deleted, err := svc.Unbind(context.Background(), "tenant-a", "wecom", "wx-1")
	if err != nil {
		t.Fatalf("unbind err: %v", err)
	}
	if deleted.UnboundAt == nil || deleted.Status != "unbound" {
		t.Fatalf("deleted = %+v", deleted)
	}
	if !sessionSvc.IsInvalidated(memberA.ID) {
		t.Fatalf("expected session invalidated for member %d", memberA.ID)
	}
}

func TestBindingServiceTenantScopeIsolation(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })

	db, err := openTestSQLite("file:binding-service-b?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&iammodel.User{}, &iammodel.Member{}, &iammodel.FederatedBinding{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	seedUser := &iammodel.User{Email: "scope@example.com", PasswordHash: "x", Status: iammodel.StatusActive}
	if err := db.Create(seedUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	member := &iammodel.Member{BaseModel: EntityModels.BaseModel{TenantUuid: "tenant-a"}, UserID: seedUser.ID, Username: "scope-u", Status: iammodel.StatusActive}
	if err := db.Create(member).Error; err != nil {
		t.Fatalf("seed member: %v", err)
	}

	repo := iamrepo.NewFederatedBindingRepository(db)
	svc := NewBindingService(repo, db, nil)

	first, err := svc.Bind(context.Background(), BindInput{
		TenantUUID:     "tenant-a",
		Provider:       "wecom",
		TenantScope:    "corp-a",
		ExternalUserID: "wx-1",
		MemberID:       member.ID,
		Source:         "sync",
	})
	if err != nil {
		t.Fatalf("bind first err: %v", err)
	}
	second, err := svc.Bind(context.Background(), BindInput{
		TenantUUID:     "tenant-a",
		Provider:       "wecom",
		TenantScope:    "corp-b",
		ExternalUserID: "wx-1",
		MemberID:       member.ID,
		Source:         "sync",
	})
	if err != nil {
		t.Fatalf("bind second err: %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("expected different binding rows for different tenant_scope, got same id=%d", first.ID)
	}
}

func TestJITServiceUniqueMatchAutoBind(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })

	db, err := openTestSQLite("file:binding-service-c?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&iammodel.User{}, &iammodel.Member{}, &iammodel.FederatedBinding{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	user := &iammodel.User{Email: "jit@example.com", Phone: "13800000000", PasswordHash: "x", Status: iammodel.StatusActive, CreatedAt: time.Now()}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	member := &iammodel.Member{BaseModel: EntityModels.BaseModel{TenantUuid: "tenant-j"}, UserID: user.ID, Username: "jit", Status: iammodel.StatusActive}
	if err := db.Create(member).Error; err != nil {
		t.Fatalf("seed member: %v", err)
	}

	repo := iamrepo.NewFederatedBindingRepository(db)
	bindingSvc := NewBindingService(repo, db, nil)
	policySvc := NewJITPolicyService()
	policySvc.Set(JITPolicy{TenantUUID: "tenant-j", Enabled: true, Mode: JITPolicyUniqueMatch})
	jitSvc := NewJITService(db, bindingSvc, policySvc)

	result, err := jitSvc.Handle(context.Background(), JITRequest{TenantUUID: "tenant-j", Provider: "wecom", ExternalUserID: "wx-j", Email: "jit@example.com"})
	if err != nil {
		t.Fatalf("jit handle err: %v", err)
	}
	if !result.Bound || result.MemberID != member.ID {
		t.Fatalf("result = %+v", result)
	}
}
