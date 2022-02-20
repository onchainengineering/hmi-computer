// Code generated by sqlc. DO NOT EDIT.

package database

import (
	"context"

	"github.com/google/uuid"
)

type querier interface {
	AcquireProvisionerJob(ctx context.Context, arg AcquireProvisionerJobParams) (ProvisionerJob, error)
	GetAPIKeyByID(ctx context.Context, id string) (APIKey, error)
	GetFileByHash(ctx context.Context, hash string) (File, error)
	GetOrganizationByID(ctx context.Context, id string) (Organization, error)
	GetOrganizationByName(ctx context.Context, name string) (Organization, error)
	GetOrganizationMemberByUserID(ctx context.Context, arg GetOrganizationMemberByUserIDParams) (OrganizationMember, error)
	GetOrganizationsByUserID(ctx context.Context, userID string) ([]Organization, error)
	GetParameterSchemasByJobID(ctx context.Context, jobID uuid.UUID) ([]ParameterSchema, error)
	GetParameterValuesByScope(ctx context.Context, arg GetParameterValuesByScopeParams) ([]ParameterValue, error)
	GetProjectByID(ctx context.Context, id uuid.UUID) (Project, error)
	GetProjectByOrganizationAndName(ctx context.Context, arg GetProjectByOrganizationAndNameParams) (Project, error)
	GetProjectImportJobResourcesByJobID(ctx context.Context, jobID uuid.UUID) ([]ProjectImportJobResource, error)
	GetProjectVersionByID(ctx context.Context, id uuid.UUID) (ProjectVersion, error)
	GetProjectVersionByProjectIDAndName(ctx context.Context, arg GetProjectVersionByProjectIDAndNameParams) (ProjectVersion, error)
	GetProjectVersionsByProjectID(ctx context.Context, projectID uuid.UUID) ([]ProjectVersion, error)
	GetProjectsByOrganizationIDs(ctx context.Context, ids []string) ([]Project, error)
	GetProvisionerDaemonByID(ctx context.Context, id uuid.UUID) (ProvisionerDaemon, error)
	GetProvisionerDaemons(ctx context.Context) ([]ProvisionerDaemon, error)
	GetProvisionerJobByID(ctx context.Context, id uuid.UUID) (ProvisionerJob, error)
	GetProvisionerLogsByIDBetween(ctx context.Context, arg GetProvisionerLogsByIDBetweenParams) ([]ProvisionerJobLog, error)
	GetUserByEmailOrUsername(ctx context.Context, arg GetUserByEmailOrUsernameParams) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	GetUserCount(ctx context.Context) (int64, error)
	GetWorkspaceAgentsByResourceIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceAgent, error)
	GetWorkspaceByID(ctx context.Context, id uuid.UUID) (Workspace, error)
	GetWorkspaceByUserIDAndName(ctx context.Context, arg GetWorkspaceByUserIDAndNameParams) (Workspace, error)
	GetWorkspaceHistoryByID(ctx context.Context, id uuid.UUID) (WorkspaceHistory, error)
	GetWorkspaceHistoryByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceHistory, error)
	GetWorkspaceHistoryByWorkspaceIDAndName(ctx context.Context, arg GetWorkspaceHistoryByWorkspaceIDAndNameParams) (WorkspaceHistory, error)
	GetWorkspaceHistoryByWorkspaceIDWithoutAfter(ctx context.Context, workspaceID uuid.UUID) (WorkspaceHistory, error)
	GetWorkspaceResourceByInstanceID(ctx context.Context, instanceID string) (WorkspaceResource, error)
	GetWorkspaceResourcesByHistoryID(ctx context.Context, workspaceHistoryID uuid.UUID) ([]WorkspaceResource, error)
	GetWorkspacesByProjectAndUserID(ctx context.Context, arg GetWorkspacesByProjectAndUserIDParams) ([]Workspace, error)
	GetWorkspacesByUserID(ctx context.Context, ownerID string) ([]Workspace, error)
	InsertAPIKey(ctx context.Context, arg InsertAPIKeyParams) (APIKey, error)
	InsertFile(ctx context.Context, arg InsertFileParams) (File, error)
	InsertOrganization(ctx context.Context, arg InsertOrganizationParams) (Organization, error)
	InsertOrganizationMember(ctx context.Context, arg InsertOrganizationMemberParams) (OrganizationMember, error)
	InsertParameterSchema(ctx context.Context, arg InsertParameterSchemaParams) (ParameterSchema, error)
	InsertParameterValue(ctx context.Context, arg InsertParameterValueParams) (ParameterValue, error)
	InsertProject(ctx context.Context, arg InsertProjectParams) (Project, error)
	InsertProjectImportJobResource(ctx context.Context, arg InsertProjectImportJobResourceParams) (ProjectImportJobResource, error)
	InsertProjectVersion(ctx context.Context, arg InsertProjectVersionParams) (ProjectVersion, error)
	InsertProvisionerDaemon(ctx context.Context, arg InsertProvisionerDaemonParams) (ProvisionerDaemon, error)
	InsertProvisionerJob(ctx context.Context, arg InsertProvisionerJobParams) (ProvisionerJob, error)
	InsertProvisionerJobLogs(ctx context.Context, arg InsertProvisionerJobLogsParams) ([]ProvisionerJobLog, error)
	InsertUser(ctx context.Context, arg InsertUserParams) (User, error)
	InsertWorkspace(ctx context.Context, arg InsertWorkspaceParams) (Workspace, error)
	InsertWorkspaceAgent(ctx context.Context, arg InsertWorkspaceAgentParams) (WorkspaceAgent, error)
	InsertWorkspaceHistory(ctx context.Context, arg InsertWorkspaceHistoryParams) (WorkspaceHistory, error)
	InsertWorkspaceResource(ctx context.Context, arg InsertWorkspaceResourceParams) (WorkspaceResource, error)
	UpdateAPIKeyByID(ctx context.Context, arg UpdateAPIKeyByIDParams) error
	UpdateProvisionerDaemonByID(ctx context.Context, arg UpdateProvisionerDaemonByIDParams) error
	UpdateProvisionerJobByID(ctx context.Context, arg UpdateProvisionerJobByIDParams) error
	UpdateProvisionerJobWithCompleteByID(ctx context.Context, arg UpdateProvisionerJobWithCompleteByIDParams) error
	UpdateWorkspaceHistoryByID(ctx context.Context, arg UpdateWorkspaceHistoryByIDParams) error
}

var _ querier = (*sqlQuerier)(nil)
