package publisher

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed locales.json
var localeData []byte
var catalog = func() map[string]map[string]string {
	var values map[string]map[string]string
	if json.Unmarshal(localeData, &values) != nil {
		panic("PUBLISHER_INVALID_LOCALE_CATALOG")
	}
	return values
}()

func Text(locale, key string) string {
	language := "en"
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		language = "zh"
	}
	if value, ok := catalog[language][key]; ok {
		return value
	}
	return key
}

type DisplayError struct {
	Err    error
	Locale string
}

func (e *DisplayError) Error() string {
	typed := AsError(e.Err)
	if field := typed.Params["field"]; field != "" {
		return fmt.Sprintf("%s [%s] (%s)", Text(e.Locale, typed.MessageKey), typed.Code, field)
	}
	return fmt.Sprintf("%s [%s]", Text(e.Locale, typed.MessageKey), typed.Code)
}
func (e *DisplayError) Unwrap() error { return e.Err }
func (e *DisplayError) ExitCode() int { return AsError(e.Err).ExitCode() }
