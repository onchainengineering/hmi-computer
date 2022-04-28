// From codersdk/templateversions.go:31:6
export interface TemplateVersionParameter {
	// Named type "github.com/coder/coder/coderd/database.ParameterValue" unknown, using "any"
	readonly ParameterValue: any
	readonly schema_id: string
	readonly default_source_value: boolean
}

// From codersdk/users.go:59:6
export interface UpdateUserProfileRequest {
	readonly email: string
	readonly username: string
}

// From codersdk/parameters.go:38:6
export interface CreateParameterRequest {
	readonly name: string
	readonly source_value: string
	readonly source_scheme: string
	readonly destination_scheme: string
}

// From codersdk/users.go:52:6
export interface CreateUserRequest {
	readonly email: string
	readonly username: string
	readonly password: string
	readonly organization_id: string
}

// From codersdk/users.go:76:6
export interface GenerateAPIKeyResponse {
	readonly key: string
}

// From codersdk/users.go:65:6
export interface LoginWithPasswordRequest {
	readonly email: string
	readonly password: string
}

// From codersdk/users.go:71:6
export interface LoginWithPasswordResponse {
	readonly session_token: string
}

// From codersdk/templates.go:17:6
export interface Template {
	readonly id: string
	readonly created_at: string
	readonly updated_at: string
	readonly organization_id: string
	readonly name: string
	readonly provisioner: string
	readonly active_version_id: string
	readonly workspace_owner_count: number
}

// From codersdk/workspaces.go:94:6
export interface UpdateWorkspaceAutostartRequest {
	readonly schedule: string
}

// From codersdk/files.go:16:6
export interface UploadResponse {
	readonly hash: string
}

// From codersdk/users.go:32:6
export interface User {
	readonly id: string
	readonly email: string
	readonly created_at: string
	readonly username: string
}

// From codersdk/workspaceresources.go:58:6
export interface WorkspaceAgentInstanceMetadata {
	readonly jail_orchestrator: string
	readonly operating_system: string
	readonly platform: string
	readonly platform_family: string
	readonly kernel_version: string
	readonly kernel_architecture: string
	readonly cloud: string
	readonly jail: string
	readonly vnc: boolean
}

// From codersdk/gitsshkey.go:14:6
export interface GitSSHKey {
	readonly user_id: string
	readonly created_at: string
	readonly updated_at: string
	readonly public_key: string
}

// From codersdk/provisionerdaemons.go:56:6
export interface ProvisionerJobLog {
	readonly id: string
	readonly created_at: string
	readonly log_source: string
	readonly log_level: string
	readonly stage: string
	readonly output: string
}

// From codersdk/users.go:17:6
export interface UsersRequest {
	readonly after_user: string
	readonly search: string
	readonly limit: number
	readonly offset: number
}

// From codersdk/workspaceagents.go:35:6
export interface AWSInstanceIdentityToken {
	readonly signature: string
	readonly document: string
}

// From codersdk/buildinfo.go:10:6
export interface BuildInfoResponse {
	readonly external_url: string
	readonly version: string
}

// From codersdk/users.go:39:6
export interface CreateFirstUserRequest {
	readonly email: string
	readonly username: string
	readonly password: string
	readonly organization: string
}

// From codersdk/users.go:85:6
export interface CreateWorkspaceRequest {
	readonly template_id: string
	readonly name: string
	readonly parameter_values: CreateParameterRequest[]
}

// From codersdk/workspaceagents.go:31:6
export interface GoogleInstanceIdentityToken {
	readonly json_web_token: string
}

// From codersdk/organizations.go:38:6
export interface CreateTemplateRequest {
	readonly name: string
	readonly template_version_id: string
	readonly parameter_values: CreateParameterRequest[]
}

// From codersdk/templateversions.go:17:6
export interface TemplateVersion {
	readonly id: string
	readonly template_id?: string
	readonly created_at: string
	readonly updated_at: string
	readonly name: string
	readonly job: ProvisionerJob
}

// From codersdk/workspaces.go:114:6
export interface UpdateWorkspaceAutostopRequest {
	readonly schedule: string
}

// From codersdk/workspaces.go:18:6
export interface Workspace {
	readonly id: string
	readonly created_at: string
	readonly updated_at: string
	readonly owner_id: string
	readonly template_id: string
	readonly template_name: string
	readonly latest_build: WorkspaceBuild
	readonly outdated: boolean
	readonly name: string
	readonly autostart_schedule: string
	readonly autostop_schedule: string
}

// From codersdk/workspacebuilds.go:17:6
export interface WorkspaceBuild {
	readonly id: string
	readonly created_at: string
	readonly updated_at: string
	readonly workspace_id: string
	readonly template_version_id: string
	readonly before_id: string
	readonly after_id: string
	readonly name: string
	readonly transition: string
	readonly initiator_id: string
	readonly job: ProvisionerJob
}

// From codersdk/users.go:94:6
export interface AuthMethods {
	readonly password: boolean
	readonly github: boolean
}

// From codersdk/organizations.go:25:6
export interface CreateTemplateVersionRequest {
	readonly template_id: string
	readonly storage_method: string
	readonly storage_source: string
	readonly provisioner: string
	readonly parameter_values: CreateParameterRequest[]
}

// From codersdk/organizations.go:17:6
export interface Organization {
	readonly id: string
	readonly name: string
	readonly created_at: string
	readonly updated_at: string
}

// From codersdk/parameters.go:26:6
export interface Parameter {
	readonly id: string
	readonly created_at: string
	readonly updated_at: string
	readonly scope: ParameterScope
	readonly scope_id: string
	readonly name: string
	readonly source_scheme: string
	readonly destination_scheme: string
}

// From codersdk/provisionerdaemons.go:46:6
export interface ProvisionerJob {
	readonly id: string
	readonly created_at: string
	readonly started_at?: string
	readonly completed_at?: string
	readonly error: string
	readonly status: ProvisionerJobStatus
	readonly worker_id?: string
}

// From codersdk/templates.go:28:6
export interface UpdateActiveTemplateVersion {
	readonly id: string
}

// From codersdk/gitsshkey.go:21:6
export interface AgentGitSSHKey {
	readonly public_key: string
	readonly private_key: string
}

// From codersdk/workspaceagents.go:40:6
export interface AzureInstanceIdentityToken {
	readonly signature: string
	readonly encoding: string
}

// From codersdk/workspaceresources.go:50:6
export interface WorkspaceAgentResourceMetadata {
	readonly memory_total: number
	readonly disk_total: number
	readonly cpu_cores: number
	readonly cpu_model: string
	readonly cpu_mhz: number
}

// From codersdk/users.go:47:6
export interface CreateFirstUserResponse {
	readonly user_id: string
	readonly organization_id: string
}

// From codersdk/workspaces.go:33:6
export interface CreateWorkspaceBuildRequest {
	readonly template_version_id: string
	readonly transition: string
	readonly dry_run: boolean
}

// From codersdk/provisionerdaemons.go:23:6
export interface ProvisionerDaemon {
	readonly id: string
	readonly created_at: string
	readonly updated_at?: string
	readonly organization_id?: string
	readonly name: string
	readonly provisioners: string[]
}

// From codersdk/templateversions.go:28:6
export interface TemplateVersionParameterSchema {
	readonly id: string
	readonly created_at: string
	readonly job_id: string
	readonly name: string
	readonly description: string
	readonly default_source_scheme: string
	readonly default_source_value: string
	readonly allow_override_source: boolean
	readonly default_destination_scheme: string
	readonly allow_override_destination: boolean
	readonly default_refresh: string
	readonly redisplay_value: boolean
	readonly validation_error: string
	readonly validation_condition: string
	readonly validation_type_system: string
	readonly validation_value_type: string
}

// From codersdk/workspaceresources.go:23:6
export interface WorkspaceResource {
	readonly id: string
	readonly created_at: string
	readonly job_id: string
	readonly workspace_transition: string
	readonly type: string
	readonly name: string
	readonly agents: WorkspaceAgent[]
}

// From codersdk/users.go:80:6
export interface CreateOrganizationRequest {
	readonly name: string
}

// From codersdk/workspaceresources.go:33:6
export interface WorkspaceAgent {
	readonly id: string
	readonly created_at: string
	readonly updated_at: string
	readonly first_connected_at?: string
	readonly last_connected_at?: string
	readonly disconnected_at?: string
	readonly status: WorkspaceAgentStatus
	readonly name: string
	readonly resource_id: string
	readonly instance_id: string
	readonly architecture: string
	readonly environment_variables: Record<string, string>
	readonly operating_system: string
	readonly startup_script: string
}


// From codersdk/workspaceagents.go:47:6
export interface WorkspaceAgentAuthenticateResponse {
	readonly session_token: string
}

// From codersdk/workspaceresources.go:15:6
export type WorkspaceAgentStatus = "connected" | "connecting" | "disconnected"

// From codersdk/parameters.go:16:6
export type ParameterScope = "organization" | "template" | "user" | "workspace"

// From codersdk/provisionerdaemons.go:26:6
export type ProvisionerJobStatus = "canceled" | "canceling" | "failed" | "pending" | "running" | "succeeded"


