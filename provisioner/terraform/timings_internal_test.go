//go:build linux || darwin

package terraform

import (
	"bufio"
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/txtar"

	"github.com/coder/coder/v2/coderd/database"
	terraform_internal "github.com/coder/coder/v2/provisioner/terraform/internal"
	"github.com/coder/coder/v2/provisionersdk/proto"
)

var (
	//go:embed testdata/timings-aggregation/simple.txtar
	inputSimple []byte
	//go:embed testdata/timings-aggregation/init.txtar
	inputInit []byte
	//go:embed testdata/timings-aggregation/error.txtar
	inputError []byte
	//go:embed testdata/timings-aggregation/complete.txtar
	inputComplete []byte
	//go:embed testdata/timings-aggregation/incomplete.txtar
	inputIncomplete []byte
	//go:embed testdata/timings-aggregation/faster-than-light.txtar
	inputFasterThanLight []byte
)

func TestAggregation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "init",
			input: inputInit,
		},
		{
			name:  "simple",
			input: inputSimple,
		},
		{
			name:  "error",
			input: inputError,
		},
		{
			name:  "complete",
			input: inputComplete,
		},
		{
			name:  "incomplete",
			input: inputIncomplete,
		},
		{
			name:  "faster-than-light",
			input: inputFasterThanLight,
		},
	}

	// nolint:paralleltest // Not since go v1.22.
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// txtar is a text-based archive format used in the stdlib for simple and elegant tests.
			//
			// We ALWAYS expect that the archive contains two or more "files":
			//   1. JSON logs generated by a terraform execution, one per line, *one file per stage*
			//   N. Expected resulting timings in JSON form, one per line
			arc := txtar.Parse(tc.input)
			require.GreaterOrEqual(t, len(arc.Files), 2)

			t.Logf("%s: %s", t.Name(), arc.Comment)

			var actualTimings []*proto.Timing

			// The last "file" MUST contain the expected timings.
			expectedTimings := arc.Files[len(arc.Files)-1]

			// Iterate over the initial "files" and extract their timings according to their stage.
			for i := 0; i < len(arc.Files)-1; i++ {
				file := arc.Files[i]
				stage := database.ProvisionerJobTimingStage(file.Name)
				require.Truef(t, stage.Valid(), "%q is not a valid stage name; acceptable values: %v",
					file.Name, database.AllProvisionerJobTimingStageValues())

				agg := newTimingAggregator(stage)
				ingestAllSpans(t, file.Data, agg)
				actualTimings = append(actualTimings, agg.aggregate()...)
			}

			// Ensure that the expected timings were produced.
			expected := terraform_internal.ParseTimingLines(t, expectedTimings.Data)
			terraform_internal.StableSortTimings(t, actualTimings) // To reduce flakiness.
			if !assert.True(t, terraform_internal.TimingsAreEqual(t, expected, actualTimings)) {
				printExpectation(t, expected)
			}
		})
	}
}

func ingestAllSpans(t *testing.T, input []byte, aggregator *timingAggregator) {
	t.Helper()

	scanner := bufio.NewScanner(bytes.NewBuffer(input))
	for scanner.Scan() {
		line := scanner.Bytes()
		log := parseTerraformLogLine(line)
		if log == nil {
			continue
		}

		ts, span, err := extractTimingSpan(log)
		if err != nil {
			// t.Logf("%s: failed span extraction on line: %q", err, line)
			continue
		}

		require.NotZerof(t, ts, "failed on line: %q", line)
		require.NotNilf(t, span, "failed on line: %q", line)

		aggregator.ingest(ts, span)
	}

	require.NoError(t, scanner.Err())
}

func printExpectation(t *testing.T, actual []*proto.Timing) {
	t.Helper()

	t.Log("expected:")
	for _, a := range actual {
		terraform_internal.PrintTiming(t, a)
	}
}
