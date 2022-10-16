// Code generated by 'make coder/scripts/apitypings/main.go'. DO NOT EDIT.

// From codersdk/apikey.go
export interface APIKey {
  readonly id: string
  readonly user_id: string
  readonly last_used: string
  readonly expires_at: string
  readonly created_at: string
  readonly updated_at: string
  readonly login_type: LoginType
  readonly scope: APIKeyScope
  readonly lifetime_seconds: number
}

// From codersdk/licenses.go
export interface AddLicenseRequest {
  readonly license: string
}

// From codersdk/gitsshkey.go
export interface AgentGitSSHKey {
  readonly public_key: string
  readonly private_key: string
}

// From codersdk/templates.go
export interface AgentStatsReportResponse {
  readonly num_comms: number
  readonly rx_bytes: number
  readonly tx_bytes: number
}

// From codersdk/roles.go
export interface AssignableRoles extends Role {
  readonly assignable: boolean
}

// From codersdk/audit.go
export type AuditDiff = Record<string, AuditDiffField>

// From codersdk/audit.go
export interface AuditDiffField {
  // eslint-disable-next-line
  readonly old?: any
  // eslint-disable-next-line
  readonly new?: any
  readonly secret: boolean
}

// From codersdk/audit.go
export interface AuditLog {
  readonly id: string
  readonly request_id: string
  readonly time: string
  readonly organization_id: string
  // Named type "net/netip.Addr" unknown, using "any"
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  readonly ip: any
  readonly user_agent: string
  readonly resource_type: ResourceType
  readonly resource_id: string
  readonly resource_target: string
  readonly resource_icon: string
  readonly action: AuditAction
  readonly diff: AuditDiff
  readonly status_code: number
  // This is likely an enum in an external package ("encoding/json.RawMessage")
  readonly additional_fields: string
  readonly description: string
  readonly user?: User
}

// From codersdk/audit.go
export interface AuditLogCountRequest {
  readonly q?: string
}

// From codersdk/audit.go
export interface AuditLogCountResponse {
  readonly count: number
}

// From codersdk/audit.go
export interface AuditLogResponse {
  readonly audit_logs: AuditLog[]
}

// From codersdk/audit.go
export interface AuditLogsRequest extends Pagination {
  readonly q?: string
}

// From codersdk/users.go
export interface AuthMethods {
  readonly password: boolean
  readonly github: boolean
  readonly oidc: boolean
}

// From codersdk/authorization.go
export interface AuthorizationCheck {
  readonly object: AuthorizationObject
  readonly action: string
}

// From codersdk/authorization.go
export interface AuthorizationObject {
  readonly resource_type: string
  readonly owner_id?: string
  readonly organization_id?: string
  readonly resource_id?: string
}

// From codersdk/authorization.go
export interface AuthorizationRequest {
  readonly checks: Record<string, AuthorizationCheck>
}

// From codersdk/authorization.go
export type AuthorizationResponse = Record<string, boolean>

// From codersdk/workspaceagents.go
export interface AzureInstanceIdentityToken {
  readonly signature: string
  readonly encoding: string
}

// From codersdk/flags.go
export interface BoolFlag {
  readonly name: string
  readonly flag: string
  readonly env_var: string
  readonly shorthand: string
  readonly description: string
  readonly enterprise: boolean
  readonly hidden: boolean
  readonly default: boolean
  readonly value: boolean
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

// From codersdk/groups.go
export interface CreateGroupRequest {
  readonly name: string
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
  readonly icon?: string
  readonly template_version_id: string
  readonly parameter_values?: CreateParameterRequest[]
  readonly max_ttl_ms?: number
  readonly min_autostart_interval_ms?: number
}

// From codersdk/templateversions.go
export interface CreateTemplateVersionDryRunRequest {
  readonly workspace_name: string
  readonly parameter_values: CreateParameterRequest[]
}

// From codersdk/organizations.go
export interface CreateTemplateVersionRequest {
  readonly name?: string
  readonly template_id?: string
  readonly storage_method: ProvisionerStorageMethod
  readonly file_id: string
  readonly provisioner: ProvisionerType
  readonly parameter_values?: CreateParameterRequest[]
}

// From codersdk/audit.go
export interface CreateTestAuditLogRequest {
  readonly action?: AuditAction
  readonly resource_type?: ResourceType
  readonly resource_id?: string
}

// From codersdk/apikey.go
export interface CreateTokenRequest {
  readonly scope: APIKeyScope
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
  readonly orphan?: boolean
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

// From codersdk/templates.go
export interface DAUEntry {
  readonly date: string
  readonly amount: number
}

// From codersdk/workspaceagents.go
export interface DERPRegion {
  readonly preferred: boolean
  readonly latency_ms: number
}

// From codersdk/flags.go
export interface DeploymentFlags {
  readonly access_url: StringFlag
  readonly wildcard_access_url: StringFlag
  readonly address: StringFlag
  readonly autobuild_poll_interval: DurationFlag
  readonly derp_server_enabled: BoolFlag
  readonly derp_server_region_id: IntFlag
  readonly derp_server_region_code: StringFlag
  readonly derp_server_region_name: StringFlag
  readonly derp_server_stun_address: StringArrayFlag
  readonly derp_config_url: StringFlag
  readonly derp_config_path: StringFlag
  readonly prom_enabled: BoolFlag
  readonly prom_address: StringFlag
  readonly pprof_enabled: BoolFlag
  readonly pprof_address: StringFlag
  readonly cache_dir: StringFlag
  readonly in_memory_database: BoolFlag
  readonly provisioner_daemon_count: IntFlag
  readonly postgres_url: StringFlag
  readonly oauth2_github_client_id: StringFlag
  readonly oauth2_github_client_secret: StringFlag
  readonly oauth2_github_allowed_organizations: StringArrayFlag
  readonly oauth2_github_allowed_teams: StringArrayFlag
  readonly oauth2_github_allow_signups: BoolFlag
  readonly oauth2_github_enterprise_base_url: StringFlag
  readonly oidc_allow_signups: BoolFlag
  readonly oidc_client_id: StringFlag
  readonly oidc_cliet_secret: StringFlag
  readonly oidc_email_domain: StringFlag
  readonly oidc_issuer_url: StringFlag
  readonly oidc_scopes: StringArrayFlag
  readonly telemetry_enable: BoolFlag
  readonly telemetry_trace_enable: BoolFlag
  readonly telemetry_url: StringFlag
  readonly tls_enable: BoolFlag
  readonly tls_cert_files: StringArrayFlag
  readonly tls_client_ca_file: StringFlag
  readonly tls_client_auth: StringFlag
  readonly tls_key_tiles: StringArrayFlag
  readonly tls_min_version: StringFlag
  readonly trace_enable: BoolFlag
  readonly secure_auth_cookie: BoolFlag
  readonly ssh_keygen_algorithm: StringFlag
  readonly auto_import_templates: StringArrayFlag
  readonly metrics_cache_refresh_interval: DurationFlag
  readonly agent_stat_refresh_interval: DurationFlag
  readonly verbose: BoolFlag
  readonly audit_logging: BoolFlag
  readonly browser_only: BoolFlag
  readonly scim_auth_header: StringFlag
  readonly user_workspace_quota: IntFlag
}

// From codersdk/flags.go
export interface DurationFlag {
  readonly name: string
  readonly flag: string
  readonly env_var: string
  readonly shorthand: string
  readonly description: string
  readonly enterprise: boolean
  readonly hidden: boolean
  // This is likely an enum in an external package ("time.Duration")
  readonly default: number
  // This is likely an enum in an external package ("time.Duration")
  readonly value: number
}

// From codersdk/features.go
export interface Entitlements {
  readonly features: Record<string, Feature>
  readonly warnings: string[]
  readonly has_license: boolean
  readonly experimental: boolean
  readonly trial: boolean
}

// From codersdk/features.go
export interface Feature {
  readonly entitlement: Entitlement
  readonly enabled: boolean
  readonly limit?: number
  readonly actual?: number
}

// From codersdk/apikey.go
export interface GenerateAPIKeyResponse {
  readonly key: string
}

// From codersdk/workspaces.go
export interface GetAppHostResponse {
  readonly host: string
}

// From codersdk/gitsshkey.go
export interface GitSSHKey {
  readonly user_id: string
  readonly created_at: string
  readonly updated_at: string
  readonly public_key: string
}

// From codersdk/groups.go
export interface Group {
  readonly id: string
  readonly name: string
  readonly organization_id: string
  readonly members: User[]
}

// From codersdk/workspaceapps.go
export interface Healthcheck {
  readonly url: string
  readonly interval: number
  readonly threshold: number
}

// From codersdk/flags.go
export interface IntFlag {
  readonly name: string
  readonly flag: string
  readonly env_var: string
  readonly shorthand: string
  readonly description: string
  readonly enterprise: boolean
  readonly hidden: boolean
  readonly default: number
  readonly value: number
}

// From codersdk/licenses.go
export interface License {
  readonly id: number
  readonly uploaded_at: string
  // eslint-disable-next-line
  readonly claims: Record<string, any>
}

// From codersdk/agentconn.go
export interface ListeningPort {
  readonly process_name: string
  readonly network: ListeningPortNetwork
  readonly port: number
}

// From codersdk/agentconn.go
export interface ListeningPortsResponse {
  readonly ports: ListeningPort[]
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
  readonly roles: Role[]
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

// From codersdk/groups.go
export interface PatchGroupRequest {
  readonly add_users: string[]
  readonly remove_users: string[]
  readonly name: string
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
  readonly canceled_at?: string
  readonly error?: string
  readonly status: ProvisionerJobStatus
  readonly worker_id?: string
  readonly file_id: string
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

// From codersdk/sse.go
export interface ServerSentEvent {
  readonly type: ServerSentEventType
  // eslint-disable-next-line
  readonly data: any
}

// From codersdk/flags.go
export interface StringArrayFlag {
  readonly name: string
  readonly flag: string
  readonly env_var: string
  readonly shorthand: string
  readonly description: string
  readonly enterprise: boolean
  readonly hidden: boolean
  readonly default: string[]
  readonly value: string[]
}

// From codersdk/flags.go
export interface StringFlag {
  readonly name: string
  readonly flag: string
  readonly env_var: string
  readonly shorthand: string
  readonly description: string
  readonly enterprise: boolean
  readonly secret: boolean
  readonly hidden: boolean
  readonly default: string
  readonly value: string
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
  readonly active_user_count: number
  readonly build_time_stats: TemplateBuildTimeStats
  readonly description: string
  readonly icon: string
  readonly max_ttl_ms: number
  readonly min_autostart_interval_ms: number
  readonly created_by_id: string
  readonly created_by_name: string
}

// From codersdk/templates.go
export interface TemplateACL {
  readonly users: TemplateUser[]
  readonly group: TemplateGroup[]
}

// From codersdk/templates.go
export interface TemplateBuildTimeStats {
  readonly start_ms?: number
  readonly stop_ms?: number
  readonly delete_ms?: number
}

// From codersdk/templates.go
export interface TemplateDAUsResponse {
  readonly entries: DAUEntry[]
}

// From codersdk/templates.go
export interface TemplateGroup extends Group {
  readonly role: TemplateRole
}

// From codersdk/templates.go
export interface TemplateUser extends User {
  readonly role: TemplateRole
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
export interface UpdateTemplateACL {
  readonly user_perms?: Record<string, TemplateRole>
  readonly group_perms?: Record<string, TemplateRole>
}

// From codersdk/templates.go
export interface UpdateTemplateMeta {
  readonly name?: string
  readonly description?: string
  readonly icon?: string
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
export interface UpdateWorkspaceRequest {
  readonly name?: string
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
  readonly last_seen_at: string
  readonly status: UserStatus
  readonly organization_ids: string[]
  readonly roles: Role[]
  readonly avatar_url: string
}

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
  readonly template_icon: string
  readonly latest_build: WorkspaceBuild
  readonly outdated: boolean
  readonly name: string
  readonly autostart_schedule?: string
  readonly ttl_ms?: number
  readonly last_used_at: string
}

// From codersdk/workspaceagents.go
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
  readonly version: string
  readonly apps: WorkspaceApp[]
  readonly latency?: Record<string, DERPRegion>
}

// From codersdk/workspaceagents.go
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

// From codersdk/workspaceagents.go
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
  readonly subdomain: boolean
  readonly sharing_level: WorkspaceAppSharingLevel
  readonly healthcheck: Healthcheck
  readonly health: WorkspaceAppHealth
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
  readonly transition: WorkspaceTransition
  readonly initiator_id: string
  readonly initiator_name: string
  readonly job: ProvisionerJob
  readonly reason: BuildReason
  readonly resources: WorkspaceResource[]
  readonly deadline?: string
  readonly status: WorkspaceStatus
}

// From codersdk/workspaces.go
export interface WorkspaceBuildsRequest extends Pagination {
  readonly WorkspaceID: string
  readonly Since: string
}

// From codersdk/workspaces.go
export interface WorkspaceFilter {
  readonly q?: string
}

// From codersdk/workspaces.go
export interface WorkspaceOptions {
  readonly include_deleted?: boolean
}

// From codersdk/workspacequota.go
export interface WorkspaceQuota {
  readonly user_workspace_count: number
  readonly user_workspace_limit: number
}

// From codersdk/workspacebuilds.go
export interface WorkspaceResource {
  readonly id: string
  readonly created_at: string
  readonly job_id: string
  readonly workspace_transition: WorkspaceTransition
  readonly type: string
  readonly name: string
  readonly hide: boolean
  readonly icon: string
  readonly agents?: WorkspaceAgent[]
  readonly metadata?: WorkspaceResourceMetadata[]
}

// From codersdk/workspacebuilds.go
export interface WorkspaceResourceMetadata {
  readonly key: string
  readonly value: string
  readonly sensitive: boolean
}

// From codersdk/apikey.go
export type APIKeyScope = "all" | "application_connect"

// From codersdk/audit.go
export type AuditAction = "create" | "delete" | "write"

// From codersdk/workspacebuilds.go
export type BuildReason = "autostart" | "autostop" | "initiator"

// From codersdk/features.go
export type Entitlement = "entitled" | "grace_period" | "not_entitled"

// From codersdk/agentconn.go
export type ListeningPortNetwork = "tcp"

// From codersdk/provisionerdaemons.go
export type LogLevel = "debug" | "error" | "info" | "trace" | "warn"

// From codersdk/provisionerdaemons.go
export type LogSource = "provisioner" | "provisioner_daemon"

// From codersdk/apikey.go
export type LoginType = "github" | "oidc" | "password" | "token"

// From codersdk/parameters.go
export type ParameterDestinationScheme =
  | "environment_variable"
  | "none"
  | "provisioner_variable"

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

// From codersdk/audit.go
export type ResourceType =
  | "api_key"
  | "git_ssh_key"
  | "organization"
  | "template"
  | "template_version"
  | "user"
  | "workspace"

// From codersdk/sse.go
export type ServerSentEventType = "data" | "error" | "ping"

// From codersdk/templates.go
export type TemplateRole = "" | "admin" | "use"

// From codersdk/users.go
export type UserStatus = "active" | "suspended"

// From codersdk/workspaceagents.go
export type WorkspaceAgentStatus = "connected" | "connecting" | "disconnected"

// From codersdk/workspaceapps.go
export type WorkspaceAppHealth =
  | "disabled"
  | "healthy"
  | "initializing"
  | "unhealthy"

// From codersdk/workspaceapps.go
export type WorkspaceAppSharingLevel = "authenticated" | "owner" | "public"

// From codersdk/workspacebuilds.go
export type WorkspaceStatus =
  | "canceled"
  | "canceling"
  | "deleted"
  | "deleting"
  | "failed"
  | "pending"
  | "running"
  | "starting"
  | "stopped"
  | "stopping"

// From codersdk/workspacebuilds.go
export type WorkspaceTransition = "delete" | "start" | "stop"
