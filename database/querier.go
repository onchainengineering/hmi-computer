// Code generated by sqlc. DO NOT EDIT.

package database

import (
	"context"

	"github.com/google/uuid"
)

type querier interface {
	AcquireProvisionerJob(ctx context.Context, arg AcquireProvisionerJobParams) (ProvisionerJob, error)
	GetAPIKeyByID(ctx context.Context, id string) (APIKey, error)
	GetOrganizationByID(ctx context.Context, id string) (Organization, error)
	GetOrganizationByName(ctx context.Context, name string) (Organization, error)
	GetOrganizationMemberByUserID(ctx context.Context, arg GetOrganizationMemberByUserIDParams) (OrganizationMember, error)
	GetOrganizationsByUserID(ctx context.Context, userID string) ([]Organization, error)
	GetParameterValuesByScope(ctx context.Context, arg GetParameterValuesByScopeParams) ([]ParameterValue, error)
	GetProjectByID(ctx context.Context, id uuid.UUID) (Project, error)
	GetProjectByOrganizationAndName(ctx context.Context, arg GetProjectByOrganizationAndNameParams) (Project, error)
	GetProjectHistoryByID(ctx context.Context, id uuid.UUID) (ProjectHistory, error)
	GetProjectHistoryByProjectID(ctx context.Context, projectID uuid.UUID) ([]ProjectHistory, error)
	GetProjectParametersByHistoryID(ctx context.Context, projectHistoryID uuid.UUID) ([]ProjectParameter, error)
	GetProjectsByOrganizationIDs(ctx context.Context, ids []string) ([]Project, error)
	GetProvisionerDaemonByID(ctx context.Context, id uuid.UUID) (ProvisionerDaemon, error)
	GetProvisionerJobByID(ctx context.Context, id uuid.UUID) (ProvisionerJob, error)
	GetUserByEmailOrUsername(ctx context.Context, arg GetUserByEmailOrUsernameParams) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	GetUserCount(ctx context.Context) (int64, error)
	GetWorkspaceAgentsByResourceIDs(ctx context.Context, ids []uuid.UUID) ([]WorkspaceAgent, error)
	GetWorkspaceByID(ctx context.Context, id uuid.UUID) (Workspace, error)
	GetWorkspaceByUserIDAndName(ctx context.Context, arg GetWorkspaceByUserIDAndNameParams) (Workspace, error)
	GetWorkspaceHistoryByID(ctx context.Context, id uuid.UUID) (WorkspaceHistory, error)
	GetWorkspaceHistoryByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceHistory, error)
	GetWorkspaceHistoryByWorkspaceIDWithoutAfter(ctx context.Context, workspaceID uuid.UUID) (WorkspaceHistory, error)
	GetWorkspaceResourcesByHistoryID(ctx context.Context, workspaceHistoryID uuid.UUID) ([]WorkspaceResource, error)
	GetWorkspacesByProjectAndUserID(ctx context.Context, arg GetWorkspacesByProjectAndUserIDParams) ([]Workspace, error)
	GetWorkspacesByUserID(ctx context.Context, ownerID string) ([]Workspace, error)
	InsertAPIKey(ctx context.Context, arg InsertAPIKeyParams) (APIKey, error)
	InsertOrganization(ctx context.Context, arg InsertOrganizationParams) (Organization, error)
	InsertOrganizationMember(ctx context.Context, arg InsertOrganizationMemberParams) (OrganizationMember, error)
	InsertParameterValue(ctx context.Context, arg InsertParameterValueParams) (ParameterValue, error)
	InsertProject(ctx context.Context, arg InsertProjectParams) (Project, error)
	InsertProjectHistory(ctx context.Context, arg InsertProjectHistoryParams) (ProjectHistory, error)
	InsertProjectParameter(ctx context.Context, arg InsertProjectParameterParams) (ProjectParameter, error)
	InsertProvisionerDaemon(ctx context.Context, arg InsertProvisionerDaemonParams) (ProvisionerDaemon, error)
	InsertProvisionerJob(ctx context.Context, arg InsertProvisionerJobParams) (ProvisionerJob, error)
	InsertUser(ctx context.Context, arg InsertUserParams) (User, error)
	InsertWorkspace(ctx context.Context, arg InsertWorkspaceParams) (Workspace, error)
	InsertWorkspaceAgent(ctx context.Context, arg InsertWorkspaceAgentParams) (WorkspaceAgent, error)
	InsertWorkspaceHistory(ctx context.Context, arg InsertWorkspaceHistoryParams) (WorkspaceHistory, error)
	InsertWorkspaceResource(ctx context.Context, arg InsertWorkspaceResourceParams) (WorkspaceResource, error)
	UpdateAPIKeyByID(ctx context.Context, arg UpdateAPIKeyByIDParams) error
	UpdateProvisionerDaemonByID(ctx context.Context, arg UpdateProvisionerDaemonByIDParams) error
	UpdateProvisionerJobByID(ctx context.Context, arg UpdateProvisionerJobByIDParams) error
	UpdateWorkspaceHistoryByID(ctx context.Context, arg UpdateWorkspaceHistoryByIDParams) error
}

var _ querier = (*sqlQuerier)(nil)
