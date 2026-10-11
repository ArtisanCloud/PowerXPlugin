package agent

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestSSEDeliversBeforeEOFAndStopsOnConsumerError(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	stop := errors.New("consumer_stop")
	delivered := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- consumeSSE(reader, func(AgentStreamEvent) error { close(delivered); return stop }) }()
	go func() { _, _ = io.WriteString(writer, "event: token\ndata: {\"payload\":{\"text\":\"x\"}}\n\n") }()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	select {
	case <-delivered:
	case <-ctx.Done():
		t.Fatal("event buffered until EOF")
	}
	select {
	case err := <-done:
		if !errors.Is(err, stop) {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("consumer error ignored")
	}
}
