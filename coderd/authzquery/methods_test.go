package authzquery_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd/authzquery"
	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/coderd/database/databasefake"
	"github.com/stretchr/testify/suite"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/rbac"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type MethodTestSuite struct {
	suite.Suite
}

func (suite *MethodTestSuite) SetupTest() {
}

func (suite *MethodTestSuite) TearDownTest() {
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestMethodTestSuite(t *testing.T) {
	suite.Run(t, new(MethodTestSuite))
}

type MethodCase struct {
	Inputs     []reflect.Value
	Assertions []AssertRBAC
}

type AssertRBAC struct {
	Object  rbac.Object
	Actions []rbac.Action
}

func (suite *MethodTestSuite) RunMethodTest(t *testing.T, testCaseF func(t *testing.T, db database.Store) MethodCase) {
	testName := suite.T().Name()
	names := strings.Split(testName, "/")
	methodName := names[len(names)-1]

	db := databasefake.New()
	rec := &coderdtest.RecordingAuthorizer{
		Wrapped: &coderdtest.FakeAuthorizer{},
	}
	az := authzquery.NewAuthzQuerier(db, rec)
	actor := rbac.Subject{
		ID:     uuid.NewString(),
		Roles:  rbac.RoleNames{},
		Groups: []string{},
		Scope:  rbac.ScopeAll,
	}
	ctx := authzquery.WithAuthorizeContext(context.Background(), actor)

	testCase := testCaseF(t, db)

	// Find the method with the name of the test.
	found := false
	azt := reflect.TypeOf(az)
MethodLoop:
	for i := 0; i < azt.NumMethod(); i++ {
		method := azt.Method(i)
		if method.Name == methodName {
			resp := reflect.ValueOf(az).Method(i).Call(append([]reflect.Value{reflect.ValueOf(ctx)}, testCase.Inputs...))
			var _ = resp
			found = true
			break MethodLoop
		}
	}

	require.True(t, found, "method %q does not exist", testName)
}

func methodInputs(inputs ...any) []reflect.Value {
	out := make([]reflect.Value, 0)
	for _, input := range inputs {
		input := input
		out = append(out, reflect.ValueOf(input))
	}
	return out
}

func asserts(inputs ...any) []AssertRBAC {
	if len(inputs)%2 != 0 {
		panic(fmt.Sprintf("Must be an even length number of args, found %d", len(inputs)))
	}

	out := make([]AssertRBAC, 0)
	for i := 0; i < len(inputs); i += 2 {
		obj, ok := inputs[i].(rbac.Objecter)
		if !ok {
			panic(fmt.Sprintf("object type '%T' not a supported key", obj))
		}

		var actions []rbac.Action
		actions, ok = inputs[i+1].([]rbac.Action)
		if !ok {
			action, ok := inputs[i+1].(rbac.Action)
			if !ok {
				// Could be the string type.
				actionAsString, ok := inputs[i+1].(string)
				if !ok {
					panic(fmt.Sprintf("action type '%T' not a supported action", obj))
				}
				action = rbac.Action(actionAsString)
			}
			actions = []rbac.Action{action}
		}

		out = append(out, AssertRBAC{
			Object:  rbac.Object{},
			Actions: actions,
		})
	}
	return out
}
