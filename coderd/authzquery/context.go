package authzquery

import (
	"context"

	"github.com/google/uuid"

	"github.com/coder/coder/coderd/rbac"
)

// TODO:
//	- We still need a system user for system functions that a user should
//	not be able to call.

type authContextKey struct{}

func WithAuthorizeSystemContext(ctx context.Context, roles rbac.ExpandableRoles) context.Context {
	// TODO: Add protections to search for user roles. If user roles are found,
	// this should panic. That is a developer error that should be caught
	// in unit tests.
	return context.WithValue(ctx, authContextKey{}, rbac.Subject{
		ID:     uuid.Nil.String(),
		Roles:  roles,
		Scope:  rbac.ScopeAll,
		Groups: []string{},
	})
}

func WithAuthorizeContext(ctx context.Context, actor rbac.Subject) context.Context {
	return context.WithValue(ctx, authContextKey{}, actor)
}

// WithWorkspaceAgentTokenContext returns a context with a workspace agent token
// authorization subject. A workspace agent authorization subject is the
// workspace owner's authorization subject + a workspace agent scope.
//
// TODO: The arguments and usage of this function are not finalized. It might
// be a bit awkward to use at present. The arguments are required to build the
// required authorization context. The arguments should be the owner of the
// workspace authorization roles.
func WithWorkspaceAgentTokenContext(ctx context.Context, workspaceID uuid.UUID, actorID uuid.UUID, roles rbac.ExpandableRoles, groups []string) context.Context {
	// TODO: This workspace ID should be applied in the scope.
	var _ = workspaceID
	return context.WithValue(ctx, authContextKey{}, rbac.Subject{
		ID:    actorID.String(),
		Roles: roles,
		Scope: rbac.Scope{
			Role: rbac.Role{
				Name:        "workspace-agent-scope",
				DisplayName: "Workspace Agent Scope",
				// TODO: More permissions are needed for the agent to work.
				Site: []rbac.Permission{
					{
						ResourceType: rbac.ResourceWorkspace.Type,
						Action:       rbac.ActionRead,
					},
					{
						ResourceType: rbac.ResourceWorkspace.Type,
						Action:       rbac.ActionRead,
					},
					// TODO: Read the workspace owner user.
				},
				Org:  map[string][]rbac.Permission{},
				User: []rbac.Permission{},
			},
			// TODO: We need to whitelist more resources such as the workspace
			// owner.
			AllowIDList: []string{workspaceID.String()},
		},
		Groups: groups,
	})
}

// ActorFromContext returns the authorization subject from the context.
// All authentication flows should set the authorization subject in the context.
// If no actor is present, the function returns false.
func ActorFromContext(ctx context.Context) (rbac.Subject, bool) {
	a, ok := ctx.Value(authContextKey{}).(rbac.Subject)
	return a, ok
}
