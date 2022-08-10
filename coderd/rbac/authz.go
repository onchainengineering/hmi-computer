package rbac

import (
	"context"
	_ "embed"

	"golang.org/x/xerrors"

	"github.com/open-policy-agent/opa/rego"
)

type Authorizer interface {
	ByRoleName(ctx context.Context, subjectID string, roleNames []string, action Action, object Object) error
	PrepareByRoleName(ctx context.Context, subjectID string, roles []string, action Action, objectType string) (PreparedAuthorized, error)
}

type PreparedAuthorized interface {
	Authorize(ctx context.Context, object Object) error
}

// Filter takes in a list of objects, and will filter the list removing all
// the elements the subject does not have permission for.
// Filter does not allocate a new slice, and will use the existing one
// passed in. This can cause memory leaks if the slice is held for a prolonged
// period of time.
func Filter[O Objecter](ctx context.Context, auth Authorizer, subjID string, subjRoles []string, action Action, objects []O) []O {
	filtered := make([]O, 0)

	for i := range objects {
		object := objects[i]
		err := auth.ByRoleName(ctx, subjID, subjRoles, action, object.RBACObject())
		if err == nil {
			filtered = append(filtered, object)
		}
	}
	return filtered
}

func FilterPart[O Objecter](ctx context.Context, auth Authorizer, subjID string, subjRoles []string, action Action, objectType string, objects []O) ([]O, error) {
	filtered := make([]O, 0)
	prepared, err := auth.PrepareByRoleName(ctx, subjID, subjRoles, action, objectType)
	if err != nil {
		return nil, xerrors.Errorf("prepare: %w", err)
	}

	for i := range objects {
		object := objects[i]
		rbacObj := object.RBACObject()
		if rbacObj.Type != objectType {
			return nil, xerrors.Errorf("object types must be %s, found %s", objectType, object.RBACObject().Type)
		}
		err := prepared.Authorize(ctx, rbacObj)
		if err == nil {
			filtered = append(filtered, object)
		}
	}
	return filtered, nil
}

// RegoAuthorizer will use a prepared rego query for performing authorize()
type RegoAuthorizer struct {
	query rego.PreparedEvalQuery
}

// Load the policy from policy.rego in this directory.
//go:embed policy.rego
var policy string

func NewAuthorizer() (*RegoAuthorizer, error) {
	ctx := context.Background()
	query, err := rego.New(
		// allowed is the `allow` field from the prepared query. This is the field to check if authorization is
		// granted.
		rego.Query("data.authz.allow"),
		rego.Module("policy.rego", policy),
	).PrepareForEval(ctx)

	if err != nil {
		return nil, xerrors.Errorf("prepare query: %w", err)
	}
	return &RegoAuthorizer{query: query}, nil
}

type authSubject struct {
	ID    string `json:"id"`
	Roles []Role `json:"roles"`
}

// ByRoleName will expand all roleNames into roles before calling Authorize().
// This is the function intended to be used outside this package.
// The role is fetched from the builtin map located in memory.
func (a RegoAuthorizer) ByRoleName(ctx context.Context, subjectID string, roleNames []string, action Action, object Object) error {
	roles, err := RolesByNames(roleNames)
	if err != nil {
		return err
	}

	return a.Authorize(ctx, subjectID, roles, action, object)
}

// Authorize allows passing in custom Roles.
// This is really helpful for unit testing, as we can create custom roles to exercise edge cases.
func (a RegoAuthorizer) Authorize(ctx context.Context, subjectID string, roles []Role, action Action, object Object) error {
	input := map[string]interface{}{
		"subject": authSubject{
			ID:    subjectID,
			Roles: roles,
		},
		"object": object,
		"action": action,
	}

	results, err := a.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return ForbiddenWithInternal(xerrors.Errorf("eval rego: %w", err), input, results)
	}

	if !results.Allowed() {
		return ForbiddenWithInternal(xerrors.Errorf("policy disallows request"), input, results)
	}

	return nil
}

func (a RegoAuthorizer) PrepareByRoleName(ctx context.Context, subjectID string, roleNames []string, action Action, objectType string) (PreparedAuthorized, error) {
	roles, err := RolesByNames(roleNames)
	if err != nil {
		return nil, err
	}

	return a.Prepare(ctx, subjectID, roles, action, objectType)
}

func (RegoAuthorizer) Prepare(ctx context.Context, subjectID string, roles []Role, action Action, objectType string) (*PartialAuthorizer, error) {
	input := map[string]interface{}{
		"subject": authSubject{
			ID:    subjectID,
			Roles: roles,
		},
		"object": map[string]string{
			"type": objectType,
		},
		"action": action,
	}

	regoPolicy := rego.New(
		rego.Query("data.authz.allow"),
		rego.Module("policy.rego", policy),
		rego.Unknowns([]string{
			"input.object.owner",
			"input.object.org_owner",
		}),
		rego.Input(input),
	)

	auth, err := newPartialAuthorizer(ctx, regoPolicy, input)
	if err != nil {
		return nil, xerrors.Errorf("new partial authorizer: %w", err)
	}

	return auth, nil
}

// CheckPartial will not authorize the request. This function is to be used for unit testing to verify the rego policy
// can be converted into ONLY queries. This ensures we can convert the queries into SQL WHERE clauses in the future.
// If this function returns an error, then there is a set of inputs that also returns support rules, which cannot
// be converted.
// Unfortunately we cannot reuse a.Prepare for this purpose as a Partial() requires an expression,
// whereas a PartialResult() requires a reference. It's an annoying detail that we need to work around.
func (a RegoAuthorizer) CheckPartial(ctx context.Context, subjectID string, roles []Role, action Action, objectType string) (*rego.PartialQueries, interface{}, error) {
	partAuth, err := a.Prepare(ctx, subjectID, roles, action, objectType)
	if err != nil {
		return nil, nil, err
	}

	result, err := rego.New(
		rego.Query("true = data.authz.allow"),
		rego.Module("policy.rego", policy),
		rego.Unknowns([]string{
			"input.object.owner",
			"input.object.org_owner",
		}),
		rego.Input(partAuth.Input),
	).Partial(ctx)
	return result, partAuth.Input, err
}
