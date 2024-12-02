package schedule

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/onchainengineering/hmi-computer/v2/coderd/database"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/coderd/schedule/cron"
)

var ErrUserCannotSetQuietHoursSchedule = xerrors.New("user cannot set custom quiet hours schedule due to deployment configuration")

type UserQuietHoursScheduleOptions struct {
	// Schedule is the cron schedule to use for quiet hours windows for all
	// workspaces owned by the user.
	//
	// This value will be set to the parsed custom schedule of the user. If the
	// user doesn't have a custom schedule set, it will be set to the default
	// schedule (and UserSet will be false). If quiet hours schedules are not
	// entitled or disabled instance-wide, this value will be nil to denote that
	// quiet hours windows should not be used.
	Schedule *cron.Schedule
	// UserSet is true if the user has set a custom schedule, false if the
	// default schedule is being used.
	UserSet bool
	// UserCanSet is true if the user is allowed to set a custom schedule. If
	// false, the user cannot set a custom schedule and the default schedule
	// will always be used.
	UserCanSet bool
}

type UserQuietHoursScheduleStore interface {
	// Get retrieves the quiet hours schedule for the given user. If the user
	// has not set a custom schedule, the default schedule will be returned. If
	// quiet hours schedules are not entitled or disabled instance-wide, this
	// will return a nil schedule.
	Get(ctx context.Context, db database.Store, userID uuid.UUID) (UserQuietHoursScheduleOptions, error)
	// Set sets the quiet hours schedule for the given user. If the given
	// schedule is an empty string, the user's custom schedule will be cleared
	// and the default schedule will be used from now on. If quiet hours
	// schedules are not entitled or disabled instance-wide, this will do
	// nothing and return a nil schedule.
	Set(ctx context.Context, db database.Store, userID uuid.UUID, rawSchedule string) (UserQuietHoursScheduleOptions, error)
}

type agplUserQuietHoursScheduleStore struct{}

var _ UserQuietHoursScheduleStore = &agplUserQuietHoursScheduleStore{}

func NewAGPLUserQuietHoursScheduleStore() UserQuietHoursScheduleStore {
	return &agplUserQuietHoursScheduleStore{}
}

func (*agplUserQuietHoursScheduleStore) Get(_ context.Context, _ database.Store, _ uuid.UUID) (UserQuietHoursScheduleOptions, error) {
	// User quiet hours windows are not supported in AGPL.
	return UserQuietHoursScheduleOptions{
		Schedule:   nil,
		UserSet:    false,
		UserCanSet: false,
	}, nil
}

func (*agplUserQuietHoursScheduleStore) Set(_ context.Context, _ database.Store, _ uuid.UUID, _ string) (UserQuietHoursScheduleOptions, error) {
	return UserQuietHoursScheduleOptions{}, ErrUserCannotSetQuietHoursSchedule
}
