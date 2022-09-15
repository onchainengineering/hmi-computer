package rbac

import (
	"fmt"

	"golang.org/x/xerrors"
)

type Scope string

const (
	ScopeAny                Scope = "any"
	ScopeApplicationConnect Scope = "application_connect"
)

var builtinScopes map[Scope]Role = map[Scope]Role{
	// ScopeAny is a special scope that allows access to all resources. During
	// authorize checks it is usually not used directly and skips scope checks.
	ScopeAny: {
		Name:        fmt.Sprintf("Scope_%s", ScopeAny),
		DisplayName: "Any operation",
		Site: permissions(map[Object][]Action{
			ResourceWildcard: {WildcardSymbol},
		}),
		Org:  map[string][]Permission{},
		User: []Permission{},
	},

	ScopeApplicationConnect: {
		Name:        fmt.Sprintf("Scope_%s", ScopeApplicationConnect),
		DisplayName: "Ability to connect to applications",
		Site: permissions(map[Object][]Action{
			ResourceWorkspaceApplicationConnect: {ActionCreate},
		}),
		Org:  map[string][]Permission{},
		User: []Permission{},
	},
}

func ScopeRole(scope Scope) (Role, error) {
	role, ok := builtinScopes[scope]
	if !ok {
		return Role{}, xerrors.Errorf("no scope named %q", scope)
	}
	return role, nil
}
