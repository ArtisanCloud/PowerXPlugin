package publisher

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const Protocol = "powerx.plugin-publisher-local/v1"
const PlanSchema = "powerx.plugin-publisher-plan/v1"
const MaxNotesBytes = 10000
const MaxRequestBytes = 64 << 10
const MaxRecordBytes = 128 << 10
const DefaultPageSize = 20
const MaxPageSize = 100

var stableVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

type Target struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}
type CreatePlan struct {
	Version      string `json:"version"`
	Channel      string `json:"channel"`
	Target       Target `json:"target"`
	ReleaseNotes string `json:"release_notes"`
}
type Plan struct {
	Schema        string `json:"schema"`
	ReleasePlanID string `json:"release_plan_id"`
	PluginID      string `json:"plugin_id"`
	CreatePlan
	CreatedAt time.Time `json:"created_at"`
}
type PlanPage struct {
	Items []Plan `json:"items"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
	Size  int    `json:"size"`
}

func (p CreatePlan) Validate() error {
	if len(p.Version) > 50 || !stableVersion.MatchString(p.Version) {
		return invalid("version")
	}
	if p.Channel != "stable" {
		return invalid("channel")
	}
	if p.Target != (Target{OS: "linux", Arch: "amd64"}) {
		return invalid("target")
	}
	if len(p.ReleaseNotes) > MaxNotesBytes || !utf8.ValidString(p.ReleaseNotes) {
		return invalid("release_notes")
	}
	return nil
}
func canonicalUUID(value string) bool {
	u, err := uuid.Parse(value)
	return err == nil && u != uuid.Nil && u.String() == value
}

// Reject duplicate object keys, unknown fields, and trailing JSON documents.
// Standard encoding/json otherwise silently accepts duplicate keys.
func decodeStrict(data []byte, target any) error {
	if !utf8.Valid(data) {
		return invalid("json")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	var value func() error
	value = func() error {
		t, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		if delim == '{' {
			keys := map[string]bool{}
			for d.More() {
				t, err := d.Token()
				if err != nil {
					return err
				}
				key, ok := t.(string)
				if !ok || keys[key] {
					return invalid("json")
				}
				keys[key] = true
				if err = value(); err != nil {
					return err
				}
			}
		} else if delim == '[' {
			for d.More() {
				if err = value(); err != nil {
					return err
				}
			}
		} else {
			return invalid("json")
		}
		_, err = d.Token()
		return err
	}
	if err := value(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return invalid("json")
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	return nil
}
