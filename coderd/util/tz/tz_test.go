package tz_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd/util/tz"
)

//nolint:paralleltest // Environment variables
func Test_TimezoneIANA(t *testing.T) {
	//nolint:paralleltest // t.Setenv
	t.Run("Env", func(t *testing.T) {
		t.Setenv("TZ", "Europe/Dublin")

		zone, err := tz.TimezoneIANA()
		assert.NoError(t, err)
		if assert.NotNil(t, zone) {
			assert.Equal(t, "Europe/Dublin", zone.String())
		}
	})

	//nolint:paralleltest // UnsetEnv
	t.Run("NoEnv", func(t *testing.T) {
		oldEnv, found := os.LookupEnv("TZ")
		if found {
			require.NoError(t, os.Unsetenv("TZ"))
			t.Cleanup(func() {
				_ = os.Setenv("TZ", oldEnv)
			})
		}

		zone, err := tz.TimezoneIANA()
		assert.NoError(t, err)
		assert.NotNil(t, zone)
	})
}
