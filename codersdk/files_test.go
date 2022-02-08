package codersdk_test

import (
	"context"
	"testing"

	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/codersdk"
	"github.com/stretchr/testify/require"
)

func TestUpload(t *testing.T) {
	t.Parallel()
	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		_, err := client.Upload(context.Background(), "wow", []byte{})
		require.Error(t, err)
	})
	t.Run("Upload", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		_ = coderdtest.CreateInitialUser(t, client)
		_, err := client.Upload(context.Background(), codersdk.ContentTypeTar, []byte{'a'})
		require.NoError(t, err)
	})
}
