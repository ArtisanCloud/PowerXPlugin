package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeSSE(t *testing.T) {
	raw := "event: token\ndata: {\"trace_id\":\"trace_001\",\"session_id\":\"session_001\",\"payload\":{\"text\":\"hi\"}}\n\n"
	events, err := DecodeSSE(strings.NewReader(raw))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, EventToken, events[0].Type)
	require.Equal(t, "trace_001", events[0].TraceID)
}

func TestDecodeSSERejectsUnknownEvent(t *testing.T) {
	_, err := DecodeSSE(strings.NewReader("event: mystery\ndata: {}\n\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrCodeStreamDecode)
}
