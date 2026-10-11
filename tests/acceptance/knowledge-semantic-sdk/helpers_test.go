package hostsnapshot_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func sha(text string) string { v := sha256.Sum256([]byte(text)); return hex.EncodeToString(v[:]) }
func ref[T any](value T) *T  { return &value }

type evidenceRecord struct {
	Name     string          `json:"name"`
	HTTP     int             `json:"http"`
	Trace    string          `json:"trace_id"`
	Response json.RawMessage `json:"response"`
}

var fixtureDocumentAPIKey string
