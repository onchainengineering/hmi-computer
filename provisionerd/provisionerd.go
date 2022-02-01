package provisionerd

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"go.uber.org/atomic"

	"cdr.dev/slog"
	"github.com/coder/coder/provisionerd/proto"
	sdkproto "github.com/coder/coder/provisionersdk/proto"
	"github.com/coder/retry"
)

// Dialer represents the function to create a daemon client connection.
type Dialer func(ctx context.Context) (proto.DRPCProvisionerDaemonClient, error)

// Provisioners maps provisioner ID to implementation.
type Provisioners map[string]sdkproto.DRPCProvisionerClient

// Options provides customizations to the behavior of a provisioner daemon.
type Options struct {
	Logger slog.Logger

	PollInterval  time.Duration
	Provisioners  Provisioners
	WorkDirectory string
}

// New creates and starts a provisioner daemon.
func New(clientDialer Dialer, opts *Options) io.Closer {
	if opts.PollInterval == 0 {
		opts.PollInterval = 5 * time.Second
	}
	ctx, ctxCancel := context.WithCancel(context.Background())
	daemon := &provisionerDaemon{
		clientDialer: clientDialer,
		opts:         opts,

		closeContext:       ctx,
		closeContextCancel: ctxCancel,
		closed:             make(chan struct{}),
	}
	go daemon.connect()
	return daemon
}

type provisionerDaemon struct {
	opts *Options

	clientDialer Dialer
	connectMutex sync.Mutex
	client       proto.DRPCProvisionerDaemonClient
	updateStream proto.DRPCProvisionerDaemon_UpdateJobClient

	closeContext       context.Context
	closeContextCancel context.CancelFunc
	closed             chan struct{}
	closeMutex         sync.Mutex
	closeError         error

	runningJob              *proto.AcquiredJob
	runningJobContext       context.Context
	runningJobContextCancel context.CancelFunc
	runningJobMutex         sync.Mutex
	isRunningJob            atomic.Bool
}

// Connnect establishes a connection to coderd.
func (p *provisionerDaemon) connect() {
	p.connectMutex.Lock()
	defer p.connectMutex.Unlock()

	var err error
	for retrier := retry.New(50*time.Millisecond, 10*time.Second); retrier.Wait(p.closeContext); {
		p.client, err = p.clientDialer(p.closeContext)
		if err != nil {
			// Warn
			p.opts.Logger.Warn(context.Background(), "failed to dial", slog.Error(err))
			continue
		}
		p.updateStream, err = p.client.UpdateJob(p.closeContext)
		if err != nil {
			p.opts.Logger.Warn(context.Background(), "create update job stream", slog.Error(err))
			continue
		}
		p.opts.Logger.Debug(context.Background(), "connected")
		break
	}

	go func() {
		if p.isClosed() {
			return
		}
		select {
		case <-p.closed:
			return
		case <-p.updateStream.Context().Done():
			// We use the update stream to detect when the connection
			// has been interrupted. This works well, because logs need
			// to buffer if a job is running in the background.
			p.opts.Logger.Debug(context.Background(), "update stream ended", slog.Error(p.updateStream.Context().Err()))
			p.connect()
		}
	}()

	go func() {
		if p.isClosed() {
			return
		}
		ticker := time.NewTicker(p.opts.PollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-p.closed:
				return
			case <-p.updateStream.Context().Done():
				return
			case <-ticker.C:
				p.acquireJob()
			}
		}
	}()
}

func (p *provisionerDaemon) acquireJob() {
	p.runningJobMutex.Lock()
	defer p.runningJobMutex.Unlock()
	if p.isRunningJob.Load() {
		p.opts.Logger.Debug(context.Background(), "skipping acquire; job is already running")
		return
	}
	var err error
	p.runningJob, err = p.client.AcquireJob(p.closeContext, &proto.Empty{})
	if err != nil {
		p.opts.Logger.Error(context.Background(), "acquire job", slog.Error(err))
		return
	}
	if p.runningJob.JobId == "" {
		p.opts.Logger.Debug(context.Background(), "no jobs available")
		return
	}
	p.runningJobContext, p.runningJobContextCancel = context.WithCancel(p.closeContext)
	p.isRunningJob.Store(true)

	p.opts.Logger.Info(context.Background(), "acquired job",
		slog.F("organization_name", p.runningJob.OrganizationName),
		slog.F("project_name", p.runningJob.ProjectName),
		slog.F("username", p.runningJob.UserName),
		slog.F("provisioner", p.runningJob.Provisioner),
	)

	go p.runJob()
}

func (p *provisionerDaemon) runJob() {
	// It's safe to cast this ProvisionerType. This data is coming directly from coderd.
	provisioner, hasProvisioner := p.opts.Provisioners[p.runningJob.Provisioner]
	if !hasProvisioner {
		p.cancelActiveJob(fmt.Sprintf("provisioner %q not registered", p.runningJob.Provisioner))
		return
	}
	defer func() {
		// Cleanup the work directory after execution.
		err := os.RemoveAll(p.opts.WorkDirectory)
		if err != nil {
			p.cancelActiveJob(fmt.Sprintf("remove all from %q directory: %s", p.opts.WorkDirectory, err))
			return
		}
		p.opts.Logger.Debug(context.Background(), "cleaned up work directory")
	}()

	err := os.MkdirAll(p.opts.WorkDirectory, 0600)
	if err != nil {
		p.cancelActiveJob(fmt.Sprintf("create work directory %q: %s", p.opts.WorkDirectory, err))
		return
	}

	p.opts.Logger.Info(context.Background(), "unpacking project source archive", slog.F("size_bytes", len(p.runningJob.ProjectSourceArchive)))
	reader := tar.NewReader(bytes.NewBuffer(p.runningJob.ProjectSourceArchive))
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			p.cancelActiveJob(fmt.Sprintf("read project source archive: %s", err))
			return
		}
		// #nosec
		path := filepath.Join(p.opts.WorkDirectory, header.Name)
		if !strings.HasPrefix(path, filepath.Clean(p.opts.WorkDirectory)) {
			p.cancelActiveJob("tar attempts to target relative upper directory")
			return
		}
		mode := header.FileInfo().Mode()
		if mode == 0 {
			mode = 0600
		}
		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(path, mode)
			if err != nil {
				p.cancelActiveJob(fmt.Sprintf("mkdir %q: %s", path, err))
				return
			}
			p.opts.Logger.Debug(context.Background(), "extracted directory", slog.F("path", path))
		case tar.TypeReg:
			file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, mode)
			if err != nil {
				p.cancelActiveJob(fmt.Sprintf("create file %q: %s", path, err))
				return
			}
			// Max file size of 10MB.
			size, err := io.CopyN(file, reader, (1<<20)*10)
			if errors.Is(err, io.EOF) {
				err = nil
			}
			if err != nil {
				p.cancelActiveJob(fmt.Sprintf("copy file %q: %s", path, err))
				return
			}
			err = file.Close()
			if err != nil {
				p.cancelActiveJob(fmt.Sprintf("close file %q: %s", path, err))
				return
			}
			p.opts.Logger.Debug(context.Background(), "extracted file",
				slog.F("size_bytes", size),
				slog.F("path", path),
				slog.F("mode", mode),
			)
		}
	}

	switch jobType := p.runningJob.Type.(type) {
	case *proto.AcquiredJob_ProjectImport_:
		p.opts.Logger.Debug(context.Background(), "acquired job is project import",
			slog.F("project_history_name", jobType.ProjectImport.ProjectHistoryName),
		)

		p.runProjectImport(provisioner, jobType)
	case *proto.AcquiredJob_WorkspaceProvision_:
		p.opts.Logger.Debug(context.Background(), "acquired job is workspace provision",
			slog.F("workspace_name", jobType.WorkspaceProvision.WorkspaceName),
			slog.F("state_length", len(jobType.WorkspaceProvision.State)),
			slog.F("parameters", jobType.WorkspaceProvision.ParameterValues),
		)

		p.runWorkspaceProvision(provisioner, jobType)
	default:
		p.cancelActiveJob(fmt.Sprintf("unknown job type %q; ensure your provisioner daemon is up-to-date", reflect.TypeOf(p.runningJob.Type).String()))
		return
	}

	p.opts.Logger.Info(context.Background(), "completed job")
	p.isRunningJob.Store(false)
}

func (p *provisionerDaemon) runProjectImport(provisioner sdkproto.DRPCProvisionerClient, job *proto.AcquiredJob_ProjectImport_) {
	stream, err := provisioner.Parse(p.runningJobContext, &sdkproto.Parse_Request{
		Directory: p.opts.WorkDirectory,
	})
	if err != nil {
		p.cancelActiveJob(fmt.Sprintf("parse source: %s", err))
		return
	}
	defer stream.Close()
	for {
		msg, err := stream.Recv()
		if err != nil {
			p.cancelActiveJob(fmt.Sprintf("recv parse source: %s", err))
			return
		}
		switch msgType := msg.Type.(type) {
		case *sdkproto.Parse_Response_Log:
			p.opts.Logger.Debug(context.Background(), "parse job logged",
				slog.F("level", msgType.Log.Level),
				slog.F("output", msgType.Log.Output),
				slog.F("project_history_id", job.ProjectImport.ProjectHistoryId),
			)

			err = p.updateStream.Send(&proto.JobUpdate{
				JobId: p.runningJob.JobId,
				ProjectImportLogs: []*proto.Log{{
					Source:    proto.LogSource_PROVISIONER,
					Level:     msgType.Log.Level,
					CreatedAt: time.Now().UTC().UnixMilli(),
					Output:    msgType.Log.Output,
				}},
			})
			if err != nil {
				p.cancelActiveJob(fmt.Sprintf("update job: %s", err))
				return
			}
		case *sdkproto.Parse_Response_Complete:
			_, err = p.client.CompleteJob(p.runningJobContext, &proto.CompletedJob{
				JobId: p.runningJob.JobId,
				Type: &proto.CompletedJob_ProjectImport_{
					ProjectImport: &proto.CompletedJob_ProjectImport{
						ParameterSchemas: msgType.Complete.ParameterSchemas,
					},
				},
			})
			if err != nil {
				p.cancelActiveJob(fmt.Sprintf("complete job: %s", err))
				return
			}
			// Return so we stop looping!
			return
		default:
			p.cancelActiveJob(fmt.Sprintf("invalid message type %q received from provisioner",
				reflect.TypeOf(msg.Type).String()))
			return
		}
	}
}

func (p *provisionerDaemon) runWorkspaceProvision(provisioner sdkproto.DRPCProvisionerClient, job *proto.AcquiredJob_WorkspaceProvision_) {
	stream, err := provisioner.Provision(p.closeContext, &sdkproto.Provision_Request{
		Directory:       p.opts.WorkDirectory,
		ParameterValues: job.WorkspaceProvision.ParameterValues,
		State:           job.WorkspaceProvision.State,
	})
	if err != nil {
		p.cancelActiveJob(fmt.Sprintf("provision: %s", err))
		return
	}
	defer stream.Close()

	for {
		msg, err := stream.Recv()
		if err != nil {
			p.cancelActiveJob(fmt.Sprintf("recv workspace provision: %s", err))
			return
		}
		switch msgType := msg.Type.(type) {
		case *sdkproto.Provision_Response_Log:
			p.opts.Logger.Debug(context.Background(), "workspace provision job logged",
				slog.F("level", msgType.Log.Level),
				slog.F("output", msgType.Log.Output),
				slog.F("workspace_history_id", job.WorkspaceProvision.WorkspaceHistoryId),
			)

			err = p.updateStream.Send(&proto.JobUpdate{
				JobId: p.runningJob.JobId,
				WorkspaceProvisionLogs: []*proto.Log{{
					Source:    proto.LogSource_PROVISIONER,
					Level:     msgType.Log.Level,
					CreatedAt: time.Now().UTC().UnixMilli(),
					Output:    msgType.Log.Output,
				}},
			})
			if err != nil {
				p.cancelActiveJob(fmt.Sprintf("send job update: %s", err))
				return
			}
		case *sdkproto.Provision_Response_Complete:
			p.opts.Logger.Info(context.Background(), "provision successful; marking job as complete",
				slog.F("resource_count", len(msgType.Complete.Resources)),
				slog.F("resources", msgType.Complete.Resources),
				slog.F("state_length", len(msgType.Complete.State)),
			)

			// Complete job may need to be async if we disconnected...
			// When we reconnect we can flush any of these cached values.
			_, err = p.client.CompleteJob(p.closeContext, &proto.CompletedJob{
				JobId: p.runningJob.JobId,
				Type: &proto.CompletedJob_WorkspaceProvision_{
					WorkspaceProvision: &proto.CompletedJob_WorkspaceProvision{
						State:     msgType.Complete.State,
						Resources: msgType.Complete.Resources,
					},
				},
			})
			if err != nil {
				p.cancelActiveJob(fmt.Sprintf("complete job: %s", err))
				return
			}
			// Return so we stop looping!
			return
		default:
			p.cancelActiveJob(fmt.Sprintf("invalid message type %q received from provisioner",
				reflect.TypeOf(msg.Type).String()))
			return
		}
	}
}

func (p *provisionerDaemon) cancelActiveJob(errMsg string) {
	p.runningJobMutex.Lock()
	defer p.runningJobMutex.Unlock()
	if !p.isRunningJob.Load() {
		p.opts.Logger.Warn(context.Background(), "skipping job cancel; none running", slog.F("error_message", errMsg))
		return
	}

	p.opts.Logger.Info(context.Background(), "canceling running job",
		slog.F("error_message", errMsg),
		slog.F("job_id", p.runningJob.JobId),
	)
	_, err := p.client.CancelJob(p.closeContext, &proto.CancelledJob{
		JobId: p.runningJob.JobId,
		Error: fmt.Sprintf("provisioner daemon: %s", errMsg),
	})
	if err != nil {
		p.opts.Logger.Error(context.Background(), "couldn't cancel job", slog.Error(err))
	}
	p.opts.Logger.Debug(context.Background(), "canceled running job")
	p.runningJobContextCancel()
	p.isRunningJob.Store(false)
}

// isClosed returns whether the API is closed or not.
func (p *provisionerDaemon) isClosed() bool {
	select {
	case <-p.closed:
		return true
	default:
		return false
	}
}

// Close ends the provisioner. It will mark any running jobs as canceled.
func (p *provisionerDaemon) Close() error {
	return p.closeWithError(nil)
}

// closeWithError closes the provisioner; subsequent reads/writes will return the error err.
func (p *provisionerDaemon) closeWithError(err error) error {
	p.closeMutex.Lock()
	defer p.closeMutex.Unlock()
	if p.isClosed() {
		return p.closeError
	}

	if p.isRunningJob.Load() {
		errMsg := "provisioner daemon was shutdown gracefully"
		if err != nil {
			errMsg = err.Error()
		}
		p.cancelActiveJob(errMsg)
	}

	p.opts.Logger.Debug(context.Background(), "closing server with error", slog.Error(err))
	p.closeError = err
	close(p.closed)
	p.closeContextCancel()

	if p.updateStream != nil {
		_ = p.client.DRPCConn().Close()
		_ = p.updateStream.Close()
	}

	return err
}
