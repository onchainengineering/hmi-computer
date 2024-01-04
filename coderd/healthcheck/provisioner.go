package healthcheck

import (
	"context"
	"time"

	"golang.org/x/mod/semver"

	"github.com/coder/coder/v2/buildinfo"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/healthcheck/health"
	"github.com/coder/coder/v2/coderd/provisionerdserver"
	"github.com/coder/coder/v2/coderd/util/apiversion"
	"github.com/coder/coder/v2/coderd/util/ptr"
	"github.com/coder/coder/v2/codersdk"
)

// @typescript-generate ProvisionerDaemonsReport
type ProvisionerDaemonsReport struct {
	Severity  health.Severity  `json:"severity"`
	Warnings  []health.Message `json:"warnings"`
	Dismissed bool             `json:"dismissed"`
	Error     *string

	Provisioners []codersdk.ProvisionerDaemon
}

type ProvisionerDaemonsReportOptions struct {
	CurrentVersion    string
	CurrentAPIVersion *apiversion.APIVersion

	// ProvisionerDaemonsFn is a function that returns ProvisionerDaemons.
	// Satisfied by database.Store.ProvisionerDaemons
	ProvisionerDaemonsFn func(context.Context) ([]database.ProvisionerDaemon, error)

	TimeNowFn     func() time.Time
	StaleInterval time.Duration

	Dismissed bool
}

func (r *ProvisionerDaemonsReport) Run(ctx context.Context, opts *ProvisionerDaemonsReportOptions) {
	r.Severity = health.SeverityOK
	r.Warnings = make([]health.Message, 0)
	r.Dismissed = opts.Dismissed
	now := opts.TimeNowFn()
	if opts.StaleInterval == 0 {
		opts.StaleInterval = provisionerdserver.DefaultHeartbeatInterval * 3
	}

	if opts.CurrentVersion == "" {
		r.Severity = health.SeverityError
		r.Error = ptr.Ref("Developer error: CurrentVersion is empty!")
		return
	}

	if opts.CurrentAPIVersion == nil {
		r.Severity = health.SeverityError
		r.Error = ptr.Ref("Developer error: CurrentAPIVersion is nil!")
		return
	}

	if opts.ProvisionerDaemonsFn == nil {
		r.Severity = health.SeverityError
		r.Error = ptr.Ref("Developer error: ProvisionerDaemonsFn is nil!")
		return
	}

	// nolint: gocritic // need an actor to fetch provisioner daemons
	daemons, err := opts.ProvisionerDaemonsFn(dbauthz.AsSystemRestricted(ctx))
	if err != nil {
		r.Severity = health.SeverityError
		r.Error = ptr.Ref("error fetching provisioner daemons: " + err.Error())
		return
	}

	if len(daemons) == 0 {
		r.Severity = health.SeverityError
		r.Error = ptr.Ref("No provisioner daemons found!")
		return
	}

	for _, daemon := range daemons {
		// Daemon never connected, skip.
		if !daemon.LastSeenAt.Valid {
			continue
		}
		// Daemon has gone away, skip.
		if now.Sub(daemon.LastSeenAt.Time) > (opts.StaleInterval) {
			continue
		}
		// For release versions, just check MAJOR.MINOR and ignore patch.
		if !semver.IsValid(daemon.Version) {
			if r.Severity.Value() < health.SeverityWarning.Value() {
				r.Severity = health.SeverityWarning
			}
			r.Warnings = append(r.Warnings, health.Messagef(health.CodeUnknown, "Provisioner daemon %q reports invalid version %q", opts.CurrentVersion, daemon.Version))
		} else if !buildinfo.VersionsMatch(opts.CurrentVersion, daemon.Version) {
			if r.Severity.Value() < health.SeverityWarning.Value() {
				r.Severity = health.SeverityWarning
			}
			r.Warnings = append(r.Warnings, health.Messagef(health.CodeProvisionerDaemonVersionMismatch, "Provisioner daemon %q has outdated version %q", daemon.Name, daemon.Version))
		}

		// Provisioner daemon API version follows different rules.
		if _, _, err := apiversion.Parse(daemon.APIVersion); err != nil {
			if r.Severity.Value() < health.SeverityError.Value() {
				r.Severity = health.SeverityError
			}
			r.Warnings = append(r.Warnings, health.Messagef(health.CodeUnknown, "Provisioner daemon %q reports invalid API version: %s", daemon.Name, err.Error()))
		} else if err := opts.CurrentAPIVersion.Validate(daemon.APIVersion); err != nil {
			if r.Severity.Value() < health.SeverityError.Value() {
				r.Severity = health.SeverityError
			}
			r.Warnings = append(r.Warnings, health.Messagef(health.CodeProvisionerDaemonAPIVersionIncompatible, "Provisioner daemon %q reports incompatible API version: %s", daemon.Name, err.Error()))
		}
	}
}
