package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const sessionRoot = "/api/v1/tenant/agent/sessions"

// SessionService is the plugin-owned service-session contract. Implementations
// must isolate trusted tenant/service actors; identity overrides are not inputs.
type SessionService interface {
	CreateSession(context.Context, CreateSessionInput) (*ServiceSession, error)
	ListSessions(context.Context, SessionPage) (*ServiceSessionList, error)
	GetSession(context.Context, string) (*ServiceSession, error)
	RenameSession(context.Context, string, string) (*ServiceSession, error)
	ArchiveSession(context.Context, string) (*ServiceSession, error)
	DeleteSession(context.Context, string) (*ServiceSession, error)
	AppendSessionMessage(context.Context, string, string, AppendSessionMessageInput) (*ServiceMessage, error)
	ListSessionMessages(context.Context, string, SessionPage) (*ServiceMessageList, error)
	InvokeSession(context.Context, string, string, string) (*ServiceInvocation, error)
	GetSessionInvocation(context.Context, string, string) (*ServiceInvocation, error)
	CancelSessionInvocation(context.Context, string, string) (*ServiceInvocation, error)
	StreamSessionEvents(context.Context, string, string, func(SessionEvent) error) error
}

type CreateSessionInput struct {
	AgentUUID string `json:"agent_uuid"`
	Title     string `json:"title"`
}
type AppendSessionMessageInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type SessionPage struct {
	Page     int
	PageSize int
}
type ServiceSession struct {
	SessionUUID string    `json:"session_uuid"`
	AgentUUID   string    `json:"agent_uuid"`
	Title       string    `json:"title"`
	Status      string    `json:"status"`
	Revision    int64     `json:"revision"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type ServiceMessage struct {
	MessageUUID string    `json:"message_uuid"`
	SessionUUID string    `json:"session_uuid"`
	Role        string    `json:"role"`
	Content     string    `json:"content"`
	Sequence    int64     `json:"sequence"`
	CreatedAt   time.Time `json:"created_at"`
}
type ServiceInvocation struct {
	InvocationUUID string     `json:"invocation_uuid"`
	SessionUUID    string     `json:"session_uuid"`
	MessageUUID    string     `json:"message_uuid"`
	TraceUUID      string     `json:"trace_uuid"`
	Status         string     `json:"status"`
	ReasonCode     string     `json:"reason_code"`
	Output         string     `json:"output"`
	CreatedAt      time.Time  `json:"created_at"`
	DeadlineAt     time.Time  `json:"deadline_at"`
	FinishedAt     *time.Time `json:"finished_at"`
}
type ServiceSessionList struct {
	Items    []ServiceSession `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}
type ServiceMessageList struct {
	Items    []ServiceMessage `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

func sessionError(status int, reason string) error {
	return &Error{StatusCode: status, Code: reason, ReasonCode: reason, Message: reason}
}
func sessionInvalid() error    { return sessionError(400, "AGENT_SESSION_INVALID_ARGUMENT") }
func sessionDependency() error { return sessionError(502, "AGENT_SESSION_UPSTREAM_DEPENDENCY") }
func sessionUUID(s string) bool {
	u, e := uuid.Parse(s)
	return e == nil && u != uuid.Nil && u.String() == s
}
func sessionPath(id string) (string, error) {
	if !sessionUUID(id) {
		return "", sessionInvalid()
	}
	return sessionRoot + "/" + id, nil
}
func invocationPath(s, i string) (string, error) {
	p, e := sessionPath(s)
	if e != nil {
		return "", e
	}
	if !sessionUUID(i) {
		return "", sessionInvalid()
	}
	return p + "/invocations/" + i, nil
}
func sessionPage(p SessionPage) (string, error) {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PageSize == 0 {
		p.PageSize = 20
	}
	if p.Page < 1 || p.Page > 1000000 || p.PageSize < 1 || p.PageSize > 100 {
		return "", sessionInvalid()
	}
	return fmt.Sprintf("?page=%d&page_size=%d", p.Page, p.PageSize), nil
}
func sessionKey(k string) bool {
	return len(k) > 0 && len(k) <= 128 && strings.TrimSpace(k) == k && !strings.ContainsAny(k, "\r\n")
}

func (c *Client) sessionRequest(ctx context.Context, method, path, key string, body any) (*http.Response, error) {
	return c.sessionRequestWithCursor(ctx, method, path, key, body, "")
}

// sessionRequestWithCursor is restricted to the idempotent event subscription.
// The cursor is an acknowledged SSE frame, never an invocation input.
func (c *Client) sessionRequestWithCursor(ctx context.Context, method, path, key string, body any, cursor string) (*http.Response, error) {
	if c == nil || c.cfg.Mode != ModeDelegated || c.tokens == nil || c.http == nil {
		return nil, sessionError(503, "AGENT_SESSION_UNAVAILABLE")
	}
	tokenValue := reflect.ValueOf(c.tokens)
	switch tokenValue.Kind() {
	case reflect.Pointer, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan:
		if tokenValue.IsNil() {
			return nil, sessionError(503, "AGENT_SESSION_UNAVAILABLE")
		}
	}
	var b io.Reader
	if body != nil {
		raw, e := json.Marshal(body)
		if e != nil {
			return nil, sessionInvalid()
		}
		b = bytes.NewReader(raw)
	}
	req, e := http.NewRequestWithContext(ctx, method, c.url(path), b)
	if e != nil {
		return nil, sessionInvalid()
	}
	token, e := c.tokens.Token(ctx)
	if e != nil || strings.TrimSpace(token) == "" {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, sessionError(503, "AGENT_SESSION_UPSTREAM_DEPENDENCY")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	if strings.HasSuffix(path, "/events") {
		req.Header.Set("Accept", "text/event-stream")
		if cursor != "" {
			req.Header.Set("Last-Event-ID", cursor)
		}
	}
	client := *c.http
	// A Host Contract is a fixed route. Never redirect service credentials or
	// repeat a mutation at a location supplied by the response.
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, e := client.Do(req)
	if e != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, sessionDependency()
	}
	return resp, nil
}

func sessionCall[T any](ctx context.Context, c *Client, method, path, key string, body any, status int) (*T, error) {
	resp, e := c.sessionRequest(ctx, method, path, key, body)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
	if e != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, sessionDependency()
	}
	if resp.StatusCode >= 400 {
		return nil, transportErrorPayload(resp, raw)
	}
	if resp.StatusCode != status || len(raw) > 8<<20 {
		return nil, sessionDependency()
	}
	var envelope struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Code != status || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil, sessionDependency()
	}
	var out T
	if json.Unmarshal(envelope.Data, &out) != nil {
		return nil, sessionDependency()
	}
	if v, ok := any(&out).(interface{ valid() bool }); ok && !v.valid() {
		return nil, sessionDependency()
	}
	resource := strings.Split(strings.Split(strings.TrimPrefix(path, sessionRoot+"/"), "?")[0], "/")
	if len(resource) > 0 && sessionUUID(resource[0]) {
		id := resource[0]
		switch v := any(&out).(type) {
		case *ServiceSession:
			if v.SessionUUID != id {
				return nil, sessionDependency()
			}
		case *ServiceMessage:
			if v.SessionUUID != id {
				return nil, sessionDependency()
			}
		case *ServiceMessageList:
			for _, m := range v.Items {
				if m.SessionUUID != id {
					return nil, sessionDependency()
				}
			}
		case *ServiceInvocation:
			if v.SessionUUID != id || (len(resource) >= 3 && resource[1] == "invocations" && v.InvocationUUID != resource[2]) {
				return nil, sessionDependency()
			}
		}
	}
	return &out, nil
}
func (s *ServiceSession) valid() bool {
	return sessionUUID(s.SessionUUID) && sessionUUID(s.AgentUUID) && s.Revision > 0 && !s.CreatedAt.IsZero() && !s.UpdatedAt.IsZero() && (s.Status == "active" || s.Status == "archived" || s.Status == "deleted")
}
func (m *ServiceMessage) valid() bool {
	return sessionUUID(m.MessageUUID) && sessionUUID(m.SessionUUID) && m.Sequence > 0 && !m.CreatedAt.IsZero() && (m.Role == "user" || m.Role == "assistant")
}
func (i *ServiceInvocation) valid() bool {
	return sessionUUID(i.InvocationUUID) && sessionUUID(i.SessionUUID) && sessionUUID(i.MessageUUID) && sessionUUID(i.TraceUUID) && !i.CreatedAt.IsZero() && !i.DeadlineAt.IsZero() && validInvocationStatus(i.Status)
}
func validInvocationStatus(s string) bool {
	return s == "running" || s == "cancelling" || s == "succeeded" || s == "failed" || s == "cancelled"
}
func (l *ServiceSessionList) valid() bool {
	if l.Items == nil || l.Page < 1 || l.PageSize < 1 || l.Total < 0 {
		return false
	}
	for _, v := range l.Items {
		if !v.valid() {
			return false
		}
	}
	return true
}
func (l *ServiceMessageList) valid() bool {
	if l.Items == nil || l.Page < 1 || l.PageSize < 1 || l.Total < 0 {
		return false
	}
	for _, v := range l.Items {
		if !v.valid() {
			return false
		}
	}
	return true
}
func (c *Client) CreateSession(ctx context.Context, in CreateSessionInput) (*ServiceSession, error) {
	if !sessionUUID(in.AgentUUID) || !utf8.ValidString(in.Title) || utf8.RuneCountInString(in.Title) > 255 {
		return nil, sessionInvalid()
	}
	return sessionCall[ServiceSession](ctx, c, "POST", sessionRoot, "", in, 201)
}
func (c *Client) ListSessions(ctx context.Context, page SessionPage) (*ServiceSessionList, error) {
	q, e := sessionPage(page)
	if e != nil {
		return nil, e
	}
	return sessionCall[ServiceSessionList](ctx, c, "GET", sessionRoot+q, "", nil, 200)
}
func (c *Client) GetSession(ctx context.Context, id string) (*ServiceSession, error) {
	p, e := sessionPath(id)
	if e != nil {
		return nil, e
	}
	return sessionCall[ServiceSession](ctx, c, "GET", p, "", nil, 200)
}
func (c *Client) RenameSession(ctx context.Context, id, title string) (*ServiceSession, error) {
	p, e := sessionPath(id)
	if e != nil {
		return nil, e
	}
	if !utf8.ValidString(title) || utf8.RuneCountInString(title) > 255 {
		return nil, sessionInvalid()
	}
	return sessionCall[ServiceSession](ctx, c, "PATCH", p, "", struct {
		Title string `json:"title"`
	}{title}, 200)
}
func (c *Client) ArchiveSession(ctx context.Context, id string) (*ServiceSession, error) {
	p, e := sessionPath(id)
	if e != nil {
		return nil, e
	}
	return sessionCall[ServiceSession](ctx, c, "POST", p+"/archive", "", nil, 200)
}
func (c *Client) DeleteSession(ctx context.Context, id string) (*ServiceSession, error) {
	p, e := sessionPath(id)
	if e != nil {
		return nil, e
	}
	return sessionCall[ServiceSession](ctx, c, "DELETE", p, "", nil, 200)
}
func (c *Client) AppendSessionMessage(ctx context.Context, id, key string, in AppendSessionMessageInput) (*ServiceMessage, error) {
	p, e := sessionPath(id)
	if e != nil {
		return nil, e
	}
	if !sessionKey(key) || strings.HasPrefix(key, "invoke:") || in.Role != "user" || strings.TrimSpace(in.Content) == "" || len(in.Content) > 65536 || !utf8.ValidString(in.Content) {
		return nil, sessionInvalid()
	}
	return sessionCall[ServiceMessage](ctx, c, "POST", p+"/messages", key, in, 201)
}
func (c *Client) ListSessionMessages(ctx context.Context, id string, page SessionPage) (*ServiceMessageList, error) {
	p, e := sessionPath(id)
	if e != nil {
		return nil, e
	}
	q, e := sessionPage(page)
	if e != nil {
		return nil, e
	}
	return sessionCall[ServiceMessageList](ctx, c, "GET", p+"/messages"+q, "", nil, 200)
}

// InvokeSession never appends a message or retries execution implicitly.
func (c *Client) InvokeSession(ctx context.Context, id, message, key string) (*ServiceInvocation, error) {
	p, e := sessionPath(id)
	if e != nil {
		return nil, e
	}
	if !sessionUUID(message) || !sessionKey(key) {
		return nil, sessionInvalid()
	}
	return sessionCall[ServiceInvocation](ctx, c, "POST", p+"/invocations", key, struct {
		MessageUUID string `json:"message_uuid"`
	}{message}, 202)
}
func (c *Client) GetSessionInvocation(ctx context.Context, id, invocation string) (*ServiceInvocation, error) {
	p, e := invocationPath(id, invocation)
	if e != nil {
		return nil, e
	}
	return sessionCall[ServiceInvocation](ctx, c, "GET", p, "", nil, 200)
}
func (c *Client) CancelSessionInvocation(ctx context.Context, id, invocation string) (*ServiceInvocation, error) {
	p, e := invocationPath(id, invocation)
	if e != nil {
		return nil, e
	}
	return sessionCall[ServiceInvocation](ctx, c, "POST", p+"/cancel", "", nil, 202)
}

var _ SessionService = (*Client)(nil)
