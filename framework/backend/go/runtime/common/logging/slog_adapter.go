package logging

import "log/slog"

type SlogAdapter struct {
	logger *slog.Logger
}

func NewSlogAdapter(logger *slog.Logger) Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return &SlogAdapter{logger: logger}
}

func (a *SlogAdapter) WithFields(fields Fields) Logger {
	if a == nil {
		return NewSlogAdapter(nil)
	}
	if len(fields) == 0 {
		return &SlogAdapter{logger: a.logger}
	}
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	return &SlogAdapter{logger: a.logger.With(attrs...)}
}

func (a *SlogAdapter) Info(msg string) {
	if a == nil || a.logger == nil {
		return
	}
	a.logger.Info(msg)
}

func (a *SlogAdapter) Warn(msg string) {
	if a == nil || a.logger == nil {
		return
	}
	a.logger.Warn(msg)
}

func (a *SlogAdapter) Error(msg string) {
	if a == nil || a.logger == nil {
		return
	}
	a.logger.Error(msg)
}

