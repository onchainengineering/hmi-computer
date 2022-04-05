// Code generated by sqlc. DO NOT EDIT.

package database

import (
	"context"

	"github.com/google/uuid"
)

type querier interface {
	AcquireProvisionerJob(ctx context.Context, arg AcquireProvisionerJobParams) (ProvisionerJob, error)
	DeleteParameterValueByID(ctx context.Context, id uuid.UUID) error
	GetAPIKeyByID(ctx context.Context, id string) (APIKey, error)
	GetFileByHash(ctx context.Context, hash string) (File, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (Organization, error)
	GetOrganizationByName(ctx context.Context, name string) (Organization, error)
	GetOrganizationMemberByUserID(ctx context.Context, arg GetOrganizationMemberByUserIDParams) (OrganizationMember, error)
	GetOrganizationsByUserID(ctx context.Context, userID uuid.UUID) ([]Organization, error)
	GetParameterSchemasByJobID(ctx context.Context, jobID uuid.UUID) ([]ParameterSchema, error)
	GetParameterValueByScopeAndName(ctx context.Context, arg GetParameterValueByScopeAndNameParams) (ParameterValue, error)
	GetParameterValuesByScope(ctx context.Context, arg GetParameterValuesByScopeParams) ([]ParameterValue, error)
	GetProvisionerDaemonByID(ctx context.Context, id uuid.UUID) (ProvisionerDaemon, error)
	GetProvisionerDaemons(ctx context.Context) ([]ProvisionerDaemon, error)
	GetProvisionerJobByID(ctx context.Context, id uuid.UUID) (ProvisionerJob, error)
	GetProvisionerJobsByIDs(ctx context.Context, ids []uuid.UUID) ([]ProvisionerJob, error)
	GetProvisionerLogsByIDBetween(ctx context.Context, arg GetProvisionerLogsByIDBetweenParams) ([]ProvisionerJobLog, error)
	GetTemplateByID(ctx context.Context, id uuid.UUID) (Template, error)
	GetTemplateByOrganizationAndName(ctx context.Context, arg GetTemplateByOrganizationAndNameParams) (Template, error)
	GetTemplateVersionByID(ctx context.Context, id uuid.UUID) (TemplateVersion, error)
	GetTemplateVersionByJobID(ctx context.Context, jobID uuid.UUID) (TemplateVersion, error)
	GetTemplateVersionByTemplateIDAndName(ctx context.Context, arg GetTemplateVersionByTemplateIDAndNameParams) (TemplateVersion, error)
	GetTemplateVersionsByTemplateID(ctx context.Context, dollar_1 uuid.UUID) ([]TemplateVersion, error)
	GetTemplatesByIDs(ctx context.Context, ids []uuid.UUID) ([]Template, error)
	GetTemplatesByOrganization(ctx context.Context, arg GetTemplatesByOrganizationParams) ([]Template, error)
	GetUserByEmailOrUsername(ctx context.Context, arg GetUserByEmailOrUsernameParams) (User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (User, error)
	GetUserCount(ctx context.Context) (int64, error)
	GetWorkspaceAgentByAuthToken(ctx context.Context, authToken uuid.UUID) (WorkspaceAgent, error)
	GetWorkspaceAgentByInstanceID(ctx context.Context, authInstanceID string) (WorkspaceAgent, error)
	GetWorkspaceAgentByResourceID(ctx context.Context, resourceID uuid.UUID) (WorkspaceAgent, error)
	GetWorkspaceBuildByID(ctx context.Context, id uuid.UUID) (WorkspaceBuild, error)
	GetWorkspaceBuildByJobID(ctx context.Context, jobID uuid.UUID) (WorkspaceBuild, error)
	GetWorkspaceBuildByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceBuild, error)
	GetWorkspaceBuildByWorkspaceIDAndName(ctx context.Context, arg GetWorkspaceBuildByWorkspaceIDAndNameParams) (WorkspaceBuild, error)
	GetWorkspaceBuildByWorkspaceIDWithoutAfter(ctx context.Context, workspaceID uuid.UUID) (WorkspaceBuild, error)
	GetWorkspaceBuildsByWorkspaceIDsWithoutAfter(ctx context.Context, ids []uuid.UUID) ([]WorkspaceBuild, error)
	GetWorkspaceByID(ctx context.Context, id uuid.UUID) (Workspace, error)
	GetWorkspaceByUserIDAndName(ctx context.Context, arg GetWorkspaceByUserIDAndNameParams) (Workspace, error)
	GetWorkspaceOwnerCountsByTemplateIDs(ctx context.Context, ids []uuid.UUID) ([]GetWorkspaceOwnerCountsByTemplateIDsRow, error)
	GetWorkspaceResourceByID(ctx context.Context, id uuid.UUID) (WorkspaceResource, error)
	GetWorkspaceResourcesByJobID(ctx context.Context, jobID uuid.UUID) ([]WorkspaceResource, error)
	GetWorkspacesByTemplateID(ctx context.Context, arg GetWorkspacesByTemplateIDParams) ([]Workspace, error)
	GetWorkspacesByUserID(ctx context.Context, arg GetWorkspacesByUserIDParams) ([]Workspace, error)
	InsertAPIKey(ctx context.Context, arg InsertAPIKeyParams) (APIKey, error)
	InsertFile(ctx context.Context, arg InsertFileParams) (File, error)
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
	InsertWorkspace(ctx context.Context, arg InsertWorkspaceParams) (Workspace, error)
	InsertWorkspaceAgent(ctx context.Context, arg InsertWorkspaceAgentParams) (WorkspaceAgent, error)
	InsertWorkspaceBuild(ctx context.Context, arg InsertWorkspaceBuildParams) (WorkspaceBuild, error)
	InsertWorkspaceResource(ctx context.Context, arg InsertWorkspaceResourceParams) (WorkspaceResource, error)
	UpdateAPIKeyByID(ctx context.Context, arg UpdateAPIKeyByIDParams) error
	UpdateProvisionerDaemonByID(ctx context.Context, arg UpdateProvisionerDaemonByIDParams) error
	UpdateProvisionerJobByID(ctx context.Context, arg UpdateProvisionerJobByIDParams) error
	UpdateProvisionerJobWithCancelByID(ctx context.Context, arg UpdateProvisionerJobWithCancelByIDParams) error
	UpdateProvisionerJobWithCompleteByID(ctx context.Context, arg UpdateProvisionerJobWithCompleteByIDParams) error
	UpdateTemplateActiveVersionByID(ctx context.Context, arg UpdateTemplateActiveVersionByIDParams) error
	UpdateTemplateDeletedByID(ctx context.Context, arg UpdateTemplateDeletedByIDParams) error
	UpdateTemplateVersionByID(ctx context.Context, arg UpdateTemplateVersionByIDParams) error
	UpdateWorkspaceAgentConnectionByID(ctx context.Context, arg UpdateWorkspaceAgentConnectionByIDParams) error
	UpdateWorkspaceBuildByID(ctx context.Context, arg UpdateWorkspaceBuildByIDParams) error
	UpdateWorkspaceDeletedByID(ctx context.Context, arg UpdateWorkspaceDeletedByIDParams) error
}

var _ querier = (*sqlQuerier)(nil)
