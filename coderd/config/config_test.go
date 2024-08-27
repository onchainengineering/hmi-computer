package config_test

import (
	"testing"

	"github.com/coder/serpent"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/coderd/coderdtest"
	"github.com/coder/coder/v2/coderd/config"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/enterprise/coderd/coderdenttest"
	"github.com/coder/coder/v2/enterprise/coderd/license"
)

// TestConfig demonstrates creating org-level overrides for deployment-level settings.
func TestConfig(t *testing.T) {
	t.Parallel()

	vals := coderdtest.DeploymentValues(t)
	vals.Experiments = []string{string(codersdk.ExperimentMultiOrganization)}
	adminClient, _, _, _ := coderdenttest.NewWithAPI(t, &coderdenttest.Options{
		Options: &coderdtest.Options{DeploymentValues: vals},
		LicenseOptions: &coderdenttest.LicenseOptions{
			Features: license.Features{
				codersdk.FeatureMultipleOrganizations: 1,
			},
		},
	})
	altOrg := coderdenttest.CreateOrganization(t, adminClient, coderdenttest.CreateOrganizationOptions{})

	t.Run("panics unless initialized", func(t *testing.T) {
		t.Parallel()

		field := config.RuntimeOrgScoped[*serpent.String]{}
		require.Panics(t, func() {
			field.StartupValue().String()
		})

		field.Init("my-field")
		require.NotPanics(t, func() {
			field.StartupValue().String()
		})
	})

	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		store := config.NewInMemoryStore()
		resolver := config.NewOrgResolver(config.NewStoreResolver(store), altOrg.ID)
		mutator := config.NewOrgMutator(config.NewStoreMutator(store), altOrg.ID)

		var (
			base     = serpent.String("system@dev.coder.com")
			override = serpent.String("dogfood@dev.coder.com")
		)

		field := config.RuntimeOrgScoped[*serpent.String]{}
		field.Init("my-field")
		// Check that no default has been set.
		require.Empty(t, field.StartupValue().String())
		// Initialize the value.
		require.NoError(t, field.Set(base.String()))
		// Validate that it returns that value.
		require.Equal(t, base.String(), field.String())
		// Validate that there is no org-level override right now.
		_, err := field.Resolve(resolver)
		require.ErrorIs(t, err, config.EntryNotFound)
		// Coalesce returns the deployment-wide value.
		val, err := field.Coalesce(resolver)
		require.NoError(t, err)
		require.Equal(t, base.String(), val.String())
		// Set an org-level override.
		require.NoError(t, field.Save(mutator, &override))
		// Coalesce now returns the org-level value.
		val, err = field.Coalesce(resolver)
		require.NoError(t, err)
		require.Equal(t, override.String(), val.String())
	})

	t.Run("complex", func(t *testing.T) {
		t.Parallel()

		store := config.NewInMemoryStore()
		resolver := config.NewOrgResolver(config.NewStoreResolver(store), altOrg.ID)
		mutator := config.NewOrgMutator(config.NewStoreMutator(store), altOrg.ID)

		field := config.Runtime[*serpent.Struct[map[string]string]]{}
		field.Init("my-field")
		var (
			base = serpent.Struct[map[string]string]{
				Value: map[string]string{"access_type": "offline"},
			}
			override = serpent.Struct[map[string]string]{
				Value: map[string]string{
					"a": "b",
					"c": "d",
				},
			}
		)

		// Check that no default has been set.
		require.Empty(t, field.StartupValue().Value)
		// Initialize the value.
		require.NoError(t, field.Set(base.String()))
		// Validate that there is no org-level override right now.
		_, err := field.Resolve(resolver)
		require.ErrorIs(t, err, config.EntryNotFound)
		// Coalesce returns the deployment-wide value.
		val, err := field.Coalesce(resolver)
		require.NoError(t, err)
		require.Equal(t, base.Value, val.Value)
		// Set an org-level override.
		require.NoError(t, field.Save(mutator, &override))
		// Coalesce now returns the org-level value.
		structVal, err := field.Resolve(resolver)
		require.NoError(t, err)
		require.Equal(t, override.Value, structVal.Value)
	})
}
