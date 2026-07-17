package adapters

import (
	"sync"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

// Bundle 为启动期单选绑定的 IAM 能力集合。
type Bundle struct {
	Directory contracts.DirectoryService
	Authz     contracts.AuthzService
	Context   contracts.IdentityContextService
}

// Registry 维护 IAM adapter 的启动期单选绑定。
type Registry struct {
	mu     sync.RWMutex
	bound  bool
	mode   contracts.IAMAdapterMode
	bundle Bundle
}

// NewRegistry 创建空注册中心。
func NewRegistry() *Registry {
	return &Registry{}
}

// Bind 在启动阶段绑定唯一 adapter。
func (r *Registry) Bind(mode contracts.IAMAdapterMode, bundle Bundle) error {
	if mode != contracts.IAMAdapterModeLocal && mode != contracts.IAMAdapterModeDelegated {
		return iamerrors.New(iamerrors.CodeModeInvalid, "invalid iam mode for adapter binding")
	}
	if bundle.Directory == nil || bundle.Authz == nil || bundle.Context == nil {
		return iamerrors.New(iamerrors.CodeAdapterNotBound, "adapter bundle requires directory/authz/context")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bound {
		return iamerrors.New(iamerrors.CodeAdapterAlreadyBind, "iam adapter has already been bound")
	}

	r.bound = true
	r.mode = mode
	r.bundle = bundle
	return nil
}

// IsBound 返回是否已绑定 adapter。
func (r *Registry) IsBound() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.bound
}

// Mode 返回当前绑定模式；未绑定时 ok=false。
func (r *Registry) Mode() (mode contracts.IAMAdapterMode, ok bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.bound {
		return "", false
	}
	return r.mode, true
}

// Directory 返回目录服务。
func (r *Registry) Directory() (contracts.DirectoryService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.bound || r.bundle.Directory == nil {
		return nil, iamerrors.New(iamerrors.CodeAdapterNotBound, "iam directory adapter not bound")
	}
	return r.bundle.Directory, nil
}

// Authz 返回授权服务。
func (r *Registry) Authz() (contracts.AuthzService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.bound || r.bundle.Authz == nil {
		return nil, iamerrors.New(iamerrors.CodeAdapterNotBound, "iam authz adapter not bound")
	}
	return r.bundle.Authz, nil
}

// IdentityContext 返回身份上下文服务。
func (r *Registry) IdentityContext() (contracts.IdentityContextService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.bound || r.bundle.Context == nil {
		return nil, iamerrors.New(iamerrors.CodeAdapterNotBound, "iam context adapter not bound")
	}
	return r.bundle.Context, nil
}
