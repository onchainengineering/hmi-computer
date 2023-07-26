package dashboard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_allActions_ordering(t *testing.T) {
	t.Parallel()

	last := -1
	for idx, entry := range allActions {
		require.Greater(t, entry.roll, last, "roll table must be in ascending order, entry %d is out of order", idx)
		last = entry.roll
	}
}
