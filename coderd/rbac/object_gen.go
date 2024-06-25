// Code generated by rbacgen/main.go. DO NOT EDIT.
package rbac

import "github.com/coder/coder/v2/coderd/rbac/policy"

// Objecter returns the RBAC object for itself.
type Objecter interface {
	RBACObject() Object
}

var (
	// ResourceWildcard
	// Valid Actions
	ResourceWildcard = Object{
		Type: "*",
	}

	// ResourceApiKey
	// Valid Actions
	//  - "ActionCreate" :: create an api key
	//  - "ActionDelete" :: delete an api key
	//  - "ActionRead" :: read api key details (secrets are not stored)
	//  - "ActionUpdate" :: update an api key, eg expires
	ResourceApiKey = Object{
		Type: "api_key",
	}

	// ResourceAssignOrgRole
	// Valid Actions
	//  - "ActionAssign" :: ability to assign org scoped roles
	//  - "ActionCreate" :: ability to create/delete/edit custom roles within an organization
	//  - "ActionDelete" :: ability to delete org scoped roles
	//  - "ActionRead" :: view what roles are assignable
	ResourceAssignOrgRole = Object{
		Type: "assign_org_role",
	}

	// ResourceAssignRole
	// Valid Actions
	//  - "ActionAssign" :: ability to assign roles
	//  - "ActionCreate" :: ability to create/delete/edit custom roles
	//  - "ActionDelete" :: ability to unassign roles
	//  - "ActionRead" :: view what roles are assignable
	ResourceAssignRole = Object{
		Type: "assign_role",
	}

	// ResourceAuditLog
	// Valid Actions
	//  - "ActionCreate" :: create new audit log entries
	//  - "ActionRead" :: read audit logs
	ResourceAuditLog = Object{
		Type: "audit_log",
	}

	// ResourceDebugInfo
	// Valid Actions
	//  - "ActionRead" :: access to debug routes
	ResourceDebugInfo = Object{
		Type: "debug_info",
	}

	// ResourceDeploymentConfig
	// Valid Actions
	//  - "ActionRead" :: read deployment config
	//  - "ActionUpdate" :: updating health information
	ResourceDeploymentConfig = Object{
		Type: "deployment_config",
	}

	// ResourceDeploymentStats
	// Valid Actions
	//  - "ActionRead" :: read deployment stats
	ResourceDeploymentStats = Object{
		Type: "deployment_stats",
	}

	// ResourceFile
	// Valid Actions
	//  - "ActionCreate" :: create a file
	//  - "ActionRead" :: read files
	ResourceFile = Object{
		Type: "file",
	}

	// ResourceGroup
	// Valid Actions
	//  - "ActionCreate" :: create a group
	//  - "ActionDelete" :: delete a group
	//  - "ActionRead" :: read groups
	//  - "ActionUpdate" :: update a group
	ResourceGroup = Object{
		Type: "group",
	}

	// ResourceLicense
	// Valid Actions
	//  - "ActionCreate" :: create a license
	//  - "ActionDelete" :: delete license
	//  - "ActionRead" :: read licenses
	ResourceLicense = Object{
		Type: "license",
	}

	// ResourceOauth2App
	// Valid Actions
	//  - "ActionCreate" :: make an OAuth2 app.
	//  - "ActionDelete" :: delete an OAuth2 app
	//  - "ActionRead" :: read OAuth2 apps
	//  - "ActionUpdate" :: update the properties of the OAuth2 app.
	ResourceOauth2App = Object{
		Type: "oauth2_app",
	}

	// ResourceOauth2AppCodeToken
	// Valid Actions
	//  - "ActionCreate" ::
	//  - "ActionDelete" ::
	//  - "ActionRead" ::
	ResourceOauth2AppCodeToken = Object{
		Type: "oauth2_app_code_token",
	}

	// ResourceOauth2AppSecret
	// Valid Actions
	//  - "ActionCreate" ::
	//  - "ActionDelete" ::
	//  - "ActionRead" ::
	//  - "ActionUpdate" ::
	ResourceOauth2AppSecret = Object{
		Type: "oauth2_app_secret",
	}

	// ResourceOrganization
	// Valid Actions
	//  - "ActionCreate" :: create an organization
	//  - "ActionDelete" :: delete an organization
	//  - "ActionRead" :: read organizations
	//  - "ActionUpdate" :: update an organization
	ResourceOrganization = Object{
		Type: "organization",
	}

	// ResourceOrganizationMember
	// Valid Actions
	//  - "ActionCreate" :: create an organization member
	//  - "ActionDelete" :: delete member
	//  - "ActionRead" :: read member
	//  - "ActionUpdate" :: update an organization member
	ResourceOrganizationMember = Object{
		Type: "organization_member",
	}

	// ResourceProvisionerDaemon
	// Valid Actions
	//  - "ActionCreate" :: create a provisioner daemon
	//  - "ActionDelete" :: delete a provisioner daemon
	//  - "ActionRead" :: read provisioner daemon
	//  - "ActionUpdate" :: update a provisioner daemon
	ResourceProvisionerDaemon = Object{
		Type: "provisioner_daemon",
	}

	// ResourceReplicas
	// Valid Actions
	//  - "ActionRead" :: read replicas
	ResourceReplicas = Object{
		Type: "replicas",
	}

	// ResourceSystem
	// Valid Actions
	//  - "ActionCreate" :: create system resources
	//  - "ActionDelete" :: delete system resources
	//  - "ActionRead" :: view system resources
	//  - "ActionUpdate" :: update system resources
	ResourceSystem = Object{
		Type: "system",
	}

	// ResourceTailnetCoordinator
	// Valid Actions
	//  - "ActionCreate" ::
	//  - "ActionDelete" ::
	//  - "ActionRead" ::
	//  - "ActionUpdate" ::
	ResourceTailnetCoordinator = Object{
		Type: "tailnet_coordinator",
	}

	// ResourceTemplate
	// Valid Actions
	//  - "ActionCreate" :: create a template
	//  - "ActionDelete" :: delete a template
	//  - "ActionRead" :: read template
	//  - "ActionUpdate" :: update a template
	//  - "ActionViewInsights" :: view insights
	ResourceTemplate = Object{
		Type: "template",
	}

	// ResourceUser
	// Valid Actions
	//  - "ActionCreate" :: create a new user
	//  - "ActionDelete" :: delete an existing user
	//  - "ActionRead" :: read user data
	//  - "ActionReadPersonal" :: read personal user data like user settings and auth links
	//  - "ActionUpdate" :: update an existing user
	//  - "ActionUpdatePersonal" :: update personal data
	ResourceUser = Object{
		Type: "user",
	}

	// ResourceWorkspace
	// Valid Actions
	//  - "ActionApplicationConnect" :: connect to workspace apps via browser
	//  - "ActionCreate" :: create a new workspace
	//  - "ActionDelete" :: delete workspace
	//  - "ActionRead" :: read workspace data to view on the UI
	//  - "ActionSSH" :: ssh into a given workspace
	//  - "ActionWorkspaceStart" :: allows starting a workspace
	//  - "ActionWorkspaceStop" :: allows stopping a workspace
	//  - "ActionUpdate" :: edit workspace settings (scheduling, permissions, parameters)
	ResourceWorkspace = Object{
		Type: "workspace",
	}

	// ResourceWorkspaceDormant
	// Valid Actions
	//  - "ActionApplicationConnect" :: connect to workspace apps via browser
	//  - "ActionCreate" :: create a new workspace
	//  - "ActionDelete" :: delete workspace
	//  - "ActionRead" :: read workspace data to view on the UI
	//  - "ActionSSH" :: ssh into a given workspace
	//  - "ActionWorkspaceStart" :: allows starting a workspace
	//  - "ActionWorkspaceStop" :: allows stopping a workspace
	//  - "ActionUpdate" :: edit workspace settings (scheduling, permissions, parameters)
	ResourceWorkspaceDormant = Object{
		Type: "workspace_dormant",
	}

	// ResourceWorkspaceProxy
	// Valid Actions
	//  - "ActionCreate" :: create a workspace proxy
	//  - "ActionDelete" :: delete a workspace proxy
	//  - "ActionRead" :: read and use a workspace proxy
	//  - "ActionUpdate" :: update a workspace proxy
	ResourceWorkspaceProxy = Object{
		Type: "workspace_proxy",
	}
)

func AllResources() []Objecter {
	return []Objecter{
		ResourceWildcard,
		ResourceApiKey,
		ResourceAssignOrgRole,
		ResourceAssignRole,
		ResourceAuditLog,
		ResourceDebugInfo,
		ResourceDeploymentConfig,
		ResourceDeploymentStats,
		ResourceFile,
		ResourceGroup,
		ResourceLicense,
		ResourceOauth2App,
		ResourceOauth2AppCodeToken,
		ResourceOauth2AppSecret,
		ResourceOrganization,
		ResourceOrganizationMember,
		ResourceProvisionerDaemon,
		ResourceReplicas,
		ResourceSystem,
		ResourceTailnetCoordinator,
		ResourceTemplate,
		ResourceUser,
		ResourceWorkspace,
		ResourceWorkspaceDormant,
		ResourceWorkspaceProxy,
	}
}

func AllActions() []policy.Action {
	return []policy.Action{
		policy.ActionApplicationConnect,
		policy.ActionAssign,
		policy.ActionCreate,
		policy.ActionDelete,
		policy.ActionRead,
		policy.ActionReadPersonal,
		policy.ActionSSH,
		policy.ActionUpdate,
		policy.ActionUpdatePersonal,
		policy.ActionUse,
		policy.ActionViewInsights,
		policy.ActionWorkspaceStart,
		policy.ActionWorkspaceStop,
	}
}
