package provisionerd_test

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/goleak"
	"golang.org/x/xerrors"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"

	"github.com/coder/coder/provisionerd"
	"github.com/coder/coder/provisionerd/proto"
	"github.com/coder/coder/provisionersdk"
	sdkproto "github.com/coder/coder/provisionersdk/proto"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestProvisionerd(t *testing.T) {
	t.Parallel()

	noopUpdateJob := func(ctx context.Context, update *proto.UpdateJobRequest) (*proto.UpdateJobResponse, error) {
		return &proto.UpdateJobResponse{}, nil
	}

	t.Run("InstantClose", func(t *testing.T) {
		t.Parallel()
		closer := createProvisionerd(t, func(ctx context.Context) (proto.DRPCProvisionerDaemonClient, error) {
			return createProvisionerDaemonClient(t, provisionerDaemonTestServer{}), nil
		}, provisionerd.Provisioners{})
		require.NoError(t, closer.Close())
	})

	t.Run("ConnectErrorClose", func(t *testing.T) {
		t.Parallel()
		completeChan := make(chan struct{})
		closer := createProvisionerd(t, func(ctx context.Context) (proto.DRPCProvisionerDaemonClient, error) {
			defer close(completeChan)
			return nil, xerrors.New("an error")
		}, provisionerd.Provisioners{})
		<-completeChan
		require.NoError(t, closer.Close())
	})

	t.Run("AcquireEmptyJob", func(t *testing.T) {
		// The provisioner daemon is supposed to skip the job acquire if
		// the job provided is empty. This is to show it successfully
		// tried to get a job, but none were available.
		t.Parallel()
		completeChan := make(chan struct{})
		closer := createProvisionerd(t, func(ctx context.Context) (proto.DRPCProvisionerDaemonClient, error) {
			acquireJobAttempt := 0
			return createProvisionerDaemonClient(t, provisionerDaemonTestServer{
				acquireJob: func(ctx context.Context, _ *proto.Empty) (*proto.AcquiredJob, error) {
					if acquireJobAttempt == 1 {
						close(completeChan)
					}
					acquireJobAttempt++
					return &proto.AcquiredJob{}, nil
				},
				updateJob: noopUpdateJob,
			}), nil
		}, provisionerd.Provisioners{})
		<-completeChan
		require.NoError(t, closer.Close())
	})

	t.Run("CloseCancelsJob", func(t *testing.T) {
		t.Parallel()
		completeChan := make(chan struct{})
		var closer io.Closer
		var closerMutex sync.Mutex
		closerMutex.Lock()
		closer = createProvisionerd(t, func(ctx context.Context) (proto.DRPCProvisionerDaemonClient, error) {
			return createProvisionerDaemonClient(t, provisionerDaemonTestServer{
				acquireJob: func(ctx context.Context, _ *proto.Empty) (*proto.AcquiredJob, error) {
					return &proto.AcquiredJob{
						JobId:       "test",
						Provisioner: "someprovisioner",
						ProjectSourceArchive: createTar(t, map[string]string{
							"test.txt": "content",
						}),
						Type: &proto.AcquiredJob_ProjectImport_{
							ProjectImport: &proto.AcquiredJob_ProjectImport{
								Metadata: &sdkproto.Provision_Metadata{},
							},
						},
					}, nil
				},
				updateJob: noopUpdateJob,
				failJob: func(ctx context.Context, job *proto.FailedJob) (*proto.Empty, error) {
					close(completeChan)
					return &proto.Empty{}, nil
				},
			}), nil
		}, provisionerd.Provisioners{
			"someprovisioner": createProvisionerClient(t, provisionerTestServer{
				parse: func(request *sdkproto.Parse_Request, stream sdkproto.DRPCProvisioner_ParseStream) error {
					closerMutex.Lock()
					defer closerMutex.Unlock()
					return closer.Close()
				},
			}),
		})
		closerMutex.Unlock()
		<-completeChan
		require.NoError(t, closer.Close())
	})

	t.Run("MaliciousTar", func(t *testing.T) {
		// Ensures tars with "../../../etc/passwd" as the path
		// are not allowed to run, and will fail the job.
		t.Parallel()
		completeChan := make(chan struct{})
		closer := createProvisionerd(t, func(ctx context.Context) (proto.DRPCProvisionerDaemonClient, error) {
			return createProvisionerDaemonClient(t, provisionerDaemonTestServer{
				acquireJob: func(ctx context.Context, _ *proto.Empty) (*proto.AcquiredJob, error) {
					return &proto.AcquiredJob{
						JobId:       "test",
						Provisioner: "someprovisioner",
						ProjectSourceArchive: createTar(t, map[string]string{
							"../../../etc/passwd": "content",
						}),
						Type: &proto.AcquiredJob_ProjectImport_{
							ProjectImport: &proto.AcquiredJob_ProjectImport{
								Metadata: &sdkproto.Provision_Metadata{},
							},
						},
					}, nil
				},
				updateJob: noopUpdateJob,
				failJob: func(ctx context.Context, job *proto.FailedJob) (*proto.Empty, error) {
					close(completeChan)
					return &proto.Empty{}, nil
				},
			}), nil
		}, provisionerd.Provisioners{
			"someprovisioner": createProvisionerClient(t, provisionerTestServer{}),
		})
		<-completeChan
		require.NoError(t, closer.Close())
	})

	t.Run("RunningPeriodicUpdate", func(t *testing.T) {
		t.Parallel()
		completeChan := make(chan struct{})
		closer := createProvisionerd(t, func(ctx context.Context) (proto.DRPCProvisionerDaemonClient, error) {
			return createProvisionerDaemonClient(t, provisionerDaemonTestServer{
				acquireJob: func(ctx context.Context, _ *proto.Empty) (*proto.AcquiredJob, error) {
					return &proto.AcquiredJob{
						JobId:       "test",
						Provisioner: "someprovisioner",
						ProjectSourceArchive: createTar(t, map[string]string{
							"test.txt": "content",
						}),
						Type: &proto.AcquiredJob_ProjectImport_{
							ProjectImport: &proto.AcquiredJob_ProjectImport{
								Metadata: &sdkproto.Provision_Metadata{},
							},
						},
					}, nil
				},
				updateJob: func(ctx context.Context, update *proto.UpdateJobRequest) (*proto.UpdateJobResponse, error) {
					close(completeChan)
					return &proto.UpdateJobResponse{}, nil
				},
				failJob: func(ctx context.Context, job *proto.FailedJob) (*proto.Empty, error) {
					return &proto.Empty{}, nil
				},
			}), nil
		}, provisionerd.Provisioners{
			"someprovisioner": createProvisionerClient(t, provisionerTestServer{
				parse: func(request *sdkproto.Parse_Request, stream sdkproto.DRPCProvisioner_ParseStream) error {
					<-stream.Context().Done()
					return nil
				},
			}),
		})
		<-completeChan
		require.NoError(t, closer.Close())
	})

	t.Run("ProjectImport", func(t *testing.T) {
		t.Parallel()
		var (
			didComplete   atomic.Bool
			didLog        atomic.Bool
			didAcquireJob atomic.Bool
			didDryRun     atomic.Bool
		)
		completeChan := make(chan struct{})
		closer := createProvisionerd(t, func(ctx context.Context) (proto.DRPCProvisionerDaemonClient, error) {
			return createProvisionerDaemonClient(t, provisionerDaemonTestServer{
				acquireJob: func(ctx context.Context, _ *proto.Empty) (*proto.AcquiredJob, error) {
					if didAcquireJob.Load() {
						close(completeChan)
						return &proto.AcquiredJob{}, nil
					}
					didAcquireJob.Store(true)
					return &proto.AcquiredJob{
						JobId:       "test",
						Provisioner: "someprovisioner",
						ProjectSourceArchive: createTar(t, map[string]string{
							"test.txt": "content",
						}),
						Type: &proto.AcquiredJob_ProjectImport_{
							ProjectImport: &proto.AcquiredJob_ProjectImport{
								Metadata: &sdkproto.Provision_Metadata{},
							},
						},
					}, nil
				},
				updateJob: func(ctx context.Context, update *proto.UpdateJobRequest) (*proto.UpdateJobResponse, error) {
					if len(update.Logs) != 0 {
						didLog.Store(true)
					}
					return &proto.UpdateJobResponse{}, nil
				},
				completeJob: func(ctx context.Context, job *proto.CompletedJob) (*proto.Empty, error) {
					didComplete.Store(true)
					return &proto.Empty{}, nil
				},
			}), nil
		}, provisionerd.Provisioners{
			"someprovisioner": createProvisionerClient(t, provisionerTestServer{
				parse: func(request *sdkproto.Parse_Request, stream sdkproto.DRPCProvisioner_ParseStream) error {
					data, err := os.ReadFile(filepath.Join(request.Directory, "test.txt"))
					require.NoError(t, err)
					require.Equal(t, "content", string(data))

					err = stream.Send(&sdkproto.Parse_Response{
						Type: &sdkproto.Parse_Response_Log{
							Log: &sdkproto.Log{
								Level:  sdkproto.LogLevel_INFO,
								Output: "hello",
							},
						},
					})
					require.NoError(t, err)

					err = stream.Send(&sdkproto.Parse_Response{
						Type: &sdkproto.Parse_Response_Complete{
							Complete: &sdkproto.Parse_Complete{},
						},
					})
					require.NoError(t, err)
					return nil
				},
				provision: func(request *sdkproto.Provision_Request, stream sdkproto.DRPCProvisioner_ProvisionStream) error {
					if request.DryRun {
						didDryRun.Store(true)
					}
					err := stream.Send(&sdkproto.Provision_Response{
						Type: &sdkproto.Provision_Response_Log{
							Log: &sdkproto.Log{
								Level:  sdkproto.LogLevel_INFO,
								Output: "hello",
							},
						},
					})
					require.NoError(t, err)

					err = stream.Send(&sdkproto.Provision_Response{
						Type: &sdkproto.Provision_Response_Complete{
							Complete: &sdkproto.Provision_Complete{
								Resources: []*sdkproto.Resource{},
							},
						},
					})
					require.NoError(t, err)
					return nil
				},
			}),
		})
		<-completeChan
		require.True(t, didLog.Load())
		require.True(t, didComplete.Load())
		require.True(t, didDryRun.Load())
		require.NoError(t, closer.Close())
	})

	t.Run("WorkspaceProvision", func(t *testing.T) {
		t.Parallel()
		var (
			didComplete   atomic.Bool
			didLog        atomic.Bool
			didAcquireJob atomic.Bool
		)
		completeChan := make(chan struct{})
		closer := createProvisionerd(t, func(ctx context.Context) (proto.DRPCProvisionerDaemonClient, error) {
			return createProvisionerDaemonClient(t, provisionerDaemonTestServer{
				acquireJob: func(ctx context.Context, _ *proto.Empty) (*proto.AcquiredJob, error) {
					if didAcquireJob.Load() {
						close(completeChan)
						return &proto.AcquiredJob{}, nil
					}
					didAcquireJob.Store(true)
					return &proto.AcquiredJob{
						JobId:       "test",
						Provisioner: "someprovisioner",
						ProjectSourceArchive: createTar(t, map[string]string{
							"test.txt": "content",
						}),
						Type: &proto.AcquiredJob_WorkspaceProvision_{
							WorkspaceProvision: &proto.AcquiredJob_WorkspaceProvision{
								Metadata: &sdkproto.Provision_Metadata{},
							},
						},
					}, nil
				},
				updateJob: func(ctx context.Context, update *proto.UpdateJobRequest) (*proto.UpdateJobResponse, error) {
					if len(update.Logs) != 0 {
						didLog.Store(true)
					}
					return &proto.UpdateJobResponse{}, nil
				},
				completeJob: func(ctx context.Context, job *proto.CompletedJob) (*proto.Empty, error) {
					didComplete.Store(true)
					return &proto.Empty{}, nil
				},
			}), nil
		}, provisionerd.Provisioners{
			"someprovisioner": createProvisionerClient(t, provisionerTestServer{
				provision: func(request *sdkproto.Provision_Request, stream sdkproto.DRPCProvisioner_ProvisionStream) error {
					err := stream.Send(&sdkproto.Provision_Response{
						Type: &sdkproto.Provision_Response_Log{
							Log: &sdkproto.Log{
								Level:  sdkproto.LogLevel_DEBUG,
								Output: "wow",
							},
						},
					})
					require.NoError(t, err)

					err = stream.Send(&sdkproto.Provision_Response{
						Type: &sdkproto.Provision_Response_Complete{
							Complete: &sdkproto.Provision_Complete{},
						},
					})
					require.NoError(t, err)
					return nil
				},
			}),
		})
		<-completeChan
		require.True(t, didLog.Load())
		require.True(t, didComplete.Load())
		require.NoError(t, closer.Close())
	})

	t.Run("WorkspaceProvisionFailComplete", func(t *testing.T) {
		t.Parallel()
		var (
			didFail       atomic.Bool
			didAcquireJob atomic.Bool
		)
		completeChan := make(chan struct{})
		closer := createProvisionerd(t, func(ctx context.Context) (proto.DRPCProvisionerDaemonClient, error) {
			return createProvisionerDaemonClient(t, provisionerDaemonTestServer{
				acquireJob: func(ctx context.Context, _ *proto.Empty) (*proto.AcquiredJob, error) {
					if didAcquireJob.Load() {
						close(completeChan)
						return &proto.AcquiredJob{}, nil
					}
					didAcquireJob.Store(true)
					return &proto.AcquiredJob{
						JobId:       "test",
						Provisioner: "someprovisioner",
						ProjectSourceArchive: createTar(t, map[string]string{
							"test.txt": "content",
						}),
						Type: &proto.AcquiredJob_WorkspaceProvision_{
							WorkspaceProvision: &proto.AcquiredJob_WorkspaceProvision{
								Metadata: &sdkproto.Provision_Metadata{},
							},
						},
					}, nil
				},
				failJob: func(ctx context.Context, job *proto.FailedJob) (*proto.Empty, error) {
					didFail.Store(true)
					return &proto.Empty{}, nil
				},
			}), nil
		}, provisionerd.Provisioners{
			"someprovisioner": createProvisionerClient(t, provisionerTestServer{
				provision: func(request *sdkproto.Provision_Request, stream sdkproto.DRPCProvisioner_ProvisionStream) error {
					return stream.Send(&sdkproto.Provision_Response{
						Type: &sdkproto.Provision_Response_Complete{
							Complete: &sdkproto.Provision_Complete{
								Error: "some error",
							},
						},
					})
				},
			}),
		})
		<-completeChan
		require.True(t, didFail.Load())
		require.NoError(t, closer.Close())
	})

	t.Run("Shutdown", func(t *testing.T) {
		t.Parallel()
		updateChan := make(chan struct{})
		completeChan := make(chan struct{})
		shutdownCtx, shutdownCtxCancel := context.WithCancel(context.Background())
		defer shutdownCtxCancel()
		server := createProvisionerd(t, func(ctx context.Context) (proto.DRPCProvisionerDaemonClient, error) {
			return createProvisionerDaemonClient(t, provisionerDaemonTestServer{
				acquireJob: func(ctx context.Context, _ *proto.Empty) (*proto.AcquiredJob, error) {
					return &proto.AcquiredJob{
						JobId:       "test",
						Provisioner: "someprovisioner",
						ProjectSourceArchive: createTar(t, map[string]string{
							"test.txt": "content",
						}),
						Type: &proto.AcquiredJob_WorkspaceProvision_{
							WorkspaceProvision: &proto.AcquiredJob_WorkspaceProvision{
								Metadata: &sdkproto.Provision_Metadata{},
							},
						},
					}, nil
				},
				updateJob: func(ctx context.Context, update *proto.UpdateJobRequest) (*proto.UpdateJobResponse, error) {
					if len(update.Logs) > 0 {
						// Close on a log so we know when the job is in progress!
						close(updateChan)
					}
					return &proto.UpdateJobResponse{}, nil
				},
				failJob: func(ctx context.Context, job *proto.FailedJob) (*proto.Empty, error) {
					close(completeChan)
					return &proto.Empty{}, nil
				},
			}), nil
		}, provisionerd.Provisioners{
			"someprovisioner": createProvisionerClient(t, provisionerTestServer{
				shutdown: func(_ context.Context, _ *sdkproto.Empty) (*sdkproto.Empty, error) {
					shutdownCtxCancel()
					return &sdkproto.Empty{}, nil
				},
				provision: func(request *sdkproto.Provision_Request, stream sdkproto.DRPCProvisioner_ProvisionStream) error {
					err := stream.Send(&sdkproto.Provision_Response{
						Type: &sdkproto.Provision_Response_Log{
							Log: &sdkproto.Log{
								Level:  sdkproto.LogLevel_DEBUG,
								Output: "in progress",
							},
						},
					})
					require.NoError(t, err)
					<-shutdownCtx.Done()
					err = stream.Send(&sdkproto.Provision_Response{
						Type: &sdkproto.Provision_Response_Complete{
							Complete: &sdkproto.Provision_Complete{
								Error: "some error",
							},
						},
					})
					require.NoError(t, err)
					return nil
				},
			}),
		})
		<-updateChan
		err := server.Shutdown(context.Background())
		require.NoError(t, err)
		<-completeChan
		require.NoError(t, server.Close())
	})
}

// Creates an in-memory tar of the files provided.
func createTar(t *testing.T, files map[string]string) []byte {
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	for path, content := range files {
		err := writer.WriteHeader(&tar.Header{
			Name: path,
			Size: int64(len(content)),
		})
		require.NoError(t, err)

		_, err = writer.Write([]byte(content))
		require.NoError(t, err)
	}

	err := writer.Flush()
	require.NoError(t, err)
	return buffer.Bytes()
}

// Creates a provisionerd implementation with the provided dialer and provisioners.
func createProvisionerd(t *testing.T, dialer provisionerd.Dialer, provisioners provisionerd.Provisioners) *provisionerd.Server {
	server := provisionerd.New(dialer, &provisionerd.Options{
		Logger:         slogtest.Make(t, nil).Named("provisionerd").Leveled(slog.LevelDebug),
		PollInterval:   50 * time.Millisecond,
		UpdateInterval: 50 * time.Millisecond,
		Provisioners:   provisioners,
		WorkDirectory:  t.TempDir(),
	})
	t.Cleanup(func() {
		_ = server.Close()
	})
	return server
}

// Creates a provisionerd protobuf client that's connected
// to the server implementation provided.
func createProvisionerDaemonClient(t *testing.T, server provisionerDaemonTestServer) proto.DRPCProvisionerDaemonClient {
	clientPipe, serverPipe := provisionersdk.TransportPipe()
	t.Cleanup(func() {
		_ = clientPipe.Close()
		_ = serverPipe.Close()
	})
	mux := drpcmux.New()
	err := proto.DRPCRegisterProvisionerDaemon(mux, &server)
	require.NoError(t, err)
	srv := drpcserver.New(mux)
	ctx, cancelFunc := context.WithCancel(context.Background())
	t.Cleanup(cancelFunc)
	go func() {
		_ = srv.Serve(ctx, serverPipe)
	}()
	return proto.NewDRPCProvisionerDaemonClient(provisionersdk.Conn(clientPipe))
}

// Creates a provisioner protobuf client that's connected
// to the server implementation provided.
func createProvisionerClient(t *testing.T, server provisionerTestServer) sdkproto.DRPCProvisionerClient {
	clientPipe, serverPipe := provisionersdk.TransportPipe()
	t.Cleanup(func() {
		_ = clientPipe.Close()
		_ = serverPipe.Close()
	})
	mux := drpcmux.New()
	err := sdkproto.DRPCRegisterProvisioner(mux, &server)
	require.NoError(t, err)
	srv := drpcserver.New(mux)
	ctx, cancelFunc := context.WithCancel(context.Background())
	t.Cleanup(cancelFunc)
	go func() {
		_ = srv.Serve(ctx, serverPipe)
	}()
	return sdkproto.NewDRPCProvisionerClient(provisionersdk.Conn(clientPipe))
}

type provisionerTestServer struct {
	shutdown  func(_ context.Context, _ *sdkproto.Empty) (*sdkproto.Empty, error)
	parse     func(request *sdkproto.Parse_Request, stream sdkproto.DRPCProvisioner_ParseStream) error
	provision func(request *sdkproto.Provision_Request, stream sdkproto.DRPCProvisioner_ProvisionStream) error
}

func (p *provisionerTestServer) Shutdown(ctx context.Context, empty *sdkproto.Empty) (*sdkproto.Empty, error) {
	return p.shutdown(ctx, empty)
}

func (p *provisionerTestServer) Parse(request *sdkproto.Parse_Request, stream sdkproto.DRPCProvisioner_ParseStream) error {
	return p.parse(request, stream)
}

func (p *provisionerTestServer) Provision(request *sdkproto.Provision_Request, stream sdkproto.DRPCProvisioner_ProvisionStream) error {
	return p.provision(request, stream)
}

// Fulfills the protobuf interface for a ProvisionerDaemon with
// passable functions for dynamic functionality.
type provisionerDaemonTestServer struct {
	acquireJob  func(ctx context.Context, _ *proto.Empty) (*proto.AcquiredJob, error)
	updateJob   func(ctx context.Context, update *proto.UpdateJobRequest) (*proto.UpdateJobResponse, error)
	failJob     func(ctx context.Context, job *proto.FailedJob) (*proto.Empty, error)
	completeJob func(ctx context.Context, job *proto.CompletedJob) (*proto.Empty, error)
}

func (p *provisionerDaemonTestServer) AcquireJob(ctx context.Context, empty *proto.Empty) (*proto.AcquiredJob, error) {
	return p.acquireJob(ctx, empty)
}

func (p *provisionerDaemonTestServer) UpdateJob(ctx context.Context, update *proto.UpdateJobRequest) (*proto.UpdateJobResponse, error) {
	return p.updateJob(ctx, update)
}

func (p *provisionerDaemonTestServer) FailJob(ctx context.Context, job *proto.FailedJob) (*proto.Empty, error) {
	return p.failJob(ctx, job)
}

func (p *provisionerDaemonTestServer) CompleteJob(ctx context.Context, job *proto.CompletedJob) (*proto.Empty, error) {
	return p.completeJob(ctx, job)
}
