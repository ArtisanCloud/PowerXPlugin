package logging

type LogrusEntry interface {
	WithFields(map[string]any) LogrusEntry
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
}

type LogrusAdapter struct {
	entry LogrusEntry
}

func NewLogrusAdapter(entry LogrusEntry) Logger {
	return &LogrusAdapter{entry: entry}
}

func (a *LogrusAdapter) WithFields(fields Fields) Logger {
	if a == nil || a.entry == nil {
		return &LogrusAdapter{}
	}
	if len(fields) == 0 {
		return &LogrusAdapter{entry: a.entry}
	}
	lf := map[string]any{}
	for k, v := range fields {
		lf[k] = v
	}
	return &LogrusAdapter{entry: a.entry.WithFields(lf)}
}

func (a *LogrusAdapter) Info(msg string) {
	if a == nil || a.entry == nil {
		return
	}
	a.entry.Info(msg)
}

func (a *LogrusAdapter) Warn(msg string) {
	if a == nil || a.entry == nil {
		return
	}
	a.entry.Warn(msg)
}

func (a *LogrusAdapter) Error(msg string) {
	if a == nil || a.entry == nil {
		return
	}
	a.entry.Error(msg)
}

