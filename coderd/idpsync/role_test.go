package idpsync_test

import (
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/database/dbgen"
	"github.com/coder/coder/v2/coderd/database/dbtestutil"
	"github.com/coder/coder/v2/coderd/idpsync"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/coderd/runtimeconfig"
	"github.com/coder/coder/v2/testutil"
)

func TestRoleSyncTable(t *testing.T) {
	t.Parallel()

	if dbtestutil.WillUsePostgres() {
		t.Skip("Skipping test because it populates a lot of db entries, which is slow on postgres.")
	}

	userClaims := jwt.MapClaims{
		"roles": []string{
			"foo", "bar", "baz",
			"create-bar", "create-baz",
			"legacy-bar", rbac.RoleOrgAuditor(),
		},
		// bad-claim is a number, and will fail any role sync
		"bad-claim": 100,
		"empty":     []string{},
	}

	testCases := []orgSetupDefinition{
		{
			Name:              "NoSync",
			OrganizationRoles: []string{},
			assertRoles: &orgRoleAssert{
				ExpectedOrgRoles: []string{},
			},
		},
		{
			Name: "SyncDisabled",
			OrganizationRoles: []string{
				rbac.RoleOrgAdmin(),
			},
			RoleSettings: &idpsync.RoleSyncSettings{},
			assertRoles: &orgRoleAssert{
				ExpectedOrgRoles: []string{
					rbac.RoleOrgAdmin(),
				},
			},
		},
		{
			// Audit role from claim
			Name: "RawAudit",
			OrganizationRoles: []string{
				rbac.RoleOrgAdmin(),
			},
			RoleSettings: &idpsync.RoleSyncSettings{
				Field:   "roles",
				Mapping: map[string][]string{},
			},
			assertRoles: &orgRoleAssert{
				ExpectedOrgRoles: []string{
					rbac.RoleOrgAuditor(),
				},
			},
		},
		{
			Name: "CustomRole",
			OrganizationRoles: []string{
				rbac.RoleOrgAdmin(),
			},
			CustomRoles: []string{"foo"},
			RoleSettings: &idpsync.RoleSyncSettings{
				Field:   "roles",
				Mapping: map[string][]string{},
			},
			assertRoles: &orgRoleAssert{
				ExpectedOrgRoles: []string{
					rbac.RoleOrgAuditor(),
					"foo",
				},
			},
		},
		{
			Name: "RoleMapping",
			OrganizationRoles: []string{
				rbac.RoleOrgAdmin(),
				"invalid", // Throw in an extra invalid role that will be removed
			},
			CustomRoles: []string{"custom"},
			RoleSettings: &idpsync.RoleSyncSettings{
				Field: "roles",
				Mapping: map[string][]string{
					"foo": {"custom", rbac.RoleOrgTemplateAdmin()},
				},
			},
			assertRoles: &orgRoleAssert{
				ExpectedOrgRoles: []string{
					rbac.RoleOrgAuditor(),
					rbac.RoleOrgTemplateAdmin(),
					"custom",
				},
			},
		},
		{
			// InvalidClaims will log an error, but do not block authentication.
			// This is to prevent a misconfigured organization from blocking
			// a user from authenticating.
			Name:              "InvalidClaim",
			OrganizationRoles: []string{rbac.RoleOrgAdmin()},
			RoleSettings: &idpsync.RoleSyncSettings{
				Field: "bad-claim",
			},
			assertRoles: &orgRoleAssert{
				ExpectedOrgRoles: []string{
					rbac.RoleOrgAdmin(),
				},
			},
		},
		{
			Name:              "NoChange",
			OrganizationRoles: []string{rbac.RoleOrgAdmin(), rbac.RoleOrgTemplateAdmin(), rbac.RoleOrgAuditor()},
			RoleSettings: &idpsync.RoleSyncSettings{
				Field: "roles",
				Mapping: map[string][]string{
					"foo": {rbac.RoleOrgAuditor(), rbac.RoleOrgTemplateAdmin()},
					"bar": {rbac.RoleOrgAdmin()},
				},
			},
			assertRoles: &orgRoleAssert{
				ExpectedOrgRoles: []string{
					rbac.RoleOrgAdmin(), rbac.RoleOrgAuditor(), rbac.RoleOrgTemplateAdmin(),
				},
			},
		},
		{
			// InvalidOriginalRole starts the user with an invalid role.
			// In practice, this should not happen, as it means a role was
			// inserted into the database that does not exist.
			// For the purposes of syncing, it does not matter, and the sync
			// should succeed.
			Name:              "InvalidOriginalRole",
			OrganizationRoles: []string{"something-bad"},
			RoleSettings: &idpsync.RoleSyncSettings{
				Field:   "roles",
				Mapping: map[string][]string{},
			},
			assertRoles: &orgRoleAssert{
				ExpectedOrgRoles: []string{
					rbac.RoleOrgAuditor(),
				},
			},
		},
		{
			Name:              "NonExistentClaim",
			OrganizationRoles: []string{rbac.RoleOrgAuditor()},
			RoleSettings: &idpsync.RoleSyncSettings{
				Field:   "not-exists",
				Mapping: map[string][]string{},
			},
			assertRoles: &orgRoleAssert{
				ExpectedOrgRoles: []string{},
			},
		},
		{
			Name:              "EmptyClaim",
			OrganizationRoles: []string{rbac.RoleOrgAuditor()},
			RoleSettings: &idpsync.RoleSyncSettings{
				Field:   "empty",
				Mapping: map[string][]string{},
			},
			assertRoles: &orgRoleAssert{
				ExpectedOrgRoles: []string{},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			db, _ := dbtestutil.NewDB(t)
			manager := runtimeconfig.NewManager()
			s := idpsync.NewAGPLSync(slogtest.Make(t, &slogtest.Options{
				IgnoreErrors: true,
			}),
				manager,
				idpsync.DeploymentSyncSettings{
					SiteRoleField: "roles",
				},
			)

			ctx := testutil.Context(t, testutil.WaitSuperLong)
			user := dbgen.User(t, db, database.User{})
			orgID := uuid.New()
			SetupOrganization(t, s, db, user, orgID, tc)

			// Do the role sync!
			err := s.SyncRoles(ctx, db, user, idpsync.RoleParams{
				SyncEnabled:  true,
				SyncSiteWide: false,
				MergedClaims: userClaims,
			})
			require.NoError(t, err)

			tc.Assert(t, orgID, db, user)
		})
	}

	// AllTogether runs the entire tabled test as a singular user and
	// deployment. This tests all organizations being synced together.
	// The reason we do them individually, is that it is much easier to
	// debug a single test case.
	t.Run("AllTogether", func(t *testing.T) {
		t.Parallel()

		db, _ := dbtestutil.NewDB(t)
		manager := runtimeconfig.NewManager()
		s := idpsync.NewAGPLSync(slogtest.Make(t, &slogtest.Options{
			IgnoreErrors: true,
		}),
			manager,
			// Also sync some site wide roles
			idpsync.DeploymentSyncSettings{
				GroupField:    "groups",
				SiteRoleField: "roles",
				// Site sync settings do not matter,
				// as we are not testing the site parse here.
				// Only the sync, assuming the parse is correct.
			},
		)

		ctx := testutil.Context(t, testutil.WaitSuperLong)
		user := dbgen.User(t, db, database.User{})

		var asserts []func(t *testing.T)

		for _, tc := range testCases {
			tc := tc

			orgID := uuid.New()
			SetupOrganization(t, s, db, user, orgID, tc)
			asserts = append(asserts, func(t *testing.T) {
				t.Run(tc.Name, func(t *testing.T) {
					t.Parallel()
					tc.Assert(t, orgID, db, user)
				})
			})
		}

		err := s.SyncRoles(ctx, db, user, idpsync.RoleParams{
			SyncEnabled:  true,
			SyncSiteWide: true,
			SiteWideRoles: []string{
				rbac.RoleTemplateAdmin().Name, // Duplicate this value to test deduplication
				rbac.RoleTemplateAdmin().Name, rbac.RoleAuditor().Name,
			},
			MergedClaims: userClaims,
		})
		require.NoError(t, err)

		for _, assert := range asserts {
			assert(t)
		}

		// Also assert site wide roles
		//nolint:gocritic // unit testing assertions
		allRoles, err := db.GetAuthorizationUserRoles(dbauthz.AsSystemRestricted(ctx), user.ID)
		require.NoError(t, err)

		allRoleIDs, err := allRoles.RoleNames()
		require.NoError(t, err)

		siteRoles := slices.DeleteFunc(allRoleIDs, func(r rbac.RoleIdentifier) bool {
			return r.IsOrgRole()
		})

		require.ElementsMatch(t, []rbac.RoleIdentifier{
			rbac.RoleTemplateAdmin(), rbac.RoleAuditor(), rbac.RoleMember(),
		}, siteRoles)
	})
}
