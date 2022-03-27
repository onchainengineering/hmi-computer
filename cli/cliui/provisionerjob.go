package cliui

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/codersdk"
)

func WorkspaceBuild(cmd *cobra.Command, client *codersdk.Client, build uuid.UUID, before time.Time) error {
	return ProvisionerJob(cmd, ProvisionerJobOptions{
		Fetch: func() (codersdk.ProvisionerJob, error) {
			build, err := client.WorkspaceBuild(cmd.Context(), build)
			return build.Job, err
		},
		Cancel: func() error {
			return client.CancelWorkspaceBuild(cmd.Context(), build)
		},
		Logs: func() (<-chan codersdk.ProvisionerJobLog, error) {
			return client.WorkspaceBuildLogsAfter(cmd.Context(), build, before)
		},
	})
}

type ProvisionerJobOptions struct {
	Fetch  func() (codersdk.ProvisionerJob, error)
	Cancel func() error
	Logs   func() (<-chan codersdk.ProvisionerJobLog, error)

	FetchInterval time.Duration
}

// ProvisionerJob renders a provisioner job with interactive cancellation.
func ProvisionerJob(cmd *cobra.Command, opts ProvisionerJobOptions) error {
	if opts.FetchInterval == 0 {
		opts.FetchInterval = time.Second
	}

	var (
		currentStage          = "Queued"
		currentStageStartedAt = time.Now().UTC()
		didLogBetweenStage    = false
		ctx, cancelFunc       = context.WithCancel(cmd.Context())

		errChan  = make(chan error)
		job      codersdk.ProvisionerJob
		jobMutex sync.Mutex
	)
	defer cancelFunc()

	printStage := func() {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), Styles.Prompt.Render("⧗")+"%s\n", Styles.Field.Render(currentStage))
	}
	printStage()

	updateStage := func(stage string, startedAt time.Time) {
		if currentStage != "" {
			prefix := ""
			if !didLogBetweenStage {
				prefix = "\033[1A\r"
			}
			mark := Styles.Checkmark
			if job.CompletedAt != nil && job.Status != codersdk.ProvisionerJobSucceeded {
				mark = Styles.Crossmark
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), prefix+mark.String()+Styles.Placeholder.Render(" %s [%dms]")+"\n", currentStage, startedAt.Sub(currentStageStartedAt).Milliseconds())
		}
		if stage == "" {
			return
		}
		currentStage = stage
		currentStageStartedAt = startedAt
		didLogBetweenStage = false
		printStage()
	}

	updateJob := func() {
		var err error
		jobMutex.Lock()
		defer jobMutex.Unlock()
		job, err = opts.Fetch()
		if err != nil {
			errChan <- xerrors.Errorf("fetch: %w", err)
			return
		}
		if job.StartedAt == nil {
			return
		}
		if currentStage != "Queued" {
			// If another stage is already running, there's no need
			// for us to notify the user we're running!
			return
		}
		updateStage("Running", *job.StartedAt)
	}
	updateJob()

	// Handles ctrl+c to cancel a job.
	stopChan := make(chan os.Signal, 1)
	defer signal.Stop(stopChan)
	go func() {
		signal.Notify(stopChan, os.Interrupt)
		select {
		case <-ctx.Done():
			return
		case _, ok := <-stopChan:
			if !ok {
				return
			}
		}
		// Stop listening for signals so another one kills it!
		signal.Stop(stopChan)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\033[2K\r\n"+Styles.FocusedPrompt.String()+Styles.Bold.Render("Gracefully canceling... wait for exit or data loss may occur!")+"\n\n")
		err := opts.Cancel()
		if err != nil {
			errChan <- xerrors.Errorf("cancel: %w", err)
			return
		}
		updateJob()
	}()

	logs, err := opts.Logs()
	if err != nil {
		return xerrors.Errorf("logs: %w", err)
	}

	ticker := time.NewTicker(opts.FetchInterval)
	for {
		select {
		case err = <-errChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			updateJob()
		case log, ok := <-logs:
			if !ok {
				// The logs stream will end when the job does,
				// so it's safe to
				updateJob()
				jobMutex.Lock()
				if job.CompletedAt != nil {
					updateStage("", *job.CompletedAt)
				}
				switch job.Status {
				case codersdk.ProvisionerJobCanceled:
					jobMutex.Unlock()
					return Canceled
				case codersdk.ProvisionerJobSucceeded:
					jobMutex.Unlock()
					return nil
				case codersdk.ProvisionerJobFailed:
				}
				jobMutex.Unlock()
				return xerrors.New(job.Error)
			}
			output := ""
			switch log.Level {
			case database.LogLevelTrace, database.LogLevelDebug, database.LogLevelError:
				output = defaultStyles.Error.Render(log.Output)
			case database.LogLevelWarn:
				output = Styles.Warn.Render(log.Output)
			case database.LogLevelInfo:
				output = log.Output
			}
			if log.Stage != currentStage && log.Stage != "" {
				jobMutex.Lock()
				updateStage(log.Stage, log.CreatedAt)
				jobMutex.Unlock()
				continue
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", Styles.Placeholder.Render(" "), output)
			didLogBetweenStage = true
		}
	}
}
