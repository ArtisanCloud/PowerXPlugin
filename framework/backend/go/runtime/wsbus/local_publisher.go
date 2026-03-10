package wsbus

import (
	"context"
	"log/slog"
)

type LocalHub interface {
	Publish(ctx context.Context, topic string, payload any, opts PublishOptions) error
}

type LocalPublisher struct {
	hub    LocalHub
	logger *slog.Logger
}

func NewLocalPublisher(hub LocalHub, logger *slog.Logger) *LocalPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &LocalPublisher{hub: hub, logger: logger}
}

func (p *LocalPublisher) Publish(ctx context.Context, topic string, payload any, opts PublishOptions) PublishResult {
	if p == nil || p.hub == nil {
		return FailureResult(ErrorCodePublisherNotConfigured, "local ws bus is not configured")
	}
	if err := p.hub.Publish(ctx, topic, payload, opts); err != nil {
		if p.logger != nil {
			p.logger.Warn("local ws bus publish failed", slog.String("error", err.Error()))
		}
		return FailureResult(ErrorCodeLocalPublishFailed, err.Error())
	}
	return SuccessResult()
}
