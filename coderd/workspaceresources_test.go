package coderd_test

import (
	"context"
	"testing"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/agent"
	"github.com/coder/coder/coderd"
	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/database"
	"github.com/coder/coder/peer"
	"github.com/coder/coder/peerbroker"
	"github.com/coder/coder/provisioner/echo"
	"github.com/coder/coder/provisionersdk/proto"
)

func TestWorkspaceResource(t *testing.T) {
	t.Parallel()
	t.Run("Get", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		user := coderdtest.CreateFirstUser(t, client)
		coderdtest.NewProvisionerDaemon(t, client)
		version := coderdtest.CreateProjectVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			Provision: []*proto.Provision_Response{{
				Type: &proto.Provision_Response_Complete{
					Complete: &proto.Provision_Complete{
						Resources: []*proto.Resource{{
							Name: "some",
							Type: "example",
							Agent: &proto.Agent{
								Id:   "something",
								Auth: &proto.Agent_Token{},
							},
						}},
					},
				},
			}},
		})
		coderdtest.AwaitProjectVersionJob(t, client, version.ID)
		project := coderdtest.CreateProject(t, client, user.OrganizationID, version.ID)
		workspace := coderdtest.CreateWorkspace(t, client, "", project.ID)
		build, err := client.CreateWorkspaceBuild(context.Background(), workspace.ID, coderd.CreateWorkspaceBuildRequest{
			ProjectVersionID: project.ActiveVersionID,
			Transition:       database.WorkspaceTransitionStart,
		})
		require.NoError(t, err)
		coderdtest.AwaitWorkspaceBuildJob(t, client, build.ID)
		resources, err := client.WorkspaceResourcesByBuild(context.Background(), build.ID)
		require.NoError(t, err)
		_, err = client.WorkspaceResource(context.Background(), resources[0].ID)
		require.NoError(t, err)
	})
}

func TestWorkspaceAgentListen(t *testing.T) {
	t.Parallel()
	client := coderdtest.New(t, nil)
	user := coderdtest.CreateFirstUser(t, client)
	daemonCloser := coderdtest.NewProvisionerDaemon(t, client)
	authToken := uuid.NewString()
	version := coderdtest.CreateProjectVersion(t, client, user.OrganizationID, &echo.Responses{
		Parse:           echo.ParseComplete,
		ProvisionDryRun: echo.ProvisionComplete,
		Provision: []*proto.Provision_Response{{
			Type: &proto.Provision_Response_Complete{
				Complete: &proto.Provision_Complete{
					Resources: []*proto.Resource{{
						Name: "example",
						Type: "aws_instance",
						Agent: &proto.Agent{
							Id: uuid.NewString(),
							Auth: &proto.Agent_Token{
								Token: authToken,
							},
						},
					}},
				},
			},
		}},
	})
	project := coderdtest.CreateProject(t, client, user.OrganizationID, version.ID)
	coderdtest.AwaitProjectVersionJob(t, client, version.ID)
	workspace := coderdtest.CreateWorkspace(t, client, "me", project.ID)
	build, err := client.CreateWorkspaceBuild(context.Background(), workspace.ID, coderd.CreateWorkspaceBuildRequest{
		ProjectVersionID: project.ActiveVersionID,
		Transition:       database.WorkspaceTransitionStart,
	})
	require.NoError(t, err)
	coderdtest.AwaitWorkspaceBuildJob(t, client, build.ID)
	daemonCloser.Close()

	agentClient := codersdk.New(client.URL)
	agentClient.SessionToken = authToken
	agentCloser := agent.New(agentClient.ListenWorkspaceAgent, &peer.ConnOptions{
		Logger: slogtest.Make(t, nil),
	})
	t.Cleanup(func() {
		_ = agentCloser.Close()
	})
	resources := coderdtest.AwaitWorkspaceAgents(t, client, build.ID)
	workspaceClient, err := client.DialWorkspaceAgent(context.Background(), resources[0].ID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = workspaceClient.DRPCConn().Close()
	})
	stream, err := workspaceClient.NegotiateConnection(context.Background())
	require.NoError(t, err)
	conn, err := peerbroker.Dial(stream, nil, &peer.ConnOptions{
		Logger: slogtest.Make(t, nil).Named("client").Leveled(slog.LevelDebug),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})
	_, err = conn.Ping()
	require.NoError(t, err)
}
