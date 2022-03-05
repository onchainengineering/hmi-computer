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
	GetProjectVersionByID(ctx context.Context, id uuid.UUID) (ProjectVersion, error)
	GetProjectVersionByProjectIDAndName(ctx context.Context, arg GetProjectVersionByProjectIDAndNameParams) (ProjectVersion, error)
	GetProjectVersionsByProjectID(ctx context.Context, projectID uuid.NullUUID) ([]ProjectVersion, error)
	GetProjectsByOrganizationIDs(ctx context.Context, ids []string) ([]Project, error)
	GetProvisionerDaemonByID(ctx context.Context, id uuid.UUID) (ProvisionerDaemon, error)
	GetProvisionerDaemons(ctx context.Context) ([]ProvisionerDaemon, error)
	GetProvisionerJobAgentByAuthToken(ctx context.Context, authToken uuid.UUID) (ProvisionerJobAgent, error)
	GetProvisionerJobAgentByInstanceID(ctx context.Context, authInstanceID string) (ProvisionerJobAgent, error)
	GetProvisionerJobAgentByResourceID(ctx context.Context, resourceID uuid.UUID) (ProvisionerJobAgent, error)
	GetProvisionerJobByID(ctx context.Context, id uuid.UUID) (ProvisionerJob, error)
	GetProvisionerJobResourceByID(ctx context.Context, id uuid.UUID) (ProvisionerJobResource, error)
	GetProvisionerJobResourcesByJobID(ctx context.Context, jobID uuid.UUID) ([]ProvisionerJobResource, error)
	GetProvisionerLogsByIDBetween(ctx context.Context, arg GetProvisionerLogsByIDBetweenParams) ([]ProvisionerJobLog, error)
	GetUserByEmailOrUsername(ctx context.Context, arg GetUserByEmailOrUsernameParams) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	GetUserCount(ctx context.Context) (int64, error)
	GetWorkspaceBuildByID(ctx context.Context, id uuid.UUID) (WorkspaceBuild, error)
	GetWorkspaceBuildByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceBuild, error)
	GetWorkspaceBuildByWorkspaceIDAndName(ctx context.Context, arg GetWorkspaceBuildByWorkspaceIDAndNameParams) (WorkspaceBuild, error)
	GetWorkspaceBuildByWorkspaceIDWithoutAfter(ctx context.Context, workspaceID uuid.UUID) (WorkspaceBuild, error)
	GetWorkspaceByID(ctx context.Context, id uuid.UUID) (Workspace, error)
	GetWorkspaceByUserIDAndName(ctx context.Context, arg GetWorkspaceByUserIDAndNameParams) (Workspace, error)
	GetWorkspaceOwnerCountsByProjectIDs(ctx context.Context, ids []uuid.UUID) ([]GetWorkspaceOwnerCountsByProjectIDsRow, error)
	GetWorkspacesByProjectAndUserID(ctx context.Context, arg GetWorkspacesByProjectAndUserIDParams) ([]Workspace, error)
	GetWorkspacesByUserID(ctx context.Context, ownerID string) ([]Workspace, error)
	InsertAPIKey(ctx context.Context, arg InsertAPIKeyParams) (APIKey, error)
	InsertFile(ctx context.Context, arg InsertFileParams) (File, error)
	InsertOrganization(ctx context.Context, arg InsertOrganizationParams) (Organization, error)
	InsertOrganizationMember(ctx context.Context, arg InsertOrganizationMemberParams) (OrganizationMember, error)
	InsertParameterSchema(ctx context.Context, arg InsertParameterSchemaParams) (ParameterSchema, error)
	InsertParameterValue(ctx context.Context, arg InsertParameterValueParams) (ParameterValue, error)
	InsertProject(ctx context.Context, arg InsertProjectParams) (Project, error)
	InsertProjectVersion(ctx context.Context, arg InsertProjectVersionParams) (ProjectVersion, error)
	InsertProvisionerDaemon(ctx context.Context, arg InsertProvisionerDaemonParams) (ProvisionerDaemon, error)
	InsertProvisionerJob(ctx context.Context, arg InsertProvisionerJobParams) (ProvisionerJob, error)
	InsertProvisionerJobAgent(ctx context.Context, arg InsertProvisionerJobAgentParams) (ProvisionerJobAgent, error)
	InsertProvisionerJobLogs(ctx context.Context, arg InsertProvisionerJobLogsParams) ([]ProvisionerJobLog, error)
	InsertProvisionerJobResource(ctx context.Context, arg InsertProvisionerJobResourceParams) (ProvisionerJobResource, error)
	InsertUser(ctx context.Context, arg InsertUserParams) (User, error)
	InsertWorkspace(ctx context.Context, arg InsertWorkspaceParams) (Workspace, error)
	InsertWorkspaceBuild(ctx context.Context, arg InsertWorkspaceBuildParams) (WorkspaceBuild, error)
	UpdateAPIKeyByID(ctx context.Context, arg UpdateAPIKeyByIDParams) error
	UpdateProvisionerDaemonByID(ctx context.Context, arg UpdateProvisionerDaemonByIDParams) error
	UpdateProvisionerJobAgentByID(ctx context.Context, arg UpdateProvisionerJobAgentByIDParams) error
	UpdateProvisionerJobByID(ctx context.Context, arg UpdateProvisionerJobByIDParams) error
	UpdateProvisionerJobWithCompleteByID(ctx context.Context, arg UpdateProvisionerJobWithCompleteByIDParams) error
	UpdateWorkspaceBuildByID(ctx context.Context, arg UpdateWorkspaceBuildByIDParams) error
}

var _ querier = (*sqlQuerier)(nil)
