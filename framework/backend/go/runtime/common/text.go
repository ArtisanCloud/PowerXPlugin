package common

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ValidText counts Unicode codepoints, matching PostgreSQL varchar(n).
func ValidText(value string, limit int, required bool) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) <= limit && strings.IndexFunc(value, unicode.IsControl) < 0 && (!required || strings.TrimSpace(value) != "")
}
