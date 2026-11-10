package gateway

import (
	"context"
	"errors"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

type StreamInput = hostcontract.StreamInput
type StreamEvent = hostcontract.StreamEvent

func invocationError(f *hostcontract.InvocationFailure, body []byte) *InvocationError {
	return &InvocationError{StatusCode: f.StatusCode, ErrorCode: f.ErrorCode, ReasonCode: f.ReasonCode, TraceID: f.TraceID, RequestID: f.RequestID, UpstreamRequestID: f.RequestID, Details: f.Details, Body: body}
}

func (c *Client) InvokeStream(ctx context.Context, input StreamInput, emit func(StreamEvent) error) error {
	if c == nil {
		return invocationError(&hostcontract.InvocationFailure{StatusCode: 503, ErrorCode: "GATEWAY_UNAVAILABLE", ReasonCode: "GATEWAY_UNAVAILABLE"}, nil)
	}
	err := hostcontract.InvokeStream(ctx, hostcontract.StreamTransport{BaseURL: c.baseURL, APIPrefix: c.apiPrefix, Client: c.httpClient, Authorization: c.authHeader, Timeout: c.streamTimeout}, input, emit)
	var failure *hostcontract.InvocationFailure
	if errors.As(err, &failure) {
		return invocationError(failure, nil)
	}
	return err
}
