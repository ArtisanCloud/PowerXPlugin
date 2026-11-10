package capability

import (
	"context"
	"errors"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

type StreamInput = hostcontract.StreamInput
type StreamEvent = hostcontract.StreamEvent
type Streamer interface {
	InvokeStream(context.Context, StreamInput, func(StreamEvent) error) error
}

func capabilityFailure(f *hostcontract.InvocationFailure, body string) *HTTPError {
	return &HTTPError{StatusCode: f.StatusCode, ErrorCode: f.ErrorCode, ReasonCode: f.ReasonCode, TraceID: f.TraceID, RequestID: f.RequestID, Details: f.Details, Body: body}
}

func (c *Client) InvokeStream(ctx context.Context, in StreamInput, emit func(StreamEvent) error) error {
	if c == nil || c.token == nil {
		return &HTTPError{StatusCode: 503, ErrorCode: "CAPABILITY_STREAM_UNAVAILABLE", ReasonCode: "CAPABILITY_STREAM_UNAVAILABLE"}
	}
	authorization := func(ctx context.Context) (string, error) {
		token, err := c.token.Token(ctx)
		if err != nil {
			return "", err
		}
		return c.authScheme + " " + token, nil
	}
	err := hostcontract.InvokeStream(ctx, hostcontract.StreamTransport{BaseURL: c.baseURL, APIPrefix: c.apiPrefix, Client: c.http, Authorization: authorization, Timeout: c.streamTimeout}, in, emit)
	var failure *hostcontract.InvocationFailure
	if errors.As(err, &failure) {
		return capabilityFailure(failure, "")
	}
	return err
}
