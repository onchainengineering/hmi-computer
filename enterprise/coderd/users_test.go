package coderd_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/coderd/coderdtest"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/coderd/schedule/cron"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/enterprise/coderd/coderdenttest"
	"github.com/coder/coder/v2/enterprise/coderd/license"
	"github.com/coder/coder/v2/testutil"
)

func TestUserQuietHours(t *testing.T) {
	t.Parallel()

	t.Run("DefaultToUTC", func(t *testing.T) {
		t.Parallel()

		adminClient, adminUser := coderdenttest.New(t, &coderdenttest.Options{
			LicenseOptions: &coderdenttest.LicenseOptions{
				Features: license.Features{
					codersdk.FeatureAdvancedTemplateScheduling: 1,
				},
			},
		})

		client, user := coderdtest.CreateAnotherUser(t, adminClient, adminUser.OrganizationID)
		ctx := testutil.Context(t, testutil.WaitLong)
		res, err := client.UserQuietHoursSchedule(ctx, user.ID.String())
		require.NoError(t, err)
		require.Equal(t, "UTC", res.Timezone)
		require.Equal(t, "00:00", res.Time)
		require.Equal(t, "CRON_TZ=UTC 0 0 * * *", res.RawSchedule)
	})

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		defaultQuietHoursSchedule := "CRON_TZ=America/Chicago 0 1 * * *"
		defaultScheduleParsed, err := cron.Daily(defaultQuietHoursSchedule)
		require.NoError(t, err)
		nextTime := defaultScheduleParsed.Next(time.Now().In(defaultScheduleParsed.Location()))
		if time.Until(nextTime) < time.Hour {
			// Use a different default schedule instead, because we want to avoid
			// the schedule "ticking over" during this test run.
			defaultQuietHoursSchedule = "CRON_TZ=America/Chicago 0 13 * * *"
			defaultScheduleParsed, err = cron.Daily(defaultQuietHoursSchedule)
			require.NoError(t, err)
		}

		dv := coderdtest.DeploymentValues(t)
		dv.UserQuietHoursSchedule.DefaultSchedule.Set(defaultQuietHoursSchedule)

		adminClient, adminUser := coderdenttest.New(t, &coderdenttest.Options{
			Options: &coderdtest.Options{
				DeploymentValues: dv,
			},
			LicenseOptions: &coderdenttest.LicenseOptions{
				Features: license.Features{
					codersdk.FeatureAdvancedTemplateScheduling: 1,
				},
			},
		})

		// Do it with another user to make sure that we're not hitting RBAC
		// errors.
		client, user := coderdtest.CreateAnotherUser(t, adminClient, adminUser.OrganizationID)

		// Get quiet hours for a user that doesn't have them set.
		ctx := testutil.Context(t, testutil.WaitLong)
		sched1, err := client.UserQuietHoursSchedule(ctx, codersdk.Me)
		require.NoError(t, err)
		require.Equal(t, defaultScheduleParsed.String(), sched1.RawSchedule)
		require.False(t, sched1.UserSet)
		require.Equal(t, defaultScheduleParsed.TimeParsed().Format("15:40"), sched1.Time)
		require.Equal(t, defaultScheduleParsed.Location().String(), sched1.Timezone)
		require.WithinDuration(t, defaultScheduleParsed.Next(time.Now()), sched1.Next, 15*time.Second)

		// Set their quiet hours.
		customQuietHoursSchedule := "CRON_TZ=Australia/Sydney 0 0 * * *"
		customScheduleParsed, err := cron.Daily(customQuietHoursSchedule)
		require.NoError(t, err)
		nextTime = customScheduleParsed.Next(time.Now().In(customScheduleParsed.Location()))
		if time.Until(nextTime) < time.Hour {
			// Use a different default schedule instead, because we want to avoid
			// the schedule "ticking over" during this test run.
			customQuietHoursSchedule = "CRON_TZ=Australia/Sydney 0 12 * * *"
			customScheduleParsed, err = cron.Daily(customQuietHoursSchedule)
			require.NoError(t, err)
		}

		sched2, err := client.UpdateUserQuietHoursSchedule(ctx, user.ID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
			Schedule: customQuietHoursSchedule,
		})
		require.NoError(t, err)
		require.Equal(t, customScheduleParsed.String(), sched2.RawSchedule)
		require.True(t, sched2.UserSet)
		require.Equal(t, customScheduleParsed.TimeParsed().Format("15:40"), sched2.Time)
		require.Equal(t, customScheduleParsed.Location().String(), sched2.Timezone)
		require.WithinDuration(t, customScheduleParsed.Next(time.Now()), sched2.Next, 15*time.Second)

		// Get quiet hours for a user that has them set.
		sched3, err := client.UserQuietHoursSchedule(ctx, user.ID.String())
		require.NoError(t, err)
		require.Equal(t, customScheduleParsed.String(), sched3.RawSchedule)
		require.True(t, sched3.UserSet)
		require.Equal(t, customScheduleParsed.TimeParsed().Format("15:40"), sched3.Time)
		require.Equal(t, customScheduleParsed.Location().String(), sched3.Timezone)
		require.WithinDuration(t, customScheduleParsed.Next(time.Now()), sched3.Next, 15*time.Second)

		// Try setting a garbage schedule.
		_, err = client.UpdateUserQuietHoursSchedule(ctx, user.ID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
			Schedule: "garbage",
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "parse daily schedule")

		// Try setting a non-daily schedule.
		_, err = client.UpdateUserQuietHoursSchedule(ctx, user.ID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
			Schedule: "CRON_TZ=America/Chicago 0 0 * * 1",
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "parse daily schedule")

		// Try setting a schedule with a timezone that doesn't exist.
		_, err = client.UpdateUserQuietHoursSchedule(ctx, user.ID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
			Schedule: "CRON_TZ=Deans/House 0 0 * * *",
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "parse daily schedule")

		// Try setting a schedule with more than one time.
		_, err = client.UpdateUserQuietHoursSchedule(ctx, user.ID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
			Schedule: "CRON_TZ=America/Chicago 0 0,12 * * *",
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "more than one time")
		_, err = client.UpdateUserQuietHoursSchedule(ctx, user.ID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
			Schedule: "CRON_TZ=America/Chicago 0-30 0 * * *",
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "more than one time")

		// We don't allow unsetting the custom schedule so we don't need to worry
		// about it in this test.
	})

	t.Run("NotEntitled", func(t *testing.T) {
		t.Parallel()

		dv := coderdtest.DeploymentValues(t)
		dv.UserQuietHoursSchedule.DefaultSchedule.Set("CRON_TZ=America/Chicago 0 0 * * *")

		client, user := coderdenttest.New(t, &coderdenttest.Options{
			Options: &coderdtest.Options{
				DeploymentValues: dv,
			},
			LicenseOptions: &coderdenttest.LicenseOptions{
				Features: license.Features{
					// Not entitled.
					// codersdk.FeatureAdvancedTemplateScheduling: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitLong)
		//nolint:gocritic // We want to test the lack of entitlement, not RBAC.
		_, err := client.UserQuietHoursSchedule(ctx, user.UserID.String())
		require.Error(t, err)
		var sdkErr *codersdk.Error
		require.ErrorAs(t, err, &sdkErr)
		require.Equal(t, http.StatusForbidden, sdkErr.StatusCode())
	})

	t.Run("UserCannotSet", func(t *testing.T) {
		t.Parallel()

		dv := coderdtest.DeploymentValues(t)
		dv.UserQuietHoursSchedule.DefaultSchedule.Set("CRON_TZ=America/Chicago 0 0 * * *")
		dv.UserQuietHoursSchedule.AllowUserCustom.Set("false")

		adminClient, adminUser := coderdenttest.New(t, &coderdenttest.Options{
			Options: &coderdtest.Options{
				DeploymentValues: dv,
			},
			LicenseOptions: &coderdenttest.LicenseOptions{
				Features: license.Features{
					codersdk.FeatureAdvancedTemplateScheduling: 1,
				},
			},
		})

		// Do it with another user to make sure that we're not hitting RBAC
		// errors.
		client, user := coderdtest.CreateAnotherUser(t, adminClient, adminUser.OrganizationID)

		// Get the schedule
		ctx := testutil.Context(t, testutil.WaitLong)
		sched, err := client.UserQuietHoursSchedule(ctx, user.ID.String())
		require.NoError(t, err)
		require.Equal(t, "CRON_TZ=America/Chicago 0 0 * * *", sched.RawSchedule)
		require.False(t, sched.UserSet)
		require.False(t, sched.UserCanSet)

		// Try to set
		_, err = client.UpdateUserQuietHoursSchedule(ctx, user.ID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
			Schedule: "CRON_TZ=America/Chicago 30 2 * * *",
		})
		require.Error(t, err)
		var sdkErr *codersdk.Error
		require.ErrorAs(t, err, &sdkErr)
		require.Equal(t, http.StatusForbidden, sdkErr.StatusCode())
		require.Contains(t, sdkErr.Message, "cannot set custom quiet hours schedule")
	})
}

func TestCreateFirstUser_Entitlements_Trial(t *testing.T) {
	t.Parallel()

	adminClient, _ := coderdenttest.New(t, &coderdenttest.Options{
		LicenseOptions: &coderdenttest.LicenseOptions{
			Trial: true,
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
	defer cancel()

	//nolint:gocritic // we need the first user so admin
	entitlements, err := adminClient.Entitlements(ctx)
	require.NoError(t, err)
	require.True(t, entitlements.Trial, "Trial license should be immediately active.")
}

// TestAssignCustomOrgRoles verifies an organization admin (not just an owner) can create
// a custom role and assign it to an organization user.
func TestAssignCustomOrgRoles(t *testing.T) {
	t.Parallel()
	dv := coderdtest.DeploymentValues(t)
	dv.Experiments = []string{string(codersdk.ExperimentCustomRoles)}

	ownerClient, owner := coderdenttest.New(t, &coderdenttest.Options{
		Options: &coderdtest.Options{
			DeploymentValues:         dv,
			IncludeProvisionerDaemon: true,
		},
		LicenseOptions: &coderdenttest.LicenseOptions{
			Features: license.Features{
				codersdk.FeatureCustomRoles: 1,
			},
		},
	})

	client, _ := coderdtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.ScopedRoleOrgAdmin(owner.OrganizationID))
	tv := coderdtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
	coderdtest.AwaitTemplateVersionJobCompleted(t, client, tv.ID)

	ctx := testutil.Context(t, testutil.WaitShort)
	// Create a custom role as an organization admin that allows making templates.
	auditorRole, err := client.PatchOrganizationRole(ctx, owner.OrganizationID, codersdk.Role{
		Name:            "org-template-admin",
		OrganizationID:  owner.OrganizationID.String(),
		DisplayName:     "Template Admin",
		SitePermissions: nil,
		OrganizationPermissions: codersdk.CreatePermissions(map[codersdk.RBACResource][]codersdk.RBACAction{
			codersdk.ResourceTemplate: codersdk.RBACResourceActions[codersdk.ResourceTemplate], // All template perms
		}),
		UserPermissions: nil,
	})
	require.NoError(t, err)

	createTemplateReq := codersdk.CreateTemplateRequest{
		Name:        "name",
		DisplayName: "Template",
		VersionID:   tv.ID,
	}
	memberClient, member := coderdtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)
	// Check the member cannot create a template
	_, err = memberClient.CreateTemplate(ctx, owner.OrganizationID, createTemplateReq)
	require.Error(t, err)

	// Assign new role to the member as the org admin
	_, err = client.UpdateOrganizationMemberRoles(ctx, owner.OrganizationID, member.ID.String(), codersdk.UpdateRoles{
		Roles: []string{auditorRole.Name},
	})
	require.NoError(t, err)

	// Now the member can create the template
	_, err = memberClient.CreateTemplate(ctx, owner.OrganizationID, createTemplateReq)
	require.NoError(t, err)
}
