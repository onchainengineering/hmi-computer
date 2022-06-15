package cli

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/xerrors"

	"github.com/coder/coder/coderd/autobuild/schedule"
	"github.com/coder/coder/coderd/util/ptr"
	"github.com/coder/coder/coderd/util/tz"

	"github.com/coder/coder/codersdk"
)

var errInvalidScheduleFormat = xerrors.New("Schedule must be in the format Mon-Fri 09:00AM America/Chicago")
var errInvalidTimeFormat = xerrors.New("Start time must be in the format hh:mm[am|pm] or HH:MM")
var errUnsupportedTimezone = xerrors.New("The location you provided looks like a timezone. Check https://ipinfo.io for your location.")

// durationDisplay formats a duration for easier display:
//   * Durations of 24 hours or greater are displays as Xd
//   * Duration is truncated to the nearest minute
//   * Empty minutes and seconds are truncated
func durationDisplay(d time.Duration) string {
	duration := d
	if duration < time.Minute {
		return "<1m"
	}
	if duration > time.Minute {
		duration = duration.Truncate(time.Minute)
	}
	days := 0
	for duration.Hours() >= 24 {
		days++
		duration -= 24 * time.Hour
	}
	durationDisplay := duration.String()
	if days > 0 {
		durationDisplay = fmt.Sprintf("%dd%s", days, durationDisplay)
	}
	if strings.HasSuffix(durationDisplay, "m0s") {
		durationDisplay = durationDisplay[:len(durationDisplay)-2]
	}
	if strings.HasSuffix(durationDisplay, "h0m") {
		durationDisplay = durationDisplay[:len(durationDisplay)-2]
	}
	return durationDisplay
}

// hasExtension returns the deadline extension of ws, if it is present.
// Note that the extension may be negative.
func hasExtension(ws codersdk.Workspace) (bool, time.Duration) {
	if ws.LatestBuild.Transition != codersdk.WorkspaceTransitionStart {
		return false, 0
	}
	if ws.LatestBuild.Job.CompletedAt == nil || ws.LatestBuild.Job.CompletedAt.IsZero() {
		return false, 0
	}
	if ws.LatestBuild.Deadline.IsZero() {
		return false, 0
	}
	if ptr.NilOrZero(ws.TTLMillis) {
		return false, 0
	}
	ttl := time.Duration(*ws.TTLMillis) * time.Millisecond
	delta := ws.LatestBuild.Deadline.Add(-ttl).Sub(*ws.LatestBuild.Job.CompletedAt)
	cutoff := time.Minute
	if delta < -cutoff {
		return true, delta
	}
	if delta > cutoff {
		return true, delta
	}

	return false, 0
}

// parseCLISchedule parses a schedule in the format HH:MM{AM|PM} [DOW] [LOCATION]
func parseCLISchedule(parts ...string) (*schedule.Schedule, error) {
	// If the user was careful and quoted the schedule, un-quote it.
	// In the case that only time was specified, this will be a no-op.
	if len(parts) == 1 {
		parts = strings.Fields(parts[0])
	}
	var loc *time.Location
	dayOfWeek := "*"
	t, err := parseTime(parts[0])
	if err != nil {
		return nil, err
	}
	hour, minute := t.Hour(), t.Minute()

	// Any additional parts get ignored.
	switch len(parts) {
	case 3:
		dayOfWeek = parts[1]
		loc, err = time.LoadLocation(parts[2])
		if err != nil {
			_, err = time.Parse("MST", parts[2])
			if err == nil {
				return nil, errUnsupportedTimezone
			}
			return nil, xerrors.Errorf("Invalid timezone %q specified: a valid IANA timezone is required", parts[2])
		}
	case 2:
		// Did they provide day-of-week or location?
		if maybeLoc, err := time.LoadLocation(parts[1]); err != nil {
			// Assume day-of-week.
			dayOfWeek = parts[1]
		} else {
			loc = maybeLoc
		}
	case 1: // already handled
	default:
		return nil, errInvalidScheduleFormat
	}

	// If location was not specified, attempt to automatically determine it as a last resort.
	if loc == nil {
		loc, err = tz.TimezoneIANA()
		if err != nil {
			return nil, xerrors.Errorf("Could not automatically determine your timezone")
		}
	}

	sched, err := schedule.Weekly(fmt.Sprintf(
		"CRON_TZ=%s %d %d * * %s",
		loc.String(),
		minute,
		hour,
		dayOfWeek,
	))
	if err != nil {
		// This will either be an invalid dayOfWeek or an invalid timezone.
		return nil, xerrors.Errorf("Invalid schedule: %w", err)
	}

	return sched, nil
}

// parseDuration parses a duration from a string.
// If units are omitted, minutes are assumed.
func parseDuration(raw string) (time.Duration, error) {
	// If the user input a raw number, assume minutes
	if isDigit(raw) {
		raw = raw + "m"
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, err
	}
	return d, nil
}

func isDigit(s string) bool {
	return strings.IndexFunc(s, func(c rune) bool {
		return c < '0' || c > '9'
	}) == -1
}

// parseTime attempts to parse a time (no date) from the given string using a number of layouts.
func parseTime(s string) (time.Time, error) {
	// Try a number of possible layouts.
	for _, layout := range []string{
		time.Kitchen, // 03:04PM
		"03:04pm",
		"3:04PM",
		"3:04pm",
		"15:04",
		"1504",
		"03PM",
		"03pm",
		"3PM",
		"3pm",
	} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, errInvalidTimeFormat
}
