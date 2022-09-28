package rbac

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"

	"github.com/stretchr/testify/require"
)

func TestCompileQuery(t *testing.T) {
	t.Parallel()
	opts := ast.ParserOptions{
		AllFutureKeywords: true,
	}
	t.Run("EmptyQuery", func(t *testing.T) {
		t.Parallel()
		expression, err := Compile(&rego.PartialQueries{
			Queries: []ast.Body{
				must(ast.ParseBody("")),
			},
			Support: []*ast.Module{},
		})
		require.NoError(t, err, "compile empty")

		require.Equal(t, "true", expression.RegoString(), "empty query is rego 'true'")
		require.Equal(t, "true", expression.SQLString(SQLConfig{}), "empty query is sql 'true'")
	})

	t.Run("TrueQuery", func(t *testing.T) {
		t.Parallel()
		expression, err := Compile(&rego.PartialQueries{
			Queries: []ast.Body{
				must(ast.ParseBody("true")),
			},
			Support: []*ast.Module{},
		})
		require.NoError(t, err, "compile")

		require.Equal(t, "true", expression.RegoString(), "true query is rego 'true'")
		require.Equal(t, "true", expression.SQLString(SQLConfig{}), "true query is sql 'true'")
	})

	t.Run("ACLIn", func(t *testing.T) {
		t.Parallel()
		expression, err := Compile(&rego.PartialQueries{
			Queries: []ast.Body{
				ast.MustParseBodyWithOpts(`"*" in input.object.acl_group_list.allUsers`, opts),
			},
			Support: []*ast.Module{},
		})
		require.NoError(t, err, "compile")

		require.Equal(t, `internal.member_2("*", input.object.acl_group_list.allUsers)`, expression.RegoString(), "convert to internal_member")
		require.Equal(t, `group_acl->allUsers ? '*'`, expression.SQLString(DefaultConfig()), "jsonb in")
	})

	t.Run("Complex", func(t *testing.T) {
		t.Parallel()
		expression, err := Compile(&rego.PartialQueries{
			Queries: []ast.Body{
				ast.MustParseBodyWithOpts(`input.object.org_owner != ""`, opts),
				ast.MustParseBodyWithOpts(`input.object.org_owner in {"a", "b", "c"}`, opts),
				ast.MustParseBodyWithOpts(`input.object.org_owner != ""`, opts),
				ast.MustParseBodyWithOpts(`"read" in input.object.acl_group_list.allUsers`, opts),
				ast.MustParseBodyWithOpts(`"read" in input.object.acl_user_list.me`, opts),
			},
			Support: []*ast.Module{},
		})
		require.NoError(t, err, "compile")
		require.Equal(t, `(organization_id :: text != '' OR `+
			`organization_id :: text = ANY(ARRAY ['a','b','c']) OR `+
			`organization_id :: text != '' OR `+
			`group_acl->allUsers ? 'read' OR `+
			`user_acl->me ? 'read')`,
			expression.SQLString(DefaultConfig()), "complex")
	})
}

//func TestRE(t *testing.T) {
//	// ^input\.object\.group_acl\.([^.]*)$
//	re := regexp.MustCompile(`^input\.object\.group_acl\.([^.]*)$`)
//
//	x := []string{"test"}
//	fmt.Sprintf("test", x)
//
//	//re.FindStringSubmatch("input.object.group_acl.allUsers")
//	fmt.Println(re.FindStringSubmatch("input.object.group_acl.allUsers"))
//}

func TestPartialCompileQuery(t *testing.T) {
	ctx := context.Background()
	defOrg := uuid.New()
	unuseID := uuid.New()

	user := subject{
		UserID: "me",
		Scope:  must(ScopeRole(ScopeAll)),
		Roles: []Role{
			must(RoleByName(RoleMember())),
			must(RoleByName(RoleOrgMember(defOrg))),
		},
	}
	var action Action = ActionRead
	object := ResourceWorkspace.InOrg(defOrg).WithOwner(unuseID.String())

	auth := NewAuthorizer()
	part, err := auth.Prepare(ctx, user.UserID, user.Roles, user.Scope, action, object.Type)
	require.NoError(t, err)

	result, err := Compile(part.partialQueries)
	require.NoError(t, err)

	fmt.Println("Rego: ", result.RegoString())
	fmt.Println("SQL: ", result.SQLString(DefaultConfig()))
}
