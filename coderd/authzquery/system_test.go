package authzquery_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/database/dbgen"
)

func (suite *MethodTestSuite) TestSystemFunctions() {
	suite.Run("UpdateUserLinkedID", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			u := dbgen.User(t, db, database.User{})
			l := dbgen.UserLink(t, db, database.UserLink{UserID: u.ID})
			return methodCase(values(database.UpdateUserLinkedIDParams{
				UserID:    u.ID,
				LinkedID:  l.LinkedID,
				LoginType: database.LoginTypeGithub,
			}), asserts())
		})
	})
	suite.Run("GetUserLinkByLinkedID", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			l := dbgen.UserLink(t, db, database.UserLink{})
			return methodCase(values(l.LinkedID), asserts())
		})
	})
	suite.Run("GetUserLinkByUserIDLoginType", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			l := dbgen.UserLink(t, db, database.UserLink{})
			return methodCase(values(database.GetUserLinkByUserIDLoginTypeParams{
				UserID:    l.UserID,
				LoginType: l.LoginType,
			}), asserts())
		})
	})
	suite.Run("GetLatestWorkspaceBuilds", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			dbgen.WorkspaceBuild(t, db, database.WorkspaceBuild{})
			dbgen.WorkspaceBuild(t, db, database.WorkspaceBuild{})
			return methodCase(values(), asserts())
		})
	})
	suite.Run("GetWorkspaceAgentByAuthToken", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			agent := dbgen.WorkspaceAgent(t, db, database.WorkspaceAgent{})
			return methodCase(values(agent.AuthToken), asserts())
		})
	})
	suite.Run("GetActiveUserCount", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values(), asserts())
		})
	})
	suite.Run("GetUnexpiredLicenses", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values(), asserts())
		})
	})
	suite.Run("GetAuthorizationUserRoles", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			u := dbgen.User(t, db, database.User{})
			return methodCase(values(u.ID), asserts())
		})
	})
	suite.Run("GetDERPMeshKey", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values(), asserts())
		})
	})
	suite.Run("InsertDERPMeshKey", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values("value"), asserts())
		})
	})
	suite.Run("InsertDeploymentID", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values("value"), asserts())
		})
	})
	suite.Run("InsertReplica", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values(database.InsertReplicaParams{
				ID: uuid.New(),
			}), asserts())
		})
	})
	suite.Run("UpdateReplica", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			replica, err := db.InsertReplica(context.Background(), database.InsertReplicaParams{ID: uuid.New()})
			require.NoError(t, err)
			return methodCase(values(database.UpdateReplicaParams{
				ID:              replica.ID,
				DatabaseLatency: 100,
			}), asserts())
		})
	})
	suite.Run("DeleteReplicasUpdatedBefore", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			_, err := db.InsertReplica(context.Background(), database.InsertReplicaParams{ID: uuid.New(), UpdatedAt: time.Now()})
			require.NoError(t, err)
			return methodCase(values(time.Now().Add(time.Hour)), asserts())
		})
	})
	suite.Run("GetReplicasUpdatedAfter", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			_, err := db.InsertReplica(context.Background(), database.InsertReplicaParams{ID: uuid.New(), UpdatedAt: time.Now()})
			require.NoError(t, err)
			return methodCase(values(time.Now().Add(time.Hour*-1)), asserts())
		})
	})
	suite.Run("GetUserCount", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values(), asserts())
		})
	})
	suite.Run("GetTemplates", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			_ = dbgen.Template(t, db, database.Template{})
			return methodCase(values(), asserts())
		})
	})
	suite.Run("UpdateWorkspaceBuildCostByID", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			b := dbgen.WorkspaceBuild(t, db, database.WorkspaceBuild{})
			return methodCase(values(database.UpdateWorkspaceBuildCostByIDParams{
				ID:        b.ID,
				DailyCost: 10,
			}), asserts())
		})
	})
	suite.Run("InsertOrUpdateLastUpdateCheck", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values("value"), asserts())
		})
	})
	suite.Run("GetLastUpdateCheck", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			err := db.InsertOrUpdateLastUpdateCheck(context.Background(), "value")
			require.NoError(t, err)
			return methodCase(values(), asserts())
		})
	})
	suite.Run("GetWorkspaceBuildsCreatedAfter", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			_ = dbgen.WorkspaceBuild(t, db, database.WorkspaceBuild{CreatedAt: time.Now().Add(-time.Hour)})
			return methodCase(values(time.Now()), asserts())
		})
	})
	suite.Run("GetWorkspaceAgentsCreatedAfter", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			_ = dbgen.WorkspaceAgent(t, db, database.WorkspaceAgent{CreatedAt: time.Now().Add(-time.Hour)})
			return methodCase(values(time.Now()), asserts())
		})
	})
	suite.Run("GetWorkspaceAppsCreatedAfter", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			_ = dbgen.WorkspaceApp(t, db, database.WorkspaceApp{CreatedAt: time.Now().Add(-time.Hour)})
			return methodCase(values(time.Now()), asserts())
		})
	})
	suite.Run("GetWorkspaceResourcesCreatedAfter", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			_ = dbgen.WorkspaceResource(t, db, database.WorkspaceResource{CreatedAt: time.Now().Add(-time.Hour)})
			return methodCase(values(time.Now()), asserts())
		})
	})
	suite.Run("GetWorkspaceResourceMetadataCreatedAfter", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			_ = dbgen.WorkspaceResourceMetadata(t, db, database.WorkspaceResourceMetadatum{})
			return methodCase(values(time.Now()), asserts())
		})
	})
	suite.Run("DeleteOldAgentStats", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values(), asserts())
		})
	})
	suite.Run("GetParameterSchemasCreatedAfter", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			_ = dbgen.ParameterSchema(t, db, database.ParameterSchema{CreatedAt: time.Now().Add(-time.Hour)})
			return methodCase(values(time.Now()), asserts())
		})
	})
	suite.Run("GetProvisionerJobsCreatedAfter", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			_ = dbgen.ProvisionerJob(t, db, database.ProvisionerJob{CreatedAt: time.Now().Add(-time.Hour)})
			return methodCase(values(time.Now()), asserts())
		})
	})
	suite.Run("InsertWorkspaceAgent", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values(database.InsertWorkspaceAgentParams{
				ID: uuid.New(),
			}), asserts())
		})
	})
	suite.Run("InsertWorkspaceApp", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values(database.InsertWorkspaceAppParams{
				ID:           uuid.New(),
				Health:       database.WorkspaceAppHealthDisabled,
				SharingLevel: database.AppSharingLevelOwner,
			}), asserts())
		})
	})
	suite.Run("InsertWorkspaceResourceMetadata", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values(database.InsertWorkspaceResourceMetadataParams{
				WorkspaceResourceID: uuid.New(),
			}), asserts())
		})
	})
	suite.Run("AcquireProvisionerJob", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			j := dbgen.ProvisionerJob(t, db, database.ProvisionerJob{
				StartedAt: sql.NullTime{Valid: false},
			})
			return methodCase(values(database.AcquireProvisionerJobParams{Types: []database.ProvisionerType{j.Provisioner}}), asserts())
		})
	})
	suite.Run("UpdateProvisionerJobWithCompleteByID", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			j := dbgen.ProvisionerJob(t, db, database.ProvisionerJob{})
			return methodCase(values(database.UpdateProvisionerJobWithCompleteByIDParams{
				ID: j.ID,
			}), asserts())
		})
	})
	suite.Run("UpdateProvisionerJobByID", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			j := dbgen.ProvisionerJob(t, db, database.ProvisionerJob{})
			return methodCase(values(database.UpdateProvisionerJobByIDParams{
				ID:        j.ID,
				UpdatedAt: time.Now(),
			}), asserts())
		})
	})
	suite.Run("InsertProvisionerJob", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values(database.InsertProvisionerJobParams{
				ID:            uuid.New(),
				Provisioner:   database.ProvisionerTypeEcho,
				StorageMethod: database.ProvisionerStorageMethodFile,
				Type:          database.ProvisionerJobTypeWorkspaceBuild,
			}), asserts())
		})
	})
	suite.Run("InsertProvisionerJobLogs", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			j := dbgen.ProvisionerJob(t, db, database.ProvisionerJob{})
			return methodCase(values(database.InsertProvisionerJobLogsParams{
				JobID: j.ID,
			}), asserts())
		})
	})
	suite.Run("InsertProvisionerDaemon", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values(database.InsertProvisionerDaemonParams{
				ID: uuid.New(),
			}), asserts())
		})
	})
	suite.Run("InsertTemplateVersionParameter", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			v := dbgen.TemplateVersion(t, db, database.TemplateVersion{})
			return methodCase(values(database.InsertTemplateVersionParameterParams{
				TemplateVersionID: v.ID,
			}), asserts())
		})
	})
	suite.Run("InsertWorkspaceResource", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			r := dbgen.WorkspaceResource(t, db, database.WorkspaceResource{})
			return methodCase(values(database.InsertWorkspaceResourceParams{
				ID:         r.ID,
				Transition: database.WorkspaceTransitionStart,
			}), asserts())
		})
	})
	suite.Run("InsertParameterSchema", func() {
		suite.RunMethodTest(func(t *testing.T, db database.Store) MethodCase {
			return methodCase(values(database.InsertParameterSchemaParams{
				ID:                       uuid.New(),
				DefaultSourceScheme:      database.ParameterSourceSchemeNone,
				DefaultDestinationScheme: database.ParameterDestinationSchemeNone,
				ValidationTypeSystem:     database.ParameterTypeSystemNone,
			}), asserts())
		})
	})
}
