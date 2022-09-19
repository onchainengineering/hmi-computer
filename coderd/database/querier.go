// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.13.0

package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type querier interface {
	// Acquires the lock for a single job that isn't started, completed,
	// canceled, and that matches an array of provisioner types.
	//
	// SKIP LOCKED is used to jump over locked rows. This prevents
	// multiple provisioners from acquiring the same jobs. See:
	// https://www.postgresql.org/docs/9.5/sql-select.html#SQL-FOR-UPDATE-SHARE
	AcquireProvisionerJob(ctx context.Context, arg AcquireProvisionerJobParams) (ProvisionerJob, error)
	DeleteAPIKeyByID(ctx context.Context, id string) error
	DeleteGitSSHKey(ctx context.Context, userID uuid.UUID) error
	DeleteLicense(ctx context.Context, id int32) (int32, error)
	DeleteOldAgentStats(ctx context.Context) error
	DeleteParameterValueByID(ctx context.Context, id uuid.UUID) error
	GetAPIKeyByID(ctx context.Context, id string) (APIKey, error)
	GetAPIKeysLastUsedAfter(ctx context.Context, lastUsed time.Time) ([]APIKey, error)
	GetActiveUserCount(ctx context.Context) (int64, error)
	GetAuditLogCount(ctx context.Context, arg GetAuditLogCountParams) (int64, error)
	// GetAuditLogsBefore retrieves `row_limit` number of audit logs before the provided
	// ID.
	GetAuditLogsOffset(ctx context.Context, arg GetAuditLogsOffsetParams) ([]GetAuditLogsOffsetRow, error)
	// This function returns roles for authorization purposes. Implied member roles
	// are included.
	GetAuthorizationUserRoles(ctx context.Context, userID uuid.UUID) (GetAuthorizationUserRolesRow, error)
	GetDeploymentID(ctx context.Context) (string, error)
	GetFileByHash(ctx context.Context, hash string) (File, error)
	GetGitSSHKey(ctx context.Context, userID uuid.UUID) (GitSSHKey, error)
	GetLatestAgentStat(ctx context.Context, agentID uuid.UUID) (AgentStat, error)
	GetLatestWorkspaceBuildByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) (WorkspaceBuild, error)
	GetLatestWorkspaceBuilds(ctx context.Context) ([]WorkspaceBuild, error)
	GetLatestWorkspaceBuildsByWorkspaceIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceBuild, error)
	GetLicenses(ctx context.Context) ([]License, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (Organization, error)
	GetOrganizationByName(ctx context.Context, name string) (Organization, error)
	GetOrganizationIDsByMemberIDs(ctx context.Context, ids []uuid.UUID) ([]GetOrganizationIDsByMemberIDsRow, error)
	GetOrganizationMemberByUserID(ctx context.Context, arg GetOrganizationMemberByUserIDParams) (OrganizationMember, error)
	GetOrganizationMembershipsByUserID(ctx context.Context, userID uuid.UUID) ([]OrganizationMember, error)
	GetOrganizations(ctx context.Context) ([]Organization, error)
	GetOrganizationsByUserID(ctx context.Context, userID uuid.UUID) ([]Organization, error)
	GetParameterSchemasByJobID(ctx context.Context, jobID uuid.UUID) ([]ParameterSchema, error)
	GetParameterSchemasCreatedAfter(ctx context.Context, createdAt time.Time) ([]ParameterSchema, error)
	GetParameterValueByScopeAndName(ctx context.Context, arg GetParameterValueByScopeAndNameParams) (ParameterValue, error)
	GetProvisionerDaemonByID(ctx context.Context, id uuid.UUID) (ProvisionerDaemon, error)
	GetProvisionerDaemons(ctx context.Context) ([]ProvisionerDaemon, error)
	GetProvisionerJobByID(ctx context.Context, id uuid.UUID) (ProvisionerJob, error)
	GetProvisionerJobsByIDs(ctx context.Context, ids []uuid.UUID) ([]ProvisionerJob, error)
	GetProvisionerJobsCreatedAfter(ctx context.Context, createdAt time.Time) ([]ProvisionerJob, error)
	GetProvisionerLogsByIDBetween(ctx context.Context, arg GetProvisionerLogsByIDBetweenParams) ([]ProvisionerJobLog, error)
	GetTemplateByID(ctx context.Context, id uuid.UUID) (Template, error)
	GetTemplateByOrganizationAndName(ctx context.Context, arg GetTemplateByOrganizationAndNameParams) (Template, error)
	GetTemplateDAUs(ctx context.Context, templateID uuid.UUID) ([]GetTemplateDAUsRow, error)
	GetTemplateVersionByID(ctx context.Context, id uuid.UUID) (TemplateVersion, error)
	GetTemplateVersionByJobID(ctx context.Context, jobID uuid.UUID) (TemplateVersion, error)
	GetTemplateVersionByTemplateIDAndName(ctx context.Context, arg GetTemplateVersionByTemplateIDAndNameParams) (TemplateVersion, error)
	GetTemplateVersionsByTemplateID(ctx context.Context, arg GetTemplateVersionsByTemplateIDParams) ([]TemplateVersion, error)
	GetTemplateVersionsCreatedAfter(ctx context.Context, createdAt time.Time) ([]TemplateVersion, error)
	GetTemplates(ctx context.Context) ([]Template, error)
	GetTemplatesWithFilter(ctx context.Context, arg GetTemplatesWithFilterParams) ([]Template, error)
	GetUnexpiredLicenses(ctx context.Context) ([]License, error)
	GetUserByEmailOrUsername(ctx context.Context, arg GetUserByEmailOrUsernameParams) (User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (User, error)
	GetUserCount(ctx context.Context) (int64, error)
	GetUserLinkByLinkedID(ctx context.Context, linkedID string) (UserLink, error)
	GetUserLinkByUserIDLoginType(ctx context.Context, arg GetUserLinkByUserIDLoginTypeParams) (UserLink, error)
	GetUsers(ctx context.Context, arg GetUsersParams) ([]User, error)
	GetUsersByIDs(ctx context.Context, arg GetUsersByIDsParams) ([]User, error)
	GetWorkspaceAgentByAuthToken(ctx context.Context, authToken uuid.UUID) (WorkspaceAgent, error)
	GetWorkspaceAgentByID(ctx context.Context, id uuid.UUID) (WorkspaceAgent, error)
	GetWorkspaceAgentByInstanceID(ctx context.Context, authInstanceID string) (WorkspaceAgent, error)
	GetWorkspaceAgentsByResourceIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceAgent, error)
	GetWorkspaceAgentsCreatedAfter(ctx context.Context, createdAt time.Time) ([]WorkspaceAgent, error)
	GetWorkspaceAppByAgentIDAndName(ctx context.Context, arg GetWorkspaceAppByAgentIDAndNameParams) (WorkspaceApp, error)
	GetWorkspaceAppsByAgentID(ctx context.Context, agentID uuid.UUID) ([]WorkspaceApp, error)
	GetWorkspaceAppsByAgentIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceApp, error)
	GetWorkspaceAppsCreatedAfter(ctx context.Context, createdAt time.Time) ([]WorkspaceApp, error)
	GetWorkspaceBuildByID(ctx context.Context, id uuid.UUID) (WorkspaceBuild, error)
	GetWorkspaceBuildByJobID(ctx context.Context, jobID uuid.UUID) (WorkspaceBuild, error)
	GetWorkspaceBuildByWorkspaceID(ctx context.Context, arg GetWorkspaceBuildByWorkspaceIDParams) ([]WorkspaceBuild, error)
	GetWorkspaceBuildByWorkspaceIDAndBuildNumber(ctx context.Context, arg GetWorkspaceBuildByWorkspaceIDAndBuildNumberParams) (WorkspaceBuild, error)
	GetWorkspaceBuildsCreatedAfter(ctx context.Context, createdAt time.Time) ([]WorkspaceBuild, error)
	GetWorkspaceByID(ctx context.Context, id uuid.UUID) (Workspace, error)
	GetWorkspaceByOwnerIDAndName(ctx context.Context, arg GetWorkspaceByOwnerIDAndNameParams) (Workspace, error)
	GetWorkspaceOwnerCountsByTemplateIDs(ctx context.Context, ids []uuid.UUID) ([]GetWorkspaceOwnerCountsByTemplateIDsRow, error)
	GetWorkspaceResourceByID(ctx context.Context, id uuid.UUID) (WorkspaceResource, error)
	GetWorkspaceResourceMetadataByResourceID(ctx context.Context, workspaceResourceID uuid.UUID) ([]WorkspaceResourceMetadatum, error)
	GetWorkspaceResourceMetadataByResourceIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceResourceMetadatum, error)
	GetWorkspaceResourceMetadataCreatedAfter(ctx context.Context, createdAt time.Time) ([]WorkspaceResourceMetadatum, error)
	GetWorkspaceResourcesByJobID(ctx context.Context, jobID uuid.UUID) ([]WorkspaceResource, error)
	GetWorkspaceResourcesByJobIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceResource, error)
	GetWorkspaceResourcesCreatedAfter(ctx context.Context, createdAt time.Time) ([]WorkspaceResource, error)
	GetWorkspaces(ctx context.Context, arg GetWorkspacesParams) ([]Workspace, error)
	InsertAPIKey(ctx context.Context, arg InsertAPIKeyParams) (APIKey, error)
	InsertAgentStat(ctx context.Context, arg InsertAgentStatParams) (AgentStat, error)
	InsertAuditLog(ctx context.Context, arg InsertAuditLogParams) (AuditLog, error)
	InsertDeploymentID(ctx context.Context, value string) error
	InsertFile(ctx context.Context, arg InsertFileParams) (File, error)
	InsertGitSSHKey(ctx context.Context, arg InsertGitSSHKeyParams) (GitSSHKey, error)
	InsertLicense(ctx context.Context, arg InsertLicenseParams) (License, error)
	InsertOrganization(ctx context.Context, arg InsertOrganizationParams) (Organization, error)
	InsertOrganizationMember(ctx context.Context, arg InsertOrganizationMemberParams) (OrganizationMember, error)
	InsertParameterSchema(ctx context.Context, arg InsertParameterSchemaParams) (ParameterSchema, error)
	InsertParameterValue(ctx context.Context, arg InsertParameterValueParams) (ParameterValue, error)
	InsertProvisionerDaemon(ctx context.Context, arg InsertProvisionerDaemonParams) (ProvisionerDaemon, error)
	InsertProvisionerJob(ctx context.Context, arg InsertProvisionerJobParams) (ProvisionerJob, error)
	InsertProvisionerJobLogs(ctx context.Context, arg InsertProvisionerJobLogsParams) ([]ProvisionerJobLog, error)
	InsertTemplate(ctx context.Context, arg InsertTemplateParams) (Template, error)
	InsertTemplateVersion(ctx context.Context, arg InsertTemplateVersionParams) (TemplateVersion, error)
	InsertUser(ctx context.Context, arg InsertUserParams) (User, error)
	InsertUserLink(ctx context.Context, arg InsertUserLinkParams) (UserLink, error)
	InsertWorkspace(ctx context.Context, arg InsertWorkspaceParams) (Workspace, error)
	InsertWorkspaceAgent(ctx context.Context, arg InsertWorkspaceAgentParams) (WorkspaceAgent, error)
	InsertWorkspaceApp(ctx context.Context, arg InsertWorkspaceAppParams) (WorkspaceApp, error)
	InsertWorkspaceBuild(ctx context.Context, arg InsertWorkspaceBuildParams) (WorkspaceBuild, error)
	InsertWorkspaceResource(ctx context.Context, arg InsertWorkspaceResourceParams) (WorkspaceResource, error)
	InsertWorkspaceResourceMetadata(ctx context.Context, arg InsertWorkspaceResourceMetadataParams) (WorkspaceResourceMetadatum, error)
	ParameterValue(ctx context.Context, id uuid.UUID) (ParameterValue, error)
	ParameterValues(ctx context.Context, arg ParameterValuesParams) ([]ParameterValue, error)
	UpdateAPIKeyByID(ctx context.Context, arg UpdateAPIKeyByIDParams) error
	UpdateGitSSHKey(ctx context.Context, arg UpdateGitSSHKeyParams) error
	UpdateMemberRoles(ctx context.Context, arg UpdateMemberRolesParams) (OrganizationMember, error)
	UpdateProvisionerDaemonByID(ctx context.Context, arg UpdateProvisionerDaemonByIDParams) error
	UpdateProvisionerJobByID(ctx context.Context, arg UpdateProvisionerJobByIDParams) error
	UpdateProvisionerJobWithCancelByID(ctx context.Context, arg UpdateProvisionerJobWithCancelByIDParams) error
	UpdateProvisionerJobWithCompleteByID(ctx context.Context, arg UpdateProvisionerJobWithCompleteByIDParams) error
	UpdateTemplateActiveVersionByID(ctx context.Context, arg UpdateTemplateActiveVersionByIDParams) error
	UpdateTemplateDeletedByID(ctx context.Context, arg UpdateTemplateDeletedByIDParams) error
	UpdateTemplateMetaByID(ctx context.Context, arg UpdateTemplateMetaByIDParams) (Template, error)
	UpdateTemplateVersionByID(ctx context.Context, arg UpdateTemplateVersionByIDParams) error
	UpdateTemplateVersionDescriptionByJobID(ctx context.Context, arg UpdateTemplateVersionDescriptionByJobIDParams) error
	UpdateUserDeletedByID(ctx context.Context, arg UpdateUserDeletedByIDParams) error
	UpdateUserHashedPassword(ctx context.Context, arg UpdateUserHashedPasswordParams) error
	UpdateUserLink(ctx context.Context, arg UpdateUserLinkParams) (UserLink, error)
	UpdateUserLinkedID(ctx context.Context, arg UpdateUserLinkedIDParams) (UserLink, error)
	UpdateUserProfile(ctx context.Context, arg UpdateUserProfileParams) (User, error)
	UpdateUserRoles(ctx context.Context, arg UpdateUserRolesParams) (User, error)
	UpdateUserStatus(ctx context.Context, arg UpdateUserStatusParams) (User, error)
	UpdateWorkspace(ctx context.Context, arg UpdateWorkspaceParams) (Workspace, error)
	UpdateWorkspaceAgentConnectionByID(ctx context.Context, arg UpdateWorkspaceAgentConnectionByIDParams) error
	UpdateWorkspaceAgentVersionByID(ctx context.Context, arg UpdateWorkspaceAgentVersionByIDParams) error
	UpdateWorkspaceAppHealthByID(ctx context.Context, arg UpdateWorkspaceAppHealthByIDParams) error
	UpdateWorkspaceAutostart(ctx context.Context, arg UpdateWorkspaceAutostartParams) error
	UpdateWorkspaceBuildByID(ctx context.Context, arg UpdateWorkspaceBuildByIDParams) error
	UpdateWorkspaceDeletedByID(ctx context.Context, arg UpdateWorkspaceDeletedByIDParams) error
	UpdateWorkspaceLastUsedAt(ctx context.Context, arg UpdateWorkspaceLastUsedAtParams) error
	UpdateWorkspaceTTL(ctx context.Context, arg UpdateWorkspaceTTLParams) error
}

var _ querier = (*sqlQuerier)(nil)
