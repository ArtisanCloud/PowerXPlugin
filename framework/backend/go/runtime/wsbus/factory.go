package wsbus

import (
	"context"
	"log/slog"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
)

type FactoryOption func(*publisherFactory)

type publisherFactory struct {
	app            *bootstrap.App
	localPublisher Publisher
	logger         *slog.Logger
}

func WithLocalPublisher(p Publisher) FactoryOption {
	return func(f *publisherFactory) {
		f.localPublisher = p
	}
}

func WithLocalHub(hub LocalHub) FactoryOption {
	return func(f *publisherFactory) {
		if hub == nil {
			return
		}
		f.localPublisher = NewLocalPublisher(hub, f.logger)
	}
}

func WithLogger(logger *slog.Logger) FactoryOption {
	return func(f *publisherFactory) {
		f.logger = logger
	}
}

func NewPublisher(app *bootstrap.App, opts ...FactoryOption) Publisher {
	factory := &publisherFactory{app: app, logger: slog.Default()}
	for _, opt := range opts {
		if opt != nil {
			opt(factory)
		}
	}
	if factory.logger == nil {
		factory.logger = slog.Default()
	}

	var inner Publisher
	if app != nil && app.Config != nil && app.Config.Standalone {
		inner = factory.localPublisher
		if inner == nil {
			inner = NewLocalPublisher(nil, factory.logger)
		}
	} else {
		inner = NewHostPublisher(app, factory.logger)
	}

	defaultTenant := ""
	if app != nil && app.Config != nil {
		defaultTenant = strings.TrimSpace(app.Config.Gateway.TenantID)
	}
	return NewAdapter(inner, defaultTenant, factory.logger)
}

func NewHostPublisher(app *bootstrap.App, logger *slog.Logger) Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	if app == nil || app.Config == nil {
		return FailurePublisher{reason: "app config is not configured"}
	}
	gatewayCfg := app.Config.Gateway
	client, err := NewHostClient(HostClientConfig{
		BaseURL:    gatewayCfg.BaseURL,
		APIPrefix:  gatewayCfg.APIPrefix,
		AuthScheme: gatewayCfg.AuthScheme,
		Token:      gatewayCfg.ToolToken,
		APIKey:     gatewayCfg.APIKey,
		TenantUUID: gatewayCfg.TenantID,
		UserAgent:  gatewayCfg.UserAgent,
		Timeout:    gatewayCfg.Timeout,
	})
	if err != nil {
		logger.Warn("failed to initialize wsbus host client", slog.String("error", err.Error()))
		return FailurePublisher{reason: err.Error()}
	}
	return client
}

type FailurePublisher struct {
	reason string
}

func (p FailurePublisher) Publish(_ context.Context, _ string, _ any, _ PublishOptions) PublishResult {
	msg := strings.TrimSpace(p.reason)
	if msg == "" {
		msg = "publisher is not configured"
	}
	return FailureResult(ErrorCodePublisherNotConfigured, msg)
}

func RegisterTopics(app *bootstrap.App, topics []string, opts PublishOptions, logger *slog.Logger) PublishResult {
	if app == nil || app.Config == nil {
		return FailureResult(ErrorCodePublisherNotConfigured, "app config is not configured")
	}
	if app.Config.Standalone {
		return SuccessResult()
	}
	if logger == nil {
		logger = slog.Default()
	}
	publisher := NewHostPublisher(app, logger)
	client, ok := publisher.(*HostClient)
	if !ok {
		return FailureResult(ErrorCodePublisherNotConfigured, "host client is not configured")
	}
	return client.RegisterTopics(context.Background(), topics, opts)
}
