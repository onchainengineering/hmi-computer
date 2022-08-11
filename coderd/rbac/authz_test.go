package rbac_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"

	"github.com/coder/coder/coderd/rbac"
	"github.com/coder/coder/testutil"
)

// subject is required because rego needs
type subject struct {
	UserID string `json:"id"`
	// For the unit test we want to pass in the roles directly, instead of just
	// by name. This allows us to test custom roles that do not exist in the product,
	// but test edge cases of the implementation.
	Roles []rbac.Role `json:"roles"`
}

type fakeObject struct {
	Owner    uuid.UUID
	OrgOwner uuid.UUID
	Type     string
	Allowed  bool
}

func (w fakeObject) RBACObject() rbac.Object {
	return rbac.Object{
		Owner: w.Owner.String(),
		OrgID: w.OrgOwner.String(),
		Type:  w.Type,
	}
}

func TestFilterError(t *testing.T) {
	t.Parallel()
	auth, err := rbac.NewAuthorizer()
	require.NoError(t, err)

	_, err = rbac.Filter(context.Background(), auth, uuid.NewString(), []string{}, rbac.ActionRead, []rbac.Object{rbac.ResourceUser, rbac.ResourceWorkspace})
	require.ErrorContains(t, err, "object types must be uniform")
}

// TestFilter ensures the filter acts the same as an individual authorize.
// It generates a random set of objects, then runs the Filter batch function
// against the singular ByRoleName function.
func TestFilter(t *testing.T) {
	t.Parallel()

	orgIDs := make([]uuid.UUID, 10)
	userIDs := make([]uuid.UUID, len(orgIDs))
	for i := range orgIDs {
		orgIDs[i] = uuid.New()
		userIDs[i] = uuid.New()
	}
	objects := make([]fakeObject, 0, len(userIDs)*len(orgIDs))
	for i := range userIDs {
		for j := range orgIDs {
			objects = append(objects, fakeObject{
				Owner:    userIDs[i],
				OrgOwner: orgIDs[j],
				Type:     rbac.ResourceWorkspace.Type,
				Allowed:  false,
			})
		}
	}

	testCases := []struct {
		Name       string
		SubjectID  string
		Roles      []string
		Action     rbac.Action
		ObjectType string
	}{
		{
			Name:       "NoRoles",
			SubjectID:  userIDs[0].String(),
			Roles:      []string{},
			ObjectType: rbac.ResourceWorkspace.Type,
			Action:     rbac.ActionRead,
		},
		{
			Name:       "Admin",
			SubjectID:  userIDs[0].String(),
			Roles:      []string{rbac.RoleOrgMember(orgIDs[0]), "auditor", rbac.RoleAdmin(), rbac.RoleMember()},
			ObjectType: rbac.ResourceWorkspace.Type,
			Action:     rbac.ActionRead,
		},
		{
			Name:       "OrgAdmin",
			SubjectID:  userIDs[0].String(),
			Roles:      []string{rbac.RoleOrgMember(orgIDs[0]), rbac.RoleOrgAdmin(orgIDs[0]), rbac.RoleMember()},
			ObjectType: rbac.ResourceWorkspace.Type,
			Action:     rbac.ActionRead,
		},
		{
			Name:       "OrgMember",
			SubjectID:  userIDs[0].String(),
			Roles:      []string{rbac.RoleOrgMember(orgIDs[0]), rbac.RoleOrgMember(orgIDs[1]), rbac.RoleMember()},
			ObjectType: rbac.ResourceWorkspace.Type,
			Action:     rbac.ActionRead,
		},
		{
			Name:      "ManyRoles",
			SubjectID: userIDs[0].String(),
			Roles: []string{
				rbac.RoleOrgMember(orgIDs[0]), rbac.RoleOrgAdmin(orgIDs[0]),
				rbac.RoleOrgMember(orgIDs[1]), rbac.RoleOrgAdmin(orgIDs[1]),
				rbac.RoleOrgMember(orgIDs[2]), rbac.RoleOrgAdmin(orgIDs[2]),
				rbac.RoleOrgMember(orgIDs[4]),
				rbac.RoleOrgMember(orgIDs[5]),
				rbac.RoleMember(),
			},
			ObjectType: rbac.ResourceWorkspace.Type,
			Action:     rbac.ActionRead,
		},
		{
			Name:       "SiteMember",
			SubjectID:  userIDs[0].String(),
			Roles:      []string{rbac.RoleMember()},
			ObjectType: rbac.ResourceUser.Type,
			Action:     rbac.ActionRead,
		},
		{
			Name:      "ReadOrgs",
			SubjectID: userIDs[0].String(),
			Roles: []string{
				rbac.RoleOrgMember(orgIDs[0]),
				rbac.RoleOrgMember(orgIDs[1]),
				rbac.RoleOrgMember(orgIDs[2]),
				rbac.RoleOrgMember(orgIDs[3]),
				rbac.RoleMember(),
			},
			ObjectType: rbac.ResourceOrganization.Type,
			Action:     rbac.ActionRead,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			localObjects := make([]fakeObject, len(objects))
			copy(localObjects, objects)

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
			defer cancel()
			auth, err := rbac.NewAuthorizer()
			require.NoError(t, err, "new auth")

			// Run auth 1 by 1
			var allowedCount int
			for i, obj := range localObjects {
				obj.Type = tc.ObjectType
				err := auth.ByRoleName(ctx, tc.SubjectID, tc.Roles, rbac.ActionRead, obj.RBACObject())
				obj.Allowed = err == nil
				if err == nil {
					allowedCount++
				}
				localObjects[i] = obj
			}

			// Run by filter
			list, err := rbac.Filter(ctx, auth, tc.SubjectID, tc.Roles, tc.Action, localObjects)
			require.NoError(t, err)
			require.Equal(t, allowedCount, len(list), "expected number of allowed")
			for _, obj := range list {
				require.True(t, obj.Allowed, "expected allowed")
			}
		})
	}
}

// TestAuthorizeDomain test the very basic roles that are commonly used.
func TestAuthorizeDomain(t *testing.T) {
	t.Parallel()
	defOrg := uuid.New()
	unuseID := uuid.New()

	user := subject{
		UserID: "me",
		Roles: []rbac.Role{
			must(rbac.RoleByName(rbac.RoleMember())),
			must(rbac.RoleByName(rbac.RoleOrgMember(defOrg))),
		},
	}

	testAuthorize(t, "Member", user, []authTestCase{
		// Org + me
		{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner(user.UserID), actions: allActions(), allow: true},
		{resource: rbac.ResourceWorkspace.InOrg(defOrg), actions: allActions(), allow: false},

		{resource: rbac.ResourceWorkspace.WithOwner(user.UserID), actions: allActions(), allow: true},

		{resource: rbac.ResourceWorkspace.All(), actions: allActions(), allow: false},

		// Other org + me
		{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner(user.UserID), actions: allActions(), allow: false},
		{resource: rbac.ResourceWorkspace.InOrg(unuseID), actions: allActions(), allow: false},

		// Other org + other user
		{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner("not-me"), actions: allActions(), allow: false},

		{resource: rbac.ResourceWorkspace.WithOwner("not-me"), actions: allActions(), allow: false},

		// Other org + other us
		{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner("not-me"), actions: allActions(), allow: false},
		{resource: rbac.ResourceWorkspace.InOrg(unuseID), actions: allActions(), allow: false},

		{resource: rbac.ResourceWorkspace.WithOwner("not-me"), actions: allActions(), allow: false},
	})

	user = subject{
		UserID: "me",
		Roles: []rbac.Role{{
			Name: "deny-all",
			// List out deny permissions explicitly
			Site: []rbac.Permission{
				{
					Negate:       true,
					ResourceType: rbac.WildcardSymbol,
					Action:       rbac.WildcardSymbol,
				},
			},
		}},
	}

	testAuthorize(t, "DeletedMember", user, []authTestCase{
		// Org + me
		{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner(user.UserID), actions: allActions(), allow: false},
		{resource: rbac.ResourceWorkspace.InOrg(defOrg), actions: allActions(), allow: false},

		{resource: rbac.ResourceWorkspace.WithOwner(user.UserID), actions: allActions(), allow: false},

		{resource: rbac.ResourceWorkspace.All(), actions: allActions(), allow: false},

		// Other org + me
		{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner(user.UserID), actions: allActions(), allow: false},
		{resource: rbac.ResourceWorkspace.InOrg(unuseID), actions: allActions(), allow: false},

		// Other org + other user
		{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner("not-me"), actions: allActions(), allow: false},

		{resource: rbac.ResourceWorkspace.WithOwner("not-me"), actions: allActions(), allow: false},

		// Other org + other use
		{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner("not-me"), actions: allActions(), allow: false},
		{resource: rbac.ResourceWorkspace.InOrg(unuseID), actions: allActions(), allow: false},

		{resource: rbac.ResourceWorkspace.WithOwner("not-me"), actions: allActions(), allow: false},
	})

	user = subject{
		UserID: "me",
		Roles: []rbac.Role{
			must(rbac.RoleByName(rbac.RoleOrgAdmin(defOrg))),
			must(rbac.RoleByName(rbac.RoleMember())),
		},
	}

	testAuthorize(t, "OrgAdmin", user, []authTestCase{
		// Org + me
		{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner(user.UserID), actions: allActions(), allow: true},
		{resource: rbac.ResourceWorkspace.InOrg(defOrg), actions: allActions(), allow: true},

		{resource: rbac.ResourceWorkspace.WithOwner(user.UserID), actions: allActions(), allow: true},

		{resource: rbac.ResourceWorkspace.All(), actions: allActions(), allow: false},

		// Other org + me
		{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner(user.UserID), actions: allActions(), allow: false},
		{resource: rbac.ResourceWorkspace.InOrg(unuseID), actions: allActions(), allow: false},

		// Other org + other user
		{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner("not-me"), actions: allActions(), allow: true},

		{resource: rbac.ResourceWorkspace.WithOwner("not-me"), actions: allActions(), allow: false},

		// Other org + other use
		{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner("not-me"), actions: allActions(), allow: false},
		{resource: rbac.ResourceWorkspace.InOrg(unuseID), actions: allActions(), allow: false},

		{resource: rbac.ResourceWorkspace.WithOwner("not-me"), actions: allActions(), allow: false},
	})

	user = subject{
		UserID: "me",
		Roles: []rbac.Role{
			must(rbac.RoleByName(rbac.RoleAdmin())),
			must(rbac.RoleByName(rbac.RoleMember())),
		},
	}

	testAuthorize(t, "SiteAdmin", user, []authTestCase{
		// Org + me
		{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner(user.UserID), actions: allActions(), allow: true},
		{resource: rbac.ResourceWorkspace.InOrg(defOrg), actions: allActions(), allow: true},

		{resource: rbac.ResourceWorkspace.WithOwner(user.UserID), actions: allActions(), allow: true},

		{resource: rbac.ResourceWorkspace.All(), actions: allActions(), allow: true},

		// Other org + me
		{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner(user.UserID), actions: allActions(), allow: true},
		{resource: rbac.ResourceWorkspace.InOrg(unuseID), actions: allActions(), allow: true},

		// Other org + other user
		{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner("not-me"), actions: allActions(), allow: true},

		{resource: rbac.ResourceWorkspace.WithOwner("not-me"), actions: allActions(), allow: true},

		// Other org + other use
		{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner("not-me"), actions: allActions(), allow: true},
		{resource: rbac.ResourceWorkspace.InOrg(unuseID), actions: allActions(), allow: true},

		{resource: rbac.ResourceWorkspace.WithOwner("not-me"), actions: allActions(), allow: true},
	})

	// In practice this is a token scope on a regular subject.
	// So this unit test does not represent a practical role. It is just
	// testing the capabilities of the RBAC system.
	user = subject{
		UserID: "me",
		Roles: []rbac.Role{
			{
				Name: "WorkspaceToken",
				// This is at the site level to prevent the token from losing access if the user
				// is kicked from the org
				Site: []rbac.Permission{
					{
						Negate:       false,
						ResourceType: rbac.ResourceWorkspace.Type,
						Action:       rbac.ActionRead,
					},
				},
			},
		},
	}

	testAuthorize(t, "WorkspaceToken", user,
		// Read Actions
		cases(func(c authTestCase) authTestCase {
			c.actions = []rbac.Action{rbac.ActionRead}
			return c
		}, []authTestCase{
			// Org + me
			{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner(user.UserID), allow: true},
			{resource: rbac.ResourceWorkspace.InOrg(defOrg), allow: true},

			{resource: rbac.ResourceWorkspace.WithOwner(user.UserID), allow: true},

			{resource: rbac.ResourceWorkspace.All(), allow: true},

			// Other org + me
			{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner(user.UserID), allow: true},
			{resource: rbac.ResourceWorkspace.InOrg(unuseID), allow: true},

			// Other org + other user
			{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner("not-me"), allow: true},

			{resource: rbac.ResourceWorkspace.WithOwner("not-me"), allow: true},

			// Other org + other use
			{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner("not-me"), allow: true},
			{resource: rbac.ResourceWorkspace.InOrg(unuseID), allow: true},

			{resource: rbac.ResourceWorkspace.WithOwner("not-me"), allow: true},
		}),
		// Not read actions
		cases(func(c authTestCase) authTestCase {
			c.actions = []rbac.Action{rbac.ActionCreate, rbac.ActionUpdate, rbac.ActionDelete}
			c.allow = false
			return c
		}, []authTestCase{
			// Org + me
			{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner(user.UserID)},
			{resource: rbac.ResourceWorkspace.InOrg(defOrg)},

			{resource: rbac.ResourceWorkspace.WithOwner(user.UserID)},

			{resource: rbac.ResourceWorkspace.All()},

			// Other org + me
			{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner(user.UserID)},
			{resource: rbac.ResourceWorkspace.InOrg(unuseID)},

			// Other org + other user
			{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner("not-me")},

			{resource: rbac.ResourceWorkspace.WithOwner("not-me")},

			// Other org + other use
			{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner("not-me")},
			{resource: rbac.ResourceWorkspace.InOrg(unuseID)},

			{resource: rbac.ResourceWorkspace.WithOwner("not-me")},
		}),
	)

	// In practice this is a token scope on a regular subject
	user = subject{
		UserID: "me",
		Roles: []rbac.Role{
			{
				Name: "ReadOnlyOrgAndUser",
				Site: []rbac.Permission{},
				Org: map[string][]rbac.Permission{
					defOrg.String(): {{
						Negate:       false,
						ResourceType: "*",
						Action:       rbac.ActionRead,
					}},
				},
				User: []rbac.Permission{
					{
						Negate:       false,
						ResourceType: "*",
						Action:       rbac.ActionRead,
					},
				},
			},
		},
	}

	testAuthorize(t, "ReadOnly", user,
		cases(func(c authTestCase) authTestCase {
			c.actions = []rbac.Action{rbac.ActionRead}
			return c
		}, []authTestCase{
			// Read
			// Org + me
			{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner(user.UserID), allow: true},
			{resource: rbac.ResourceWorkspace.InOrg(defOrg), allow: true},

			{resource: rbac.ResourceWorkspace.WithOwner(user.UserID), allow: true},

			{resource: rbac.ResourceWorkspace.All(), allow: false},

			// Other org + me
			{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner(user.UserID), allow: false},
			{resource: rbac.ResourceWorkspace.InOrg(unuseID), allow: false},

			// Other org + other user
			{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner("not-me"), allow: true},

			{resource: rbac.ResourceWorkspace.WithOwner("not-me"), allow: false},

			// Other org + other use
			{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner("not-me"), allow: false},
			{resource: rbac.ResourceWorkspace.InOrg(unuseID), allow: false},

			{resource: rbac.ResourceWorkspace.WithOwner("not-me"), allow: false},
		}),

		// Pass non-read actions
		cases(func(c authTestCase) authTestCase {
			c.actions = []rbac.Action{rbac.ActionCreate, rbac.ActionUpdate, rbac.ActionDelete}
			c.allow = false
			return c
		}, []authTestCase{
			// Read
			// Org + me
			{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner(user.UserID)},
			{resource: rbac.ResourceWorkspace.InOrg(defOrg)},

			{resource: rbac.ResourceWorkspace.WithOwner(user.UserID)},

			{resource: rbac.ResourceWorkspace.All()},

			// Other org + me
			{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner(user.UserID)},
			{resource: rbac.ResourceWorkspace.InOrg(unuseID)},

			// Other org + other user
			{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner("not-me")},

			{resource: rbac.ResourceWorkspace.WithOwner("not-me")},

			// Other org + other use
			{resource: rbac.ResourceWorkspace.InOrg(unuseID).WithOwner("not-me")},
			{resource: rbac.ResourceWorkspace.InOrg(unuseID)},

			{resource: rbac.ResourceWorkspace.WithOwner("not-me")},
		}))
}

// TestAuthorizeLevels ensures level overrides are acting appropriately
//nolint:paralleltest
func TestAuthorizeLevels(t *testing.T) {
	defOrg := uuid.New()
	unusedID := uuid.New()

	user := subject{
		UserID: "me",
		Roles: []rbac.Role{
			must(rbac.RoleByName(rbac.RoleAdmin())),
			{
				Name: "org-deny:" + defOrg.String(),
				Org: map[string][]rbac.Permission{
					defOrg.String(): {
						{
							Negate:       true,
							ResourceType: "*",
							Action:       "*",
						},
					},
				},
			},
			{
				Name: "user-deny-all",
				// List out deny permissions explicitly
				User: []rbac.Permission{
					{
						Negate:       true,
						ResourceType: rbac.WildcardSymbol,
						Action:       rbac.WildcardSymbol,
					},
				},
			},
		},
	}

	testAuthorize(t, "AdminAlwaysAllow", user,
		cases(func(c authTestCase) authTestCase {
			c.actions = allActions()
			c.allow = true
			return c
		}, []authTestCase{
			// Org + me
			{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner(user.UserID)},
			{resource: rbac.ResourceWorkspace.InOrg(defOrg)},

			{resource: rbac.ResourceWorkspace.WithOwner(user.UserID)},

			{resource: rbac.ResourceWorkspace.All()},

			// Other org + me
			{resource: rbac.ResourceWorkspace.InOrg(unusedID).WithOwner(user.UserID)},
			{resource: rbac.ResourceWorkspace.InOrg(unusedID)},

			// Other org + other user
			{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner("not-me")},

			{resource: rbac.ResourceWorkspace.WithOwner("not-me")},

			// Other org + other use
			{resource: rbac.ResourceWorkspace.InOrg(unusedID).WithOwner("not-me")},
			{resource: rbac.ResourceWorkspace.InOrg(unusedID)},

			{resource: rbac.ResourceWorkspace.WithOwner("not-me")},
		}))

	user = subject{
		UserID: "me",
		Roles: []rbac.Role{
			{
				Name: "site-noise",
				Site: []rbac.Permission{
					{
						Negate:       true,
						ResourceType: "random",
						Action:       rbac.WildcardSymbol,
					},
				},
			},
			must(rbac.RoleByName(rbac.RoleOrgAdmin(defOrg))),
			{
				Name: "user-deny-all",
				// List out deny permissions explicitly
				User: []rbac.Permission{
					{
						Negate:       true,
						ResourceType: rbac.WildcardSymbol,
						Action:       rbac.WildcardSymbol,
					},
				},
			},
		},
	}

	testAuthorize(t, "OrgAllowAll", user,
		cases(func(c authTestCase) authTestCase {
			c.actions = allActions()
			return c
		}, []authTestCase{
			// Org + me
			{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner(user.UserID), allow: true},
			{resource: rbac.ResourceWorkspace.InOrg(defOrg), allow: true},

			{resource: rbac.ResourceWorkspace.WithOwner(user.UserID), allow: false},

			{resource: rbac.ResourceWorkspace.All(), allow: false},

			// Other org + me
			{resource: rbac.ResourceWorkspace.InOrg(unusedID).WithOwner(user.UserID), allow: false},
			{resource: rbac.ResourceWorkspace.InOrg(unusedID), allow: false},

			// Other org + other user
			{resource: rbac.ResourceWorkspace.InOrg(defOrg).WithOwner("not-me"), allow: true},

			{resource: rbac.ResourceWorkspace.WithOwner("not-me"), allow: false},

			// Other org + other use
			{resource: rbac.ResourceWorkspace.InOrg(unusedID).WithOwner("not-me"), allow: false},
			{resource: rbac.ResourceWorkspace.InOrg(unusedID), allow: false},

			{resource: rbac.ResourceWorkspace.WithOwner("not-me"), allow: false},
		}))
}

// cases applies a given function to all test cases. This makes generalities easier to create.
func cases(opt func(c authTestCase) authTestCase, cases []authTestCase) []authTestCase {
	if opt == nil {
		return cases
	}
	for i := range cases {
		cases[i] = opt(cases[i])
	}
	return cases
}

type authTestCase struct {
	resource rbac.Object
	actions  []rbac.Action
	allow    bool
}

func testAuthorize(t *testing.T, name string, subject subject, sets ...[]authTestCase) {
	t.Helper()
	authorizer, err := rbac.NewAuthorizer()
	require.NoError(t, err)
	for _, cases := range sets {
		for _, c := range cases {
			t.Run(name, func(t *testing.T) {
				for _, a := range c.actions {
					ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
					t.Cleanup(cancel)
					authError := authorizer.Authorize(ctx, subject.UserID, subject.Roles, a, c.resource)
					// Logging only
					if authError != nil {
						var uerr *rbac.UnauthorizedError
						xerrors.As(authError, &uerr)
						d, _ := json.Marshal(uerr.Input())
						t.Logf("input: %s", string(d))
						t.Logf("internal error: %+v", uerr.Internal().Error())
						t.Logf("output: %+v", uerr.Output())
					} else {
						d, _ := json.Marshal(map[string]interface{}{
							"subject": subject,
							"object":  c.resource,
							"action":  a,
						})
						t.Log(string(d))
					}

					if c.allow {
						require.NoError(t, authError, "expected no error for testcase action %s", a)
					} else {
						require.Error(t, authError, "expected unauthorized")
					}

					partialAuthz, err := authorizer.Prepare(ctx, subject.UserID, subject.Roles, a, c.resource.Type)
					require.NoError(t, err, "make prepared authorizer")

					// Also check the rego policy can form a valid partial query result.
					// This ensures we can convert the queries into SQL WHERE clauses in the future.
					// If this function returns 'Support' sections, then we cannot convert the query into SQL.
					if len(partialAuthz.PartialQueries.Support) > 0 {
						d, _ := json.Marshal(partialAuthz.Input)
						t.Logf("input: %s", string(d))
						for _, q := range partialAuthz.PartialQueries.Queries {
							t.Logf("query: %+v", q.String())
						}
						for _, s := range partialAuthz.PartialQueries.Support {
							t.Logf("support: %+v", s.String())
						}
					}
					require.Equal(t, 0, len(partialAuthz.PartialQueries.Support), "expected 0 support rules")

					partialErr := partialAuthz.Authorize(ctx, c.resource)
					if authError != nil {
						require.Error(t, partialErr, "partial error blocked valid request (false negative)")
					} else {
						require.NoError(t, partialErr, "partial allowed invalid request (false positive)")
					}
				}
			})
		}
	}
}

// allActions is a helper function to return all the possible actions types.
func allActions() []rbac.Action {
	return []rbac.Action{rbac.ActionCreate, rbac.ActionRead, rbac.ActionUpdate, rbac.ActionDelete}
}
