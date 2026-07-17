package provider

import "strings"

type Mode string

const (
	ModeLocal     Mode = "local"
	ModeDelegated Mode = "delegated"
)

func (m Mode) String() string {
	return string(m)
}

func NormalizeMode(raw string) (Mode, bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return "", false, nil
	case string(ModeLocal):
		return ModeLocal, true, nil
	case string(ModeDelegated):
		return ModeDelegated, true, nil
	default:
		return "", false, ErrModeInvalid
	}
}
