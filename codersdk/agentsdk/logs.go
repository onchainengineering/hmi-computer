package agentsdk

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"cdr.dev/slog"
	"github.com/coder/coder/codersdk"
	"github.com/coder/retry"
)

type startupLogsWriter struct {
	// The mutex is only required to ensure that calling Close is
	// concurrently safe with calling Write.
	mu     sync.Mutex // Protects following.
	closed bool
	buf    bytes.Buffer // Buffer to track partial lines.

	ctx   context.Context
	send  func(ctx context.Context, log ...StartupLog) error
	level codersdk.LogLevel
}

func (w *startupLogsWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	closed := w.closed
	w.mu.Unlock()
	if closed {
		return 0, xerrors.New("writer is closed")
	}

	n := len(p)
	for len(p) > 0 {
		nl := bytes.IndexByte(p, '\n')
		if nl == -1 {
			break
		}
		cr := 0
		if nl > 0 && p[nl-1] == '\r' {
			cr = 1
		}

		var partial []byte
		w.mu.Lock()
		if w.buf.Len() > 0 {
			partial = w.buf.Bytes()
			w.buf.Reset()
		}
		w.mu.Unlock()

		err := w.send(w.ctx, StartupLog{
			CreatedAt: time.Now().UTC(), // UTC, like database.Now().
			Level:     w.level,
			Output:    string(partial) + string(p[:nl-cr]),
		})
		if err != nil {
			return n - len(p), err
		}
		p = p[nl+1:]
	}
	if len(p) > 0 {
		w.mu.Lock()
		_, err := w.buf.Write(p)
		w.mu.Unlock()
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (w *startupLogsWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return xerrors.New("writer is already closed")
	}
	w.closed = true
	if w.buf.Len() > 0 {
		defer w.buf.Reset()
		return w.send(w.ctx, StartupLog{
			CreatedAt: time.Now().UTC(), // UTC, like database.Now().
			Level:     w.level,
			Output:    w.buf.String(),
		})
	}
	return nil
}

// StartupLogsWriter returns an io.WriteCloser that sends logs to the
// server. When closed, any remaining partially written logs will be
// sent to the server. If the context passed to StartupLogsWriter is
// canceled, any remaining logs will be discarded.
func StartupLogsWriter(ctx context.Context, sender func(ctx context.Context, log ...StartupLog) error, level codersdk.LogLevel) io.WriteCloser {
	return &startupLogsWriter{
		ctx:   ctx,
		send:  sender,
		level: level,
	}
}

// SendStartupLogs will send agent startup logs to the server. Calls to
// sendLog are non-blocking and will return an error if flushAndClose
// has been called. Calling sendLog concurrently is not supported. If
// the context passed to flushAndClose is canceled, any remaining logs
// will be discarded.
func StartupLogsSender(patchStartupLogs func(ctx context.Context, req PatchStartupLogs) error, logger slog.Logger) (sendLog func(ctx context.Context, log ...StartupLog) error, flushAndClose func(context.Context) error) {
	// The main context is used to close the sender goroutine and cancel
	// any outbound requests to the API. The shudown context is used to
	// signal the sender goroutine to flush logs and then exit.
	ctx, cancel := context.WithCancel(context.Background())
	shutdownCtx, shutdown := context.WithCancel(ctx)

	// Synchronous sender, there can only be one outbound send at a time.
	sendDone := make(chan struct{})
	send := make(chan []StartupLog, 1)
	go func() {
		// Set flushTimeout and backlogLimit so that logs are uploaded
		// once every 250ms or when 100 logs have been added to the
		// backlog, whichever comes first.
		flushTimeout := 250 * time.Millisecond
		backlogLimit := 100

		flush := time.NewTicker(flushTimeout)

		var backlog []StartupLog
		defer func() {
			flush.Stop()
			if len(backlog) > 0 {
				logger.Warn(ctx, "startup logs sender exiting early, discarding logs", slog.F("discarded_logs_count", len(backlog)))
			}
			logger.Debug(ctx, "startup logs sender exited")
			close(sendDone)
		}()

		done := false
		for {
			flushed := false
			select {
			case <-ctx.Done():
				return
			case <-shutdownCtx.Done():
				done = true

				// Check queued logs before flushing.
				select {
				case logs := <-send:
					backlog = append(backlog, logs...)
				default:
				}
			case <-flush.C:
				flushed = true
			case logs := <-send:
				backlog = append(backlog, logs...)
				flushed = len(backlog) >= backlogLimit
			}

			if (done || flushed) && len(backlog) > 0 {
				flush.Stop() // Lower the chance of a double flush.

				// Retry uploading logs until successful or a specific
				// error occurs. Note that we use the main context here,
				// meaning these requests won't be interrupted by
				// shutdown.
				for r := retry.New(time.Second, 5*time.Second); r.Wait(ctx); {
					err := patchStartupLogs(ctx, PatchStartupLogs{
						Logs: backlog,
					})
					if err == nil {
						break
					}

					if errors.Is(err, context.Canceled) {
						return
					}
					// This error is expected to be codersdk.Error, but it has
					// private fields so we can't fake it in tests.
					var statusErr interface{ StatusCode() int }
					if errors.As(err, &statusErr) {
						if statusErr.StatusCode() == http.StatusRequestEntityTooLarge {
							logger.Warn(ctx, "startup logs too large, discarding logs", slog.F("discarded_logs_count", len(backlog)), slog.Error(err))
							break
						}
					}
					logger.Error(ctx, "startup logs sender failed to upload logs, retrying later", slog.F("logs_count", len(backlog)), slog.Error(err))
				}
				if ctx.Err() != nil {
					return
				}
				backlog = nil

				// Anchor flush to the last log upload.
				flush.Reset(flushTimeout)
			}
			if done {
				return
			}
		}
	}()

	var queue []StartupLog
	sendLog = func(callCtx context.Context, log ...StartupLog) error {
		select {
		case <-shutdownCtx.Done():
			return xerrors.Errorf("closed: %w", ctx.Err())
		case <-callCtx.Done():
			return callCtx.Err()
		case queue = <-send:
			// Queue has not been captured by sender yet, re-use.
		default:
		}

		queue = append(queue, log...)
		send <- queue // Non-blocking.
		queue = nil

		return nil
	}
	flushAndClose = func(callCtx context.Context) error {
		defer cancel()
		shutdown()
		select {
		case <-sendDone:
			return nil
		case <-callCtx.Done():
			cancel()
			<-sendDone
			return callCtx.Err()
		}
	}
	return sendLog, flushAndClose
}
