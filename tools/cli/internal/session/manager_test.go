package session

import (
	"testing"
	"time"
)

func TestManager_CreateAndPersistSession(t *testing.T) {
	mgr := &Manager{store: NewStore(), active: make(map[string]*Session)}

	s, err := mgr.CreateSession("plugin.test", "1.0.0", "/tmp/plugin", "tenant-A")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if s.PluginID != "plugin.test" || s.Version != "1.0.0" {
		t.Fatalf("unexpected session data: %+v", s)
	}

	loaded, err := mgr.GetSession(s.ID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if loaded.ID != s.ID {
		t.Fatalf("expected same session id")
	}

	sessions, err := mgr.ListSessions()
	if err != nil || len(sessions) != 1 {
		t.Fatalf("list sessions: %+v, err=%v", sessions, err)
	}
}

func TestManager_RecordReloadMetrics(t *testing.T) {
	mgr := &Manager{store: NewStore(), active: make(map[string]*Session)}

	s, _ := mgr.CreateSession("plugin.test", "1.0.0", "/tmp/plugin", "")

	if err := mgr.RecordReload(s.ID, 100, true, ""); err != nil {
		t.Fatalf("record reload: %v", err)
	}
	updated, _ := mgr.GetSession(s.ID)
	if updated.Metrics.ReloadCount != 1 || updated.Metrics.AvgReloadTime != 100 {
		t.Fatalf("metrics not updated: %+v", updated.Metrics)
	}

	if err := mgr.RecordReload(s.ID, 50, false, "build failed"); err != nil {
		t.Fatalf("record reload failure: %v", err)
	}
	updated, _ = mgr.GetSession(s.ID)
	if updated.Status != StatusError {
		t.Fatalf("expected error status after failed reload")
	}
}

func TestManager_StopAndDeleteSession(t *testing.T) {
	mgr := &Manager{store: NewStore(), active: make(map[string]*Session)}

	s, _ := mgr.CreateSession("plugin.test", "1.0.0", "/tmp/plugin", "")

	if err := mgr.StopSession(s.ID); err != nil {
		t.Fatalf("stop session: %v", err)
	}
	stopped, _ := mgr.GetSession(s.ID)
	if stopped.Status != StatusStopped {
		t.Fatalf("expected stopped status")
	}

	if err := mgr.DeleteSession(s.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if _, err := mgr.GetSession(s.ID); err == nil {
		t.Fatalf("expected error retrieving deleted session")
	}
}

func TestManager_CleanupExpired(t *testing.T) {
	mgr := &Manager{store: NewStore(), active: make(map[string]*Session)}

	s, _ := mgr.CreateSession("plugin.test", "1.0.0", "/tmp/plugin", "")
	s.CreatedAt = time.Now().Add(-8 * 24 * time.Hour)
	if err := mgr.UpdateSession(s); err != nil {
		t.Fatalf("update session: %v", err)
	}
	mgr.mu.Lock()
	delete(mgr.active, s.ID)
	mgr.mu.Unlock()

	if err := mgr.CleanupExpired(); err != nil {
		t.Fatalf("cleanup expired: %v", err)
	}

	sessions, _ := mgr.ListSessions()
	if len(sessions) != 0 {
		t.Fatalf("expected expired session to be removed")
	}
}
