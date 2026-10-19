package parceltool

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestVolumeExactDecimalAndInvalidDimensions(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{`{"length_cm":12.5,"width_cm":4,"height_cm":2}`, "100"},
		{`{"length_cm":0.1,"width_cm":0.2,"height_cm":0.3}`, "0.006"},
		{`{"length_cm":123456789012345678901234567890.125,"width_cm":2,"height_cm":1}`, "246913578024691357802469135780.25"},
	} {
		var values map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(tc.input), &values))
		got, err := Volume(values)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
	for _, raw := range []string{
		`{"length_cm":0,"width_cm":4,"height_cm":2}`,
		`{"length_cm":-1,"width_cm":4,"height_cm":2}`,
		`{"length_cm":"12","width_cm":4,"height_cm":2}`,
		`{"length_cm":1e3,"width_cm":4,"height_cm":2}`,
		`{"length_cm":1,"width_cm":4}`,
		`{"length_cm":1,"width_cm":4,"height_cm":2,"extra":1}`,
		`{"length_cm":99999999999999999999999999999999999999999999999999999999999999999999999999999999,"width_cm":10,"height_cm":2}`,
	} {
		var values map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(raw), &values))
		_, err := Volume(values)
		require.ErrorIs(t, err, ErrInvalid)
	}
}
