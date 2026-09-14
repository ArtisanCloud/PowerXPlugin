package runtimeexample

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/media"
	dto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/media"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/google/uuid"
)

// LocalMedia is bounded (64 objects, 1 MiB each, 256 tickets), process-local
// development storage. No Core requests,
// filesystem paths or production persistence are hidden behind this adapter.
type LocalMedia struct {
	baseURL string
	mu      sync.Mutex
	objects map[string]*mediaObject
	tickets map[string]mediaTicket
	now     func() time.Time
}
type mediaObject struct {
	tenant   string
	asset    dto.HostAsset
	variant  *dto.Variant
	checksum string
	data     []byte
	uploaded bool
}
type mediaTicket struct {
	id, method string
	expires    time.Time
}

const MediaTransferPath = "/api/v1/local-media/transfer"

var _ fw.Service = (*LocalMedia)(nil)

func NewLocalMedia(baseURL string) *LocalMedia {
	return &LocalMedia{baseURL: strings.TrimRight(baseURL, "/"), objects: map[string]*mediaObject{}, tickets: map[string]mediaTicket{}, now: time.Now}
}
func mediaError(status int, code string) error {
	return &hostapi.HTTPError{StatusCode: status, ReasonCode: "MEDIA_" + code}
}
func mediaTenant(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	v, ok := authx.TenantUUIDFromContext(ctx)
	id, err := uuid.Parse(v)
	if !ok || err != nil || id == uuid.Nil || id.String() != v {
		return "", mediaError(401, "UNAUTHORIZED")
	}
	return v, nil
}
func (s *LocalMedia) object(ctx context.Context, id string) (*mediaObject, error) {
	tenant, err := mediaTenant(ctx)
	if err != nil {
		return nil, err
	}
	o := s.objects[id]
	if o == nil || o.tenant != tenant {
		return nil, mediaError(404, "NOT_FOUND")
	}
	return o, nil
}
func validMedia(name, mime, checksum string, size int64) bool {
	sum, err := hex.DecodeString(checksum)
	return strings.TrimSpace(name) != "" && len(name) <= 256 && strings.Contains(mime, "/") && !strings.ContainsAny(mime, "\r\n") && size >= 0 && size <= 1<<20 && err == nil && len(sum) == 32 && hex.EncodeToString(sum) == checksum
}
func (s *LocalMedia) CreateAsset(ctx context.Context, in dto.CreateAssetInput) (*dto.HostAsset, error) {
	tenant, err := mediaTenant(ctx)
	if err != nil {
		return nil, err
	}
	if !validMedia(in.Name, in.MimeType, in.Checksum, in.SizeBytes) {
		return nil, mediaError(400, "INVALID_ARGUMENT")
	}
	// Owner assignment requires a verified membership adapter; never accept arbitrary owners.
	if in.OwnerSubjectUUID != "" {
		return nil, mediaError(403, "OWNER_ASSIGNMENT_UNAVAILABLE")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.objects) >= 64 {
		return nil, mediaError(409, "LOCAL_CAPACITY_EXCEEDED")
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	a := dto.HostAsset{AssetUUID: uuid.NewString(), Name: in.Name, MimeType: in.MimeType, SizeBytes: in.SizeBytes, Status: "pending_upload", CreatedAt: now, UpdatedAt: now}
	s.objects[a.AssetUUID] = &mediaObject{tenant: tenant, asset: a, checksum: in.Checksum}
	return &a, nil
}
func (s *LocalMedia) GetAsset(ctx context.Context, id string) (*dto.HostAsset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, e := s.object(ctx, id)
	if e != nil {
		return nil, e
	}
	if o.variant != nil {
		return nil, mediaError(404, "NOT_FOUND")
	}
	a := o.asset
	return &a, nil
}
func (s *LocalMedia) ListAssets(ctx context.Context, in dto.ListAssetsInput) (*dto.ListAssetsOutput, error) {
	tenant, e := mediaTenant(ctx)
	if e != nil {
		return nil, e
	}
	if in.Page < 0 || in.PageSize < 0 || in.PageSize > 100 {
		return nil, mediaError(400, "INVALID_ARGUMENT")
	}
	page, size := in.Page, in.PageSize
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []dto.Asset{}
	for id, o := range s.objects {
		if o.tenant == tenant && o.variant == nil && strings.Contains(o.asset.Name, in.Keyword) {
			items = append(items, dto.Asset{UUID: id, Name: o.asset.Name, MimeType: o.asset.MimeType})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UUID < items[j].UUID })
	total := len(items)
	if page > total/size+1 {
		items = []dto.Asset{}
	} else {
		start := (page - 1) * size
		end := start + size
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}
		items = items[start:end]
	}
	return &dto.ListAssetsOutput{Items: items, Total: int64(total), Page: page, PageSize: size}, nil
}
func (s *LocalMedia) UpdateAsset(ctx context.Context, id string, in dto.UpdateAssetInput) (*dto.HostAsset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, e := s.object(ctx, id)
	if e != nil {
		return nil, e
	}
	if o.variant != nil {
		return nil, mediaError(404, "NOT_FOUND")
	}
	if len(in.Tags) > 0 {
		return nil, mediaError(400, "TAGS_UNSUPPORTED")
	}
	if in.Name != nil {
		if strings.TrimSpace(*in.Name) == "" || len(*in.Name) > 256 {
			return nil, mediaError(400, "INVALID_ARGUMENT")
		}
		o.asset.Name = *in.Name
	}
	o.asset.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
	a := o.asset
	return &a, nil
}
func (s *LocalMedia) DeleteAsset(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, e := s.object(ctx, id)
	if e != nil {
		return e
	}
	if o.variant != nil {
		return mediaError(404, "NOT_FOUND")
	}
	for key, obj := range s.objects {
		if key == id || (obj.variant != nil && obj.variant.AssetUUID == id) {
			delete(s.objects, key)
		}
	}
	for key, t := range s.tickets {
		if s.objects[t.id] == nil {
			delete(s.tickets, key)
		}
	}
	return nil
}
func (s *LocalMedia) CreateVariant(ctx context.Context, id string, in dto.CreateVariantInput) (*dto.Variant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	parent, e := s.object(ctx, id)
	if e != nil {
		return nil, e
	}
	if parent.variant != nil {
		return nil, mediaError(404, "NOT_FOUND")
	}
	if !validMedia(in.VariantType, in.MimeType, in.Checksum, in.SizeBytes) || len(in.Name) > 256 {
		return nil, mediaError(400, "INVALID_ARGUMENT")
	}
	if len(s.objects) >= 64 {
		return nil, mediaError(409, "LOCAL_CAPACITY_EXCEEDED")
	}
	v := dto.Variant{VariantUUID: uuid.NewString(), AssetUUID: id, Name: in.Name, MimeType: in.MimeType, SizeBytes: in.SizeBytes, Status: "pending_upload"}
	s.objects[v.VariantUUID] = &mediaObject{tenant: parent.tenant, variant: &v, checksum: in.Checksum}
	out := v
	return &out, nil
}
func cloneVariant(v *dto.Variant) *dto.Variant {
	out := *v
	if v.CompletedAt != nil {
		t := *v.CompletedAt
		out.CompletedAt = &t
	}
	return &out
}
func (s *LocalMedia) GetVariant(ctx context.Context, id string) (*dto.Variant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, e := s.object(ctx, id)
	if e != nil {
		return nil, e
	}
	if o.variant == nil {
		return nil, mediaError(404, "NOT_FOUND")
	}
	return cloneVariant(o.variant), nil
}
func (s *LocalMedia) ticket(ctx context.Context, asset, id, method string, ttl int64) (*dto.TransferTicket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, e := s.object(ctx, id)
	if e != nil {
		return nil, e
	}
	if asset != "" {
		if o.variant == nil || o.variant.AssetUUID != asset {
			return nil, mediaError(404, "NOT_FOUND")
		}
	} else if o.variant != nil {
		return nil, mediaError(404, "NOT_FOUND")
	}
	state := o.asset.Status
	if o.variant != nil {
		state = o.variant.Status
	}
	if method == "GET" && state != "ready" || method == "PUT" && state == "ready" {
		return nil, mediaError(409, "INVALID_STATE")
	}
	if ttl == 0 {
		ttl = 900
	}
	if ttl < 60 || ttl > 3600 {
		return nil, mediaError(400, "INVALID_ARGUMENT")
	}
	now := s.now()
	for k, t := range s.tickets {
		if !now.Before(t.expires) {
			delete(s.tickets, k)
		}
	}
	if len(s.tickets) >= 256 {
		return nil, mediaError(409, "LOCAL_CAPACITY_EXCEEDED")
	}
	key := uuid.NewString() + uuid.NewString()
	expires := now.Add(time.Duration(ttl) * time.Second)
	s.tickets[key] = mediaTicket{id, method, expires}
	return &dto.TransferTicket{URL: s.baseURL + MediaTransferPath, Method: method, ExpiresAt: expires.UTC().Format(time.RFC3339), Headers: map[string]string{"X-Local-Media-Ticket": key}}, nil
}
func (s *LocalMedia) PresignUpload(ctx context.Context, id string) (*dto.TransferTicket, error) {
	return s.ticket(ctx, "", id, "PUT", 0)
}
func (s *LocalMedia) PresignDownload(ctx context.Context, id string) (*dto.TransferTicket, error) {
	return s.ticket(ctx, "", id, "GET", 0)
}
func (s *LocalMedia) PresignVariantUpload(ctx context.Context, a, v string, in dto.VariantTicketInput) (*dto.TransferTicket, error) {
	return s.ticket(ctx, a, v, "PUT", in.ExpiresInSeconds)
}
func (s *LocalMedia) PresignVariantDownload(ctx context.Context, a, v string, in dto.VariantTicketInput) (*dto.TransferTicket, error) {
	return s.ticket(ctx, a, v, "GET", in.ExpiresInSeconds)
}
func (s *LocalMedia) complete(ctx context.Context, asset, id string, in dto.CompleteUploadInput) (*mediaObject, error) {
	o, e := s.object(ctx, id)
	if e != nil {
		return nil, e
	}
	if asset != "" && (o.variant == nil || o.variant.AssetUUID != asset) || asset == "" && o.variant != nil {
		return nil, mediaError(404, "NOT_FOUND")
	}
	if !o.uploaded || in.Checksum != o.checksum {
		return nil, mediaError(409, "UPLOAD_INCOMPLETE")
	}
	if o.variant != nil {
		o.variant.Status = "ready"
		if o.variant.CompletedAt == nil {
			now := s.now().UTC()
			o.variant.CompletedAt = &now
		}
	} else {
		o.asset.Status = "ready"
		o.asset.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
	}
	return o, nil
}
func (s *LocalMedia) CompleteUpload(ctx context.Context, id string, in dto.CompleteUploadInput) (*dto.HostAsset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, e := s.complete(ctx, "", id, in)
	if e != nil {
		return nil, e
	}
	out := o.asset
	return &out, nil
}
func (s *LocalMedia) CompleteVariantUpload(ctx context.Context, a, v string, in dto.CompleteUploadInput) (*dto.Variant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, e := s.complete(ctx, a, v, in)
	if e != nil {
		return nil, e
	}
	return cloneVariant(o.variant), nil
}

// ServeTransfer uses an unguessable, expiring, single-use upload ticket. No
// cookies, request tenant overrides, or Core credentials authorize this endpoint.
func (s *LocalMedia) ServeTransfer(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("X-Local-Media-Ticket")
	s.mu.Lock()
	t, ok := s.tickets[key]
	o := s.objects[t.id]
	if !ok || o == nil || !s.now().Before(t.expires) {
		s.mu.Unlock()
		w.WriteHeader(404)
		return
	}
	if r.Method != t.method {
		s.mu.Unlock()
		w.WriteHeader(405)
		return
	}
	if r.Method == "GET" {
		data := append([]byte(nil), o.data...)
		mime := o.asset.MimeType
		if o.variant != nil {
			mime = o.variant.MimeType
		}
		s.mu.Unlock()
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Content-Disposition", "attachment")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(data)
		return
	}
	s.mu.Unlock()
	data, e := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
	if e != nil || len(data) > 1<<20 {
		w.WriteHeader(400)
		return
	}
	sum := sha256.Sum256(data)
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok = s.tickets[key]
	o = s.objects[t.id]
	if !ok || o == nil || !s.now().Before(t.expires) {
		w.WriteHeader(404)
		return
	}
	size := o.asset.SizeBytes
	state := o.asset.Status
	if o.variant != nil {
		size = o.variant.SizeBytes
		state = o.variant.Status
	}
	if state == "ready" || hex.EncodeToString(sum[:]) != o.checksum || int64(len(data)) != size {
		w.WriteHeader(409)
		return
	}
	o.data = data
	o.uploaded = true
	delete(s.tickets, key)
	w.WriteHeader(204)
}
