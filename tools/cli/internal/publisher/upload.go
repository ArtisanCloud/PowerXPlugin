package publisher

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/powerx-plugin/cli/internal/publisher/wire"
)

func fmtInt64(n int64) string { return strconv.FormatInt(n, 10) }

const UploadSchema = "powerx.plugin-publisher-upload/v1"

type UploadInput struct {
	ConnectionID string `json:"connection_id"`
	BuildID      string `json:"build_id"`
}
type Upload struct {
	ContractVersion string `json:"contract_version"`
	BundleSHA256    string `json:"bundle_sha256"`
	Schema          string `json:"schema"`
	UploadID        string `json:"upload_id"`
	OperationID     string `json:"operation_id"`
	UploadInput
	UserID         string         `json:"user_id"`
	DeveloperID    string         `json:"developer_id"`
	PluginID       string         `json:"plugin_id"`
	PlanID         string         `json:"release_plan_id"`
	Version        string         `json:"version"`
	Artifact       Artifact       `json:"artifact"`
	RequestSHA256  string         `json:"request_sha256"`
	Stage          string         `json:"stage"`
	Attempts       int            `json:"attempts"`
	BytesSent      int64          `json:"bytes_sent"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	Receipt        *UploadReceipt `json:"receipt,omitempty"`
	CurrentVersion *MarketVersion `json:"current_version,omitempty"`
	Error          *Error         `json:"error,omitempty"`
}

func uploadActive(stage string) bool {
	return stage == "queued" || stage == "uploading" || stage == "reconciling" || stage == "processing"
}
func (e *Engine) uploadDir() (*directory, error) { return e.state.child("uploads", true) }
func (e *Engine) saveUpload(u *Upload, replace bool) error {
	u.UpdatedAt = time.Now().UTC()
	d, err := e.uploadDir()
	if err != nil {
		return err
	}
	defer d.close()
	b, err := json.Marshal(u)
	if int64(len(b)) > MaxRecordBytes {
		return failure(TooLarge, nil)
	}
	if err != nil {
		return failure(StorageFailure, err)
	}
	return d.write(u.UploadID+".json", b, replace)
}
func (e *Engine) GetUpload(id string) (Upload, error) {
	if !canonicalUUID(id) {
		return Upload{}, invalid("upload_id")
	}
	d, err := e.uploadDir()
	if err != nil {
		return Upload{}, err
	}
	defer d.close()
	b, err := d.read(id+".json", MaxRecordBytes)
	var u Upload
	if err != nil {
		return u, err
	}
	if decodeStrict(b, &u) != nil || u.ContractVersion != MarketContractVersion || u.BundleSHA256 != MarketBundleSHA256 || u.Schema != UploadSchema || u.UploadID != id || !canonicalUUID(u.OperationID) || !canonicalUUID(u.ConnectionID) || !canonicalUUID(u.BuildID) || !canonicalUUID(u.UserID) || !canonicalUUID(u.DeveloperID) || u.PluginID != e.plugin.ID {
		return Upload{}, failure(InvalidRecord, nil)
	}
	e.mu.Lock()
	_, running := e.uploadJobs[id]
	e.mu.Unlock()
	if uploadActive(u.Stage) && !running {
		u.Stage = "pending"
		u.Error = failure(MarketPending, nil)
	}
	return u, nil
}
func (e *Engine) ListUploads() ([]Upload, error) {
	d, err := e.uploadDir()
	if err != nil {
		return nil, err
	}
	defer d.close()
	names, err := d.names()
	if err != nil {
		return nil, err
	}
	out := []Upload{}
	for _, name := range names {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		u, err := e.GetUpload(strings.TrimSuffix(name, ".json"))
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
func uploadFingerprint(p Plan, a Artifact) (string, error) {
	b, _ := json.Marshal(map[string]any{"plugin_id": p.PluginID, "version": p.Version, "channel": p.Channel, "os": p.Target.OS, "arch": p.Target.Arch, "sha256": a.SHA256, "size": a.Size, "release_notes": p.ReleaseNotes})
	_, hash, err := wire.Fingerprint("package-upload", b)
	if err != nil {
		return "", failure(MarketProtocol, err)
	}
	return hash, nil
}
func (e *Engine) StartUpload(ctx context.Context, input UploadInput) (Upload, error) {
	if !canonicalUUID(input.ConnectionID) || !canonicalUUID(input.BuildID) {
		return Upload{}, invalid("upload")
	}
	m, err := e.GetMarket(input.ConnectionID)
	if err != nil {
		return Upload{}, err
	}
	if m.Identity == nil || m.VerifiedAt == nil {
		return Upload{}, failure(MarketCredential, nil)
	}
	b, err := e.GetBuild(input.BuildID)
	if err != nil {
		return Upload{}, err
	}
	if b.Stage != "ready" || b.Artifact == nil {
		return Upload{}, failure(ArtifactInvalid, nil)
	}
	p, err := e.Get(b.ReleasePlanID)
	if err != nil {
		return Upload{}, err
	}
	digest, err := uploadFingerprint(p, *b.Artifact)
	if err != nil {
		return Upload{}, err
	}
	// Cross-process admission lock. Never create two active operations for the same
	// frozen artifact/market, including a pending operation from a previous process.
	d, err := e.uploadDir()
	if err != nil {
		return Upload{}, err
	}
	defer d.close()
	unlock, err := d.lock()
	if err != nil {
		return Upload{}, failure(Conflict, err)
	}
	defer unlock()
	old, err := e.ListUploads()
	if err != nil {
		return Upload{}, err
	}
	for _, u := range old {
		if u.ConnectionID == m.ConnectionID && u.BuildID == b.BuildID && u.Stage != "failed" {
			return Upload{}, failure(Conflict, nil)
		}
	}
	now := time.Now().UTC()
	u := Upload{ContractVersion: MarketContractVersion, BundleSHA256: MarketBundleSHA256, Schema: UploadSchema, UploadID: uuid.NewString(), OperationID: uuid.NewString(), UploadInput: input, UserID: m.Identity.User.ID, DeveloperID: m.Identity.Developer.DeveloperID, PluginID: p.PluginID, PlanID: p.ReleasePlanID, Version: p.Version, Artifact: *b.Artifact, RequestSHA256: digest, Stage: "queued", CreatedAt: now, UpdatedAt: now}
	if err = e.saveUpload(&u, false); err != nil {
		return Upload{}, err
	}
	return e.launchUpload(ctx, u, false)
}
func (e *Engine) ResumeUpload(ctx context.Context, id string) (Upload, error) {
	u, err := e.GetUpload(id)
	if err != nil {
		return u, err
	}
	return e.launchUpload(ctx, u, true)
}
func (e *Engine) launchUpload(ctx context.Context, u Upload, reconcile bool) (Upload, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closing {
		return u, failure(Conflict, nil)
	}
	if _, ok := e.uploadJobs[u.UploadID]; ok {
		return u, failure(Conflict, nil)
	}
	d, err := e.uploadDir()
	if err != nil {
		return u, err
	}
	lockdir, err := d.child(u.UploadID, true)
	d.close()
	if err != nil {
		return u, err
	}
	unlock, err := lockdir.lock()
	if err != nil {
		lockdir.close()
		return u, failure(Conflict, err)
	}
	run, cancel := context.WithCancel(ctx)
	e.uploadJobs[u.UploadID] = cancel
	u.Stage = "reconciling"
	u.Error = nil
	if err = e.saveUpload(&u, true); err != nil {
		delete(e.uploadJobs, u.UploadID)
		cancel()
		unlock()
		lockdir.close()
		return u, err
	}
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer unlock()
		defer lockdir.close()
		defer cancel()
		defer func() { e.mu.Lock(); delete(e.uploadJobs, u.UploadID); e.mu.Unlock() }()
		e.runUpload(run, u, reconcile)
	}()
	return u, nil
}
func (e *Engine) CancelUpload(id string) (Upload, error) {
	e.mu.Lock()
	cancel := e.uploadJobs[id]
	e.mu.Unlock()
	if cancel == nil {
		u, err := e.GetUpload(id)
		if err != nil {
			return u, err
		}
		if u.Stage == "pending" {
			return u, failure(Conflict, nil)
		}
		return u, nil
	}
	cancel()
	return e.GetUpload(id)
}
func (e *Engine) WaitUpload(id string) (Upload, error) {
	for {
		e.mu.Lock()
		_, ok := e.uploadJobs[id]
		e.mu.Unlock()
		if !ok {
			return e.GetUpload(id)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
func (e *Engine) runUpload(ctx context.Context, u Upload, reconcile bool) {
	ctx = e.traceRequests(ctx, "uploads", u.UploadID, u.OperationID)
	finish := func(stage string, err error) {
		u.Stage = stage
		if err != nil {
			if ctx.Err() != nil && stage == "pending" {
				err = failure(MarketPending, nil)
			}
			u.Error = AsError(err)
		} else {
			u.Error = nil
		}
		_ = e.saveUpload(&u, true)
	}
	defer func() {
		if ctx.Err() == nil {
			return
		}
		e.mu.Lock()
		closing := e.closing
		e.mu.Unlock()
		if closing {
			return
		}
		check, cancel := context.WithTimeout(context.WithoutCancel(ctx), 8*time.Second)
		defer cancel()
		m, err := e.VerifyMarket(check, u.ConnectionID)
		if err != nil {
			return
		}
		if m.Identity.User.ID != u.UserID || m.Identity.Developer.DeveloperID != u.DeveloperID {
			return
		}
		token, err := e.credentials.Get(check, u.ConnectionID)
		if err != nil {
			return
		}
		var op marketOperation
		_, err = marketRequest(check, m, token, "/publisher/operations/"+u.OperationID, "OperationResponse", &op, nil)
		if err != nil || op.OperationID != u.OperationID || op.PluginID != u.PluginID || op.Type != "package-upload" || op.SHA256 != u.RequestSHA256 {
			return
		}
		if op.Status == "succeeded" {
			if e.acceptReceipt(check, &u, m, token, op.Result) == nil {
				finish("succeeded", nil)
			}
		} else if op.Status == "failed" && op.Error != nil && !op.Error.Retryable {
			finish("failed", op.Error)
		}
	}()
	m, err := e.VerifyMarket(ctx, u.ConnectionID)
	if err != nil {
		finish("pending", err)
		return
	}
	if m.Identity.User.ID != u.UserID || m.Identity.Developer.DeveloperID != u.DeveloperID {
		finish("pending", failure(MarketIdentity, nil))
		return
	}
	token, err := e.credentials.Get(ctx, m.ConnectionID)
	if err != nil {
		finish("pending", failure(MarketKeyring, err))
		return
	}
	var p Plan
	pollStart := time.Now()
	operationStatus := "not_found"
	retryAfter := 0
	if reconcile || u.Attempts > 0 {
		operationStatus = "unknown"
	}
	for {
		if ctx.Err() != nil {
			finish("pending", failure(MarketPending, nil))
			return
		}
		if operationStatus == "unknown" || operationStatus == "processing" {
			var op marketOperation
			status, queryErr := marketRequest(ctx, m, token, "/publisher/operations/"+u.OperationID, "OperationResponse", &op, nil)
			if queryErr != nil {
				if status == 404 {
					operationStatus = "not_found"
					retryAfter = 0
				} else {
					finish("pending", queryErr)
					return
				}
			} else {
				if op.OperationID != u.OperationID || op.PluginID != u.PluginID || op.Type != "package-upload" || op.SHA256 != u.RequestSHA256 {
					finish("pending", failure(MarketProtocol, nil))
					return
				}
				switch op.Status {
				case "succeeded":
					if err = e.acceptReceipt(ctx, &u, m, token, op.Result); err != nil {
						finish("pending", err)
					} else {
						finish("succeeded", nil)
					}
					return
				case "failed":
					if op.Error == nil {
						finish("pending", failure(MarketProtocol, nil))
						return
					}
					if !op.Error.Retryable {
						finish("failed", op.Error)
						return
					}
					operationStatus = "failed_retryable"
				case "processing":
					operationStatus = "processing"
					retryAfter = op.RetryAfter
				default:
					finish("pending", failure(MarketProtocol, nil))
					return
				}
			}
		}
		elapsed := int(time.Since(u.CreatedAt).Seconds())
		if elapsed < 0 {
			finish("pending", failure(MarketProtocol, nil))
			return
		}
		wait := retryAfter
		if operationStatus == "processing" && wait < 2 {
			wait = 2
		}
		// A fresh poll window never resets the persisted overall/attempt budget.
		rc, _ := json.Marshal(map[string]any{"credential_status": "active", "identity_matches": true, "plugin_allowed": true, "scope_allowed": (operationStatus == "processing" || slices.Contains(m.Identity.Credential.Scopes, "packages:write")), "request_matches": true, "operation_status": operationStatus, "retention_expired": time.Since(u.CreatedAt) > 7*24*time.Hour, "feature_available": slices.Contains(m.Features, "package-upload") && slices.Contains(m.Features, "idempotent-writes"), "attempts": u.Attempts, "elapsed_seconds": elapsed, "poll_elapsed_seconds": int(time.Since(pollStart).Seconds()), "retry_after_seconds": retryAfter})
		action, policyErr := wire.RecoveryAction(rc)
		if policyErr != nil {
			finish("pending", failure(MarketProtocol, policyErr))
			return
		}
		switch action {
		case "poll_operation":
			u.Stage = "processing"
			if e.saveUpload(&u, true) != nil {
				return
			}
			if !waitUpload(ctx, time.Duration(wait)*time.Second) {
				finish("pending", failure(MarketPending, nil))
				return
			}
			continue
		case "retry_same_operation":
			if u.Attempts > 0 {
				jitter, randomErr := rand.Int(rand.Reader, big.NewInt(int64(1<<min(u.Attempts-1, 4))*1000+1))
				if randomErr != nil {
					finish("pending", failure(StorageFailure, randomErr))
					return
				}
				delay := time.Duration(jitter.Int64()) * time.Millisecond
				if delay < time.Duration(wait)*time.Second {
					delay = time.Duration(wait) * time.Second
				}
				if delay >= time.Until(u.CreatedAt.Add(7200*time.Second)) || !waitUpload(ctx, delay) {
					finish("pending", failure(MarketPending, nil))
					return
				}
			}
		case "wait_capability":
			finish("pending", failure(MarketCapability, nil))
			return
		case "deny", "reauthorize":
			finish("pending", failure(MarketCredential, nil))
			return
		default:
			finish("pending", failure(MarketPending, nil))
			return
		}

		p, err = e.Get(u.PlanID)
		if err != nil {
			finish("pending", err)
			return
		}
		verifyContext, verifyCancel := context.WithDeadline(ctx, u.CreatedAt.Add(7200*time.Second))
		a, verifyErr := e.verifyArtifact(verifyContext, u.BuildID)
		verifyCancel()
		if verifyErr != nil {
			if errors.Is(verifyErr, context.Canceled) || errors.Is(verifyErr, context.DeadlineExceeded) {
				verifyErr = failure(MarketPending, nil)
			}
			finish("pending", verifyErr)
			return
		}
		digest, fingerprintErr := uploadFingerprint(p, a)
		if fingerprintErr != nil || digest != u.RequestSHA256 || a != u.Artifact {
			finish("pending", failure(ArtifactChanged, fingerprintErr))
			return
		}
		u.Attempts++
		u.Stage = "uploading"
		u.BytesSent = 0
		if e.saveUpload(&u, true) != nil {
			return
		}
		var receipt UploadReceipt
		uploadContext, uploadCancel := context.WithDeadline(ctx, u.CreatedAt.Add(7200*time.Second))
		status, sendErr := e.sendArtifact(uploadContext, m, token, &u, p, &receipt)
		uploadCancel()
		if sendErr == nil && (status == 200 || status == 201) {
			if err = e.acceptReceipt(ctx, &u, m, token, &receipt); err != nil {
				finish("pending", err)
			} else {
				finish("succeeded", nil)
			}
			return
		}
		if sendErr != nil && AsError(sendErr).Code == MarketPending {
			finish("pending", sendErr)
			return
		}
		waitResponse := 0
		if sendErr != nil {
			var processing *marketProcessing
			if errors.As(sendErr, &processing) {
				op := processing.Operation
				if op.OperationID != u.OperationID || op.PluginID != u.PluginID || op.Type != "package-upload" || op.SHA256 != u.RequestSHA256 {
					finish("pending", failure(MarketProtocol, nil))
					return
				}
				waitResponse = max(2, processing.Wait)
			}
			var remote *marketError
			if errors.As(sendErr, &remote) {
				waitResponse = remote.RetryAfter
			}

			if errors.As(sendErr, &remote) && !remote.Retryable {
				if remote.Status == 401 || remote.Status == 403 {
					finish("pending", sendErr)
				} else {
					finish("failed", sendErr)
				}
				return
			}
		}
		// Ambiguous response, 202, retryable HTTP or lost socket: query first. Never
		// allocate a replacement operation or infer failure from a transport error.
		pollStart = time.Now()
		if waitResponse > 0 {
			var processing *marketProcessing
			if !responseWaitAllowed(u.CreatedAt, pollStart, time.Now(), waitResponse, errors.As(sendErr, &processing)) {
				finish("pending", failure(MarketPending, nil))
				return
			}
			u.Stage = "processing"
			if e.saveUpload(&u, true) != nil {
				return
			}
			if !waitUpload(ctx, time.Duration(waitResponse)*time.Second) {
				finish("pending", failure(MarketPending, nil))
				return
			}
		}
		operationStatus = "unknown"
		retryAfter = 0
		u.Stage = "reconciling"
		if e.saveUpload(&u, true) != nil {
			return
		}
	}
}
func waitUpload(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
func (e *Engine) acceptReceipt(ctx context.Context, u *Upload, m Market, token string, r *UploadReceipt) error {
	if r == nil || r.OperationID != u.OperationID || r.PluginID != u.PluginID || r.Version != u.Version || r.Artifact.SHA256 != u.Artifact.SHA256 || r.Artifact.Size != u.Artifact.Size || r.Artifact.OS != "linux" || r.Artifact.Arch != "amd64" || !wire.TrustedMarketLink(m.ConsoleBase, r.URL) {
		return failure(MarketProtocol, nil)
	}
	u.Receipt = r
	u.CurrentVersion = nil
	u.Stage = "uploaded"
	if err := e.saveUpload(u, true); err != nil {
		return err
	}
	var v MarketVersion
	_, err := marketRequest(ctx, m, token, "/publisher/plugins/"+u.PluginID+"/versions/"+r.VersionID, "MarketVersionResponse", &v, nil)
	if err != nil {
		return err
	}
	if v.VersionID != r.VersionID || v.PluginID != u.PluginID || v.Version != u.Version || !wire.TrustedMarketLink(m.ConsoleBase, v.URL) {
		return failure(MarketProtocol, nil)
	}
	found := false
	for _, a := range v.Artifacts {
		if a == r.Artifact {
			found = true
		}
	}
	if !found {
		return failure(MarketProtocol, nil)
	}
	u.CurrentVersion = &v
	return nil
}
func (e *Engine) sendArtifact(ctx context.Context, m Market, token string, u *Upload, p Plan, out *UploadReceipt) (int, error) {
	dir, err := e.work.existing(u.BuildID, true)
	if err != nil {
		return 0, err
	}
	defer dir.close()
	f, err := dir.openRead("artifact.zip")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	reader, writer := io.Pipe()
	form := multipart.NewWriter(writer)
	defer reader.Close()
	finished := make(chan error, 1)
	var progressMu sync.Mutex
	go func() {
		var writeErr error
		for _, field := range [][2]string{{"version", p.Version}, {"channel", p.Channel}, {"os", p.Target.OS}, {"arch", p.Target.Arch}, {"sha256", u.Artifact.SHA256}, {"size", fmtInt64(u.Artifact.Size)}, {"release_notes", p.ReleaseNotes}} {
			if writeErr = form.WriteField(field[0], field[1]); writeErr != nil {
				break
			}
		}
		if writeErr == nil {
			part, err := form.CreatePart(textproto.MIMEHeader{"Content-Disposition": {`form-data; name="file"; filename="artifact.zip"`}, "Content-Type": {"application/zip"}})
			writeErr = err
			if err == nil {
				h := sha256.New()
				var n int64
				lastProgress := time.Now()
				progress := &uploadProgressWriter{write: func(b []byte) (int, error) {
					n, err := part.Write(b)
					u.BytesSent += int64(n)
					if time.Since(lastProgress) >= 500*time.Millisecond {
						if saveErr := e.saveUpload(u, true); saveErr != nil {
							return n, saveErr
						}
						lastProgress = time.Now()
					}
					return n, err
				}}
				n, writeErr = io.Copy(io.MultiWriter(progress, h), io.LimitReader(f, MaxPackageBytes+1))
				if writeErr == nil && (n != u.Artifact.Size || hex.EncodeToString(h.Sum(nil)) != u.Artifact.SHA256) {
					writeErr = failure(ArtifactChanged, nil)
				}
				progressMu.Lock()
				u.BytesSent = n
				progressMu.Unlock()
			}
		}
		if writeErr == nil {
			u.Stage = "processing"
			writeErr = e.saveUpload(u, true)
		}
		if writeErr == nil {
			writeErr = form.Close()
		}
		_ = writer.CloseWithError(writeErr)
		finished <- writeErr
	}()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.APIRoot+"/publisher/plugins/"+u.PluginID+"/package", reader)
	if err != nil {
		reader.Close()
		<-finished
		return 0, failure(MarketProtocol, err)
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Idempotency-Key", u.OperationID)
	status, err := marketRequest(ctx, m, token, "", "UploadReceiptResponse", out, req)
	reader.Close()
	writeErr := <-finished
	progressMu.Lock()
	defer progressMu.Unlock()
	if writeErr != nil && err == nil {
		return status, failure(MarketUnavailable, writeErr)
	}
	return status, err
}

type uploadProgressWriter struct{ write func([]byte) (int, error) }

func (w *uploadProgressWriter) Write(b []byte) (int, error) { return w.write(b) }

// The first 202 wait belongs to the polling window as well as the total
// operation budget. Never shorten a server wait to fit either deadline.
func responseWaitAllowed(created, pollStart, now time.Time, wait int, processing bool) bool {
	if wait < 1 || wait > 300 {
		return false
	}
	delay := time.Duration(wait) * time.Second
	if delay >= created.Add(7200*time.Second).Sub(now) {
		return false
	}
	return !processing || delay < pollStart.Add(120*time.Second).Sub(now)
}
