package eventbridge

import "errors"

var ErrReliableHostRequired = errors.New("EVENT_FABRIC_RELIABLE_HOST_REQUIRED")

// TaskBusProvider is a factory for producing a TaskBus-backed emitter.
// 实际实现依赖 framework runtime/host SDK；本仓库仅提供接口占位以支持 DI 与测试注入。
type TaskBusProvider interface {
	NewEmitter() (Emitter, error)
}

// NewTaskBusEmitterAdapter returns a provider function compatible with Factory.WithTaskBusProvider.
// 当前为占位：如果没有注入真实 provider，将返回明确错误。
func NewTaskBusEmitterAdapter(provider TaskBusProvider) func() (Emitter, error) {
	return func() (Emitter, error) {
		if provider == nil {
			return nil, errors.New("taskbus provider is nil")
		}
		emitter, err := provider.NewEmitter()
		if host, ok := provider.(interface{ RequiresReliableHost() bool }); ok && host.RequiresReliableHost() && err != nil {
			return nil, errors.Join(ErrReliableHostRequired, err)
		}
		return emitter, err
	}
}
