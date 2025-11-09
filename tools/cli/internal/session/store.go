package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Store handles session persistence
type Store struct {
	baseDir string
}

// NewStore creates a new session store
func NewStore() *Store {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		homeDir = "."
	}

	baseDir := filepath.Join(homeDir, ".px-plugin", "sessions")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		fmt.Printf("Warning: failed to create sessions directory: %v\n", err)
	}

	return &Store{
		baseDir: baseDir,
	}
}

// Save saves a session to disk
func (s *Store) Save(session *Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	filename := s.getSessionPath(session.ID)
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// Load loads a session from disk
func (s *Store) Load(id string) (*Session, error) {
	if id == "" {
		return nil, fmt.Errorf("session id is empty")
	}

	filename := s.getSessionPath(id)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// Delete deletes a session from disk
func (s *Store) Delete(id string) error {
	if id == "" {
		return fmt.Errorf("session id is empty")
	}

	filename := s.getSessionPath(id)
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	return nil
}

// List lists all sessions
func (s *Store) List() ([]*Session, error) {
	var sessions []*Session

	// Read all files in the sessions directory
	files, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return sessions, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	for _, file := range files {
		// Skip non-files
		if !file.IsDir() {
			continue
		}

		// Read session file
		filename := filepath.Join(s.baseDir, file.Name(), "session.json")
		data, err := os.ReadFile(filename)
		if err != nil {
			// Skip unreadable files
			continue
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			// Skip corrupted sessions
			continue
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// Cleanup removes expired sessions
func (s *Store) Cleanup() error {
	sessions, err := s.List()
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if session.IsExpired() {
			if err := s.Delete(session.ID); err != nil {
				fmt.Printf("Warning: failed to delete expired session %s: %v\n", session.ID, err)
			}
		}
	}

	return nil
}

// getSessionPath returns the file path for a session
func (s *Store) getSessionPath(id string) string {
	return filepath.Join(s.baseDir, id+".json")
}
