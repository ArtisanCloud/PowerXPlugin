package hostcontract

import (
	"bytes"
	"encoding/json"
	"errors"
)

// DecodeData accepts only the published Core response envelope. It does not
// interpret a raw object or null as a successful business response.
func DecodeData(raw []byte, out any) error {
	var envelope struct {
		Success *bool           `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if envelope.Success != nil && !*envelope.Success {
		return errors.New("host.response_rejected")
	}
	data := bytes.TrimSpace(envelope.Data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return errors.New("host.response_data_required")
	}
	return json.Unmarshal(data, out)
}
