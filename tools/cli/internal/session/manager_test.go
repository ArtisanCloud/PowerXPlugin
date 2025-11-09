package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// In-memory store for testing
type memStore struct {
	sessions map[string]*Session
}

func newMemStore() *memStore {
	return &memStore{
		sessions: make(map[string]*Session),
	}
}

func (m *memStore) Save(session *Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *memStore) Load(id string) (*Session, error) {
	session, ok := m.sessions[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return session, nil
}

func (m *memStore) Delete(id string) error {
	delete(m.sessions, id)
	return nil
}

func (m *memStore) List() ([]*Session, error) {
	var sessions []*Session
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (m *memStore) Cleanup() error {
	now := time.Now()
	for id, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			delete(m.sessions, id)
		}
	}
	return nil
}

// TestManager_CreateSession tests creating a new session
func TestManager_CreateSession(t *testing.T) {
	manager := &Manager{
		store:  newMemStore(),
		active: make(map[string]*Session),
	}

	session, err := manager.CreateSession("test-plugin", "1.0.0", "/path/to/plugin", "default")
	if err != nil {
		t.Errorf("CreateSession failed: %v", err)
	}

	if session == nil {
		t.Error("CreateSession returned nil session")
	}

	if session.PluginID != "test-plugin" {
		t.Errorf("Expected PluginID 'test-plugin', got %q", session.PluginID)
	}

	if session.Status != StatusActive {
		t.Errorf("Expected Status Active, got %v", session.Status)
	}

	if session.ReloadToken == "" {
		t.Error("Expected non-empty ReloadToken")
	}

	// Check that session is in active map
	manager.mu.RLock()
	if _, ok := manager.active[session.ID]; !ok {
		t.Error("Session not found in active map")
	}
	manager.mu.RUnlock()
}

// TestManager_CreateSession_Validation tests validation in CreateSession
func TestManager_CreateSession_Validation(t *testing.T) {
	manager := &Manager{
		store:  newMemStore(),
		active: make(map[string]*Session),
	}

	// Test empty pluginID
	_, err := manager.CreateSession("", "1.0.0", "/path/to/plugin", "default")
	if err == nil {
		t.Error("CreateSession should fail with empty pluginID")
	}

	// Test empty entryPath
	_, err = manager.CreateSession("test-plugin", "1.0.0", "", "default")
	if err == nil {
		t.Error("CreateSession should fail with empty entryPath")
	}
}

// TestManager_GetSession tests retrieving a session
func TestManager_GetSession(t *testing.T) {
	store := newMemStore()
	manager := &Manager{
		store:  store,
		active: make(map[string]*Session),
	}

	// Create a session
	original, err := manager.CreateSession("test-plugin", "1.0.0", "/path/to/plugin", "default")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Retrieve the session
	retrieved, err := manager.GetSession(original.ID)
	if err != nil {
		t.Errorf("GetSession failed: %v", err)
	}

	if retrieved == nil {
		t.Error("GetSession returned nil")
	}

	if retrieved.ID != original.ID {
		t.Errorf("Expected ID %q, got %q", original.ID, retrieved.ID)
	}

	if retrieved.PluginID != original.PluginID {
		t.Errorf("Expected PluginID %q, got %q", original.PluginID, retrieved.PluginID)
	}
}

// TestManager_UpdateSession tests updating a session
func TestManager_UpdateSession(t *testing.T) {
	store := newMemStore()
	manager := &Manager{
		store:  store,
		active: make(map[string]*Session),
	}

	// Create a session
	session, err := manager.CreateSession("test-plugin", "1.0.0", "/path/to/plugin", "default")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Update the session
	session.Metrics.ReloadCount = 5
	err = manager.UpdateSession(session)
	if err != nil {
		t.Errorf("UpdateSession failed: %v", err)
	}

	// Verify the update
	retrieved, err := manager.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if retrieved.Metrics.ReloadCount != 5 {
		t.Errorf("Expected ReloadCount 5, got %d", retrieved.Metrics.ReloadCount)
	}

	// Test stopping a session
	session.Status = StatusStopped
	err = manager.UpdateSession(session)
	if err != nil {
		t.Errorf("UpdateSession failed: %v", err)
	}

	// Verify it's removed from active map
	manager.mu.RLock()
	if _, ok := manager.active[session.ID]; ok {
		t.Error("Stopped session should be removed from active map")
	}
	manager.mu.RUnlock()
}

// TestManager_StopSession tests stopping a session
func TestManager_StopSession(t *testing.T) {
	store := newMemStore()
	manager := &Manager{
		store:  store,
		active: make(map[string]*Session),
	}

	// Create a session
	session, err := manager.CreateSession("test-plugin", "1.0.0", "/path/to/plugin", "default")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Stop the session
	err = manager.StopSession(session.ID)
	if err != nil {
		t.Errorf("StopSession failed: %v", err)
	}

	// Verify it's stopped
	retrieved, err := manager.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if retrieved.Status != StatusStopped {
		t.Errorf("Expected Status Stopped, got %v", retrieved.Status)
	}
}

// TestManager_DeleteSession tests deleting a session
func TestManager_DeleteSession(t *testing.T) {
	store := newMemStore()
	manager := &Manager{
		store:  store,
		active: make(map[string]*Session),
	}

	// Create a session
	session, err := manager.CreateSession("test-plugin", "1.0.0", "/path/to/plugin", "default")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Delete the session
	err = manager.DeleteSession(session.ID)
	if err != nil {
		t.Errorf("DeleteSession failed: %v", err)
	}

	// Verify it's deleted
	_, err = manager.GetSession(session.ID)
	if err == nil {
		t.Error("GetSession should fail after deletion")
	}

	// Verify it's removed from active map
	manager.mu.RLock()
	if _, ok := manager.active[session.ID]; ok {
		t.Error("Deleted session should be removed from active map")
	}
	manager.mu.RUnlock()
}

// TestManager_ListSessions tests listing all sessions
func TestManager_ListSessions(t *testing.T) {
	store := newMemStore()
	manager := &Manager{
		store:  store,
		active: make(map[string]*Session),
	}

	// Create multiple sessions
	_, err := manager.CreateSession("plugin1", "1.0.0", "/path/to/plugin1", "tenant1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	_, err = manager.CreateSession("plugin2", "1.0.0", "/path/to/plugin2", "tenant2")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// List sessions
	sessions, err := manager.ListSessions()
	if err != nil {
		t.Errorf("ListSessions failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

// TestManager_RecordReload tests recording reload events
func TestManager_RecordReload(t *testing.T) {
	store := newMemStore()
	manager := &Manager{
		store:  store,
		active: make(map[string]*Session),
	}

	// Create a session
	session, err := manager.CreateSession("test-plugin", "1.0.0", "/path/to/plugin", "default")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Record a successful reload
	err = manager.RecordReload(session.ID, 1000, true, "")
	if err != nil {
		t.Errorf("RecordReload failed: %v", err)
	}

	// Verify metrics updated
	retrieved, err := manager.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if retrieved.Metrics.ReloadCount != 1 {
		t.Errorf("Expected ReloadCount 1, got %d", retrieved.Metrics.ReloadCount)
	}

	if retrieved.Metrics.AvgReloadTime != 1000 {
		t.Errorf("Expected AvgReloadTime 1000, got %d", retrieved.Metrics.AvgReloadTime)
	}

	if retrieved.Metrics.SuccessRate != 1.0 {
		t.Errorf("Expected SuccessRate 1.0, got %f", retrieved.Metrics.SuccessRate)
	}

	// Record a failed reload
	err = manager.RecordReload(session.ID, 2000, false, "build failed")
	if err != nil {
		t.Errorf("RecordReload failed: %v", err)
	}

	// Verify status changed to error
	retrieved, err = manager.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if retrieved.Status != StatusError {
		t.Errorf("Expected Status Error, got %v", retrieved.Status)
	}

	if retrieved.Metrics.LastError != "build failed" {
		t.Errorf("Expected LastError 'build failed', got %q", retrieved.Metrics.LastError)
	}
}

// TestManager_GetActiveSessionIDs tests getting active session IDs
func TestManager_GetActiveSessionIDs(t *testing.T) {
	store := newMemStore()
	manager := &Manager{
		store:  store,
		active: make(map[string]*Session),
	}

	// Create sessions
	s1, err := manager.CreateSession("plugin1", "1.0.0", "/path/to/plugin1", "tenant1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	s2, err := manager.CreateSession("plugin2", "1.0.0", "/path/to/plugin2", "tenant2")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Stop one session
	manager.StopSession(s2.ID)

	// Get active IDs
	ids := manager.GetActiveSessionIDs()
	if len(ids) != 1 {
		t.Errorf("Expected 1 active session, got %d", len(ids))
	}

	if ids[0] != s1.ID {
		t.Errorf("Expected active ID %q, got %q", s1.ID, ids[0])
	}
}

// TestManager_CleanupExpired tests cleaning up expired sessions
func TestManager_CleanupExpired(t *testing.T) {
	store := newMemStore()
	manager := &Manager{
		store:  store,
		active: make(map[string]*Session),
	}

	// Create an expired session
	expiredSession := &Session{
		ID:          "expired",
		PluginID:    "test-plugin",
		Version:     "1.0.0",
		EntryPath:   "/path/to/plugin",
		Tenant:      "default",
		Status:      StatusActive,
		CreatedAt:   time.Now().Add(-2 * time.Hour),
		ExpiresAt:   time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		SessionID:   "expired",
		ReloadToken: "token",
		Metrics: SessionMetrics{
			ReloadCount:     0,
			TotalReloadTime: 0,
			AvgReloadTime:   0,
			SuccessRate:     1.0,
		},
	}
	store.Save(expiredSession)

	// Create an active session
	_, err := manager.CreateSession("test-plugin", "1.0.0", "/path/to/plugin", "default")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Cleanup
	err = manager.CleanupExpired()
	if err != nil {
		t.Errorf("CleanupExpired failed: %v", err)
	}

	// Verify expired session is gone
	sessions, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session after cleanup, got %d", len(sessions))
	}
}
