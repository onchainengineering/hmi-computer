package authztest_test

import (
	"testing"

	"github.com/coder/coder/coderd/authz/authztest"
	"github.com/stretchr/testify/require"
)

func Test_AllPermissions(t *testing.T) {
	// If this changes, then we might have to fix some other tests. This constant
	// is the basis for understanding the permutation counts.
	const totalUniquePermissions int = 270
	require.Equal(t, len(authztest.AllPermissions()), totalUniquePermissions, "expected set size")
}
