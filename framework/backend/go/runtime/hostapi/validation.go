package hostapi

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"
)

func ValidText(s string, max int) bool {
	return len(s) > 0 && len(s) <= max && utf8.ValidString(s) && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00\r\n")
}
func ValidKey(s string, max int) bool {
	if !ValidText(s, max) {
		return false
	}
	for i, c := range []byte(s) {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			continue
		}
		if i == 0 || !strings.ContainsRune("._:-", rune(c)) {
			return false
		}
	}
	return true
}

// ValidJSON rejects duplicate keys, invalid UTF-8, NUL and excessive nesting.
func ValidJSON(raw []byte) bool {
	if len(raw) == 0 {
		return true
	}
	if len(raw) > 256<<10 || !utf8.Valid(raw) {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var value func(int) bool
	value = func(depth int) bool {
		t, err := d.Token()
		if err != nil {
			return false
		}
		switch v := t.(type) {
		case string:
			return !strings.ContainsRune(v, 0)
		case json.Delim:
			if depth >= 64 {
				return false
			}
			if v == '{' {
				seen := map[string]bool{}
				for d.More() {
					k, err := d.Token()
					if err != nil {
						return false
					}
					s, ok := k.(string)
					if !ok || seen[s] || strings.ContainsRune(s, 0) {
						return false
					}
					seen[s] = true
					if !value(depth + 1) {
						return false
					}
				}
				end, err := d.Token()
				return err == nil && end == json.Delim('}')
			}
			if v == '[' {
				for d.More() {
					if !value(depth + 1) {
						return false
					}
				}
				end, err := d.Token()
				return err == nil && end == json.Delim(']')
			}
			return false
		default:
			return true
		}
	}
	if !value(0) {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}
