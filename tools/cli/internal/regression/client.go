package regression

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//go:embed i18n/*.json
var languages embed.FS

func Message(locale, key string) string {
	if locale != "en" && locale != "zh" {
		locale = "zh"
	}
	b, _ := languages.ReadFile("i18n/" + locale + ".json")
	var values map[string]string
	_ = json.Unmarshal(b, &values)
	return values[key]
}

type Error struct {
	Code    string
	RunUUID string
	Status  int
	Locale  string
}

func (e *Error) Error() string {
	key := "invalid"
	if e.Code == "REGRESSION_STREAM_INTERRUPTED" {
		key = "interrupted"
	}
	if e.Code == "REGRESSION_NOT_PASSED" {
		key = "blocked"
	}
	return fmt.Sprintf("%s [%s] %s", Message(e.Locale, key), e.Code, e.RunUUID)
}
func (e *Error) ExitCode() int {
	if e.Code == "REGRESSION_NOT_PASSED" {
		return 2
	}
	if e.Code == "REGRESSION_STREAM_INTERRUPTED" {
		return 3
	}
	return 1
}

type Client struct {
	BaseURL, Token, TenantUUID, PluginID, Locale string
	HTTP                                         *http.Client
}
type Run struct {
	UUID         string `json:"uuid"`
	TenantUUID   string `json:"tenant_uuid"`
	Phase        string `json:"phase"`
	Verdict      string `json:"verdict"`
	CleanupState string `json:"cleanup_state"`
	Revision     uint64 `json:"revision"`
}
type Detail struct {
	Run            Run             `json:"run"`
	LastEventUUID  *string         `json:"last_event_uuid"`
	CaseExecutions json.RawMessage `json:"case_executions"`
}

func ValidUUID(v string) bool {
	parsed, err := uuid.Parse(v)
	return err == nil && parsed != uuid.Nil && parsed.String() == v
}
func (c *Client) request(ctx context.Context, method, path string, body any) (*http.Response, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") || base.User != nil || base.RawQuery != "" || c.Token == "" {
		return nil, &Error{Code: "REGRESSION_INVALID_REQUEST", Locale: c.Locale}
	}
	var input io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		input = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+"/admin/regression"+path, input)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	if c.TenantUUID != "" {
		if !ValidUUID(c.TenantUUID) {
			return nil, &Error{Code: "REGRESSION_INVALID_REQUEST", Locale: c.Locale}
		}
		req.Header.Set("X-Tenant-UUID", c.TenantUUID)
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, &Error{Code: "REGRESSION_STREAM_INTERRUPTED", Locale: c.Locale}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		defer response.Body.Close()
		var data struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.NewDecoder(response.Body).Decode(&data)
		if data.Error.Code == "" {
			data.Error.Code = "REGRESSION_INVALID_REQUEST"
		}
		return nil, &Error{Code: data.Error.Code, Status: response.StatusCode, Locale: c.Locale}
	}
	return response, nil
}
func (c *Client) read(ctx context.Context, method, path string, body any, out any) error {
	response, err := c.request(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if json.NewDecoder(response.Body).Decode(&envelope) != nil || !envelope.Success {
		return &Error{Code: "REGRESSION_INVALID_REQUEST", Locale: c.Locale}
	}
	return json.Unmarshal(envelope.Data, out)
}
func (c *Client) List(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	err := c.read(ctx, "GET", "/cases", nil, &out)
	return out, err
}
func (c *Client) Create(ctx context.Context, requestUUID, profile string, cases []string, inputs json.RawMessage) (Detail, error) {
	var out Detail
	if !ValidUUID(requestUUID) || !ValidUUID(profile) || len(cases) == 0 {
		return out, &Error{Code: "REGRESSION_INVALID_REQUEST", Locale: c.Locale}
	}
	for _, id := range cases {
		if !ValidUUID(id) {
			return out, &Error{Code: "REGRESSION_INVALID_REQUEST", Locale: c.Locale}
		}
	}
	body := map[string]any{"client_request_uuid": requestUUID, "profile_uuid": profile, "case_uuids": cases, "inputs": inputs}
	err := c.read(ctx, "POST", "/runs", body, &out)
	return out, err
}
func (c *Client) Get(ctx context.Context, id string) (Detail, error) {
	var out Detail
	if !ValidUUID(id) {
		return out, &Error{Code: "REGRESSION_INVALID_REQUEST", Locale: c.Locale}
	}
	err := c.read(ctx, "GET", "/runs/"+id, nil, &out)
	return out, err
}
func (c *Client) Report(ctx context.Context, id, format string) ([]byte, error) {
	if !ValidUUID(id) || (format != "json" && format != "junit") {
		return nil, &Error{Code: "REGRESSION_INVALID_REQUEST", Locale: c.Locale}
	}
	response, err := c.request(ctx, "GET", "/runs/"+id+"/report?format="+format, nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}
func (c *Client) Wait(ctx context.Context, d Detail) (Detail, error) {
	if !ValidUUID(d.Run.UUID) || !ValidUUID(d.Run.TenantUUID) || strings.TrimSpace(c.PluginID) == "" {
		return d, &Error{Code: "REGRESSION_INVALID_REQUEST", RunUUID: d.Run.UUID, Locale: c.Locale}
	}
	path := "/runs/" + d.Run.UUID + "/events"
	if d.LastEventUUID != nil {
		path += "?after_event_uuid=" + url.QueryEscape(*d.LastEventUUID)
	}
	response, err := c.request(ctx, "GET", path, nil)
	if err != nil {
		var coded *Error
		if errors.As(err, &coded) {
			coded.RunUUID = d.Run.UUID
		}
		return d, err
	}
	defer response.Body.Close()
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		return d, &Error{Code: "REGRESSION_STREAM_INTERRUPTED", RunUUID: d.Run.UUID, Locale: c.Locale}
	}
	scan := bufio.NewScanner(response.Body)
	scan.Buffer(make([]byte, 4096), 8*1024*1024)
	var data strings.Builder
	consume := func() error {
		if data.Len() == 0 {
			return nil
		}
		var envelope struct {
			Protocol   string `json:"protocol"`
			EventType  string `json:"event_type"`
			PluginID   string `json:"plugin_id"`
			TenantUUID string `json:"tenant_uuid"`
			TraceID    string `json:"trace_id"`
			RequestID  string `json:"request_id"`
			Payload    struct {
				UUID       string `json:"uuid"`
				TenantUUID string `json:"tenant_uuid"`
				RunUUID    string `json:"run_uuid"`
				Revision   uint64 `json:"revision"`
				Sequence   uint64 `json:"sequence"`
				Meta       struct {
					TenantUUID     string `json:"tenant_uuid"`
					SourcePlugin   string `json:"source_plugin"`
					RequestID      string `json:"request_id"`
					TraceID        string `json:"trace_id"`
					OccurredAt     string `json:"occurred_at"`
					PayloadVersion string `json:"payload_version"`
				} `json:"meta"`
				Payload struct {
					Run Run `json:"run"`
				} `json:"payload"`
			} `json:"payload"`
		}
		if json.Unmarshal([]byte(data.String()), &envelope) != nil {
			return &Error{Code: "REGRESSION_EVENT_META_INVALID", Locale: c.Locale}
		}
		data.Reset()
		p, m := envelope.Payload, envelope.Payload.Meta
		_, timeError := time.Parse(time.RFC3339Nano, m.OccurredAt)
		if envelope.Protocol != "sse" || envelope.EventType != "powerx.regression.run.updated.v1" || envelope.PluginID != c.PluginID || envelope.TenantUUID != d.Run.TenantUUID || !ValidUUID(p.UUID) || p.RunUUID != d.Run.UUID || p.TenantUUID != d.Run.TenantUUID || p.Sequence != p.Revision || p.Revision == 0 || m.TenantUUID != p.TenantUUID || m.SourcePlugin != c.PluginID || strings.TrimSpace(m.RequestID) == "" || strings.TrimSpace(m.TraceID) == "" || timeError != nil || !strings.HasSuffix(m.OccurredAt, "Z") || m.PayloadVersion != "v1" || envelope.RequestID != m.RequestID || envelope.TraceID != m.TraceID || p.Payload.Run.UUID != p.RunUUID || p.Payload.Run.TenantUUID != p.TenantUUID || p.Payload.Run.Revision != p.Revision {
			return &Error{Code: "REGRESSION_EVENT_META_INVALID", RunUUID: d.Run.UUID, Locale: c.Locale}
		}
		if p.Revision > d.Run.Revision {
			d.Run = p.Payload.Run
			d.LastEventUUID = &p.UUID
		}
		return nil
	}
	for scan.Scan() {
		line := scan.Text()
		if line == "" {
			if err := consume(); err != nil {
				return d, err
			}
		} else if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := consume(); err != nil {
		return d, err
	}
	if scan.Err() != nil || !terminal(d.Run.Phase) || d.Run.CleanupState == "running" {
		return d, &Error{Code: "REGRESSION_STREAM_INTERRUPTED", RunUUID: d.Run.UUID, Locale: c.Locale}
	}
	// One final snapshot read includes all case updates; this is not polling.
	final, err := c.Get(ctx, d.Run.UUID)
	if err != nil {
		var coded *Error
		if errors.As(err, &coded) {
			coded.RunUUID = d.Run.UUID
		}
		return d, err
	}
	d = final
	if d.Run.Verdict != "passed" {
		return d, &Error{Code: "REGRESSION_NOT_PASSED", RunUUID: d.Run.UUID, Locale: c.Locale}
	}
	return d, nil
}
func terminal(phase string) bool {
	return phase == "completed" || phase == "cancelled" || phase == "interrupted"
}
