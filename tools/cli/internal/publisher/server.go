package publisher

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const BootstrapTTL = 5 * time.Minute
const SessionTTL = time.Hour
const SessionFile = ".px-plugin/publisher/tool-session.json"
const directBuildFeature = "direct-build-apple-silicon"

type ServerOptions struct{ Listen, Origin string }
type ServerInfo struct {
	Protocol           string    `json:"protocol"`
	InstanceID         string    `json:"instance_id"`
	Endpoint           string    `json:"endpoint"`
	SessionFile        string    `json:"session_file"`
	BootstrapExpiresAt time.Time `json:"bootstrap_expires_at"`
	PID                int       `json:"pid"`
}
type Bootstrap struct {
	ServerInfo
	BootstrapToken string `json:"bootstrap_token"`
}
type Session struct {
	Token     string    `json:"session_token"`
	ExpiresAt time.Time `json:"expires_at"`
}
type LocalInfo struct {
	Protocol         string           `json:"protocol"`
	Plugin           Plugin           `json:"plugin"`
	Features         []string         `json:"features"`
	BuildTargets     []Target         `json:"build_targets"`
	BuildEnvironment BuildEnvironment `json:"build_environment"`
}

type Server struct {
	info    ServerInfo
	engine  *Engine
	handler *localHandler
	http    *http.Server
	cancel  context.CancelFunc
	done    chan struct{}
	err     error
}

type localHandler struct {
	engine                           *Engine
	host, origin                     string
	mu                               sync.Mutex
	bootstrapHash, sessionHash       [32]byte
	bootstrapExpires, sessionExpires time.Time
	consumed                         bool
	now                              func() time.Time
}

func validateEndpoint(listen, origin string) error {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return invalid("listen")
	}
	ip := net.ParseIP(host)
	number, portErr := strconv.Atoi(port)
	if ip == nil || !ip.IsLoopback() || portErr != nil || number < 0 || number > 65535 {
		return invalid("listen")
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" && u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Path != "" || u.Opaque != "" {
		return invalid("origin")
	}
	originIP := net.ParseIP(u.Hostname())
	if u.Hostname() != "localhost" && (originIP == nil || !originIP.IsLoopback()) {
		return invalid("origin")
	}
	if u.Host == "" || strings.HasSuffix(u.Host, ":") || strings.ContainsAny(origin, "\r\n") || u.String() != origin {
		return invalid("origin")
	}
	if p := u.Port(); p != "" {
		n, e := strconv.Atoi(p)
		if e != nil || n < 1 || n > 65535 {
			return invalid("origin")
		}
	}
	return nil
}
func secret() (string, error) {
	var value [32]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", failure(StorageFailure, err)
	}
	return base64.RawURLEncoding.EncodeToString(value[:]), nil
}

// Start binds only loopback and serves the fixed local publisher API.
// Arbitrary shell and business-runtime routes are absent.
func Start(ctx context.Context, engine *Engine, opts ServerOptions) (*Server, error) {
	if err := validateEndpoint(opts.Listen, opts.Origin); err != nil {
		return nil, err
	}
	release, err := engine.state.lock()
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", opts.Listen)
	if err != nil {
		release()
		return nil, failure(StorageFailure, err)
	}
	token, err := secret()
	if err != nil {
		listener.Close()
		release()
		return nil, err
	}
	info := ServerInfo{Protocol: Protocol, InstanceID: uuid.NewString(), Endpoint: "http://" + listener.Addr().String(), SessionFile: SessionFile, BootstrapExpiresAt: time.Now().UTC().Add(BootstrapTTL), PID: os.Getpid()}
	bootstrap := Bootstrap{ServerInfo: info, BootstrapToken: token}
	data, err := json.Marshal(bootstrap)
	if err == nil {
		err = engine.state.write("tool-session.json", append(data, '\n'), true)
	}
	if err != nil {
		listener.Close()
		release()
		return nil, AsError(err)
	}
	handler := &localHandler{engine: engine, host: listener.Addr().String(), origin: opts.Origin, bootstrapHash: sha256.Sum256([]byte(token)), bootstrapExpires: info.BootstrapExpiresAt, now: time.Now}
	serveCtx, cancel := context.WithCancel(ctx)
	srv := &Server{info: info, engine: engine, handler: handler, cancel: cancel, done: make(chan struct{})}
	srv.http = &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	served := make(chan error, 1)
	go func() { served <- srv.http.Serve(listener) }()
	go func() {
		var serveErr error
		received := false
		select {
		case <-serveCtx.Done():
		case serveErr = <-served:
			received = true
		}
		shutdownCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		shutdownErr := srv.http.Shutdown(shutdownCtx)
		stop()
		if shutdownErr != nil {
			_ = srv.http.Close()
		}
		if !received {
			serveErr = <-served
		}
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		srv.engine.cancelBuilds()
		if cleanup := srv.removeOwnedBootstrap(); serveErr == nil {
			serveErr = cleanup
		}
		if serveErr == nil {
			serveErr = shutdownErr
		}
		release()
		srv.err = serveErr
		cancel()
		close(srv.done)
	}()
	return srv, nil
}
func (s *Server) Info() ServerInfo { s.handler.mu.Lock(); defer s.handler.mu.Unlock(); return s.info }

// RenewBootstrap is invoked only by the managed parent's private OS signal;
// it has no public HTTP route and does not cancel jobs or change operation IDs.
func (s *Server) RenewBootstrap() error {
	token, err := secret()
	if err != nil {
		return err
	}
	s.handler.mu.Lock()
	defer s.handler.mu.Unlock()
	info := s.info
	info.BootstrapExpiresAt = s.handler.now().UTC().Add(BootstrapTTL)
	raw, err := json.Marshal(Bootstrap{ServerInfo: info, BootstrapToken: token})
	if err != nil {
		return failure(StorageFailure, err)
	}
	if err = s.engine.state.write("tool-session.json", append(raw, '\n'), true); err != nil {
		return err
	}
	s.info.BootstrapExpiresAt = info.BootstrapExpiresAt
	s.handler.bootstrapHash = sha256.Sum256([]byte(token))
	s.handler.bootstrapExpires = info.BootstrapExpiresAt
	s.handler.consumed = false
	return nil
}
func (s *Server) Wait() error  { <-s.done; return s.err }
func (s *Server) Close() error { s.cancel(); return s.Wait() }
func (s *Server) removeOwnedBootstrap() error {
	data, err := s.engine.state.read("tool-session.json", MaxRequestBytes)
	if err != nil {
		if AsError(err).Code == NotFound {
			return nil
		}
		return err
	}
	var b Bootstrap
	if decodeStrict(data, &b) != nil {
		return failure(InvalidRecord, nil)
	}
	if b.InstanceID != s.info.InstanceID {
		return nil
	}
	return s.engine.state.remove("tool-session.json")
}

func (h *localHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Host != h.host || len(r.Header.Values("Origin")) != 1 || r.Header.Get("Origin") != h.origin {
		writeError(w, failure(Forbidden, nil))
		return
	}
	if r.URL.RawQuery != "" && r.URL.Path != "/v1/plans" && r.URL.Path != "/v1/builds" && !(r.Method == http.MethodGet && (strings.HasPrefix(r.URL.Path, "/v1/uploads/") || strings.HasPrefix(r.URL.Path, "/v1/submissions/")) && strings.HasSuffix(r.URL.Path, "/requests")) && !(strings.HasPrefix(r.URL.Path, "/v1/markets/") && (strings.HasSuffix(r.URL.Path, "/versions") || strings.HasSuffix(r.URL.Path, "/publications"))) {
		writeError(w, invalid("query"))
		return
	}
	if len(r.Header.Values("Authorization")) != 1 {
		writeError(w, failure(Unauthorized, nil))
		return
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") || len(auth) != len("Bearer ")+43 {
		writeError(w, failure(Unauthorized, nil))
		return
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if r.URL.Path == "/v1/session" {
		if r.Method != http.MethodPost {
			writeError(w, failure(Method, nil))
			return
		}
		if r.ContentLength != 0 {
			writeError(w, invalid("body"))
			return
		}
		h.exchange(w, token)
		return
	}
	h.mu.Lock()
	valid := h.now().Before(h.sessionExpires) && subtle.ConstantTimeCompare(h.sessionHash[:], hashToken(token)) == 1
	h.mu.Unlock()
	if !valid {
		writeError(w, failure(Unauthorized, nil))
		return
	}
	switch {
	case r.URL.Path == "/v1/info":
		if r.Method != http.MethodGet {
			writeError(w, failure(Method, nil))
			return
		}
		writeData(w, http.StatusOK, LocalInfo{Protocol: Protocol, Plugin: h.engine.Plugin(), Features: []string{"release-plans", "local-builds", "market-connections", "package-upload", "operation-recovery", "publication-submit", "managed-executor", "container-build", "credential-bind", directBuildFeature}, BuildTargets: h.engine.BuildTargets(), BuildEnvironment: h.engine.BuildEnvironment()})
	case r.URL.Path == "/v1/build-environment/start":
		if r.Method != http.MethodPost || r.ContentLength != 0 || r.URL.RawQuery != "" {
			writeError(w, invalid("body"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		value, err := h.engine.StartBuildEnvironment(ctx)
		if err != nil {
			writeError(w, AsError(err))
			return
		}
		writeData(w, http.StatusOK, value)
	case r.URL.Path == "/v1/submissions" || strings.HasPrefix(r.URL.Path, "/v1/submissions/"):
		h.serveSubmission(w, r)
	case r.URL.Path == "/v1/markets" || strings.HasPrefix(r.URL.Path, "/v1/markets/") || r.URL.Path == "/v1/uploads" || strings.HasPrefix(r.URL.Path, "/v1/uploads/"):
		h.serveMarket(w, r)
	case r.URL.Path == "/v1/builds" || strings.HasPrefix(r.URL.Path, "/v1/builds/"):
		h.serveBuild(w, r)
	case r.URL.Path == "/v1/plans" && r.Method == http.MethodPost:
		if r.URL.RawQuery != "" {
			writeError(w, invalid("query"))
			return
		}
		contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || contentType != "application/json" {
			writeError(w, failure(MediaType, nil))
			return
		}
		data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxRequestBytes))
		if err != nil {
			var large *http.MaxBytesError
			if errors.As(err, &large) {
				writeError(w, failure(TooLarge, nil))
			} else {
				writeError(w, invalid("body"))
			}
			return
		}
		var input CreatePlan
		if err = decodeStrict(data, &input); err != nil {
			writeError(w, invalid("body"))
			return
		}
		plan, err := h.engine.Create(input)
		if err != nil {
			writeError(w, AsError(err))
			return
		}
		writeData(w, http.StatusCreated, plan)
	case r.URL.Path == "/v1/plans" && r.Method == http.MethodGet:
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			writeError(w, invalid("pagination"))
			return
		}
		for key, values := range query {
			if key != "page" && key != "size" || len(values) != 1 {
				writeError(w, invalid("pagination"))
				return
			}
		}
		page, size := 1, DefaultPageSize
		if value, ok := query["page"]; ok {
			page, err = strconv.Atoi(value[0])
			if err != nil {
				writeError(w, invalid("pagination"))
				return
			}
		}
		if value, ok := query["size"]; ok {
			size, err = strconv.Atoi(value[0])
			if err != nil {
				writeError(w, invalid("pagination"))
				return
			}
		}
		out, err := h.engine.List(page, size)
		if err != nil {
			writeError(w, AsError(err))
			return
		}
		writeData(w, http.StatusOK, out)
	case strings.HasPrefix(r.URL.Path, "/v1/plans/"):
		if r.Method != http.MethodGet {
			writeError(w, failure(Method, nil))
			return
		}
		plan, err := h.engine.Get(strings.TrimPrefix(r.URL.Path, "/v1/plans/"))
		if err != nil {
			writeError(w, AsError(err))
			return
		}
		writeData(w, http.StatusOK, plan)
	case r.URL.Path == "/v1/plans":
		writeError(w, failure(Method, nil))
	default:
		writeError(w, failure(NotFound, nil))
	}
}
func hashToken(token string) []byte { hash := sha256.Sum256([]byte(token)); return hash[:] }
func (h *localHandler) exchange(w http.ResponseWriter, token string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.consumed || !h.now().Before(h.bootstrapExpires) || subtle.ConstantTimeCompare(h.bootstrapHash[:], hashToken(token)) != 1 {
		writeError(w, failure(Unauthorized, nil))
		return
	}
	session, err := secret()
	if err != nil {
		writeError(w, AsError(err))
		return
	}
	h.consumed = true
	h.bootstrapHash = [32]byte{}
	h.sessionHash = sha256.Sum256([]byte(session))
	h.sessionExpires = h.now().UTC().Add(SessionTTL)
	writeData(w, http.StatusCreated, Session{Token: session, ExpiresAt: h.sessionExpires})
}
func writeData(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Data any `json:"data"`
	}{value})
}
func writeError(w http.ResponseWriter, err *Error) {
	status := http.StatusInternalServerError
	switch err.Code {
	case InvalidInput, InvalidWorkspace:
		status = http.StatusBadRequest
	case Unauthorized:
		status = http.StatusUnauthorized
	case Forbidden:
		status = http.StatusForbidden
	case NotFound:
		status = http.StatusNotFound
	case Conflict, WorkspaceChanged, BuildBusy, SourceChanged, ArtifactChanged, SubmissionChanged, SubmissionState:
		status = http.StatusConflict
	case TooLarge:
		status = http.StatusRequestEntityTooLarge
	case MediaType:
		status = http.StatusUnsupportedMediaType
	case Method:
		status = http.StatusMethodNotAllowed
	case ArtifactInvalid, SourceUnsafe, NativeUnsupported:
		status = http.StatusUnprocessableEntity
	case SubmissionCapability:
		status = http.StatusServiceUnavailable
	case Unsupported, BuildPlatform:
		status = http.StatusNotImplemented
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error *Error `json:"error"`
	}{err})
}
func (h *localHandler) serveBuild(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/v1/builds" {
		switch r.Method {
		case http.MethodPost:
			if r.URL.RawQuery != "" {
				writeError(w, invalid("query"))
				return
			}
			t, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || t != "application/json" {
				writeError(w, failure(MediaType, nil))
				return
			}
			data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxRequestBytes))
			if err != nil {
				writeError(w, failure(TooLarge, nil))
				return
			}
			var input struct {
				PlanID   string `json:"release_plan_id"`
				Executor string `json:"executor"`
			}
			if decodeStrict(data, &input) != nil {
				writeError(w, invalid("body"))
				return
			}
			b, err := h.engine.StartBuildWithExecutor(context.Background(), input.PlanID, input.Executor)
			if err != nil {
				writeError(w, AsError(err))
				return
			}
			writeData(w, http.StatusAccepted, b)
		case http.MethodGet:
			q, err := url.ParseQuery(r.URL.RawQuery)
			if err != nil {
				writeError(w, invalid("pagination"))
				return
			}
			page, size := 1, DefaultPageSize
			for key, values := range q {
				if len(values) != 1 || key != "page" && key != "size" {
					writeError(w, invalid("pagination"))
					return
				}
				n, err := strconv.Atoi(values[0])
				if err != nil {
					writeError(w, invalid("pagination"))
					return
				}
				if key == "page" {
					page = n
				} else {
					size = n
				}
			}
			b, err := h.engine.ListBuilds(page, size)
			if err != nil {
				writeError(w, AsError(err))
				return
			}
			writeData(w, http.StatusOK, b)
		default:
			writeError(w, failure(Method, nil))
		}
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/builds/"), "/")
	if len(parts) > 2 || !canonicalUUID(parts[0]) {
		writeError(w, invalid("build_id"))
		return
	}
	id := parts[0]
	action := "get"
	if len(parts) == 2 {
		action = parts[1]
	}
	if action == "cancel" || action == "verify" {
		if r.Method != http.MethodPost {
			writeError(w, failure(Method, nil))
			return
		}
		if r.ContentLength != 0 {
			writeError(w, invalid("body"))
			return
		}
	} else if r.Method != http.MethodGet {
		writeError(w, failure(Method, nil))
		return
	}
	switch action {
	case "get":
		b, err := h.engine.GetBuild(id)
		if err != nil {
			writeError(w, AsError(err))
			return
		}
		writeData(w, http.StatusOK, b)
	case "cancel":
		b, err := h.engine.CancelBuild(id)
		if err != nil {
			writeError(w, AsError(err))
			return
		}
		writeData(w, http.StatusOK, b)
	case "verify":
		a, err := h.engine.VerifyArtifact(id)
		if err != nil {
			writeError(w, AsError(err))
			return
		}
		writeData(w, http.StatusOK, a)
	case "logs":
		text, err := h.engine.BuildLog(id)
		if err != nil {
			writeError(w, AsError(err))
			return
		}
		writeData(w, http.StatusOK, struct {
			Text string `json:"text"`
		}{text})
	default:
		writeError(w, failure(NotFound, nil))
	}
}
