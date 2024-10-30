package dormancy_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"cdr.dev/slog/sloggers/slogtest"

	"github.com/coder/coder/v2/coderd/audit"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbmem"
	"github.com/coder/coder/v2/enterprise/coderd/dormancy"
	"github.com/coder/coder/v2/testutil"
)

func TestCheckInactiveUsers(t *testing.T) {
	t.Parallel()

	// Predefine job settings
	interval := time.Millisecond
	dormancyPeriod := 90 * 24 * time.Hour

	// Add some dormant accounts
	logger := slogtest.Make(t, &slogtest.Options{IgnoreErrors: true})
	db := dbmem.New()

	ctx, cancelFunc := context.WithCancel(context.Background())
	t.Cleanup(cancelFunc)

	inactiveUser1 := setupUser(ctx, t, db, "dormant-user-1@coder.com", database.UserStatusActive, time.Now().Add(-dormancyPeriod).Add(-time.Minute))
	inactiveUser2 := setupUser(ctx, t, db, "dormant-user-2@coder.com", database.UserStatusActive, time.Now().Add(-dormancyPeriod).Add(-time.Hour))
	inactiveUser3 := setupUser(ctx, t, db, "dormant-user-3@coder.com", database.UserStatusActive, time.Now().Add(-dormancyPeriod).Add(-6*time.Hour))

	activeUser1 := setupUser(ctx, t, db, "active-user-1@coder.com", database.UserStatusActive, time.Now().Add(-dormancyPeriod).Add(time.Minute))
	activeUser2 := setupUser(ctx, t, db, "active-user-2@coder.com", database.UserStatusActive, time.Now().Add(-dormancyPeriod).Add(time.Hour))
	activeUser3 := setupUser(ctx, t, db, "active-user-3@coder.com", database.UserStatusActive, time.Now().Add(-dormancyPeriod).Add(6*time.Hour))

	suspendedUser1 := setupUser(ctx, t, db, "suspended-user-1@coder.com", database.UserStatusSuspended, time.Now().Add(-dormancyPeriod).Add(-time.Minute))
	suspendedUser2 := setupUser(ctx, t, db, "suspended-user-2@coder.com", database.UserStatusSuspended, time.Now().Add(-dormancyPeriod).Add(-time.Hour))
	suspendedUser3 := setupUser(ctx, t, db, "suspended-user-3@coder.com", database.UserStatusSuspended, time.Now().Add(-dormancyPeriod).Add(-6*time.Hour))

	mAudit := audit.NewMock()
	// Run the periodic job
	closeFunc := dormancy.CheckInactiveUsersWithOptions(ctx, logger, db, mAudit, interval, dormancyPeriod)
	t.Cleanup(closeFunc)

	var rows []database.GetUsersRow
	var err error
	require.Eventually(t, func() bool {
		rows, err = db.GetUsers(ctx, database.GetUsersParams{})
		if err != nil {
			return false
		}

		var dormant, suspended int
		for _, row := range rows {
			if row.Status == database.UserStatusDormant {
				dormant++
			} else if row.Status == database.UserStatusSuspended {
				suspended++
			}
		}
		// 6 users in total, 3 dormant, 3 suspended
		return len(rows) == 9 && dormant == 3 && suspended == 3
	}, testutil.WaitShort, testutil.IntervalMedium)

	require.Len(t, mAudit.AuditLogs(), 3)

	allUsers := ignoreUpdatedAt(database.ConvertUserRows(rows))

	// Verify user status
	expectedUsers := []database.User{
		asDormant(inactiveUser1),
		asDormant(inactiveUser2),
		asDormant(inactiveUser3),
		activeUser1,
		activeUser2,
		activeUser3,
		suspendedUser1,
		suspendedUser2,
		suspendedUser3,
	}
	require.ElementsMatch(t, allUsers, expectedUsers)
}

func setupUser(ctx context.Context, t *testing.T, db database.Store, email string, status database.UserStatus, lastSeenAt time.Time) database.User {
	t.Helper()

	user, err := db.InsertUser(ctx, database.InsertUserParams{ID: uuid.New(), LoginType: database.LoginTypePassword, Username: uuid.NewString()[:8], Email: email})
	require.NoError(t, err)
	// At the beginning of the test all users are marked as active
	user, err = db.UpdateUserStatus(ctx, database.UpdateUserStatusParams{ID: user.ID, Status: status})
	require.NoError(t, err)
	user, err = db.UpdateUserLastSeenAt(ctx, database.UpdateUserLastSeenAtParams{ID: user.ID, LastSeenAt: lastSeenAt})
	require.NoError(t, err)
	return user
}

func asDormant(user database.User) database.User {
	user.Status = database.UserStatusDormant
	return user
}

func ignoreUpdatedAt(rows []database.User) []database.User {
	for i := range rows {
		rows[i].UpdatedAt = time.Time{}
	}
	return rows
}
