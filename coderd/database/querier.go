// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.19.1

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
	CleanTailnetCoordinators(ctx context.Context) error
	DeleteAPIKeyByID(ctx context.Context, id string) error
	DeleteAPIKeysByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteApplicationConnectAPIKeysByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteCoordinator(ctx context.Context, id uuid.UUID) error
	DeleteGitSSHKey(ctx context.Context, userID uuid.UUID) error
	DeleteGroupByID(ctx context.Context, id uuid.UUID) error
	DeleteGroupMemberFromGroup(ctx context.Context, arg DeleteGroupMemberFromGroupParams) error
	DeleteGroupMembersByOrgAndUser(ctx context.Context, arg DeleteGroupMembersByOrgAndUserParams) error
	DeleteLicense(ctx context.Context, id int32) (int32, error)
	// If an agent hasn't connected in the last 7 days, we purge it's logs.
	// Logs can take up a lot of space, so it's important we clean up frequently.
	DeleteOldWorkspaceAgentStartupLogs(ctx context.Context) error
	DeleteOldWorkspaceAgentStats(ctx context.Context) error
	DeleteReplicasUpdatedBefore(ctx context.Context, updatedAt time.Time) error
	DeleteTailnetAgent(ctx context.Context, arg DeleteTailnetAgentParams) (DeleteTailnetAgentRow, error)
	DeleteTailnetClient(ctx context.Context, arg DeleteTailnetClientParams) (DeleteTailnetClientRow, error)
	GetAPIKeyByID(ctx context.Context, id string) (APIKey, error)
	// there is no unique constraint on empty token names
	GetAPIKeyByName(ctx context.Context, arg GetAPIKeyByNameParams) (APIKey, error)
	GetAPIKeysByLoginType(ctx context.Context, loginType LoginType) ([]APIKey, error)
	GetAPIKeysByUserID(ctx context.Context, arg GetAPIKeysByUserIDParams) ([]APIKey, error)
	GetAPIKeysLastUsedAfter(ctx context.Context, lastUsed time.Time) ([]APIKey, error)
	GetActiveUserCount(ctx context.Context) (int64, error)
	GetAppSecurityKey(ctx context.Context) (string, error)
	// GetAuditLogsBefore retrieves `row_limit` number of audit logs before the provided
	// ID.
	GetAuditLogsOffset(ctx context.Context, arg GetAuditLogsOffsetParams) ([]GetAuditLogsOffsetRow, error)
	// This function returns roles for authorization purposes. Implied member roles
	// are included.
	GetAuthorizationUserRoles(ctx context.Context, userID uuid.UUID) (GetAuthorizationUserRolesRow, error)
	GetDERPMeshKey(ctx context.Context) (string, error)
	GetDefaultProxyConfig(ctx context.Context) (GetDefaultProxyConfigRow, error)
	GetDeploymentDAUs(ctx context.Context, tzOffset int32) ([]GetDeploymentDAUsRow, error)
	GetDeploymentID(ctx context.Context) (string, error)
	GetDeploymentWorkspaceAgentStats(ctx context.Context, createdAt time.Time) (GetDeploymentWorkspaceAgentStatsRow, error)
	GetDeploymentWorkspaceStats(ctx context.Context) (GetDeploymentWorkspaceStatsRow, error)
	GetFileByHashAndCreator(ctx context.Context, arg GetFileByHashAndCreatorParams) (File, error)
	GetFileByID(ctx context.Context, id uuid.UUID) (File, error)
	// Get all templates that use a file.
	GetFileTemplates(ctx context.Context, fileID uuid.UUID) ([]GetFileTemplatesRow, error)
	GetGitAuthLink(ctx context.Context, arg GetGitAuthLinkParams) (GitAuthLink, error)
	GetGitSSHKey(ctx context.Context, userID uuid.UUID) (GitSSHKey, error)
	GetGroupByID(ctx context.Context, id uuid.UUID) (Group, error)
	GetGroupByOrgAndName(ctx context.Context, arg GetGroupByOrgAndNameParams) (Group, error)
	GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]User, error)
	GetGroupsByOrganizationID(ctx context.Context, organizationID uuid.UUID) ([]Group, error)
	GetHungProvisionerJobs(ctx context.Context, updatedAt time.Time) ([]ProvisionerJob, error)
	GetLastUpdateCheck(ctx context.Context) (string, error)
	GetLatestWorkspaceBuildByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) (WorkspaceBuild, error)
	GetLatestWorkspaceBuilds(ctx context.Context) ([]WorkspaceBuild, error)
	GetLatestWorkspaceBuildsByWorkspaceIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceBuild, error)
	GetLicenseByID(ctx context.Context, id int32) (License, error)
	GetLicenses(ctx context.Context) ([]License, error)
	GetLogoURL(ctx context.Context) (string, error)
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
	GetTemplateAverageBuildTime(ctx context.Context, arg GetTemplateAverageBuildTimeParams) (GetTemplateAverageBuildTimeRow, error)
	GetTemplateByID(ctx context.Context, id uuid.UUID) (Template, error)
	GetTemplateByOrganizationAndName(ctx context.Context, arg GetTemplateByOrganizationAndNameParams) (Template, error)
	GetTemplateDAUs(ctx context.Context, arg GetTemplateDAUsParams) ([]GetTemplateDAUsRow, error)
	// GetTemplateDailyInsights returns all daily intervals between start and end
	// time, if end time is a partial day, it will be included in the results and
	// that interval will be less than 24 hours. If there is no data for a selected
	// interval/template, it will be included in the results with 0 active users.
	GetTemplateDailyInsights(ctx context.Context, arg GetTemplateDailyInsightsParams) ([]GetTemplateDailyInsightsRow, error)
	// GetTemplateInsights has a granularity of 5 minutes where if a session/app was
	// in use, we will add 5 minutes to the total usage for that session (per user).
	GetTemplateInsights(ctx context.Context, arg GetTemplateInsightsParams) (GetTemplateInsightsRow, error)
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
	// This will never return deleted users.
	GetUsers(ctx context.Context, arg GetUsersParams) ([]GetUsersRow, error)
	// This shouldn't check for deleted, because it's frequently used
	// to look up references to actions. eg. a user could build a workspace
	// for another user, then be deleted... we still want them to appear!
	GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]User, error)
	GetWorkspaceAgentByAuthToken(ctx context.Context, authToken uuid.UUID) (WorkspaceAgent, error)
	GetWorkspaceAgentByID(ctx context.Context, id uuid.UUID) (WorkspaceAgent, error)
	GetWorkspaceAgentByInstanceID(ctx context.Context, authInstanceID string) (WorkspaceAgent, error)
	GetWorkspaceAgentLifecycleStateByID(ctx context.Context, id uuid.UUID) (GetWorkspaceAgentLifecycleStateByIDRow, error)
	GetWorkspaceAgentMetadata(ctx context.Context, workspaceAgentID uuid.UUID) ([]WorkspaceAgentMetadatum, error)
	GetWorkspaceAgentStartupLogsAfter(ctx context.Context, arg GetWorkspaceAgentStartupLogsAfterParams) ([]WorkspaceAgentStartupLog, error)
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
	GetWorkspaceByAgentID(ctx context.Context, agentID uuid.UUID) (Workspace, error)
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
	GetWorkspaces(ctx context.Context, arg GetWorkspacesParams) ([]GetWorkspacesRow, error)
	GetWorkspacesEligibleForTransition(ctx context.Context, now time.Time) ([]Workspace, error)
	InsertAPIKey(ctx context.Context, arg InsertAPIKeyParams) (APIKey, error)
	// We use the organization_id as the id
	// for simplicity since all users is
	// every member of the org.
	InsertAllUsersGroup(ctx context.Context, organizationID uuid.UUID) (Group, error)
	InsertAuditLog(ctx context.Context, arg InsertAuditLogParams) (AuditLog, error)
	InsertDERPMeshKey(ctx context.Context, value string) error
	InsertDeploymentID(ctx context.Context, value string) error
	InsertFile(ctx context.Context, arg InsertFileParams) (File, error)
	InsertGitAuthLink(ctx context.Context, arg InsertGitAuthLinkParams) (GitAuthLink, error)
	InsertGitSSHKey(ctx context.Context, arg InsertGitSSHKeyParams) (GitSSHKey, error)
	InsertGroup(ctx context.Context, arg InsertGroupParams) (Group, error)
	InsertGroupMember(ctx context.Context, arg InsertGroupMemberParams) error
	InsertLicense(ctx context.Context, arg InsertLicenseParams) (License, error)
	InsertOrganization(ctx context.Context, arg InsertOrganizationParams) (Organization, error)
	InsertOrganizationMember(ctx context.Context, arg InsertOrganizationMemberParams) (OrganizationMember, error)
	InsertProvisionerDaemon(ctx context.Context, arg InsertProvisionerDaemonParams) (ProvisionerDaemon, error)
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
	InsertWorkspaceAgentMetadata(ctx context.Context, arg InsertWorkspaceAgentMetadataParams) error
	InsertWorkspaceAgentStartupLogs(ctx context.Context, arg InsertWorkspaceAgentStartupLogsParams) ([]WorkspaceAgentStartupLog, error)
	InsertWorkspaceAgentStat(ctx context.Context, arg InsertWorkspaceAgentStatParams) (WorkspaceAgentStat, error)
	InsertWorkspaceApp(ctx context.Context, arg InsertWorkspaceAppParams) (WorkspaceApp, error)
	InsertWorkspaceBuild(ctx context.Context, arg InsertWorkspaceBuildParams) error
	InsertWorkspaceBuildParameters(ctx context.Context, arg InsertWorkspaceBuildParametersParams) error
	InsertWorkspaceProxy(ctx context.Context, arg InsertWorkspaceProxyParams) (WorkspaceProxy, error)
	InsertWorkspaceResource(ctx context.Context, arg InsertWorkspaceResourceParams) (WorkspaceResource, error)
	InsertWorkspaceResourceMetadata(ctx context.Context, arg InsertWorkspaceResourceMetadataParams) ([]WorkspaceResourceMetadatum, error)
	RegisterWorkspaceProxy(ctx context.Context, arg RegisterWorkspaceProxyParams) (WorkspaceProxy, error)
	// Non blocking lock. Returns true if the lock was acquired, false otherwise.
	//
	// This must be called from within a transaction. The lock will be automatically
	// released when the transaction ends.
	TryAcquireLock(ctx context.Context, pgTryAdvisoryXactLock int64) (bool, error)
	UpdateAPIKeyByID(ctx context.Context, arg UpdateAPIKeyByIDParams) error
	UpdateGitAuthLink(ctx context.Context, arg UpdateGitAuthLinkParams) (GitAuthLink, error)
	UpdateGitSSHKey(ctx context.Context, arg UpdateGitSSHKeyParams) (GitSSHKey, error)
	UpdateGroupByID(ctx context.Context, arg UpdateGroupByIDParams) (Group, error)
	UpdateMemberRoles(ctx context.Context, arg UpdateMemberRolesParams) (OrganizationMember, error)
	UpdateProvisionerJobByID(ctx context.Context, arg UpdateProvisionerJobByIDParams) error
	UpdateProvisionerJobWithCancelByID(ctx context.Context, arg UpdateProvisionerJobWithCancelByIDParams) error
	UpdateProvisionerJobWithCompleteByID(ctx context.Context, arg UpdateProvisionerJobWithCompleteByIDParams) error
	UpdateReplica(ctx context.Context, arg UpdateReplicaParams) (Replica, error)
	UpdateTemplateACLByID(ctx context.Context, arg UpdateTemplateACLByIDParams) error
	UpdateTemplateActiveVersionByID(ctx context.Context, arg UpdateTemplateActiveVersionByIDParams) error
	UpdateTemplateDeletedByID(ctx context.Context, arg UpdateTemplateDeletedByIDParams) error
	UpdateTemplateMetaByID(ctx context.Context, arg UpdateTemplateMetaByIDParams) error
	UpdateTemplateScheduleByID(ctx context.Context, arg UpdateTemplateScheduleByIDParams) error
	UpdateTemplateVersionByID(ctx context.Context, arg UpdateTemplateVersionByIDParams) error
	UpdateTemplateVersionDescriptionByJobID(ctx context.Context, arg UpdateTemplateVersionDescriptionByJobIDParams) error
	UpdateTemplateVersionGitAuthProvidersByJobID(ctx context.Context, arg UpdateTemplateVersionGitAuthProvidersByJobIDParams) error
	UpdateUserDeletedByID(ctx context.Context, arg UpdateUserDeletedByIDParams) error
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
	UpdateWorkspaceAgentMetadata(ctx context.Context, arg UpdateWorkspaceAgentMetadataParams) error
	UpdateWorkspaceAgentStartupByID(ctx context.Context, arg UpdateWorkspaceAgentStartupByIDParams) error
	UpdateWorkspaceAgentStartupLogOverflowByID(ctx context.Context, arg UpdateWorkspaceAgentStartupLogOverflowByIDParams) error
	UpdateWorkspaceAppHealthByID(ctx context.Context, arg UpdateWorkspaceAppHealthByIDParams) error
	UpdateWorkspaceAutostart(ctx context.Context, arg UpdateWorkspaceAutostartParams) error
	UpdateWorkspaceBuildByID(ctx context.Context, arg UpdateWorkspaceBuildByIDParams) error
	UpdateWorkspaceBuildCostByID(ctx context.Context, arg UpdateWorkspaceBuildCostByIDParams) error
	UpdateWorkspaceDeletedByID(ctx context.Context, arg UpdateWorkspaceDeletedByIDParams) error
	UpdateWorkspaceLastUsedAt(ctx context.Context, arg UpdateWorkspaceLastUsedAtParams) error
	UpdateWorkspaceLockedDeletingAt(ctx context.Context, arg UpdateWorkspaceLockedDeletingAtParams) error
	// This allows editing the properties of a workspace proxy.
	UpdateWorkspaceProxy(ctx context.Context, arg UpdateWorkspaceProxyParams) (WorkspaceProxy, error)
	UpdateWorkspaceProxyDeleted(ctx context.Context, arg UpdateWorkspaceProxyDeletedParams) error
	UpdateWorkspaceTTL(ctx context.Context, arg UpdateWorkspaceTTLParams) error
	UpdateWorkspacesDeletingAtByTemplateID(ctx context.Context, arg UpdateWorkspacesDeletingAtByTemplateIDParams) error
	UpsertAppSecurityKey(ctx context.Context, value string) error
	// The default proxy is implied and not actually stored in the database.
	// So we need to store it's configuration here for display purposes.
	// The functional values are immutable and controlled implicitly.
	UpsertDefaultProxy(ctx context.Context, arg UpsertDefaultProxyParams) error
	UpsertLastUpdateCheck(ctx context.Context, value string) error
	UpsertLogoURL(ctx context.Context, value string) error
	UpsertOAuthSigningKey(ctx context.Context, value string) error
	UpsertServiceBanner(ctx context.Context, value string) error
	UpsertTailnetAgent(ctx context.Context, arg UpsertTailnetAgentParams) (TailnetAgent, error)
	UpsertTailnetClient(ctx context.Context, arg UpsertTailnetClientParams) (TailnetClient, error)
	UpsertTailnetCoordinator(ctx context.Context, id uuid.UUID) (TailnetCoordinator, error)
}

var _ sqlcQuerier = (*sqlQuerier)(nil)
