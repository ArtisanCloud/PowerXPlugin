package notifications

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
)

type failureRoundTripper func(*http.Request) (*http.Response, error)

func (f failureRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type failureBody struct {
	cancel context.CancelFunc
	closed bool
}

func (b *failureBody) Read([]byte) (int, error) {
	if b.cancel != nil {
		b.cancel()
	}
	return 0, io.ErrUnexpectedEOF
}
func (b *failureBody) Close() error { b.closed = true; return nil }

func TestReadFailureAndCancellationContract(t *testing.T) {
	for _, scenario := range []string{"read_failure", "read_cancel", "token_cancel", "token_failure", "empty_token"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			body := &failureBody{}
			if scenario == "read_cancel" {
				body.cancel = cancel
			}
			calls := 0
			token := func(context.Context) (string, error) {
				switch scenario {
				case "token_cancel":
					cancel()
					return "", ctx.Err()
				case "token_failure":
					return "", io.ErrUnexpectedEOF
				case "empty_token":
					return "", nil
				default:
					return "test-sts", nil
				}
			}
			transport := failureRoundTripper(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: body}, nil
			})
			c := &Client{baseURL: "http://test.invalid", http: &http.Client{Transport: transport}, tokens: TokenProviderFunc(token)}
			err := func() error { var out map[string]any; return c.doJSON(ctx, "GET", "/test", nil, &out) }()
			if scenario == "read_cancel" || scenario == "token_cancel" {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation lost: %v", err)
				}
			} else {
				expected := 503
				if scenario == "read_failure" {
					expected = 502
				}
				var typed *HTTPError
				if !errors.As(err, &typed) || typed.StatusCode != expected || typed.ReasonCode != "NOTIFICATION_UPSTREAM_DEPENDENCY" {
					t.Fatalf("error=%#v", err)
				}
			}
			if scenario == "read_failure" || scenario == "read_cancel" {
				if calls != 1 || !body.closed {
					t.Fatalf("calls=%d closed=%v", calls, body.closed)
				}
			} else if calls != 0 {
				t.Fatalf("HTTP called after token failure: %d", calls)
			}
		})
	}
}
