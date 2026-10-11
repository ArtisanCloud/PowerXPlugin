package hostcontract

import "testing"

func TestDecodeDataRejectsMissingNullAndRawObjects(t *testing.T) {
	for _, raw := range []string{"", "{}", `{"data":null}`, `{"id":"not-enveloped"}`} {
		var out map[string]any
		if DecodeData([]byte(raw), &out) == nil {
			t.Fatalf("accepted=%s", raw)
		}
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := DecodeData([]byte(`{"data":{"id":"object"}}`), &out); err != nil || out.ID != "object" {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}
