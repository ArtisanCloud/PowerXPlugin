package agent

import "encoding/json"

func DecodeWSMessage(message []byte) (AgentStreamEvent, error) {
	var ev AgentStreamEvent
	if err := json.Unmarshal(message, &ev); err != nil {
		return ev, &Error{Code: ErrCodeStreamDecode, Message: err.Error()}
	}
	if !IsKnownEventType(ev.Type) {
		return ev, &Error{Code: ErrCodeStreamDecode, Message: "unknown event type: " + ev.Type}
	}
	return ev, nil
}

func MapWSCloseError(err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: ErrCodeDisconnected, Message: err.Error()}
}
