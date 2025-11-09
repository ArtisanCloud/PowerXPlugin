package session

import "time"

// Status represents the status of a session
type Status string

const (
	StatusActive  Status = "active"
	StatusError   Status = "error"
	StatusStopped Status = "stopped"
)

// Session represents a dev session
type Session struct {
	ID           string         `json:"id"`
	PluginID     string         `json:"pluginId"`
	Version      string         `json:"version"`
	EntryPath    string         `json:"entryPath"`
	Tenant       string         `json:"tenant,omitempty"`

	SessionID    string         `json:"sessionId"`
	ReloadToken  string         `json:"reloadToken"`
	Status       Status         `json:"status"`

	CreatedAt    time.Time      `json:"createdAt"`
	LastReloadAt *time.Time     `json:"lastReloadAt,omitempty"`

	Metrics      SessionMetrics `json:"metrics"`
}

// SessionMetrics holds session metrics
type SessionMetrics struct {
	ReloadCount     int     `json:"reloadCount"`
	TotalReloadTime int64   `json:"totalReloadTimeMs"` // in milliseconds
	AvgReloadTime   float64 `json:"avgReloadTimeMs"`
	SuccessRate     float64 `json:"successRate"`       // 0-1
	LastError       string  `json:"lastError,omitempty"`
}

// UpdateMetrics updates session metrics
func (s *Session) UpdateMetrics(duration int64, success bool) {
	s.Metrics.ReloadCount++

	// Update total reload time
	s.Metrics.TotalReloadTime += duration

	// Update average reload time
	s.Metrics.AvgReloadTime = float64(s.Metrics.TotalReloadTime) / float64(s.Metrics.ReloadCount)

	// Update success rate
	// This is a simplified calculation - in production you might track failures separately
	if success {
		// For now, just calculate based on failures vs reloads
		// In a real implementation, you'd track failures separately
		failures := s.Metrics.ReloadCount - int(float64(s.Metrics.ReloadCount)*s.Metrics.SuccessRate)
		newSuccesses := s.Metrics.ReloadCount - failures
		if newSuccesses > 0 {
			s.Metrics.SuccessRate = float64(newSuccesses) / float64(s.Metrics.ReloadCount)
		}
	}

	now := time.Now()
	s.LastReloadAt = &now
}

// IsExpired checks if the session is expired (older than 7 days)
func (s *Session) IsExpired() bool {
	return time.Since(s.CreatedAt) > 7*24*time.Hour
}
