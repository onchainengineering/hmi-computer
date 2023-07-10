package coderd_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/coderd/schedule"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/enterprise/coderd/coderdenttest"
	"github.com/coder/coder/enterprise/coderd/license"
	"github.com/coder/coder/testutil"
)

func TestUserQuietHours(t *testing.T) {
	t.Parallel()

	defaultQuietHoursSchedule := "CRON_TZ=America/Chicago 0 0 * * *"
	defaultScheduleParsed, err := schedule.Daily(defaultQuietHoursSchedule)
	require.NoError(t, err)
	nextTime := defaultScheduleParsed.Next(time.Now().In(defaultScheduleParsed.Location()))
	if time.Until(nextTime) < time.Hour {
		// Use a different default schedule instead, because we want to avoid
		// the schedule "ticking over" during this test run.
		defaultQuietHoursSchedule = "CRON_TZ=America/Chicago 0 12 * * *"
		defaultScheduleParsed, err = schedule.Daily(defaultQuietHoursSchedule)
		require.NoError(t, err)
	}

	dv := coderdtest.DeploymentValues(t)
	dv.UserQuietHoursSchedule.DefaultSchedule.Set(defaultQuietHoursSchedule)
	dv.UserQuietHoursSchedule.WindowDuration.Set("8h") // default is 4h

	client := coderdenttest.New(t, &coderdenttest.Options{
		Options: &coderdtest.Options{
			DeploymentValues: dv,
		},
	})
	user := coderdtest.CreateFirstUser(t, client)
	_ = coderdenttest.AddLicense(t, client, coderdenttest.LicenseOptions{
		Features: license.Features{
			codersdk.FeatureAdvancedTemplateScheduling: 1,
		},
	})

	// Get quiet hours for a user that doesn't have them set.
	ctx := testutil.Context(t, testutil.WaitLong)
	sched1, err := client.UserQuietHoursSchedule(ctx, codersdk.Me)
	require.NoError(t, err)
	require.Equal(t, defaultScheduleParsed.String(), sched1.RawSchedule)
	require.False(t, sched1.UserSet)
	require.Equal(t, defaultScheduleParsed.Time(), sched1.Time)
	require.Equal(t, defaultScheduleParsed.Location().String(), sched1.Timezone)
	require.Equal(t, dv.UserQuietHoursSchedule.WindowDuration.Value(), sched1.Duration)
	require.WithinDuration(t, defaultScheduleParsed.Next(time.Now()), sched1.Next, 15*time.Second)

	// Set their quiet hours.
	customQuietHoursSchedule := "CRON_TZ=Australia/Sydney 0 0 * * *"
	customScheduleParsed, err := schedule.Daily(customQuietHoursSchedule)
	require.NoError(t, err)
	nextTime = customScheduleParsed.Next(time.Now().In(customScheduleParsed.Location()))
	if time.Until(nextTime) < time.Hour {
		// Use a different default schedule instead, because we want to avoid
		// the schedule "ticking over" during this test run.
		customQuietHoursSchedule = "CRON_TZ=Australia/Sydney 0 12 * * *"
		customScheduleParsed, err = schedule.Daily(customQuietHoursSchedule)
		require.NoError(t, err)
	}

	sched2, err := client.UpdateUserQuietHoursSchedule(ctx, user.UserID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
		Schedule: customQuietHoursSchedule,
	})
	require.NoError(t, err)
	require.Equal(t, customScheduleParsed.String(), sched2.RawSchedule)
	require.True(t, sched2.UserSet)
	require.Equal(t, customScheduleParsed.Time(), sched2.Time)
	require.Equal(t, customScheduleParsed.Location().String(), sched2.Timezone)
	require.Equal(t, dv.UserQuietHoursSchedule.WindowDuration.Value(), sched2.Duration)
	require.WithinDuration(t, customScheduleParsed.Next(time.Now()), sched2.Next, 15*time.Second)

	// Get quiet hours for a user that has them set.
	sched3, err := client.UserQuietHoursSchedule(ctx, user.UserID.String())
	require.NoError(t, err)
	require.Equal(t, customScheduleParsed.String(), sched3.RawSchedule)
	require.True(t, sched3.UserSet)
	require.Equal(t, customScheduleParsed.Time(), sched3.Time)
	require.Equal(t, customScheduleParsed.Location().String(), sched3.Timezone)
	require.Equal(t, dv.UserQuietHoursSchedule.WindowDuration.Value(), sched3.Duration)
	require.WithinDuration(t, customScheduleParsed.Next(time.Now()), sched3.Next, 15*time.Second)

	// Try setting a garbage schedule.
	_, err = client.UpdateUserQuietHoursSchedule(ctx, user.UserID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
		Schedule: "garbage",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "parse daily schedule")

	// Try setting a non-daily schedule.
	_, err = client.UpdateUserQuietHoursSchedule(ctx, user.UserID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
		Schedule: "CRON_TZ=America/Chicago 0 0 * * 1",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "parse daily schedule")

	// Try setting a schedule with a timezone that doesn't exist.
	_, err = client.UpdateUserQuietHoursSchedule(ctx, user.UserID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
		Schedule: "CRON_TZ=Deans/House 0 0 * * *",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "parse daily schedule")

	// Try setting a schedule with more than one time.
	_, err = client.UpdateUserQuietHoursSchedule(ctx, user.UserID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
		Schedule: "CRON_TZ=America/Chicago 0 0,12 * * *",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "more than one time")
	_, err = client.UpdateUserQuietHoursSchedule(ctx, user.UserID.String(), codersdk.UpdateUserQuietHoursScheduleRequest{
		Schedule: "CRON_TZ=America/Chicago 0-30 0 * * *",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "more than one time")

	// We don't allow unsetting the custom schedule so we don't need to worry
	// about it in this test.
}
