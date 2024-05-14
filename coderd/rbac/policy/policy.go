package policy

const WildcardSymbol = "*"

// Action represents the allowed actions to be done on an object.
type Action string

const (
	ActionCreate Action = "create"
	ActionRead   Action = "read"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"

	ActionUse                Action = "use"
	ActionSSH                Action = "ssh"
	ActionApplicationConnect Action = "application_connect"
	ActionViewInsights       Action = "view_insights"

	ActionWorkspaceBuild Action = "build"

	ActionAssign Action = "assign"

	ActionReadPersonal   Action = "read_personal"
	ActionUpdatePersonal Action = "update_personal"
)

type PermissionDefinition struct {
	// name is optional. Used to override "Type" for function naming.
	Name string
	// Actions are a map of actions to some description of what the action
	// should represent. The key in the actions map is the verb to use
	// in the rbac policy.
	Actions map[Action]ActionDefinition
}

type ActionDefinition struct {
	// Human friendly description to explain the action.
	Description string
}

func actDef(description string) ActionDefinition {
	return ActionDefinition{
		Description: description,
	}
}

var workspaceActions = map[Action]ActionDefinition{
	ActionCreate: actDef("create a new workspace"),
	ActionRead:   actDef("read workspace data to view on the UI"),
	// TODO: Make updates more granular
	ActionUpdate: actDef("edit workspace settings (scheduling, permissions, parameters)"),
	ActionDelete: actDef("delete workspace"),

	// Workspace provisioning
	ActionWorkspaceBuild: actDef("allows starting, stopping, and updating a workspace"),

	// Running a workspace
	ActionSSH:                actDef("ssh into a given workspace"),
	ActionApplicationConnect: actDef("connect to workspace apps via browser"),
}

// RBACPermissions is indexed by the type
var RBACPermissions = map[string]PermissionDefinition{
	// Wildcard is every object, and the action "*" provides all actions.
	// So can grant all actions on all types.
	WildcardSymbol: {
		Name:    "Wildcard",
		Actions: map[Action]ActionDefinition{},
	},
	"user": {
		Actions: map[Action]ActionDefinition{
			// Actions deal with site wide user objects.
			ActionRead:   actDef("read user data"),
			ActionCreate: actDef("create a new user"),
			ActionUpdate: actDef("update an existing user"),
			ActionDelete: actDef("delete an existing user"),

			ActionReadPersonal:   actDef("read personal user data like password"),
			ActionUpdatePersonal: actDef("update personal data"),
			// ActionReadPublic: actDef(fieldOwner, "read public user data"),
		},
	},
	"workspace": {
		Actions: workspaceActions,
	},
	// Dormant workspaces have the same perms as workspaces.
	"workspace_dormant": {
		Actions: workspaceActions,
	},
	"workspace_proxy": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef("create a workspace proxy"),
			ActionDelete: actDef("delete a workspace proxy"),
			ActionUpdate: actDef("update a workspace proxy"),
			ActionRead:   actDef("read and use a workspace proxy"),
		},
	},
	"license": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef("create a license"),
			ActionRead:   actDef("read licenses"),
			ActionDelete: actDef("delete license"),
			// Licenses are immutable, so update makes no sense
		},
	},
	"audit_log": {
		Actions: map[Action]ActionDefinition{
			ActionRead: actDef("read audit logs"),
		},
	},
	"deployment_config": {
		Actions: map[Action]ActionDefinition{
			ActionRead: actDef("read deployment config"),
		},
	},
	"deployment_stats": {
		Actions: map[Action]ActionDefinition{
			ActionRead: actDef("read deployment stats"),
		},
	},
	"replicas": {
		Actions: map[Action]ActionDefinition{
			ActionRead: actDef("read replicas"),
		},
	},
	"template": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef("create a template"),
			// TODO: Create a use permission maybe?
			ActionRead:         actDef("read template"),
			ActionUpdate:       actDef("update a template"),
			ActionDelete:       actDef("delete a template"),
			ActionViewInsights: actDef("view insights"),
		},
	},
	"group": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef("create a group"),
			ActionRead:   actDef("read groups"),
			ActionDelete: actDef("delete a group"),
			ActionUpdate: actDef("update a group"),
		},
	},
	"file": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef("create a file"),
			ActionRead:   actDef("read files"),
		},
	},
	"provisioner_daemon": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef("create a provisioner daemon"),
			// TODO: Move to use?
			ActionRead:   actDef("read provisioner daemon"),
			ActionUpdate: actDef("update a provisioner daemon"),
			ActionDelete: actDef("delete a provisioner daemon"),
		},
	},
	"organization": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef("create an organization"),
			ActionRead:   actDef("read organizations"),
			ActionUpdate: actDef("update an organization"),
			ActionDelete: actDef("delete an organization"),
		},
	},
	"organization_member": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef("create an organization member"),
			ActionRead:   actDef("read member"),
			ActionUpdate: actDef("update a organization member"),
			ActionDelete: actDef("delete member"),
		},
	},
	"debug_info": {
		Actions: map[Action]ActionDefinition{
			ActionUse: actDef("access to debug routes"),
		},
	},
	"system": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef("create system resources"),
			ActionRead:   actDef("view system resources"),
			ActionUpdate: actDef("update system resources"),
			ActionDelete: actDef("delete system resources"),
		},
	},
	"api_key": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef("create an api key"),
			ActionRead:   actDef("read api key details (secrets are not stored)"),
			ActionDelete: actDef("delete an api key"),
		},
	},
	"tailnet_coordinator": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef(""),
			ActionRead:   actDef(""),
			ActionUpdate: actDef(""),
			ActionDelete: actDef(""),
		},
	},
	"assign_role": {
		Actions: map[Action]ActionDefinition{
			ActionAssign: actDef("ability to assign roles"),
			ActionRead:   actDef("view what roles are assignable"),
			ActionDelete: actDef("ability to delete roles"),
		},
	},
	"assign_org_role": {
		Actions: map[Action]ActionDefinition{
			ActionAssign: actDef("ability to assign org scoped roles"),
			ActionRead:   actDef("view what roles are assignable"),
			ActionDelete: actDef("ability to delete org scoped roles"),
		},
	},
	"oauth2_app": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef("make an OAuth2 app."),
			ActionRead:   actDef("read OAuth2 apps"),
			ActionUpdate: actDef("update the properties of the OAuth2 app."),
			ActionDelete: actDef("delete an OAuth2 app"),
		},
	},
	"oauth2_app_secret": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef(""),
			ActionRead:   actDef(""),
			ActionUpdate: actDef(""),
			ActionDelete: actDef(""),
		},
	},
	"oauth2_app_code_token": {
		Actions: map[Action]ActionDefinition{
			ActionCreate: actDef(""),
			ActionRead:   actDef(""),
			ActionDelete: actDef(""),
		},
	},
}
