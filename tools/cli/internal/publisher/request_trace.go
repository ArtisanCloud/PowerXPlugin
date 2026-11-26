package publisher

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"
)

const RequestTraceSchema = "powerx.plugin-publisher-request/v1"
const maxRequestTraceBytes = 4096

// A trace describes an HTTP attempt, never the business outcome. No URL,
// credential, headers, request body or response body is persisted.
type RequestTrace struct {
	Schema          string     `json:"schema"`
	TaskKind        string     `json:"task_kind"`
	TaskID          string     `json:"task_id"`
	OperationID     string     `json:"operation_id"`
	ClientRequestID string     `json:"client_request_id"`
	ServerRequestID string     `json:"server_request_id,omitempty"`
	Method          string     `json:"method"`
	ResponseType    string     `json:"response_type"`
	HTTPStatus      int        `json:"http_status"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
}

type RequestTracePage struct {
	Items []RequestTrace `json:"items"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
}

type requestTraceKey struct{}
type requestTraceSink struct {
	engine                *Engine
	kind, task, operation string
}

func (e *Engine) traceRequests(ctx context.Context, kind, task, operation string) context.Context {
	return context.WithValue(ctx, requestTraceKey{}, requestTraceSink{e, kind, task, operation})
}

func (e *Engine) requestTraceDir(kind, task string) (*directory, error) {
	if (kind != "uploads" && kind != "submissions") || !canonicalUUID(task) {
		return nil, invalid("request_trace")
	}
	d, err := e.state.child("requests", true)
	if err != nil {
		return nil, err
	}
	defer d.close()
	k, err := d.child(kind, true)
	if err != nil {
		return nil, err
	}
	defer k.close()
	return k.child(task, true)
}

func (sink requestTraceSink) save(trace RequestTrace, replace bool) error {
	trace.Schema, trace.TaskKind, trace.TaskID, trace.OperationID = RequestTraceSchema, sink.kind, sink.task, sink.operation
	d, err := sink.engine.requestTraceDir(sink.kind, sink.task)
	if err != nil {
		return err
	}
	defer d.close()
	raw, err := json.Marshal(trace)
	if err != nil || len(raw) > maxRequestTraceBytes {
		return failure(StorageFailure, err)
	}
	return d.write(trace.ClientRequestID+".json", raw, replace)
}

func (e *Engine) RequestTraces(kind, task string, page, size int) (RequestTracePage, error) {
	out := RequestTracePage{Items: []RequestTrace{}, Page: page, Size: size}
	if page < 1 || page > 1000000 || size < 1 || size > 100 {
		return out, invalid("pagination")
	}
	var operation string
	switch kind {
	case "uploads":
		u, err := e.GetUpload(task)
		if err != nil {
			return out, err
		}
		operation = u.OperationID
	case "submissions":
		s, err := e.GetSubmission(task)
		if err != nil {
			return out, err
		}
		operation = s.OperationID
	default:
		return out, invalid("request_trace")
	}
	d, err := e.requestTraceDir(kind, task)
	if err != nil {
		return out, err
	}
	defer d.close()
	names, err := d.names()
	if err != nil {
		return out, err
	}
	items := []RequestTrace{}
	for _, name := range names {
		if strings.HasPrefix(name, ".pending-") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if !strings.HasSuffix(name, ".json") || !canonicalUUID(id) {
			return out, failure(InvalidRecord, nil)
		}
		raw, err := d.read(name, maxRequestTraceBytes)
		if err != nil {
			return out, err
		}
		var trace RequestTrace
		if decodeStrict(raw, &trace) != nil || trace.Schema != RequestTraceSchema || trace.TaskKind != kind || trace.TaskID != task || trace.OperationID != operation || trace.ClientRequestID != id || (trace.ServerRequestID != "" && !canonicalUUID(trace.ServerRequestID)) || (trace.Method != "GET" && trace.Method != "POST") || trace.StartedAt.IsZero() || trace.HTTPStatus != 0 && (trace.HTTPStatus < 100 || trace.HTTPStatus > 599) || !slices.Contains([]string{"DiscoveryResponse", "MeResponse", "PluginResponse", "VersionsPageResponse", "MarketVersionResponse", "UploadReceiptResponse", "OperationResponse", "PublicationOptionsResponse", "PublicationsPageResponse", "SubmissionReceiptResponse"}, trace.ResponseType) {
			return out, failure(InvalidRecord, nil)
		}
		items = append(items, trace)
	}
	slices.SortFunc(items, func(a, b RequestTrace) int {
		if n := b.StartedAt.Compare(a.StartedAt); n != 0 {
			return n
		}
		return strings.Compare(a.ClientRequestID, b.ClientRequestID)
	})
	out.Total = len(items)
	start := min((page-1)*size, len(items))
	out.Items = items[start:min(start+size, len(items))]
	return out, nil
}

type marketRequestError struct {
	requestID string
	cause     error
}

func (e *marketRequestError) Error() string { return e.cause.Error() }
func (e *marketRequestError) Unwrap() error { return e.cause }
