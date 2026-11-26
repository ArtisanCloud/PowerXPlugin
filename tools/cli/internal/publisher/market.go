package publisher

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/powerx-plugin/cli/internal/publisher/wire"
)

var publishTokenPattern = regexp.MustCompile(`^pxpub_[A-Za-z0-9_-]{43}$`)

func fmtInt(n int) string { return strconv.Itoa(n) }

const MarketContractVersion = "1.0.0-rc.2"
const MarketBundleSHA256 = "89141c00e0b99b6a803312119e4cbcb3e64224bf458e234e35fc5ffbdeacd8fa"

const MarketUnavailable = "PUBLISHER_MARKET_UNAVAILABLE"
const MarketProtocol = "PUBLISHER_MARKET_PROTOCOL"
const MarketCapability = "PUBLISHER_MARKET_CAPABILITY"
const MarketCredential = "PUBLISHER_MARKET_CREDENTIAL"
const MarketIdentity = "PUBLISHER_MARKET_IDENTITY"
const MarketKeyring = "PUBLISHER_MARKET_KEYRING"
const MarketPending = "PUBLISHER_MARKET_PENDING"
const MarketRejected = "PUBLISHER_MARKET_REJECTED"

type MarketInput struct {
	Name        string `json:"name"`
	APIRoot     string `json:"api_root"`
	ConsoleBase string `json:"console_base"`
}
type MarketIdentityInfo struct {
	ID          string `json:"user_id,omitempty"`
	DeveloperID string `json:"developer_id,omitempty"`
	Name        string `json:"name"`
}
type MarketMe struct {
	User       MarketIdentityInfo `json:"user"`
	Developer  MarketIdentityInfo `json:"developer"`
	Credential struct {
		TokenID   string    `json:"token_id"`
		PluginIDs []string  `json:"plugin_ids"`
		Scopes    []string  `json:"scopes"`
		ExpiresAt time.Time `json:"expires_at"`
	} `json:"credential"`
	ServerTime time.Time `json:"server_time"`
}
type Market struct {
	ContractVersion string `json:"contract_version"`
	BundleSHA256    string `json:"bundle_sha256"`
	ConnectionID    string `json:"connection_id"`
	MarketInput
	Identity   *MarketMe  `json:"identity,omitempty"`
	Features   []string   `json:"features"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
}
type marketDiscovery struct {
	Features    []string `json:"features"`
	ConsoleBase string   `json:"console_base_url"`
}
type MarketArtifact struct {
	ArtifactID string `json:"artifact_id"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	SHA256     string `json:"sha256"`
	Size       int64  `json:"size"`
}
type UploadReceipt struct {
	OperationID string         `json:"operation_id"`
	PluginID    string         `json:"plugin_id"`
	VersionID   string         `json:"version_id"`
	Version     string         `json:"version"`
	Artifact    MarketArtifact `json:"artifact"`
	URL         string         `json:"market_version_url"`
}
type MarketVersion struct {
	VersionID    string           `json:"version_id"`
	PluginID     string           `json:"plugin_id"`
	Version      string           `json:"version"`
	Channel      string           `json:"channel"`
	Status       string           `json:"availability_status"`
	ReleaseNotes string           `json:"release_notes"`
	MinCore      string           `json:"min_core"`
	CreatedAt    time.Time        `json:"created_at"`
	PublishedAt  *time.Time       `json:"published_at"`
	Artifacts    []MarketArtifact `json:"artifacts"`
	URL          string           `json:"market_version_url"`
}
type marketOperation struct {
	OperationID string         `json:"operation_id"`
	Type        string         `json:"type"`
	PluginID    string         `json:"plugin_id"`
	SHA256      string         `json:"request_sha256"`
	Status      string         `json:"status"`
	RetryAfter  int            `json:"retry_after_seconds"`
	Result      *UploadReceipt `json:"result"`
	Error       *marketError   `json:"error"`
}
type marketError struct {
	Status     int    `json:"code"`
	Code       string `json:"error"`
	Retryable  bool   `json:"retryable"`
	RetryAfter int    `json:"-"`
}

func (m *marketError) Error() string { return MarketRejected }

// Connections are immutable trust boundaries. A different API/console requires a
// new UUID and credential binding; a server-supplied URL never grants trust.
func marketURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || strings.HasSuffix(raw, "/") || u.String() != raw {
		return false
	}
	if !wire.TrustedMarketLink(raw, raw) {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	ip := net.ParseIP(u.Hostname())
	return u.Scheme == "http" && ip != nil && ip.IsLoopback()
}
func (e *Engine) marketDir() (*directory, error) { return e.state.child("markets", true) }
func (e *Engine) CreateMarket(input MarketInput) (Market, error) {
	if strings.TrimSpace(input.Name) == "" || len(input.Name) > 200 || !utf8.ValidString(input.Name) || strings.ContainsAny(input.Name, "\x00\r\n") || !marketURL(input.APIRoot) || !marketURL(input.ConsoleBase) {
		return Market{}, invalid("market")
	}
	m := Market{ContractVersion: MarketContractVersion, BundleSHA256: MarketBundleSHA256, ConnectionID: uuid.NewString(), MarketInput: input, Features: []string{}}
	return m, e.saveMarket(m, false)
}
func (e *Engine) saveMarket(m Market, replace bool) error {
	d, err := e.marketDir()
	if err != nil {
		return err
	}
	defer d.close()
	b, err := json.Marshal(m)
	if int64(len(b)) > MaxRecordBytes {
		return failure(TooLarge, nil)
	}
	if err != nil {
		return failure(StorageFailure, err)
	}
	return d.write(m.ConnectionID+".json", b, replace)
}
func (e *Engine) GetMarket(id string) (Market, error) {
	if !canonicalUUID(id) {
		return Market{}, invalid("connection_id")
	}
	d, err := e.marketDir()
	if err != nil {
		return Market{}, err
	}
	defer d.close()
	b, err := d.read(id+".json", MaxRecordBytes)
	var m Market
	if err != nil {
		return m, err
	}
	if decodeStrict(b, &m) != nil || m.ContractVersion != MarketContractVersion || m.BundleSHA256 != MarketBundleSHA256 || m.ConnectionID != id || !marketURL(m.APIRoot) || !marketURL(m.ConsoleBase) {
		return Market{}, failure(InvalidRecord, nil)
	}
	return m, nil
}
func (e *Engine) ListMarkets() ([]Market, error) {
	d, err := e.marketDir()
	if err != nil {
		return nil, err
	}
	defer d.close()
	names, err := d.names()
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	out := []Market{}
	for _, name := range names {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		m, err := e.GetMarket(strings.TrimSuffix(name, ".json"))
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}
func (e *Engine) lockMarket(id string) (func(), error) {
	if !canonicalUUID(id) {
		return nil, invalid("connection_id")
	}
	d, err := e.marketDir()
	if err != nil {
		return nil, err
	}
	sub, err := d.child(id, true)
	d.close()
	if err != nil {
		return nil, err
	}
	release, err := sub.lock()
	if err != nil {
		sub.close()
		return nil, failure(Conflict, err)
	}
	return func() { release(); sub.close() }, nil
}
func (e *Engine) BindMarket(ctx context.Context, id, token string) (Market, error) {
	if !publishTokenPattern.MatchString(token) {
		return Market{}, failure(MarketCredential, nil)
	}
	unlock, lockErr := e.lockMarket(id)
	if lockErr != nil {
		return Market{}, lockErr
	}
	defer unlock()
	m, err := e.GetMarket(id)
	if err != nil {
		return m, err
	}
	verified, err := e.verifyMarket(ctx, m, token)
	if err != nil {
		return m, err
	}
	serialized, serializeErr := json.Marshal(verified)
	if serializeErr != nil {
		return m, failure(StorageFailure, serializeErr)
	}
	if int64(len(serialized)) > MaxRecordBytes {
		return m, failure(TooLarge, nil)
	}
	if err = e.credentials.Set(ctx, id, token); err != nil {
		return m, failure(MarketKeyring, err)
	}
	if err = e.saveMarket(verified, true); err != nil {
		return m, err
	}
	return verified, nil
}
func (e *Engine) VerifyMarket(ctx context.Context, id string) (Market, error) {
	unlock, lockErr := e.lockMarket(id)
	if lockErr != nil {
		return Market{}, lockErr
	}
	defer unlock()
	m, err := e.GetMarket(id)
	if err != nil {
		return m, err
	}
	token, err := e.credentials.Get(ctx, id)
	if err != nil {
		return m, failure(MarketKeyring, err)
	}
	m, err = e.verifyMarket(ctx, m, token)
	if err == nil {
		err = e.saveMarket(m, true)
	}
	return m, err
}
func (e *Engine) DisconnectMarket(ctx context.Context, id string) (Market, error) {
	unlock, lockErr := e.lockMarket(id)
	if lockErr != nil {
		return Market{}, lockErr
	}
	defer unlock()
	m, err := e.GetMarket(id)
	if err != nil {
		return m, err
	}
	if err = e.credentials.Delete(ctx, id); err != nil {
		return m, failure(MarketKeyring, err)
	}
	m.VerifiedAt = nil
	m.Features = []string{}
	return m, e.saveMarket(m, true)
}
func (e *Engine) verifyMarket(ctx context.Context, m Market, token string) (Market, error) {
	var d marketDiscovery
	if _, err := marketRequest(ctx, m, "", "/publisher/discovery", "DiscoveryResponse", &d, nil); err != nil {
		return m, err
	}
	if d.ConsoleBase != m.ConsoleBase {
		return m, failure(MarketProtocol, nil)
	}
	var me MarketMe
	if _, err := marketRequest(ctx, m, token, "/publisher/me", "MeResponse", &me, nil); err != nil {
		return m, err
	}
	if m.Identity != nil && (m.Identity.User.ID != me.User.ID || m.Identity.Developer.DeveloperID != me.Developer.DeveloperID) {
		return m, failure(MarketIdentity, nil)
	}
	if !slices.Contains(me.Credential.PluginIDs, e.plugin.ID) || !slices.Contains(me.Credential.Scopes, "plugins:read") || !me.Credential.ExpiresAt.After(me.ServerTime) {
		return m, failure(MarketCredential, nil)
	}
	var plugin struct {
		ID string `json:"plugin_id"`
	}
	if _, err := marketRequest(ctx, m, token, "/publisher/plugins/"+e.plugin.ID, "PluginResponse", &plugin, nil); err != nil {
		return m, err
	}
	if plugin.ID != e.plugin.ID {
		return m, failure(MarketProtocol, nil)
	}
	now := time.Now().UTC()
	m.VerifiedAt = &now
	m.Identity = &me
	m.Features = d.Features
	return m, nil
}
func marketClient(upload bool) *http.Client {
	header := 30 * time.Second
	total := 30 * time.Second
	if upload {
		header = 1800 * time.Second
		total = 1860 * time.Second
	}
	dial := (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	if upload {
		dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			c, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			return &uploadConn{Conn: c}, nil
		}
	}
	return &http.Client{Timeout: total, Transport: &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, DialContext: dial, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: header, DisableKeepAlives: true}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// Only strict, bounded contract data crosses this boundary. Never surface the
// remote message, raw body, token, URL or transport error to UI/logs.
func marketRequest(ctx context.Context, m Market, token, path, definition string, out any, request *http.Request) (statusCode int, callErr error) {
	if request == nil {
		var err error
		request, err = http.NewRequestWithContext(ctx, http.MethodGet, m.APIRoot+path, nil)
		if err != nil {
			return 0, failure(MarketProtocol, err)
		}
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	request.Header.Set("Accept", "application/json")
	trace := RequestTrace{ClientRequestID: uuid.NewString(), Method: request.Method, ResponseType: definition, StartedAt: time.Now().UTC()}
	request.Header.Set("X-Request-ID", trace.ClientRequestID)
	sink, traced := ctx.Value(requestTraceKey{}).(requestTraceSink)
	if traced {
		if err := sink.save(trace, false); err != nil {
			return 0, err
		}
	}
	defer func() {
		finished := time.Now().UTC()
		trace.FinishedAt = &finished
		trace.HTTPStatus = statusCode
		if traced {
			if err := sink.save(trace, true); err != nil {
				callErr = failure(StorageFailure, err)
			}
		}
		if callErr != nil && trace.ServerRequestID != "" {
			callErr = &marketRequestError{requestID: trace.ServerRequestID, cause: callErr}
		}
	}()
	isUpload := request.Method == http.MethodPost && strings.HasPrefix(request.Header.Get("Content-Type"), "multipart/form-data;")
	client := marketClient(isUpload)
	if request.Method == http.MethodPost && !isUpload {
		client.Timeout = 60 * time.Second
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, failure(MarketUnavailable, err)
	}
	defer response.Body.Close()
	if id := response.Header.Get("X-Request-ID"); canonicalUUID(id) {
		trace.ServerRequestID = id
	}
	retryAfter := 0
	if rawWait := response.Header.Get("Retry-After"); rawWait != "" {
		if !regexp.MustCompile(`^[1-9][0-9]{0,8}$`).MatchString(rawWait) {
			return response.StatusCode, failure(MarketPending, nil)
		}
		retryAfter, _ = strconv.Atoi(rawWait)
	}
	if retryAfter > 300 {
		return response.StatusCode, failure(MarketPending, nil)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
	if err != nil || len(raw) > 8<<20 {
		return response.StatusCode, failure(MarketProtocol, err)
	}
	media, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return response.StatusCode, failure(MarketProtocol, err)
	}
	if token != "" && (response.Header.Get("Cache-Control") != "no-store" || !canonicalUUID(response.Header.Get("X-Request-ID"))) {
		return response.StatusCode, failure(MarketProtocol, nil)
	}
	if response.StatusCode >= 400 {
		if _, err = wire.Validate("Error", raw); err != nil {
			return response.StatusCode, failure(MarketProtocol, err)
		}
		var e marketError
		_ = json.Unmarshal(raw, &e)
		if e.Status != response.StatusCode {
			return response.StatusCode, failure(MarketProtocol, nil)
		}
		e.RetryAfter = retryAfter
		return response.StatusCode, &e
	}
	if response.StatusCode != 200 && response.StatusCode != 201 && response.StatusCode != 202 {
		return response.StatusCode, failure(MarketProtocol, nil)
	}
	if response.StatusCode == 202 {
		definition = "OperationProcessingResponse"
	}
	if _, err = wire.Validate(definition, raw); err != nil {
		return response.StatusCode, failure(MarketProtocol, err)
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return response.StatusCode, failure(MarketProtocol, nil)
	}
	if response.StatusCode == 202 {
		var operation marketOperation
		if json.Unmarshal(envelope.Data, &operation) != nil || retryAfter == 0 || retryAfter != operation.RetryAfter {
			return response.StatusCode, failure(MarketProtocol, nil)
		}
		return response.StatusCode, &marketProcessing{Operation: operation, Wait: retryAfter}
	}
	if out != nil && response.StatusCode != 202 {
		if json.Unmarshal(envelope.Data, out) != nil {
			return response.StatusCode, failure(MarketProtocol, nil)
		}
	}
	return response.StatusCode, nil
}
func (e *Engine) MarketVersions(ctx context.Context, id string, page, size int) (json.RawMessage, error) {
	if page < 1 || page > 1000000 || size < 1 || size > 100 {
		return nil, invalid("pagination")
	}
	m, err := e.VerifyMarket(ctx, id)
	if err != nil {
		return nil, err
	}
	token, err := e.credentials.Get(ctx, id)
	if err != nil {
		return nil, failure(MarketKeyring, err)
	}
	var out json.RawMessage
	_, err = marketRequest(ctx, m, token, "/publisher/plugins/"+e.plugin.ID+"/versions?"+url.Values{"page": {fmtInt(page)}, "size": {fmtInt(size)}}.Encode(), "VersionsPageResponse", &out, nil)
	if err != nil {
		return nil, err
	}
	var list struct {
		Items []MarketVersion `json:"items"`
	}
	_ = json.Unmarshal(out, &list)
	for _, v := range list.Items {
		if v.PluginID != e.plugin.ID || !wire.TrustedMarketLink(m.ConsoleBase, v.URL) {
			return nil, failure(MarketProtocol, nil)
		}
	}
	return out, nil
}

type uploadConn struct{ net.Conn }

func (c *uploadConn) Write(b []byte) (int, error) {
	if err := c.SetWriteDeadline(time.Now().Add(60 * time.Second)); err != nil {
		return 0, err
	}
	return c.Conn.Write(b)
}

type marketProcessing struct {
	Operation marketOperation
	Wait      int
}

func (*marketProcessing) Error() string { return MarketPending }
