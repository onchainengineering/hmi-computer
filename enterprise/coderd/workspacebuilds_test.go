package coderd_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/coderd/coderdtest"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/enterprise/coderd/coderdenttest"
	"github.com/coder/coder/v2/enterprise/coderd/license"
	"github.com/coder/coder/v2/enterprise/coderd/schedule"
	"github.com/coder/coder/v2/testutil"
)

func TestWorkspaceBuild(t *testing.T) {
	t.Parallel()
	t.Run("TemplateRequiresActiveVersion", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t, testutil.WaitMedium)
		ownerClient, owner := coderdenttest.New(t, &coderdenttest.Options{
			Options: &coderdtest.Options{
				IncludeProvisionerDaemon: true,
				TemplateScheduleStore:    schedule.NewEnterpriseTemplateScheduleStore(agplUserQuietHoursScheduleStore()),
			},
			LicenseOptions: &coderdenttest.LicenseOptions{
				Features: license.Features{
					codersdk.FeatureAdvancedTemplateScheduling: 1,
					codersdk.FeatureTemplateRBAC:               1,
				},
			},
		})

		// Create an initial version.
		oldVersion := coderdtest.CreateTemplateVersion(t, ownerClient, owner.OrganizationID, nil)
		// Create a template that mandates the promoted version.
		// This should be enforced for everyone except template admins.
		template := coderdtest.CreateTemplate(t, ownerClient, owner.OrganizationID, oldVersion.ID)
		coderdtest.AwaitTemplateVersionJobCompleted(t, ownerClient, oldVersion.ID)
		require.Equal(t, oldVersion.ID, template.ActiveVersionID)
		template, err := ownerClient.UpdateTemplateMeta(ctx, template.ID, codersdk.UpdateTemplateMeta{
			RequireActiveVersion: true,
		})
		require.NoError(t, err)

		// Create a new version that we will promote.
		activeVersion := coderdtest.CreateTemplateVersion(t, ownerClient, owner.OrganizationID, nil, func(ctvr *codersdk.CreateTemplateVersionRequest) {
			ctvr.TemplateID = template.ID
		})
		coderdtest.AwaitTemplateVersionJobCompleted(t, ownerClient, activeVersion.ID)
		err = ownerClient.UpdateActiveTemplateVersion(ctx, template.ID, codersdk.UpdateActiveTemplateVersion{
			ID: activeVersion.ID,
		})
		require.NoError(t, err)

		templateAdminClient, _ := coderdtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())
		templateACLAdminClient, templateACLAdmin := coderdtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)
		templateGroupACLAdminClient, templateGroupACLAdmin := coderdtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)
		memberClient, _ := coderdtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)

		// Create a group so we can also test group template admin ownership.
		group, err := ownerClient.CreateGroup(ctx, owner.OrganizationID, codersdk.CreateGroupRequest{
			Name: "test",
		})
		require.NoError(t, err)

		// Add the user who gains template admin via group membership.
		group, err = ownerClient.PatchGroup(ctx, group.ID, codersdk.PatchGroupRequest{
			AddUsers: []string{templateGroupACLAdmin.ID.String()},
		})
		require.NoError(t, err)

		// Update the template for both users and groups.
		err = ownerClient.UpdateTemplateACL(ctx, template.ID, codersdk.UpdateTemplateACL{
			UserPerms: map[string]codersdk.TemplateRole{
				templateACLAdmin.ID.String(): codersdk.TemplateRoleAdmin,
			},
			GroupPerms: map[string]codersdk.TemplateRole{
				group.ID.String(): codersdk.TemplateRoleAdmin,
			},
		})
		require.NoError(t, err)

		type testcase struct {
			Name               string
			Client             *codersdk.Client
			ExpectedStatusCode int
		}

		cases := []testcase{
			{
				Name:               "OwnerOK",
				Client:             ownerClient,
				ExpectedStatusCode: http.StatusOK,
			},
			{
				Name:               "TemplateAdminOK",
				Client:             templateAdminClient,
				ExpectedStatusCode: http.StatusOK,
			},
			{
				Name:               "TemplateACLAdminOK",
				Client:             templateACLAdminClient,
				ExpectedStatusCode: http.StatusOK,
			},
			{
				Name:               "TemplateGroupACLAdminOK",
				Client:             templateGroupACLAdminClient,
				ExpectedStatusCode: http.StatusOK,
			},
			{
				Name:               "MemberFails",
				Client:             memberClient,
				ExpectedStatusCode: http.StatusUnauthorized,
			},
		}

		for _, c := range cases {
			t.Run(c.Name, func(t *testing.T) {
				_, err = c.Client.CreateWorkspace(context.Background(), owner.OrganizationID, codersdk.Me, codersdk.CreateWorkspaceRequest{
					TemplateVersionID: oldVersion.ID,
					Name:              "abc123",
					AutomaticUpdates:  codersdk.AutomaticUpdatesNever,
				})
				if c.ExpectedStatusCode == http.StatusOK {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
					cerr, ok := codersdk.AsError(err)
					require.True(t, ok)
					require.Equal(t, c.ExpectedStatusCode, cerr.StatusCode())
				}
			})
		}
	})
}
