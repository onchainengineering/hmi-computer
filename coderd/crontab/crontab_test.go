package crontab

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Parse(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name          string
		spec          string
		at            time.Time
		expectedNext  time.Time
		expectedError string
	}{
		{
			name:          "with timezone",
			spec:          "CRON_TZ=US/Central 30 9 1-5",
			at:            time.Date(2022, 4, 1, 14, 29, 0, 0, time.UTC),
			expectedNext:  time.Date(2022, 4, 1, 14, 30, 0, 0, time.UTC),
			expectedError: "",
		},
		{
			name:          "without timezone",
			spec:          "30 9 1-5",
			at:            time.Date(2022, 4, 1, 9, 29, 0, 0, time.Local),
			expectedNext:  time.Date(2022, 4, 1, 9, 30, 0, 0, time.Local),
			expectedError: "",
		},
		{
			name:          "invalid schedule",
			spec:          "asdfasdfasdfsd",
			at:            time.Time{},
			expectedNext:  time.Time{},
			expectedError: "parse schedule: expected exactly 3 fields, found 1: [asdfasdfasdfsd]",
		},
		{
			name:          "invalid location",
			spec:          "CRON_TZ=Fictional/Country 30 9 1-5",
			at:            time.Time{},
			expectedNext:  time.Time{},
			expectedError: "parse schedule: provided bad location Fictional/Country: unknown time zone Fictional/Country",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual, err := Parse(tc.spec)
			if tc.expectedError == "" {
				nextTime := actual.Next(tc.at)
				require.NoError(t, err)
				require.Equal(t, tc.expectedNext, nextTime)
				require.Equal(t, tc.spec, actual.String())
			} else {
				require.EqualError(t, err, tc.expectedError)
				require.Nil(t, actual)
			}
		})
	}
}
