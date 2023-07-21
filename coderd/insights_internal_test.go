package coderd

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseInsightsStartAndEndTime(t *testing.T) {
	t.Parallel()

	layout := insightsTimeLayout
	now := time.Now().UTC()
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	thisHour := time.Date(y, m, d, now.Hour(), 0, 0, 0, time.UTC)
	thisHourRoundUp := thisHour.Add(time.Hour)

	helsinki, err := time.LoadLocation("Europe/Helsinki")
	require.NoError(t, err)

	type args struct {
		startTime string
		endTime   string
	}
	tests := []struct {
		name          string
		args          args
		wantStartTime time.Time
		wantEndTime   time.Time
		wantOk        bool
	}{
		{
			name: "Week",
			args: args{
				startTime: "2023-07-10T00:00:00Z",
				endTime:   "2023-07-17T00:00:00Z",
			},
			wantStartTime: time.Date(2023, 7, 10, 0, 0, 0, 0, time.UTC),
			wantEndTime:   time.Date(2023, 7, 17, 0, 0, 0, 0, time.UTC),
			wantOk:        true,
		},
		{
			name: "Today",
			args: args{
				startTime: today.Format(layout),
				endTime:   thisHour.Format(layout),
			},
			wantStartTime: time.Date(2023, 7, today.Day(), 0, 0, 0, 0, time.UTC),
			wantEndTime:   time.Date(2023, 7, today.Day(), thisHour.Hour(), 0, 0, 0, time.UTC),
			wantOk:        true,
		},
		{
			name: "Today with minutes and seconds",
			args: args{
				startTime: today.Format(layout),
				endTime:   thisHour.Add(time.Minute + time.Second).Format(layout),
			},
			wantOk: false,
		},
		{
			name: "Today (hour round up)",
			args: args{
				startTime: today.Format(layout),
				endTime:   thisHourRoundUp.Format(layout),
			},
			wantStartTime: time.Date(2023, 7, today.Day(), 0, 0, 0, 0, time.UTC),
			wantEndTime:   time.Date(2023, 7, today.Day(), thisHourRoundUp.Hour(), 0, 0, 0, time.UTC),
			wantOk:        true,
		},
		{
			name: "Other timezone week",
			args: args{
				startTime: "2023-07-10T00:00:00+03:00",
				endTime:   "2023-07-17T00:00:00+03:00",
			},
			wantStartTime: time.Date(2023, 7, 10, 0, 0, 0, 0, helsinki),
			wantEndTime:   time.Date(2023, 7, 17, 0, 0, 0, 0, helsinki),
			wantOk:        true,
		},
		{
			name: "Mixed timezone week",
			args: args{
				startTime: "2023-07-10T00:00:00Z",
				endTime:   "2023-07-17T00:00:00+03:00",
			},
			wantOk: false,
		},
		{
			name: "Bad format",
			args: args{
				startTime: "2023-07-10",
				endTime:   "2023-07-17",
			},
			wantOk: false,
		},
		{
			name: "Zero time",
			args: args{
				startTime: (time.Time{}).Format(layout),
				endTime:   (time.Time{}).Format(layout),
			},
			wantOk: false,
		},
		{
			name: "Time in future",
			args: args{
				startTime: today.AddDate(0, 0, 1).Format(layout),
				endTime:   today.AddDate(0, 0, 2).Format(layout),
			},
			wantOk: false,
		},
		{
			name: "End before start",
			args: args{
				startTime: today.Format(layout),
				endTime:   today.AddDate(0, 0, -1).Format(layout),
			},
			wantOk: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rw := httptest.NewRecorder()
			gotStartTime, gotEndTime, gotOk := parseInsightsStartAndEndTime(context.Background(), rw, tt.args.startTime, tt.args.endTime)

			// assert.Equal is unable to test location equality, so we
			// use assert.WithinDuration.
			assert.Equal(t, tt.wantOk, gotOk)
			assert.WithinDuration(t, tt.wantStartTime, gotStartTime, 0)
			assert.True(t, tt.wantStartTime.Equal(gotStartTime))
			assert.WithinDuration(t, tt.wantEndTime, gotEndTime, 0)
			assert.True(t, tt.wantEndTime.Equal(gotEndTime))
		})
	}
}
