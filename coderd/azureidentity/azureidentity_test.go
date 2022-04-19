package azureidentity_test

import (
	"context"
	"crypto/x509"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd/azureidentity"
)

const (
	signature = `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`
)

func TestValidate(t *testing.T) {
	t.Parallel()
	vm, err := azureidentity.Validate(context.Background(), signature, x509.VerifyOptions{})
	require.NoError(t, err)
	require.Equal(t, "bd8e7443-24a0-41f3-b949-8baf4fd1c573", vm)
}
