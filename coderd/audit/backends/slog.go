package backends

import (
	"context"

	"github.com/fatih/structs"

	"cdr.dev/slog"
	"github.com/coder/coder/coderd/audit"
	"github.com/coder/coder/coderd/database"
)

type slogBackend struct {
	log slog.Logger
}

func NewSlogBackend(logger slog.Logger) audit.Backend {
	return slogBackend{log: logger}
}

func (slogBackend) Decision() audit.FilterDecision {
	return audit.FilterDecisionExport
}

func (b slogBackend) Export(ctx context.Context, alog database.AuditLog) error {
	m := structs.Map(alog)
	fields := make([]slog.Field, 0, len(m))
	for k, v := range m {
		fields = append(fields, slog.F(k, v))
	}

	b.log.Info(ctx, "audit_log", fields...)
	return nil
}
