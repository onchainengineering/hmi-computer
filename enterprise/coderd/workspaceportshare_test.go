package coderd_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onchainengineering/hmi-computer/v2/coderd/coderdtest"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/coderd/rbac"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/codersdk"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/enterprise/coderd/coderdenttest"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/enterprise/coderd/license"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/testutil"
)

func TestWorkspacePortShare(t *testing.T) {
	t.Parallel()

	ownerClient, owner := coderdenttest.New(t, &coderdenttest.Options{
		Options: &coderdtest.Options{
			IncludeProvisionerDaemon: true,
		},
		LicenseOptions: &coderdenttest.LicenseOptions{
			Features: license.Features{
				codersdk.FeatureControlSharedPorts: 1,
			},
		},
	})
	client, user := coderdtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())
	r := setupWorkspaceAgent(t, client, codersdk.CreateFirstUserResponse{
		UserID:         user.ID,
		OrganizationID: owner.OrganizationID,
	}, 0)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
	defer cancel()

	// try to update port share with template max port share level owner
	_, err := client.UpsertWorkspaceAgentPortShare(ctx, r.workspace.ID, codersdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  r.sdkAgent.Name,
		Port:       8080,
		ShareLevel: codersdk.WorkspaceAgentPortShareLevelPublic,
		Protocol:   codersdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.Error(t, err, "Port sharing level not allowed")

	// update the template max port share level to public
	var level codersdk.WorkspaceAgentPortShareLevel = codersdk.WorkspaceAgentPortShareLevelPublic
	client.UpdateTemplateMeta(ctx, r.workspace.TemplateID, codersdk.UpdateTemplateMeta{
		MaxPortShareLevel: &level,
	})

	// OK
	ps, err := client.UpsertWorkspaceAgentPortShare(ctx, r.workspace.ID, codersdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  r.sdkAgent.Name,
		Port:       8080,
		ShareLevel: codersdk.WorkspaceAgentPortShareLevelPublic,
		Protocol:   codersdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.NoError(t, err)
	require.EqualValues(t, codersdk.WorkspaceAgentPortShareLevelPublic, ps.ShareLevel)
}
