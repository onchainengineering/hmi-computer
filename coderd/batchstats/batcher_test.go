package batchstats_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"

	"github.com/coder/coder/coderd/batchstats"
	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/database/dbgen"
	"github.com/coder/coder/coderd/database/dbtestutil"
	"github.com/coder/coder/coderd/rbac"
	"github.com/coder/coder/codersdk/agentsdk"
	"github.com/coder/coder/cryptorand"
)

func TestBatchStats(t *testing.T) {
	t.Parallel()

	batchSize := batchstats.DefaultBatchSize

	// Given: a fresh batcher with no data
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	log := slogtest.Make(t, &slogtest.Options{IgnoreErrors: true}).Leveled(slog.LevelDebug)
	store, _ := dbtestutil.NewDB(t)

	// Set up some test dependencies.
	deps1 := setupDeps(t, store)
	deps2 := setupDeps(t, store)
	tick := make(chan time.Time)
	flushed := make(chan bool)

	b, err := batchstats.New(
		batchstats.WithStore(store),
		batchstats.WithBatchSize(batchSize),
		batchstats.WithLogger(log),
		batchstats.WithTicker(tick),
		batchstats.WithFlushed(flushed),
	)
	require.NoError(t, err)

	// Given: no data points are added for workspace
	// When: it becomes time to report stats
	done := make(chan struct{})
	t.Cleanup(func() {
		close(done)
	})
	go func() {
		b.Run(ctx)
	}()
	t1 := time.Now()
	// Signal a tick and wait for a flush to complete.
	tick <- t1
	f := <-flushed
	require.False(t, f, "flush should not have been forced")
	t.Logf("flush 1 completed")

	// Then: it should report no stats.
	stats, err := store.GetWorkspaceAgentStats(ctx, t1)
	require.NoError(t, err, "should not error getting stats")
	require.Empty(t, stats, "should have no stats for workspace")

	// Given: a single data point is added for workspace
	t2 := time.Now()
	t.Logf("inserting 1 stat")
	require.NoError(t, b.Add(deps1.Agent.ID, deps1.User.ID, deps1.Template.ID, deps1.Workspace.ID, randAgentSDKStats(t)))

	// When: it becomes time to report stats
	// Signal a tick and wait for a flush to complete.
	tick <- t2
	f = <-flushed // Wait for a flush to complete.
	require.False(t, f, "flush should not have been forced")
	t.Logf("flush 2 completed")

	// Then: it should report a single stat.
	stats, err = store.GetWorkspaceAgentStats(ctx, t2)
	require.NoError(t, err, "should not error getting stats")
	require.Len(t, stats, 1, "should have stats for workspace")

	// Given: a lot of data points are added for both workspaces
	// (equal to batch size)
	t3 := time.Now()
	t.Logf("inserting %d stats", batchSize)
	for i := 0; i < batchSize; i++ {
		if i%2 == 0 {
			require.NoError(t, b.Add(deps1.Agent.ID, deps1.User.ID, deps1.Template.ID, deps1.Workspace.ID, randAgentSDKStats(t)))
		} else {
			require.NoError(t, b.Add(deps2.Agent.ID, deps2.User.ID, deps2.Template.ID, deps2.Workspace.ID, randAgentSDKStats(t)))
		}
	}

	// When: the buffer is full
	// Wait for a flush to complete. This should be forced by filling the buffer.
	f = <-flushed
	require.True(t, f, "flush should have been forced")
	t.Logf("flush 3 completed")

	// Then: it should immediately flush its stats to store.
	stats, err = store.GetWorkspaceAgentStats(ctx, t3)
	require.NoError(t, err, "should not error getting stats")
	require.Len(t, stats, 2, "should have stats for both workspaces")

	// Ensure that a subsequent flush does not push stale data.
	t4 := time.Now()
	tick <- t4
	f = <-flushed
	require.False(t, f, "flush should not have been forced")
	t.Logf("flush 4 completed")

	stats, err = store.GetWorkspaceAgentStats(ctx, t4)
	require.NoError(t, err, "should not error getting stats")
	require.Len(t, stats, 0, "should have no stats for workspace")
}

// randAgentSDKStats returns a random agentsdk.Stats
func randAgentSDKStats(t *testing.T, opts ...func(*agentsdk.Stats)) agentsdk.Stats {
	t.Helper()
	s := agentsdk.Stats{
		ConnectionsByProto: map[string]int64{
			"ssh":              mustRandInt64n(t, 9) + 1,
			"vscode":           mustRandInt64n(t, 9) + 1,
			"jetbrains":        mustRandInt64n(t, 9) + 1,
			"reconnecting_pty": mustRandInt64n(t, 9) + 1,
		},
		ConnectionCount:             mustRandInt64n(t, 99) + 1,
		ConnectionMedianLatencyMS:   float64(mustRandInt64n(t, 99) + 1),
		RxPackets:                   mustRandInt64n(t, 99) + 1,
		RxBytes:                     mustRandInt64n(t, 99) + 1,
		TxPackets:                   mustRandInt64n(t, 99) + 1,
		TxBytes:                     mustRandInt64n(t, 99) + 1,
		SessionCountVSCode:          mustRandInt64n(t, 9) + 1,
		SessionCountJetBrains:       mustRandInt64n(t, 9) + 1,
		SessionCountReconnectingPTY: mustRandInt64n(t, 9) + 1,
		SessionCountSSH:             mustRandInt64n(t, 9) + 1,
		Metrics:                     []agentsdk.AgentMetric{},
	}
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

// deps is a set of test dependencies.
type deps struct {
	Agent     database.WorkspaceAgent
	Template  database.Template
	User      database.User
	Workspace database.Workspace
}

// setupDeps sets up a set of test dependencies.
// It creates an organization, user, template, workspace, and agent
// along with all the other miscellaneous plumbing required to link
// them together.
func setupDeps(t *testing.T, store database.Store) deps {
	t.Helper()

	org := dbgen.Organization(t, store, database.Organization{})
	user := dbgen.User(t, store, database.User{})
	_, err := store.InsertOrganizationMember(context.Background(), database.InsertOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Roles:          []string{rbac.RoleOrgMember(org.ID)},
	})
	require.NoError(t, err)
	tv := dbgen.TemplateVersion(t, store, database.TemplateVersion{
		OrganizationID: org.ID,
		CreatedBy:      user.ID,
	})
	tpl := dbgen.Template(t, store, database.Template{
		CreatedBy:       user.ID,
		OrganizationID:  org.ID,
		ActiveVersionID: tv.ID,
	})
	ws := dbgen.Workspace(t, store, database.Workspace{
		TemplateID:     tpl.ID,
		OwnerID:        user.ID,
		OrganizationID: org.ID,
		LastUsedAt:     time.Now().Add(-time.Hour),
	})
	pj := dbgen.ProvisionerJob(t, store, database.ProvisionerJob{
		InitiatorID:    user.ID,
		OrganizationID: org.ID,
	})
	_ = dbgen.WorkspaceBuild(t, store, database.WorkspaceBuild{
		TemplateVersionID: tv.ID,
		WorkspaceID:       ws.ID,
		JobID:             pj.ID,
	})
	res := dbgen.WorkspaceResource(t, store, database.WorkspaceResource{
		Transition: database.WorkspaceTransitionStart,
		JobID:      pj.ID,
	})
	agt := dbgen.WorkspaceAgent(t, store, database.WorkspaceAgent{
		ResourceID: res.ID,
	})
	return deps{
		Agent:     agt,
		Template:  tpl,
		User:      user,
		Workspace: ws,
	}
}

// mustRandInt64n returns a random int64 in the range [0, n).
func mustRandInt64n(t *testing.T, n int64) int64 {
	t.Helper()
	i, err := cryptorand.Intn(int(n))
	require.NoError(t, err)
	return int64(i)
}
