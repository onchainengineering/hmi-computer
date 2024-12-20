package coderd_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onchainengineering/hmi-computer/v2/coderd/coderdtest"
	"github.com/onchainengineering/hmi-computer/v2/provisioner/echo"
	"github.com/onchainengineering/hmi-computer/v2/provisionersdk/proto"
	"github.com/onchainengineering/hmi-computer/v2/testutil"
)

func TestProvisionerJobLogs(t *testing.T) {
	t.Parallel()
	t.Run("StreamAfterComplete", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{IncludeProvisionerDaemon: true})
		user := coderdtest.CreateFirstUser(t, client)
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionApply: []*proto.Response{{
				Type: &proto.Response_Log{
					Log: &proto.Log{
						Level:  proto.LogLevel_INFO,
						Output: "log-output",
					},
				},
			}, {
				Type: &proto.Response_Apply{
					Apply: &proto.ApplyComplete{},
				},
			}},
		})
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		coderdtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := coderdtest.CreateWorkspace(t, client, template.ID)
		coderdtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		logs, closer, err := client.WorkspaceBuildLogsAfter(ctx, workspace.LatestBuild.ID, 0)
		require.NoError(t, err)
		defer closer.Close()
		for {
			log, ok := <-logs
			t.Logf("got log: [%s] %s %s", log.Level, log.Stage, log.Output)
			if !ok {
				return
			}
		}
	})

	t.Run("StreamWhileRunning", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{IncludeProvisionerDaemon: true})
		user := coderdtest.CreateFirstUser(t, client)
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionApply: []*proto.Response{{
				Type: &proto.Response_Log{
					Log: &proto.Log{
						Level:  proto.LogLevel_INFO,
						Output: "log-output",
					},
				},
			}, {
				Type: &proto.Response_Apply{
					Apply: &proto.ApplyComplete{},
				},
			}},
		})
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		coderdtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := coderdtest.CreateWorkspace(t, client, template.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		logs, closer, err := client.WorkspaceBuildLogsAfter(ctx, workspace.LatestBuild.ID, 0)
		require.NoError(t, err)
		defer closer.Close()
		for {
			_, ok := <-logs
			if !ok {
				return
			}
		}
	})
}
