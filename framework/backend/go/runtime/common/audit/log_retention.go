package audit

import "time"

// DefaultRetention specifies 180 days per spec.
const DefaultRetention = 180 * 24 * time.Hour

// Record models a single audit event that can be retained/purged.
type Record struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Actor     string    `json:"actor"`
	Metadata  any       `json:"metadata,omitempty"`
}

// RetentionPolicy encapsulates retention duration and purge helper.
type RetentionPolicy struct {
	TTL   time.Duration
	Clock func() time.Time
}

// NewRetentionPolicy returns policy with default TTL if not specified.
func NewRetentionPolicy(ttl time.Duration) RetentionPolicy {
	if ttl <= 0 {
		ttl = DefaultRetention
	}
	return RetentionPolicy{TTL: ttl, Clock: time.Now}
}

// IsExpired reports whether a timestamp is older than TTL.
func (p RetentionPolicy) IsExpired(ts time.Time) bool {
	clock := p.Clock
	if clock == nil {
		clock = time.Now
	}
	return clock().After(ts.Add(p.TTL))
}

// Purge filters expired records and returns kept + purged slices.
func (p RetentionPolicy) Purge(records []Record) (kept []Record, purged []Record) {
	for _, rec := range records {
		if p.IsExpired(rec.Timestamp) {
			purged = append(purged, rec)
			continue
		}
		kept = append(kept, rec)
	}
	return
}
