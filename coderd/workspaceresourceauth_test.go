package coderd_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/provisioner/echo"
	"github.com/coder/coder/provisionersdk/proto"
)

func TestPostWorkspaceAuthGoogleInstanceIdentity(t *testing.T) {
	t.Parallel()
	t.Run("Expired", func(t *testing.T) {
		t.Parallel()
		instanceID := "instanceidentifier"
		validator, metadata := coderdtest.NewGoogleInstanceIdentity(t, instanceID, true)
		client := coderdtest.New(t, &coderdtest.Options{
			GoogleTokenValidator: validator,
		})
		_, err := client.AuthWorkspaceGoogleInstanceIdentity(context.Background(), "", metadata)
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode())
	})

	t.Run("InstanceNotFound", func(t *testing.T) {
		t.Parallel()
		instanceID := "instanceidentifier"
		validator, metadata := coderdtest.NewGoogleInstanceIdentity(t, instanceID, false)
		client := coderdtest.New(t, &coderdtest.Options{
			GoogleTokenValidator: validator,
		})
		_, err := client.AuthWorkspaceGoogleInstanceIdentity(context.Background(), "", metadata)
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode())
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		instanceID := "instanceidentifier"
		validator, metadata := coderdtest.NewGoogleInstanceIdentity(t, instanceID, false)
		client := coderdtest.New(t, &coderdtest.Options{
			GoogleTokenValidator: validator,
		})
		user := coderdtest.CreateFirstUser(t, client)
		coderdtest.NewProvisionerDaemon(t, client)
		version := coderdtest.CreateProjectVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionDryRun: []*proto.Provision_Response{{
				Type: &proto.Provision_Response_Complete{
					Complete: &proto.Provision_Complete{
						Resources: []*proto.Resource{{
							Name: "somename",
							Type: "someinstance",
							Agent: &proto.Agent{
								Auth: &proto.Agent_InstanceId{
									InstanceId: "",
								},
							},
						}},
					},
				},
			}},
			Provision: []*proto.Provision_Response{{
				Type: &proto.Provision_Response_Complete{
					Complete: &proto.Provision_Complete{
						Resources: []*proto.Resource{{
							Name: "somename",
							Type: "someinstance",
							Agent: &proto.Agent{
								Auth: &proto.Agent_InstanceId{
									InstanceId: instanceID,
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
		coderdtest.AwaitWorkspaceBuildJob(t, client, workspace.LatestBuild.ID)

		_, err := client.AuthWorkspaceGoogleInstanceIdentity(context.Background(), "", metadata)
		require.NoError(t, err)
	})
}
