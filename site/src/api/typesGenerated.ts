// Code generated by 'make coder/scripts/apitypings/main.go'. DO NOT EDIT.

// From codersdk/users.go
export interface APIKey {
  readonly id: string
  readonly user_id: string
  readonly last_used: string
  readonly expires_at: string
  readonly created_at: string
  readonly updated_at: string
  readonly login_type: LoginType
  readonly lifetime_seconds: number
}

// From codersdk/workspaceagents.go
export interface AWSInstanceIdentityToken {
  readonly signature: string
  readonly document: string
}

// From codersdk/gitsshkey.go
export interface AgentGitSSHKey {
  readonly public_key: string
  readonly private_key: string
}

// From codersdk/roles.go
export interface AssignableRoles extends Role {
  readonly assignable: boolean
}

// From codersdk/users.go
export interface AuthMethods {
  readonly password: boolean
  readonly github: boolean
  readonly oidc: boolean
}

// From codersdk/workspaceagents.go
export interface AzureInstanceIdentityToken {
  readonly signature: string
  readonly encoding: string
}

// From codersdk/buildinfo.go
export interface BuildInfoResponse {
  readonly external_url: string
  readonly version: string
}

// From codersdk/parameters.go
export interface ComputedParameter extends Parameter {
  readonly source_value: string
  readonly schema_id: string
  readonly default_source_value: boolean
}

// From codersdk/users.go
export interface CreateFirstUserRequest {
  readonly email: string
  readonly username: string
  readonly password: string
  readonly organization: string
}

// From codersdk/users.go
export interface CreateFirstUserResponse {
  readonly user_id: string
  readonly organization_id: string
}

// From codersdk/users.go
export interface CreateOrganizationRequest {
  readonly name: string
}

// From codersdk/parameters.go
export interface CreateParameterRequest {
  readonly copy_from_parameter?: string
  readonly name: string
  readonly source_value: string
  readonly source_scheme: ParameterSourceScheme
  readonly destination_scheme: ParameterDestinationScheme
}

// From codersdk/organizations.go
export interface CreateTemplateRequest {
  readonly name: string
  readonly description?: string
  readonly template_version_id: string
  readonly parameter_values?: CreateParameterRequest[]
  readonly max_ttl_ms?: number
  readonly min_autostart_interval_ms?: number
}

// From codersdk/templateversions.go
export interface CreateTemplateVersionDryRunRequest {
  readonly WorkspaceName: string
  readonly ParameterValues: CreateParameterRequest[]
}

// From codersdk/organizations.go
export interface CreateTemplateVersionRequest {
  readonly template_id?: string
  readonly storage_method: ProvisionerStorageMethod
  readonly storage_source: string
  readonly provisioner: ProvisionerType
  readonly parameter_values?: CreateParameterRequest[]
}

// From codersdk/users.go
export interface CreateUserRequest {
  readonly email: string
  readonly username: string
  readonly password: string
  readonly organization_id: string
}

// From codersdk/workspaces.go
export interface CreateWorkspaceBuildRequest {
  readonly template_version_id?: string
  readonly transition: WorkspaceTransition
  readonly dry_run?: boolean
  readonly state?: string
  readonly parameter_values?: CreateParameterRequest[]
}

// From codersdk/organizations.go
export interface CreateWorkspaceRequest {
  readonly template_id: string
  readonly name: string
  readonly autostart_schedule?: string
  readonly ttl_ms?: number
  readonly parameter_values?: CreateParameterRequest[]
}

// From codersdk/features.go
export interface Entitlements {
  readonly features: Record<string, Feature>
  readonly warnings: string[]
  readonly has_license: boolean
}

// From codersdk/features.go
export interface Feature {
  readonly entitlement: Entitlement
  readonly enabled: boolean
  readonly limit?: number
  readonly actual?: number
}

// From codersdk/users.go
export interface GenerateAPIKeyResponse {
  readonly key: string
}

// From codersdk/gitsshkey.go
export interface GitSSHKey {
  readonly user_id: string
  readonly created_at: string
  readonly updated_at: string
  readonly public_key: string
}

// From codersdk/workspaceagents.go
export interface GoogleInstanceIdentityToken {
  readonly json_web_token: string
}

// From codersdk/users.go
export interface LoginWithPasswordRequest {
  readonly email: string
  readonly password: string
}

// From codersdk/users.go
export interface LoginWithPasswordResponse {
  readonly session_token: string
}

// From codersdk/organizations.go
export interface Organization {
  readonly id: string
  readonly name: string
  readonly created_at: string
  readonly updated_at: string
}

// From codersdk/organizationmember.go
export interface OrganizationMember {
  readonly user_id: string
  readonly organization_id: string
  readonly created_at: string
  readonly updated_at: string
  readonly roles: string[]
}

// From codersdk/pagination.go
export interface Pagination {
  readonly after_id?: string
  readonly limit?: number
  readonly offset?: number
}

// From codersdk/parameters.go
export interface Parameter {
  readonly id: string
  readonly scope: ParameterScope
  readonly scope_id: string
  readonly name: string
  readonly source_scheme: ParameterSourceScheme
  readonly destination_scheme: ParameterDestinationScheme
  readonly created_at: string
  readonly updated_at: string
}

// From codersdk/parameters.go
export interface ParameterSchema {
  readonly id: string
  readonly created_at: string
  readonly job_id: string
  readonly name: string
  readonly description: string
  readonly default_source_scheme: ParameterSourceScheme
  readonly default_source_value: string
  readonly allow_override_source: boolean
  readonly default_destination_scheme: ParameterDestinationScheme
  readonly allow_override_destination: boolean
  readonly default_refresh: string
  readonly redisplay_value: boolean
  readonly validation_error: string
  readonly validation_condition: string
  readonly validation_type_system: string
  readonly validation_value_type: string
  readonly validation_contains?: string[]
}

// From codersdk/provisionerdaemons.go
export interface ProvisionerDaemon {
  readonly id: string
  readonly created_at: string
  readonly updated_at?: string
  readonly name: string
  readonly provisioners: ProvisionerType[]
}

// From codersdk/provisionerdaemons.go
export interface ProvisionerJob {
  readonly id: string
  readonly created_at: string
  readonly started_at?: string
  readonly completed_at?: string
  readonly error?: string
  readonly status: ProvisionerJobStatus
  readonly worker_id?: string
  readonly storage_source: string
}

// From codersdk/provisionerdaemons.go
export interface ProvisionerJobLog {
  readonly id: string
  readonly created_at: string
  readonly log_source: LogSource
  readonly log_level: LogLevel
  readonly stage: string
  readonly output: string
}

// From codersdk/workspaces.go
export interface PutExtendWorkspaceRequest {
  readonly deadline: string
}

// From codersdk/error.go
export interface Response {
  readonly message: string
  readonly detail?: string
  readonly validations?: ValidationError[]
}

// From codersdk/roles.go
export interface Role {
  readonly name: string
  readonly display_name: string
}

// From codersdk/templates.go
export interface Template {
  readonly id: string
  readonly created_at: string
  readonly updated_at: string
  readonly organization_id: string
  readonly name: string
  readonly provisioner: ProvisionerType
  readonly active_version_id: string
  readonly workspace_owner_count: number
  readonly description: string
  readonly max_ttl_ms: number
  readonly min_autostart_interval_ms: number
  readonly created_by_id: string
  readonly created_by_name: string
}

// From codersdk/templateversions.go
export interface TemplateVersion {
  readonly id: string
  readonly template_id?: string
  readonly organization_id?: string
  readonly created_at: string
  readonly updated_at: string
  readonly name: string
  readonly job: ProvisionerJob
  readonly readme: string
  readonly created_by_id: string
  readonly created_by_name: string
}

// From codersdk/templates.go
export interface TemplateVersionsByTemplateRequest extends Pagination {
  readonly template_id: string
}

// From codersdk/templates.go
export interface UpdateActiveTemplateVersion {
  readonly id: string
}

// From codersdk/users.go
export interface UpdateRoles {
  readonly roles: string[]
}

// From codersdk/templates.go
export interface UpdateTemplateMeta {
  readonly name?: string
  readonly description?: string
  readonly max_ttl_ms?: number
  readonly min_autostart_interval_ms?: number
}

// From codersdk/users.go
export interface UpdateUserPasswordRequest {
  readonly old_password: string
  readonly password: string
}

// From codersdk/users.go
export interface UpdateUserProfileRequest {
  readonly username: string
}

// From codersdk/workspaces.go
export interface UpdateWorkspaceAutostartRequest {
  readonly schedule?: string
}

// From codersdk/workspaces.go
export interface UpdateWorkspaceTTLRequest {
  readonly ttl_ms?: number
}

// From codersdk/files.go
export interface UploadResponse {
  readonly hash: string
}

// From codersdk/users.go
export interface User {
  readonly id: string
  readonly username: string
  readonly email: string
  readonly created_at: string
  readonly status: UserStatus
  readonly organization_ids: string[]
  readonly roles: Role[]
}

// From codersdk/users.go
export interface UserAuthorization {
  readonly object: UserAuthorizationObject
  readonly action: string
}

// From codersdk/users.go
export interface UserAuthorizationObject {
  readonly resource_type: string
  readonly owner_id?: string
  readonly organization_id?: string
  readonly resource_id?: string
}

// From codersdk/users.go
export interface UserAuthorizationRequest {
  readonly checks: Record<string, UserAuthorization>
}

// From codersdk/users.go
export type UserAuthorizationResponse = Record<string, boolean>

// From codersdk/users.go
export interface UserRoles {
  readonly roles: string[]
  readonly organization_roles: Record<string, string[]>
}

// From codersdk/users.go
export interface UsersRequest extends Pagination {
  readonly q?: string
}

// From codersdk/error.go
export interface ValidationError {
  readonly field: string
  readonly detail: string
}

// From codersdk/workspaces.go
export interface Workspace {
  readonly id: string
  readonly created_at: string
  readonly updated_at: string
  readonly owner_id: string
  readonly owner_name: string
  readonly template_id: string
  readonly template_name: string
  readonly latest_build: WorkspaceBuild
  readonly outdated: boolean
  readonly name: string
  readonly autostart_schedule?: string
  readonly ttl_ms?: number
}

// From codersdk/workspaceresources.go
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
  readonly instance_id?: string
  readonly architecture: string
  readonly environment_variables: Record<string, string>
  readonly operating_system: string
  readonly startup_script?: string
  readonly directory?: string
  readonly apps: WorkspaceApp[]
  // Named type "tailscale.com/types/key.NodePublic" unknown, using "any"
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  readonly wireguard_public_key: any
  // Named type "tailscale.com/types/key.DiscoPublic" unknown, using "any"
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  readonly disco_public_key: any
  // Named type "inet.af/netaddr.IPPrefix" unknown, using "any"
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  readonly ipv6: any
}

// From codersdk/workspaceagents.go
export interface WorkspaceAgentAuthenticateResponse {
  readonly session_token: string
}

// From codersdk/workspaceresources.go
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

// From codersdk/workspaceresources.go
export interface WorkspaceAgentResourceMetadata {
  readonly memory_total: number
  readonly disk_total: number
  readonly cpu_cores: number
  readonly cpu_model: string
  readonly cpu_mhz: number
}

// From codersdk/workspaceapps.go
export interface WorkspaceApp {
  readonly id: string
  readonly name: string
  readonly command?: string
  readonly icon?: string
}

// From codersdk/workspacebuilds.go
export interface WorkspaceBuild {
  readonly id: string
  readonly created_at: string
  readonly updated_at: string
  readonly workspace_id: string
  readonly workspace_name: string
  readonly workspace_owner_id: string
  readonly workspace_owner_name: string
  readonly template_version_id: string
  readonly build_number: number
  readonly name: string
  readonly transition: WorkspaceTransition
  readonly initiator_id: string
  readonly initiator_name: string
  readonly job: ProvisionerJob
  readonly deadline: string
  readonly reason: BuildReason
}

// From codersdk/workspaces.go
export interface WorkspaceBuildsRequest extends Pagination {
  readonly WorkspaceID: string
}

// From codersdk/workspaces.go
export interface WorkspaceFilter {
  readonly q?: string
}

// From codersdk/workspaces.go
export interface WorkspaceOptions {
  readonly include_deleted?: boolean
}

// From codersdk/workspaceresources.go
export interface WorkspaceResource {
  readonly id: string
  readonly created_at: string
  readonly job_id: string
  readonly workspace_transition: WorkspaceTransition
  readonly type: string
  readonly name: string
  readonly agents?: WorkspaceAgent[]
  readonly metadata?: WorkspaceResourceMetadata[]
}

// From codersdk/workspaceresources.go
export interface WorkspaceResourceMetadata {
  readonly key: string
  readonly value: string
  readonly sensitive: boolean
}

// From codersdk/workspacebuilds.go
export type BuildReason = "autostart" | "autostop" | "initiator"

// From codersdk/features.go
export type Entitlement = "entitled" | "grace_period" | "not_entitled"

// From codersdk/provisionerdaemons.go
export type LogLevel = "debug" | "error" | "info" | "trace" | "warn"

// From codersdk/provisionerdaemons.go
export type LogSource = "provisioner" | "provisioner_daemon"

// From codersdk/users.go
export type LoginType = "github" | "oidc" | "password"

// From codersdk/parameters.go
export type ParameterDestinationScheme = "environment_variable" | "none" | "provisioner_variable"

// From codersdk/parameters.go
export type ParameterScope = "import_job" | "template" | "workspace"

// From codersdk/parameters.go
export type ParameterSourceScheme = "data" | "none"

// From codersdk/parameters.go
export type ParameterTypeSystem = "hcl" | "none"

// From codersdk/provisionerdaemons.go
export type ProvisionerJobStatus =
  | "canceled"
  | "canceling"
  | "failed"
  | "pending"
  | "running"
  | "succeeded"

// From codersdk/organizations.go
export type ProvisionerStorageMethod = "file"

// From codersdk/organizations.go
export type ProvisionerType = "echo" | "terraform"

// From codersdk/users.go
export type UserStatus = "active" | "suspended"

// From codersdk/workspaceresources.go
export type WorkspaceAgentStatus = "connected" | "connecting" | "disconnected"

// From codersdk/workspacebuilds.go
export type WorkspaceTransition = "delete" | "start" | "stop"
