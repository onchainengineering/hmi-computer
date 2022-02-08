package coderd_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/codersdk"
)

func TestPostFiles(t *testing.T) {
	t.Parallel()
	t.Run("BadContentType", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		_ = coderdtest.CreateInitialUser(t, client)
		_, err := client.UploadFile(context.Background(), "bad", []byte{'a'})
		require.Error(t, err)
	})

	t.Run("MassiveBody", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		_ = coderdtest.CreateInitialUser(t, client)
		_, err := client.UploadFile(context.Background(), codersdk.ContentTypeTar, make([]byte, 11*(10<<20)))
		require.Error(t, err)
	})

	t.Run("Insert", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t)
		_ = coderdtest.CreateInitialUser(t, client)
		_, err := client.UploadFile(context.Background(), codersdk.ContentTypeTar, make([]byte, 10<<20))
		require.NoError(t, err)
	})
}
