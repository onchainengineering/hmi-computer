// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type sqlcQuerier interface {
	// Blocks until the lock is acquired.
	//
	// This must be called from within a transaction. The lock will be automatically
	// released when the transaction ends.
	AcquireLock(ctx context.Context, pgAdvisoryXactLock int64) error
	// Acquires the lock for a single job that isn't started, completed,
	// canceled, and that matches an array of provisioner types.
	//
	// SKIP LOCKED is used to jump over locked rows. This prevents
	// multiple provisioners from acquiring the same jobs. See:
	// https://www.postgresql.org/docs/9.5/sql-select.html#SQL-FOR-UPDATE-SHARE
	AcquireProvisionerJob(ctx context.Context, arg AcquireProvisionerJobParams) (ProvisionerJob, error)
	// Bumps the workspace deadline by the template's configured "activity_bump"
	// duration (default 1h). If the workspace bump will cross an autostart
	// threshold, then the bump is autostart + TTL. This is the deadline behavior if
	// the workspace was to autostart from a stopped state.
	//
	// Max deadline is respected, and the deadline will never be bumped past it.
	// The deadline will never decrease.
	// We only bump if the template has an activity bump duration set.
	// We only bump if the raw interval is positive and non-zero.
	// We only bump if workspace shutdown is manual.
	// We only bump when 5% of the deadline has elapsed.
	ActivityBumpWorkspace(ctx context.Context, arg ActivityBumpWorkspaceParams) error
	// AllUserIDs returns all UserIDs regardless of user status or deletion.
	AllUserIDs(ctx context.Context) ([]uuid.UUID, error)
	// Archiving templates is a soft delete action, so is reversible.
	// Archiving prevents the version from being used and discovered
	// by listing.
	// Only unused template versions will be archived, which are any versions not
	// referenced by the latest build of a workspace.
	ArchiveUnusedTemplateVersions(ctx context.Context, arg ArchiveUnusedTemplateVersionsParams) ([]uuid.UUID, error)
	BatchUpdateWorkspaceLastUsedAt(ctx context.Context, arg BatchUpdateWorkspaceLastUsedAtParams) error
	CleanTailnetCoordinators(ctx context.Context) error
	CleanTailnetLostPeers(ctx context.Context) error
	CleanTailnetTunnels(ctx context.Context) error
	DeleteAPIKeyByID(ctx context.Context, id string) error
	DeleteAPIKeysByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteAllTailnetClientSubscriptions(ctx context.Context, arg DeleteAllTailnetClientSubscriptionsParams) error
	DeleteAllTailnetTunnels(ctx context.Context, arg DeleteAllTailnetTunnelsParams) error
	DeleteApplicationConnectAPIKeysByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteCoordinator(ctx context.Context, id uuid.UUID) error
	DeleteExternalAuthLink(ctx context.Context, arg DeleteExternalAuthLinkParams) error
	DeleteGitSSHKey(ctx context.Context, userID uuid.UUID) error
	DeleteGroupByID(ctx context.Context, id uuid.UUID) error
	DeleteGroupMemberFromGroup(ctx context.Context, arg DeleteGroupMemberFromGroupParams) error
	DeleteLicense(ctx context.Context, id int32) (int32, error)
	DeleteOAuth2ProviderAppByID(ctx context.Context, id uuid.UUID) error
	DeleteOAuth2ProviderAppCodeByID(ctx context.Context, id uuid.UUID) error
	DeleteOAuth2ProviderAppCodesByAppAndUserID(ctx context.Context, arg DeleteOAuth2ProviderAppCodesByAppAndUserIDParams) error
	DeleteOAuth2ProviderAppSecretByID(ctx context.Context, id uuid.UUID) error
	DeleteOAuth2ProviderAppTokensByAppAndUserID(ctx context.Context, arg DeleteOAuth2ProviderAppTokensByAppAndUserIDParams) error
	// Delete provisioner daemons that have been created at least a week ago
	// and have not connected to coderd since a week.
	// A provisioner daemon with "zeroed" last_seen_at column indicates possible
	// connectivity issues (no provisioner daemon activity since registration).
	DeleteOldProvisionerDaemons(ctx context.Context) error
	// If an agent hasn't connected in the last 7 days, we purge it's logs.
	// Logs can take up a lot of space, so it's important we clean up frequently.
	DeleteOldWorkspaceAgentLogs(ctx context.Context) error
	DeleteOldWorkspaceAgentStats(ctx context.Context) error
	DeleteReplicasUpdatedBefore(ctx context.Context, updatedAt time.Time) error
	DeleteTailnetAgent(ctx context.Context, arg DeleteTailnetAgentParams) (DeleteTailnetAgentRow, error)
	DeleteTailnetClient(ctx context.Context, arg DeleteTailnetClientParams) (DeleteTailnetClientRow, error)
	DeleteTailnetClientSubscription(ctx context.Context, arg DeleteTailnetClientSubscriptionParams) error
	DeleteTailnetPeer(ctx context.Context, arg DeleteTailnetPeerParams) (DeleteTailnetPeerRow, error)
	DeleteTailnetTunnel(ctx context.Context, arg DeleteTailnetTunnelParams) (DeleteTailnetTunnelRow, error)
	DeleteWorkspaceAgentPortShare(ctx context.Context, arg DeleteWorkspaceAgentPortShareParams) error
	DeleteWorkspaceAgentPortSharesByTemplate(ctx context.Context, templateID uuid.UUID) error
	FavoriteWorkspace(ctx context.Context, id uuid.UUID) error
	GetAPIKeyByID(ctx context.Context, id string) (APIKey, error)
	// there is no unique constraint on empty token names
	GetAPIKeyByName(ctx context.Context, arg GetAPIKeyByNameParams) (APIKey, error)
	GetAPIKeysByLoginType(ctx context.Context, loginType LoginType) ([]APIKey, error)
	GetAPIKeysByUserID(ctx context.Context, arg GetAPIKeysByUserIDParams) ([]APIKey, error)
	GetAPIKeysLastUsedAfter(ctx context.Context, lastUsed time.Time) ([]APIKey, error)
	GetActiveUserCount(ctx context.Context) (int64, error)
	GetActiveWorkspaceBuildsByTemplateID(ctx context.Context, templateID uuid.UUID) ([]WorkspaceBuild, error)
	GetAllTailnetAgents(ctx context.Context) ([]TailnetAgent, error)
	// For PG Coordinator HTMLDebug
	GetAllTailnetCoordinators(ctx context.Context) ([]TailnetCoordinator, error)
	GetAllTailnetPeers(ctx context.Context) ([]TailnetPeer, error)
	GetAllTailnetTunnels(ctx context.Context) ([]TailnetTunnel, error)
	GetAppSecurityKey(ctx context.Context) (string, error)
	GetApplicationName(ctx context.Context) (string, error)
	// GetAuditLogsBefore retrieves `row_limit` number of audit logs before the provided
	// ID.
	GetAuditLogsOffset(ctx context.Context, arg GetAuditLogsOffsetParams) ([]GetAuditLogsOffsetRow, error)
	// This function returns roles for authorization purposes. Implied member roles
	// are included.
	GetAuthorizationUserRoles(ctx context.Context, userID uuid.UUID) (GetAuthorizationUserRolesRow, error)
	GetDBCryptKeys(ctx context.Context) ([]DBCryptKey, error)
	GetDERPMeshKey(ctx context.Context) (string, error)
	GetDefaultOrganization(ctx context.Context) (Organization, error)
	GetDefaultProxyConfig(ctx context.Context) (GetDefaultProxyConfigRow, error)
	GetDeploymentDAUs(ctx context.Context, tzOffset int32) ([]GetDeploymentDAUsRow, error)
	GetDeploymentID(ctx context.Context) (string, error)
	GetDeploymentWorkspaceAgentStats(ctx context.Context, createdAt time.Time) (GetDeploymentWorkspaceAgentStatsRow, error)
	GetDeploymentWorkspaceStats(ctx context.Context) (GetDeploymentWorkspaceStatsRow, error)
	GetExternalAuthLink(ctx context.Context, arg GetExternalAuthLinkParams) (ExternalAuthLink, error)
	GetExternalAuthLinksByUserID(ctx context.Context, userID uuid.UUID) ([]ExternalAuthLink, error)
	GetFileByHashAndCreator(ctx context.Context, arg GetFileByHashAndCreatorParams) (File, error)
	GetFileByID(ctx context.Context, id uuid.UUID) (File, error)
	// Get all templates that use a file.
	GetFileTemplates(ctx context.Context, fileID uuid.UUID) ([]GetFileTemplatesRow, error)
	GetGitSSHKey(ctx context.Context, userID uuid.UUID) (GitSSHKey, error)
	GetGroupByID(ctx context.Context, id uuid.UUID) (Group, error)
	GetGroupByOrgAndName(ctx context.Context, arg GetGroupByOrgAndNameParams) (Group, error)
	// If the group is a user made group, then we need to check the group_members table.
	// If it is the "Everyone" group, then we need to check the organization_members table.
	GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]User, error)
	GetGroupsByOrganizationID(ctx context.Context, organizationID uuid.UUID) ([]Group, error)
	GetHealthSettings(ctx context.Context) (string, error)
	GetHungProvisionerJobs(ctx context.Context, updatedAt time.Time) ([]ProvisionerJob, error)
	GetJFrogXrayScanByWorkspaceAndAgentID(ctx context.Context, arg GetJFrogXrayScanByWorkspaceAndAgentIDParams) (JfrogXrayScan, error)
	GetLastUpdateCheck(ctx context.Context) (string, error)
	GetLatestWorkspaceBuildByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) (WorkspaceBuild, error)
	GetLatestWorkspaceBuilds(ctx context.Context) ([]WorkspaceBuild, error)
	GetLatestWorkspaceBuildsByWorkspaceIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceBuild, error)
	GetLicenseByID(ctx context.Context, id int32) (License, error)
	GetLicenses(ctx context.Context) ([]License, error)
	GetLogoURL(ctx context.Context) (string, error)
	GetOAuth2ProviderAppByID(ctx context.Context, id uuid.UUID) (OAuth2ProviderApp, error)
	GetOAuth2ProviderAppCodeByID(ctx context.Context, id uuid.UUID) (OAuth2ProviderAppCode, error)
	GetOAuth2ProviderAppCodeByPrefix(ctx context.Context, secretPrefix []byte) (OAuth2ProviderAppCode, error)
	GetOAuth2ProviderAppSecretByID(ctx context.Context, id uuid.UUID) (OAuth2ProviderAppSecret, error)
	GetOAuth2ProviderAppSecretByPrefix(ctx context.Context, secretPrefix []byte) (OAuth2ProviderAppSecret, error)
	GetOAuth2ProviderAppSecretsByAppID(ctx context.Context, appID uuid.UUID) ([]OAuth2ProviderAppSecret, error)
	GetOAuth2ProviderAppTokenByPrefix(ctx context.Context, hashPrefix []byte) (OAuth2ProviderAppToken, error)
	GetOAuth2ProviderApps(ctx context.Context) ([]OAuth2ProviderApp, error)
	GetOAuth2ProviderAppsByUserID(ctx context.Context, userID uuid.UUID) ([]GetOAuth2ProviderAppsByUserIDRow, error)
	GetOAuthSigningKey(ctx context.Context) (string, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (Organization, error)
	GetOrganizationByName(ctx context.Context, name string) (Organization, error)
	GetOrganizationIDsByMemberIDs(ctx context.Context, ids []uuid.UUID) ([]GetOrganizationIDsByMemberIDsRow, error)
	GetOrganizationMemberByUserID(ctx context.Context, arg GetOrganizationMemberByUserIDParams) (OrganizationMember, error)
	GetOrganizationMembershipsByUserID(ctx context.Context, userID uuid.UUID) ([]OrganizationMember, error)
	GetOrganizations(ctx context.Context) ([]Organization, error)
	GetOrganizationsByUserID(ctx context.Context, userID uuid.UUID) ([]Organization, error)
	GetParameterSchemasByJobID(ctx context.Context, jobID uuid.UUID) ([]ParameterSchema, error)
	GetPreviousTemplateVersion(ctx context.Context, arg GetPreviousTemplateVersionParams) (TemplateVersion, error)
	GetProvisionerDaemons(ctx context.Context) ([]ProvisionerDaemon, error)
	GetProvisionerJobByID(ctx context.Context, id uuid.UUID) (ProvisionerJob, error)
	GetProvisionerJobsByIDs(ctx context.Context, ids []uuid.UUID) ([]ProvisionerJob, error)
	GetProvisionerJobsByIDsWithQueuePosition(ctx context.Context, ids []uuid.UUID) ([]GetProvisionerJobsByIDsWithQueuePositionRow, error)
	GetProvisionerJobsCreatedAfter(ctx context.Context, createdAt time.Time) ([]ProvisionerJob, error)
	GetProvisionerLogsAfterID(ctx context.Context, arg GetProvisionerLogsAfterIDParams) ([]ProvisionerJobLog, error)
	GetQuotaAllowanceForUser(ctx context.Context, userID uuid.UUID) (int64, error)
	GetQuotaConsumedForUser(ctx context.Context, ownerID uuid.UUID) (int64, error)
	GetReplicaByID(ctx context.Context, id uuid.UUID) (Replica, error)
	GetReplicasUpdatedAfter(ctx context.Context, updatedAt time.Time) ([]Replica, error)
	GetServiceBanner(ctx context.Context) (string, error)
	GetTailnetAgents(ctx context.Context, id uuid.UUID) ([]TailnetAgent, error)
	GetTailnetClientsForAgent(ctx context.Context, agentID uuid.UUID) ([]TailnetClient, error)
	GetTailnetPeers(ctx context.Context, id uuid.UUID) ([]TailnetPeer, error)
	GetTailnetTunnelPeerBindings(ctx context.Context, srcID uuid.UUID) ([]GetTailnetTunnelPeerBindingsRow, error)
	GetTailnetTunnelPeerIDs(ctx context.Context, srcID uuid.UUID) ([]GetTailnetTunnelPeerIDsRow, error)
	// GetTemplateAppInsights returns the aggregate usage of each app in a given
	// timeframe. The result can be filtered on template_ids, meaning only user data
	// from workspaces based on those templates will be included.
	GetTemplateAppInsights(ctx context.Context, arg GetTemplateAppInsightsParams) ([]GetTemplateAppInsightsRow, error)
	// GetTemplateAppInsightsByTemplate is used for Prometheus metrics. Keep
	// in sync with GetTemplateAppInsights and UpsertTemplateUsageStats.
	GetTemplateAppInsightsByTemplate(ctx context.Context, arg GetTemplateAppInsightsByTemplateParams) ([]GetTemplateAppInsightsByTemplateRow, error)
	GetTemplateAverageBuildTime(ctx context.Context, arg GetTemplateAverageBuildTimeParams) (GetTemplateAverageBuildTimeRow, error)
	GetTemplateByID(ctx context.Context, id uuid.UUID) (Template, error)
	GetTemplateByOrganizationAndName(ctx context.Context, arg GetTemplateByOrganizationAndNameParams) (Template, error)
	GetTemplateDAUs(ctx context.Context, arg GetTemplateDAUsParams) ([]GetTemplateDAUsRow, error)
	// GetTemplateInsights returns the aggregate user-produced usage of all
	// workspaces in a given timeframe. The template IDs, active users, and
	// usage_seconds all reflect any usage in the template, including apps.
	//
	// When combining data from multiple templates, we must make a guess at
	// how the user behaved for the 30 minute interval. In this case we make
	// the assumption that if the user used two workspaces for 15 minutes,
	// they did so sequentially, thus we sum the usage up to a maximum of
	// 30 minutes with LEAST(SUM(n), 30).
	GetTemplateInsights(ctx context.Context, arg GetTemplateInsightsParams) (GetTemplateInsightsRow, error)
	// GetTemplateInsightsByInterval returns all intervals between start and end
	// time, if end time is a partial interval, it will be included in the results and
	// that interval will be shorter than a full one. If there is no data for a selected
	// interval/template, it will be included in the results with 0 active users.
	GetTemplateInsightsByInterval(ctx context.Context, arg GetTemplateInsightsByIntervalParams) ([]GetTemplateInsightsByIntervalRow, error)
	// GetTemplateInsightsByTemplate is used for Prometheus metrics. Keep
	// in sync with GetTemplateInsights and UpsertTemplateUsageStats.
	GetTemplateInsightsByTemplate(ctx context.Context, arg GetTemplateInsightsByTemplateParams) ([]GetTemplateInsightsByTemplateRow, error)
	// GetTemplateParameterInsights does for each template in a given timeframe,
	// look for the latest workspace build (for every workspace) that has been
	// created in the timeframe and return the aggregate usage counts of parameter
	// values.
	GetTemplateParameterInsights(ctx context.Context, arg GetTemplateParameterInsightsParams) ([]GetTemplateParameterInsightsRow, error)
	GetTemplateUsageStats(ctx context.Context, arg GetTemplateUsageStatsParams) ([]TemplateUsageStat, error)
	GetTemplateVersionByID(ctx context.Context, id uuid.UUID) (TemplateVersion, error)
	GetTemplateVersionByJobID(ctx context.Context, jobID uuid.UUID) (TemplateVersion, error)
	GetTemplateVersionByTemplateIDAndName(ctx context.Context, arg GetTemplateVersionByTemplateIDAndNameParams) (TemplateVersion, error)
	GetTemplateVersionParameters(ctx context.Context, templateVersionID uuid.UUID) ([]TemplateVersionParameter, error)
	GetTemplateVersionVariables(ctx context.Context, templateVersionID uuid.UUID) ([]TemplateVersionVariable, error)
	GetTemplateVersionsByIDs(ctx context.Context, ids []uuid.UUID) ([]TemplateVersion, error)
	GetTemplateVersionsByTemplateID(ctx context.Context, arg GetTemplateVersionsByTemplateIDParams) ([]TemplateVersion, error)
	GetTemplateVersionsCreatedAfter(ctx context.Context, createdAt time.Time) ([]TemplateVersion, error)
	GetTemplates(ctx context.Context) ([]Template, error)
	GetTemplatesWithFilter(ctx context.Context, arg GetTemplatesWithFilterParams) ([]Template, error)
	GetUnexpiredLicenses(ctx context.Context) ([]License, error)
	// GetUserActivityInsights returns the ranking with top active users.
	// The result can be filtered on template_ids, meaning only user data
	// from workspaces based on those templates will be included.
	// Note: The usage_seconds and usage_seconds_cumulative differ only when
	// requesting deployment-wide (or multiple template) data. Cumulative
	// produces a bloated value if a user has used multiple templates
	// simultaneously.
	GetUserActivityInsights(ctx context.Context, arg GetUserActivityInsightsParams) ([]GetUserActivityInsightsRow, error)
	GetUserByEmailOrUsername(ctx context.Context, arg GetUserByEmailOrUsernameParams) (User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (User, error)
	GetUserCount(ctx context.Context) (int64, error)
	// GetUserLatencyInsights returns the median and 95th percentile connection
	// latency that users have experienced. The result can be filtered on
	// template_ids, meaning only user data from workspaces based on those templates
	// will be included.
	GetUserLatencyInsights(ctx context.Context, arg GetUserLatencyInsightsParams) ([]GetUserLatencyInsightsRow, error)
	GetUserLinkByLinkedID(ctx context.Context, linkedID string) (UserLink, error)
	GetUserLinkByUserIDLoginType(ctx context.Context, arg GetUserLinkByUserIDLoginTypeParams) (UserLink, error)
	GetUserLinksByUserID(ctx context.Context, userID uuid.UUID) ([]UserLink, error)
	GetUserWorkspaceBuildParameters(ctx context.Context, arg GetUserWorkspaceBuildParametersParams) ([]GetUserWorkspaceBuildParametersRow, error)
	// This will never return deleted users.
	GetUsers(ctx context.Context, arg GetUsersParams) ([]GetUsersRow, error)
	// This shouldn't check for deleted, because it's frequently used
	// to look up references to actions. eg. a user could build a workspace
	// for another user, then be deleted... we still want them to appear!
	GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]User, error)
	GetWorkspaceAgentAndLatestBuildByAuthToken(ctx context.Context, authToken uuid.UUID) (GetWorkspaceAgentAndLatestBuildByAuthTokenRow, error)
	GetWorkspaceAgentByID(ctx context.Context, id uuid.UUID) (WorkspaceAgent, error)
	GetWorkspaceAgentByInstanceID(ctx context.Context, authInstanceID string) (WorkspaceAgent, error)
	GetWorkspaceAgentLifecycleStateByID(ctx context.Context, id uuid.UUID) (GetWorkspaceAgentLifecycleStateByIDRow, error)
	GetWorkspaceAgentLogSourcesByAgentIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceAgentLogSource, error)
	GetWorkspaceAgentLogsAfter(ctx context.Context, arg GetWorkspaceAgentLogsAfterParams) ([]WorkspaceAgentLog, error)
	GetWorkspaceAgentMetadata(ctx context.Context, arg GetWorkspaceAgentMetadataParams) ([]WorkspaceAgentMetadatum, error)
	GetWorkspaceAgentPortShare(ctx context.Context, arg GetWorkspaceAgentPortShareParams) (WorkspaceAgentPortShare, error)
	GetWorkspaceAgentScriptsByAgentIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceAgentScript, error)
	GetWorkspaceAgentStats(ctx context.Context, createdAt time.Time) ([]GetWorkspaceAgentStatsRow, error)
	GetWorkspaceAgentStatsAndLabels(ctx context.Context, createdAt time.Time) ([]GetWorkspaceAgentStatsAndLabelsRow, error)
	GetWorkspaceAgentsByResourceIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceAgent, error)
	GetWorkspaceAgentsCreatedAfter(ctx context.Context, createdAt time.Time) ([]WorkspaceAgent, error)
	GetWorkspaceAgentsInLatestBuildByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceAgent, error)
	GetWorkspaceAppByAgentIDAndSlug(ctx context.Context, arg GetWorkspaceAppByAgentIDAndSlugParams) (WorkspaceApp, error)
	GetWorkspaceAppsByAgentID(ctx context.Context, agentID uuid.UUID) ([]WorkspaceApp, error)
	GetWorkspaceAppsByAgentIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceApp, error)
	GetWorkspaceAppsCreatedAfter(ctx context.Context, createdAt time.Time) ([]WorkspaceApp, error)
	GetWorkspaceBuildByID(ctx context.Context, id uuid.UUID) (WorkspaceBuild, error)
	GetWorkspaceBuildByJobID(ctx context.Context, jobID uuid.UUID) (WorkspaceBuild, error)
	GetWorkspaceBuildByWorkspaceIDAndBuildNumber(ctx context.Context, arg GetWorkspaceBuildByWorkspaceIDAndBuildNumberParams) (WorkspaceBuild, error)
	GetWorkspaceBuildParameters(ctx context.Context, workspaceBuildID uuid.UUID) ([]WorkspaceBuildParameter, error)
	GetWorkspaceBuildsByWorkspaceID(ctx context.Context, arg GetWorkspaceBuildsByWorkspaceIDParams) ([]WorkspaceBuild, error)
	GetWorkspaceBuildsCreatedAfter(ctx context.Context, createdAt time.Time) ([]WorkspaceBuild, error)
	GetWorkspaceByAgentID(ctx context.Context, agentID uuid.UUID) (GetWorkspaceByAgentIDRow, error)
	GetWorkspaceByID(ctx context.Context, id uuid.UUID) (Workspace, error)
	GetWorkspaceByOwnerIDAndName(ctx context.Context, arg GetWorkspaceByOwnerIDAndNameParams) (Workspace, error)
	GetWorkspaceByWorkspaceAppID(ctx context.Context, workspaceAppID uuid.UUID) (Workspace, error)
	GetWorkspaceProxies(ctx context.Context) ([]WorkspaceProxy, error)
	// Finds a workspace proxy that has an access URL or app hostname that matches
	// the provided hostname. This is to check if a hostname matches any workspace
	// proxy.
	//
	// The hostname must be sanitized to only contain [a-zA-Z0-9.-] before calling
	// this query. The scheme, port and path should be stripped.
	//
	GetWorkspaceProxyByHostname(ctx context.Context, arg GetWorkspaceProxyByHostnameParams) (WorkspaceProxy, error)
	GetWorkspaceProxyByID(ctx context.Context, id uuid.UUID) (WorkspaceProxy, error)
	GetWorkspaceProxyByName(ctx context.Context, name string) (WorkspaceProxy, error)
	GetWorkspaceResourceByID(ctx context.Context, id uuid.UUID) (WorkspaceResource, error)
	GetWorkspaceResourceMetadataByResourceIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceResourceMetadatum, error)
	GetWorkspaceResourceMetadataCreatedAfter(ctx context.Context, createdAt time.Time) ([]WorkspaceResourceMetadatum, error)
	GetWorkspaceResourcesByJobID(ctx context.Context, jobID uuid.UUID) ([]WorkspaceResource, error)
	GetWorkspaceResourcesByJobIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceResource, error)
	GetWorkspaceResourcesCreatedAfter(ctx context.Context, createdAt time.Time) ([]WorkspaceResource, error)
	GetWorkspaceUniqueOwnerCountByTemplateIDs(ctx context.Context, templateIds []uuid.UUID) ([]GetWorkspaceUniqueOwnerCountByTemplateIDsRow, error)
	// build_params is used to filter by build parameters if present.
	// It has to be a CTE because the set returning function 'unnest' cannot
	// be used in a WHERE clause.
	GetWorkspaces(ctx context.Context, arg GetWorkspacesParams) ([]GetWorkspacesRow, error)
	GetWorkspacesEligibleForTransition(ctx context.Context, now time.Time) ([]Workspace, error)
	InsertAPIKey(ctx context.Context, arg InsertAPIKeyParams) (APIKey, error)
	// We use the organization_id as the id
	// for simplicity since all users is
	// every member of the org.
	InsertAllUsersGroup(ctx context.Context, organizationID uuid.UUID) (Group, error)
	InsertAuditLog(ctx context.Context, arg InsertAuditLogParams) (AuditLog, error)
	InsertDBCryptKey(ctx context.Context, arg InsertDBCryptKeyParams) error
	InsertDERPMeshKey(ctx context.Context, value string) error
	InsertDeploymentID(ctx context.Context, value string) error
	InsertExternalAuthLink(ctx context.Context, arg InsertExternalAuthLinkParams) (ExternalAuthLink, error)
	InsertFile(ctx context.Context, arg InsertFileParams) (File, error)
	InsertGitSSHKey(ctx context.Context, arg InsertGitSSHKeyParams) (GitSSHKey, error)
	InsertGroup(ctx context.Context, arg InsertGroupParams) (Group, error)
	InsertGroupMember(ctx context.Context, arg InsertGroupMemberParams) error
	InsertLicense(ctx context.Context, arg InsertLicenseParams) (License, error)
	// Inserts any group by name that does not exist. All new groups are given
	// a random uuid, are inserted into the same organization. They have the default
	// values for avatar, display name, and quota allowance (all zero values).
	// If the name conflicts, do nothing.
	InsertMissingGroups(ctx context.Context, arg InsertMissingGroupsParams) ([]Group, error)
	InsertOAuth2ProviderApp(ctx context.Context, arg InsertOAuth2ProviderAppParams) (OAuth2ProviderApp, error)
	InsertOAuth2ProviderAppCode(ctx context.Context, arg InsertOAuth2ProviderAppCodeParams) (OAuth2ProviderAppCode, error)
	InsertOAuth2ProviderAppSecret(ctx context.Context, arg InsertOAuth2ProviderAppSecretParams) (OAuth2ProviderAppSecret, error)
	InsertOAuth2ProviderAppToken(ctx context.Context, arg InsertOAuth2ProviderAppTokenParams) (OAuth2ProviderAppToken, error)
	InsertOrganization(ctx context.Context, arg InsertOrganizationParams) (Organization, error)
	InsertOrganizationMember(ctx context.Context, arg InsertOrganizationMemberParams) (OrganizationMember, error)
	InsertProvisionerJob(ctx context.Context, arg InsertProvisionerJobParams) (ProvisionerJob, error)
	InsertProvisionerJobLogs(ctx context.Context, arg InsertProvisionerJobLogsParams) ([]ProvisionerJobLog, error)
	InsertReplica(ctx context.Context, arg InsertReplicaParams) (Replica, error)
	InsertTemplate(ctx context.Context, arg InsertTemplateParams) error
	InsertTemplateVersion(ctx context.Context, arg InsertTemplateVersionParams) error
	InsertTemplateVersionParameter(ctx context.Context, arg InsertTemplateVersionParameterParams) (TemplateVersionParameter, error)
	InsertTemplateVersionVariable(ctx context.Context, arg InsertTemplateVersionVariableParams) (TemplateVersionVariable, error)
	InsertUser(ctx context.Context, arg InsertUserParams) (User, error)
	// InsertUserGroupsByName adds a user to all provided groups, if they exist.
	InsertUserGroupsByName(ctx context.Context, arg InsertUserGroupsByNameParams) error
	InsertUserLink(ctx context.Context, arg InsertUserLinkParams) (UserLink, error)
	InsertWorkspace(ctx context.Context, arg InsertWorkspaceParams) (Workspace, error)
	InsertWorkspaceAgent(ctx context.Context, arg InsertWorkspaceAgentParams) (WorkspaceAgent, error)
	InsertWorkspaceAgentLogSources(ctx context.Context, arg InsertWorkspaceAgentLogSourcesParams) ([]WorkspaceAgentLogSource, error)
	InsertWorkspaceAgentLogs(ctx context.Context, arg InsertWorkspaceAgentLogsParams) ([]WorkspaceAgentLog, error)
	InsertWorkspaceAgentMetadata(ctx context.Context, arg InsertWorkspaceAgentMetadataParams) error
	InsertWorkspaceAgentScripts(ctx context.Context, arg InsertWorkspaceAgentScriptsParams) ([]WorkspaceAgentScript, error)
	InsertWorkspaceAgentStats(ctx context.Context, arg InsertWorkspaceAgentStatsParams) error
	InsertWorkspaceApp(ctx context.Context, arg InsertWorkspaceAppParams) (WorkspaceApp, error)
	InsertWorkspaceAppStats(ctx context.Context, arg InsertWorkspaceAppStatsParams) error
	InsertWorkspaceBuild(ctx context.Context, arg InsertWorkspaceBuildParams) error
	InsertWorkspaceBuildParameters(ctx context.Context, arg InsertWorkspaceBuildParametersParams) error
	InsertWorkspaceProxy(ctx context.Context, arg InsertWorkspaceProxyParams) (WorkspaceProxy, error)
	InsertWorkspaceResource(ctx context.Context, arg InsertWorkspaceResourceParams) (WorkspaceResource, error)
	InsertWorkspaceResourceMetadata(ctx context.Context, arg InsertWorkspaceResourceMetadataParams) ([]WorkspaceResourceMetadatum, error)
	ListWorkspaceAgentPortShares(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceAgentPortShare, error)
	ReduceWorkspaceAgentShareLevelToAuthenticatedByTemplate(ctx context.Context, templateID uuid.UUID) error
	RegisterWorkspaceProxy(ctx context.Context, arg RegisterWorkspaceProxyParams) (WorkspaceProxy, error)
	RemoveUserFromAllGroups(ctx context.Context, userID uuid.UUID) error
	RevokeDBCryptKey(ctx context.Context, activeKeyDigest string) error
	// Non blocking lock. Returns true if the lock was acquired, false otherwise.
	//
	// This must be called from within a transaction. The lock will be automatically
	// released when the transaction ends.
	TryAcquireLock(ctx context.Context, pgTryAdvisoryXactLock int64) (bool, error)
	// This will always work regardless of the current state of the template version.
	UnarchiveTemplateVersion(ctx context.Context, arg UnarchiveTemplateVersionParams) error
	UnfavoriteWorkspace(ctx context.Context, id uuid.UUID) error
	UpdateAPIKeyByID(ctx context.Context, arg UpdateAPIKeyByIDParams) error
	UpdateExternalAuthLink(ctx context.Context, arg UpdateExternalAuthLinkParams) (ExternalAuthLink, error)
	UpdateGitSSHKey(ctx context.Context, arg UpdateGitSSHKeyParams) (GitSSHKey, error)
	UpdateGroupByID(ctx context.Context, arg UpdateGroupByIDParams) (Group, error)
	UpdateInactiveUsersToDormant(ctx context.Context, arg UpdateInactiveUsersToDormantParams) ([]UpdateInactiveUsersToDormantRow, error)
	UpdateMemberRoles(ctx context.Context, arg UpdateMemberRolesParams) (OrganizationMember, error)
	UpdateOAuth2ProviderAppByID(ctx context.Context, arg UpdateOAuth2ProviderAppByIDParams) (OAuth2ProviderApp, error)
	UpdateOAuth2ProviderAppSecretByID(ctx context.Context, arg UpdateOAuth2ProviderAppSecretByIDParams) (OAuth2ProviderAppSecret, error)
	UpdateProvisionerDaemonLastSeenAt(ctx context.Context, arg UpdateProvisionerDaemonLastSeenAtParams) error
	UpdateProvisionerJobByID(ctx context.Context, arg UpdateProvisionerJobByIDParams) error
	UpdateProvisionerJobWithCancelByID(ctx context.Context, arg UpdateProvisionerJobWithCancelByIDParams) error
	UpdateProvisionerJobWithCompleteByID(ctx context.Context, arg UpdateProvisionerJobWithCompleteByIDParams) error
	UpdateReplica(ctx context.Context, arg UpdateReplicaParams) (Replica, error)
	UpdateTemplateACLByID(ctx context.Context, arg UpdateTemplateACLByIDParams) error
	UpdateTemplateAccessControlByID(ctx context.Context, arg UpdateTemplateAccessControlByIDParams) error
	UpdateTemplateActiveVersionByID(ctx context.Context, arg UpdateTemplateActiveVersionByIDParams) error
	UpdateTemplateDeletedByID(ctx context.Context, arg UpdateTemplateDeletedByIDParams) error
	UpdateTemplateMetaByID(ctx context.Context, arg UpdateTemplateMetaByIDParams) error
	UpdateTemplateScheduleByID(ctx context.Context, arg UpdateTemplateScheduleByIDParams) error
	UpdateTemplateVersionByID(ctx context.Context, arg UpdateTemplateVersionByIDParams) error
	UpdateTemplateVersionDescriptionByJobID(ctx context.Context, arg UpdateTemplateVersionDescriptionByJobIDParams) error
	UpdateTemplateVersionExternalAuthProvidersByJobID(ctx context.Context, arg UpdateTemplateVersionExternalAuthProvidersByJobIDParams) error
	UpdateTemplateWorkspacesLastUsedAt(ctx context.Context, arg UpdateTemplateWorkspacesLastUsedAtParams) error
	UpdateUserAppearanceSettings(ctx context.Context, arg UpdateUserAppearanceSettingsParams) (User, error)
	UpdateUserDeletedByID(ctx context.Context, id uuid.UUID) error
	UpdateUserHashedPassword(ctx context.Context, arg UpdateUserHashedPasswordParams) error
	UpdateUserLastSeenAt(ctx context.Context, arg UpdateUserLastSeenAtParams) (User, error)
	UpdateUserLink(ctx context.Context, arg UpdateUserLinkParams) (UserLink, error)
	UpdateUserLinkedID(ctx context.Context, arg UpdateUserLinkedIDParams) (UserLink, error)
	UpdateUserLoginType(ctx context.Context, arg UpdateUserLoginTypeParams) (User, error)
	UpdateUserProfile(ctx context.Context, arg UpdateUserProfileParams) (User, error)
	UpdateUserQuietHoursSchedule(ctx context.Context, arg UpdateUserQuietHoursScheduleParams) (User, error)
	UpdateUserRoles(ctx context.Context, arg UpdateUserRolesParams) (User, error)
	UpdateUserStatus(ctx context.Context, arg UpdateUserStatusParams) (User, error)
	UpdateWorkspace(ctx context.Context, arg UpdateWorkspaceParams) (Workspace, error)
	UpdateWorkspaceAgentConnectionByID(ctx context.Context, arg UpdateWorkspaceAgentConnectionByIDParams) error
	UpdateWorkspaceAgentLifecycleStateByID(ctx context.Context, arg UpdateWorkspaceAgentLifecycleStateByIDParams) error
	UpdateWorkspaceAgentLogOverflowByID(ctx context.Context, arg UpdateWorkspaceAgentLogOverflowByIDParams) error
	UpdateWorkspaceAgentMetadata(ctx context.Context, arg UpdateWorkspaceAgentMetadataParams) error
	UpdateWorkspaceAgentStartupByID(ctx context.Context, arg UpdateWorkspaceAgentStartupByIDParams) error
	UpdateWorkspaceAppHealthByID(ctx context.Context, arg UpdateWorkspaceAppHealthByIDParams) error
	UpdateWorkspaceAutomaticUpdates(ctx context.Context, arg UpdateWorkspaceAutomaticUpdatesParams) error
	UpdateWorkspaceAutostart(ctx context.Context, arg UpdateWorkspaceAutostartParams) error
	UpdateWorkspaceBuildCostByID(ctx context.Context, arg UpdateWorkspaceBuildCostByIDParams) error
	UpdateWorkspaceBuildDeadlineByID(ctx context.Context, arg UpdateWorkspaceBuildDeadlineByIDParams) error
	UpdateWorkspaceBuildProvisionerStateByID(ctx context.Context, arg UpdateWorkspaceBuildProvisionerStateByIDParams) error
	UpdateWorkspaceDeletedByID(ctx context.Context, arg UpdateWorkspaceDeletedByIDParams) error
	UpdateWorkspaceDormantDeletingAt(ctx context.Context, arg UpdateWorkspaceDormantDeletingAtParams) (Workspace, error)
	UpdateWorkspaceLastUsedAt(ctx context.Context, arg UpdateWorkspaceLastUsedAtParams) error
	// This allows editing the properties of a workspace proxy.
	UpdateWorkspaceProxy(ctx context.Context, arg UpdateWorkspaceProxyParams) (WorkspaceProxy, error)
	UpdateWorkspaceProxyDeleted(ctx context.Context, arg UpdateWorkspaceProxyDeletedParams) error
	UpdateWorkspaceTTL(ctx context.Context, arg UpdateWorkspaceTTLParams) error
	UpdateWorkspacesDormantDeletingAtByTemplateID(ctx context.Context, arg UpdateWorkspacesDormantDeletingAtByTemplateIDParams) error
	UpsertAppSecurityKey(ctx context.Context, value string) error
	UpsertApplicationName(ctx context.Context, value string) error
	// The default proxy is implied and not actually stored in the database.
	// So we need to store it's configuration here for display purposes.
	// The functional values are immutable and controlled implicitly.
	UpsertDefaultProxy(ctx context.Context, arg UpsertDefaultProxyParams) error
	UpsertHealthSettings(ctx context.Context, value string) error
	UpsertJFrogXrayScanByWorkspaceAndAgentID(ctx context.Context, arg UpsertJFrogXrayScanByWorkspaceAndAgentIDParams) error
	UpsertLastUpdateCheck(ctx context.Context, value string) error
	UpsertLogoURL(ctx context.Context, value string) error
	UpsertOAuthSigningKey(ctx context.Context, value string) error
	UpsertProvisionerDaemon(ctx context.Context, arg UpsertProvisionerDaemonParams) (ProvisionerDaemon, error)
	UpsertServiceBanner(ctx context.Context, value string) error
	UpsertTailnetAgent(ctx context.Context, arg UpsertTailnetAgentParams) (TailnetAgent, error)
	UpsertTailnetClient(ctx context.Context, arg UpsertTailnetClientParams) (TailnetClient, error)
	UpsertTailnetClientSubscription(ctx context.Context, arg UpsertTailnetClientSubscriptionParams) error
	UpsertTailnetCoordinator(ctx context.Context, id uuid.UUID) (TailnetCoordinator, error)
	UpsertTailnetPeer(ctx context.Context, arg UpsertTailnetPeerParams) (TailnetPeer, error)
	UpsertTailnetTunnel(ctx context.Context, arg UpsertTailnetTunnelParams) (TailnetTunnel, error)
	// This query aggregates the workspace_agent_stats and workspace_app_stats data
	// into a single table for efficient storage and querying. Half-hour buckets are
	// used to store the data, and the minutes are summed for each user and template
	// combination. The result is stored in the template_usage_stats table.
	UpsertTemplateUsageStats(ctx context.Context) error
	UpsertWorkspaceAgentPortShare(ctx context.Context, arg UpsertWorkspaceAgentPortShareParams) (WorkspaceAgentPortShare, error)
}

var _ sqlcQuerier = (*sqlQuerier)(nil)
