package realtime

import (
	"context"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/google/uuid"
)

const (
	WSIdentityExpired = "IAM_WS_CREDENTIAL_EXPIRED"
	WSIdentityRevoked = "IAM_WS_IDENTITY_REVOKED"
	WSTopicForbidden  = "IAM_WS_TOPIC_FORBIDDEN"
)

// IdentityRevocation is supplied by a trusted logout/IAM event adapter. Empty
// fields do not match every connection: at least one selector is required.
type IdentityRevocation struct{ PluginID, TenantUUID, UserUUID, MemberUUID, PolicyVersion string }
type IdentityLifecycle struct {
	mu       sync.Mutex
	revision uint64
	leases   map[string]*IdentityLease
}

func NewIdentityLifecycle() *IdentityLifecycle {
	return &IdentityLifecycle{leases: map[string]*IdentityLease{}}
}
func (r *IdentityLifecycle) Revision() uint64 { r.mu.Lock(); defer r.mu.Unlock(); return r.revision }

type IdentityLeaseOptions struct {
	Revalidator      contracts.WSIdentityRevalidationService
	Identity         *contracts.DelegatedIdentity
	PluginID         string
	TTL              time.Duration
	ObservedRevision uint64
	AuthorizeTopic   func(*contracts.DelegatedIdentity, string) bool
	Close            func(string)
}
type IdentityLease struct {
	revalidator   contracts.WSIdentityRevalidationService
	ctx           context.Context
	cancel        context.CancelFunc
	ID            string
	ExpiresAt     time.Time
	identity      contracts.DelegatedIdentity
	authorize     func(*contracts.DelegatedIdentity, string) bool
	closeCallback func(string)
	owner         *IdentityLifecycle
	timer         *time.Timer
	once          sync.Once
	done          chan struct{}
}

func BoundIdentityExpiry(identity *contracts.DelegatedIdentity, ttl time.Duration, now time.Time) (time.Time, error) {
	if identity == nil || ttl <= 0 {
		return time.Time{}, iamerrors.New(iamerrors.CodeContextInvalidArgument, iamerrors.CodeContextInvalidArgument)
	}
	if !now.Before(identity.UserSessionExpiresAt) {
		return time.Time{}, iamerrors.New(iamerrors.CodeUserSessionExpired, iamerrors.CodeUserSessionExpired)
	}
	expiry := now.Add(ttl)
	if expiry.After(identity.UserSessionExpiresAt) {
		expiry = identity.UserSessionExpiresAt
	}
	if identity.WSRevalidationExpiresAt == nil {
		return time.Time{}, iamerrors.New("IAM_CONTEXT_WS_HANDLE_REQUIRED", "IAM_CONTEXT_WS_HANDLE_REQUIRED")
	}
	if expiry.After(*identity.WSRevalidationExpiresAt) {
		expiry = *identity.WSRevalidationExpiresAt
	}
	if !now.Before(expiry) {
		return time.Time{}, iamerrors.New("IAM_CONTEXT_WS_HANDLE_EXPIRED", "IAM_CONTEXT_WS_HANDLE_EXPIRED")
	}
	return expiry, nil
}

func (r *IdentityLifecycle) Open(opts IdentityLeaseOptions) (*IdentityLease, error) {
	if r == nil || opts.Close == nil || opts.AuthorizeTopic == nil || opts.Revalidator == nil {
		return nil, iamerrors.New(iamerrors.CodeAdapterNotBound, iamerrors.CodeAdapterNotBound)
	}
	now := time.Now()
	if err := opts.Identity.Validate(opts.PluginID, now); err != nil {
		return nil, err
	}
	expiry, err := BoundIdentityExpiry(opts.Identity, opts.TTL, now)
	if err != nil {
		return nil, err
	}
	identity := *opts.Identity
	identity.Permissions = append([]string{}, identity.Permissions...)
	identity.Roles = append([]string{}, identity.Roles...)
	if identity.MemberUUID != nil {
		member := *identity.MemberUUID
		identity.MemberUUID = &member
	}
	if identity.WSRevalidationExpiresAt != nil {
		expiry := *identity.WSRevalidationExpiresAt
		identity.WSRevalidationExpiresAt = &expiry
	}
	ctx, cancel := context.WithCancel(context.Background())
	lease := &IdentityLease{ctx: ctx, cancel: cancel, revalidator: opts.Revalidator, ID: uuid.NewString(), ExpiresAt: expiry, identity: identity, authorize: opts.AuthorizeTopic, closeCallback: opts.Close, owner: r, done: make(chan struct{})}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.leases == nil {
		cancel()
		return nil, iamerrors.New(iamerrors.CodeAdapterNotBound, iamerrors.CodeAdapterNotBound)
	}
	if r.revision != opts.ObservedRevision {
		cancel()
		return nil, iamerrors.New("IAM_CONTEXT_POLICY_CHANGED", "IAM_CONTEXT_POLICY_CHANGED")
	}
	r.leases[lease.ID] = lease
	lease.timer = time.AfterFunc(time.Until(expiry), func() { lease.Close(WSIdentityExpired) })
	return lease, nil
}

func (l *IdentityLease) Done() <-chan struct{} { return l.done }
func (l *IdentityLease) Revalidate(ctx context.Context) (*contracts.WSIdentity, error) {
	select {
	case <-l.done:
		return nil, iamerrors.New(WSIdentityRevoked, WSIdentityRevoked)
	default:
	}
	if !time.Now().Before(l.ExpiresAt) {
		l.Close(WSIdentityExpired)
		return nil, iamerrors.New(WSIdentityExpired, WSIdentityExpired)
	}
	requestCtx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(l.ctx, cancel)
	defer func() { stop(); cancel() }()
	current, err := l.revalidator.RevalidateWSIdentity(requestCtx, contracts.WSIdentityQuery{Handle: l.identity.WSRevalidationHandle})
	if err != nil {
		code := iamerrors.CodeOf(err)
		if code == "" {
			code = iamerrors.CodeContextUnavailable
		}
		l.Close(code)
		return nil, err
	}
	if err = current.Validate(l.identity.PluginID, time.Now()); err != nil {
		l.Close(iamerrors.CodeOf(err))
		return nil, err
	}
	v := current.DelegatedIdentity
	base := l.identity
	sameMember := (v.MemberUUID == nil && base.MemberUUID == nil) || (v.MemberUUID != nil && base.MemberUUID != nil && *v.MemberUUID == *base.MemberUUID)
	if !sameMember || v.TenantUUID != base.TenantUUID || v.UserUUID != base.UserUUID || v.IsRoot != base.IsRoot || v.WSRevalidationHandle != base.WSRevalidationHandle || !v.UserSessionExpiresAt.Equal(base.UserSessionExpiresAt) || !v.WSRevalidationExpiresAt.Equal(*base.WSRevalidationExpiresAt) || v.SourcePath != base.SourcePath || v.SourceMethod != base.SourceMethod || v.Transport != base.Transport || v.SourceRequestID != base.SourceRequestID || v.SourceTraceID != base.SourceTraceID || v.SourceRequiredPermission != base.SourceRequiredPermission {
		err = iamerrors.New("IAM_WS_IDENTITY_MISMATCH", "IAM_WS_IDENTITY_MISMATCH")
		l.Close(iamerrors.CodeOf(err))
		return nil, err
	}
	select {
	case <-l.done:
		return nil, iamerrors.New(WSIdentityRevoked, WSIdentityRevoked)
	default:
		if !time.Now().Before(l.ExpiresAt) || ctx.Err() != nil {
			l.Close(WSIdentityExpired)
			return nil, iamerrors.New(WSIdentityExpired, WSIdentityExpired)
		}
		return current, nil
	}
}
func (l *IdentityLease) AuthorizeTopic(ctx context.Context, topic string) error {
	current, err := l.Revalidate(ctx)
	if err != nil {
		return err
	}
	identity := current.DelegatedIdentity
	if topic == "" || !l.authorize(&identity, topic) {
		err = iamerrors.New(WSTopicForbidden, WSTopicForbidden)
		l.Close(WSTopicForbidden)
		return err
	}
	select {
	case <-l.done:
		return iamerrors.New(WSIdentityRevoked, WSIdentityRevoked)
	default:
		if ctx.Err() != nil {
			l.Close(WSIdentityRevoked)
			return iamerrors.New(WSIdentityRevoked, WSIdentityRevoked)
		}
		if !time.Now().Before(l.ExpiresAt) {
			l.Close(WSIdentityExpired)
			return iamerrors.New(WSIdentityExpired, WSIdentityExpired)
		}
		return nil
	}
}
func (l *IdentityLease) Close(reason string) {
	l.once.Do(func() {
		close(l.done)
		l.cancel()
		l.owner.mu.Lock()
		delete(l.owner.leases, l.ID)
		if l.timer != nil {
			l.timer.Stop()
		}
		l.owner.mu.Unlock()
		l.closeCallback(reason)
	})
}
func (r *IdentityLifecycle) Revoke(selector IdentityRevocation) int {
	if selector == (IdentityRevocation{}) {
		return 0
	}
	r.mu.Lock()
	r.revision++
	var selected []*IdentityLease
	for _, l := range r.leases {
		v := l.identity
		member := ""
		if v.MemberUUID != nil {
			member = *v.MemberUUID
		}
		if (selector.PluginID == "" || selector.PluginID == v.PluginID) && (selector.TenantUUID == "" || selector.TenantUUID == v.TenantUUID) && (selector.UserUUID == "" || selector.UserUUID == v.UserUUID) && (selector.MemberUUID == "" || selector.MemberUUID == member) && (selector.PolicyVersion == "" || selector.PolicyVersion == v.PolicyVersion) {
			selected = append(selected, l)
		}
	}
	r.mu.Unlock()
	for _, l := range selected {
		l.Close(WSIdentityRevoked)
	}
	return len(selected)
}
func (r *IdentityLifecycle) Close() {
	r.mu.Lock()
	r.revision++
	var selected []*IdentityLease
	for _, l := range r.leases {
		selected = append(selected, l)
	}
	r.leases = nil
	r.mu.Unlock()
	for _, l := range selected {
		l.Close(WSIdentityRevoked)
	}
}
