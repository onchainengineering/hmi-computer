package tailnet_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/exp/slices"
	"golang.org/x/xerrors"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/database/dbmock"
	"github.com/coder/coder/coderd/database/dbtestutil"
	"github.com/coder/coder/coderd/database/pubsub"
	"github.com/coder/coder/enterprise/tailnet"
	agpl "github.com/coder/coder/tailnet"
	"github.com/coder/coder/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestPGCoordinatorSingle_ClientWithoutAgent(t *testing.T) {
	t.Parallel()
	if !dbtestutil.WillUsePostgres() {
		t.Skip("test only with postgres")
	}
	store, ps := dbtestutil.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitSuperLong)
	defer cancel()
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	coordinator, err := tailnet.NewPGCoord(ctx, logger, ps, store)
	require.NoError(t, err)
	defer coordinator.Close()

	agentID := uuid.New()
	client := newTestClient(t, coordinator, agentID)
	defer client.close()
	client.sendNode(&agpl.Node{PreferredDERP: 10})
	require.Eventually(t, func() bool {
		clients, err := store.GetTailnetClientsForAgent(ctx, agentID)
		if err != nil && !xerrors.Is(err, sql.ErrNoRows) {
			t.Fatalf("database error: %v", err)
		}
		if len(clients) == 0 {
			return false
		}
		var node agpl.Node
		err = json.Unmarshal(clients[0].Node, &node)
		assert.NoError(t, err)
		assert.Equal(t, 10, node.PreferredDERP)
		return true
	}, testutil.WaitShort, testutil.IntervalFast)

	err = client.close()
	require.NoError(t, err)
	<-client.errChan
	<-client.closeChan
	assertEventuallyNoClientsForAgent(ctx, t, store, agentID)
}

func TestPGCoordinatorSingle_AgentWithoutClients(t *testing.T) {
	t.Parallel()
	if !dbtestutil.WillUsePostgres() {
		t.Skip("test only with postgres")
	}
	store, ps := dbtestutil.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitSuperLong)
	defer cancel()
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	coordinator, err := tailnet.NewPGCoord(ctx, logger, ps, store)
	require.NoError(t, err)
	defer coordinator.Close()

	agent := newTestAgent(t, coordinator, "agent")
	defer agent.close()
	agent.sendNode(&agpl.Node{PreferredDERP: 10})
	require.Eventually(t, func() bool {
		agents, err := store.GetTailnetAgents(ctx, agent.id)
		if err != nil && !xerrors.Is(err, sql.ErrNoRows) {
			t.Fatalf("database error: %v", err)
		}
		if len(agents) == 0 {
			return false
		}
		var node agpl.Node
		err = json.Unmarshal(agents[0].Node, &node)
		assert.NoError(t, err)
		assert.Equal(t, 10, node.PreferredDERP)
		return true
	}, testutil.WaitShort, testutil.IntervalFast)
	err = agent.close()
	require.NoError(t, err)
	<-agent.errChan
	<-agent.closeChan
	assertEventuallyNoAgents(ctx, t, store, agent.id)
}

func TestPGCoordinatorSingle_AgentWithClient(t *testing.T) {
	t.Parallel()
	if !dbtestutil.WillUsePostgres() {
		t.Skip("test only with postgres")
	}
	store, ps := dbtestutil.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitSuperLong)
	defer cancel()
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	coordinator, err := tailnet.NewPGCoord(ctx, logger, ps, store)
	require.NoError(t, err)
	defer coordinator.Close()

	agent := newTestAgent(t, coordinator, "original")
	defer agent.close()
	agent.sendNode(&agpl.Node{PreferredDERP: 10})

	client := newTestClient(t, coordinator, agent.id)
	defer client.close()

	agentNodes := client.recvNodes(ctx, t)
	require.Len(t, agentNodes, 1)
	assert.Equal(t, 10, agentNodes[0].PreferredDERP)
	client.sendNode(&agpl.Node{PreferredDERP: 11})
	clientNodes := agent.recvNodes(ctx, t)
	require.Len(t, clientNodes, 1)
	assert.Equal(t, 11, clientNodes[0].PreferredDERP)

	// Ensure an update to the agent node reaches the connIO!
	agent.sendNode(&agpl.Node{PreferredDERP: 12})
	agentNodes = client.recvNodes(ctx, t)
	require.Len(t, agentNodes, 1)
	assert.Equal(t, 12, agentNodes[0].PreferredDERP)

	// Close the agent WebSocket so a new one can connect.
	err = agent.close()
	require.NoError(t, err)
	_ = agent.recvErr(ctx, t)
	agent.waitForClose(ctx, t)

	// Create a new agent connection. This is to simulate a reconnect!
	agent = newTestAgent(t, coordinator, "reconnection", agent.id)
	// Ensure the existing listening connIO sends its node immediately!
	clientNodes = agent.recvNodes(ctx, t)
	require.Len(t, clientNodes, 1)
	assert.Equal(t, 11, clientNodes[0].PreferredDERP)

	// Send a bunch of updates in rapid succession, and test that we eventually get the latest.  We don't want the
	// coordinator accidentally reordering things.
	for d := 13; d < 36; d++ {
		agent.sendNode(&agpl.Node{PreferredDERP: d})
	}
	for {
		nodes := client.recvNodes(ctx, t)
		if !assert.Len(t, nodes, 1) {
			break
		}
		if nodes[0].PreferredDERP == 35 {
			// got latest!
			break
		}
	}

	err = agent.close()
	require.NoError(t, err)
	_ = agent.recvErr(ctx, t)
	agent.waitForClose(ctx, t)

	err = client.close()
	require.NoError(t, err)
	_ = client.recvErr(ctx, t)
	client.waitForClose(ctx, t)

	assertEventuallyNoAgents(ctx, t, store, agent.id)
	assertEventuallyNoClientsForAgent(ctx, t, store, agent.id)
}

func TestPGCoordinatorSingle_MissedHeartbeats(t *testing.T) {
	t.Parallel()
	if !dbtestutil.WillUsePostgres() {
		t.Skip("test only with postgres")
	}
	store, ps := dbtestutil.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitSuperLong)
	defer cancel()
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	coordinator, err := tailnet.NewPGCoord(ctx, logger, ps, store)
	require.NoError(t, err)
	defer coordinator.Close()

	agent := newTestAgent(t, coordinator, "agent")
	defer agent.close()
	agent.sendNode(&agpl.Node{PreferredDERP: 10})

	client := newTestClient(t, coordinator, agent.id)
	defer client.close()

	assertEventuallyHasDERPs(ctx, t, client, 10)
	client.sendNode(&agpl.Node{PreferredDERP: 11})
	assertEventuallyHasDERPs(ctx, t, agent, 11)

	// simulate a second coordinator via DB calls only --- our goal is to test broken heart-beating, so we can't use a
	// real coordinator
	fCoord2 := &fakeCoordinator{
		ctx:   ctx,
		t:     t,
		store: store,
		id:    uuid.New(),
	}
	// heatbeat until canceled
	ctx2, cancel2 := context.WithCancel(ctx)
	go func() {
		t := time.NewTicker(tailnet.HeartbeatPeriod)
		defer t.Stop()
		for {
			select {
			case <-ctx2.Done():
				return
			case <-t.C:
				fCoord2.heartbeat()
			}
		}
	}()
	fCoord2.heartbeat()
	fCoord2.agentNode(agent.id, &agpl.Node{PreferredDERP: 12})
	assertEventuallyHasDERPs(ctx, t, client, 12)

	fCoord3 := &fakeCoordinator{
		ctx:   ctx,
		t:     t,
		store: store,
		id:    uuid.New(),
	}
	start := time.Now()
	fCoord3.heartbeat()
	fCoord3.agentNode(agent.id, &agpl.Node{PreferredDERP: 13})
	assertEventuallyHasDERPs(ctx, t, client, 13)

	// when the fCoord3 misses enough heartbeats, the real coordinator should send an update with the
	// node from fCoord2 for the agent.
	assertEventuallyHasDERPs(ctx, t, client, 12)
	assert.Greater(t, time.Since(start), tailnet.HeartbeatPeriod*tailnet.MissedHeartbeats)

	// stop fCoord2 heartbeats, which should cause us to revert to the original agent mapping
	cancel2()
	assertEventuallyHasDERPs(ctx, t, client, 10)

	// send fCoord3 heartbeat, which should trigger us to consider that mapping valid again.
	fCoord3.heartbeat()
	assertEventuallyHasDERPs(ctx, t, client, 13)

	err = agent.close()
	require.NoError(t, err)
	_ = agent.recvErr(ctx, t)
	agent.waitForClose(ctx, t)

	err = client.close()
	require.NoError(t, err)
	_ = client.recvErr(ctx, t)
	client.waitForClose(ctx, t)

	assertEventuallyNoClientsForAgent(ctx, t, store, agent.id)
}

func TestPGCoordinatorSingle_SendsHeartbeats(t *testing.T) {
	t.Parallel()
	if !dbtestutil.WillUsePostgres() {
		t.Skip("test only with postgres")
	}
	store, ps := dbtestutil.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitSuperLong)
	defer cancel()
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)

	mu := sync.Mutex{}
	heartbeats := []time.Time{}
	unsub, err := ps.SubscribeWithErr(tailnet.EventHeartbeats, func(_ context.Context, msg []byte, err error) {
		assert.NoError(t, err)
		mu.Lock()
		defer mu.Unlock()
		heartbeats = append(heartbeats, time.Now())
	})
	require.NoError(t, err)
	defer unsub()

	start := time.Now()
	coordinator, err := tailnet.NewPGCoord(ctx, logger, ps, store)
	require.NoError(t, err)
	defer coordinator.Close()

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		if len(heartbeats) < 2 {
			return false
		}
		require.Greater(t, heartbeats[0].Sub(start), time.Duration(0))
		require.Greater(t, heartbeats[1].Sub(start), time.Duration(0))
		return assert.Greater(t, heartbeats[1].Sub(heartbeats[0]), tailnet.HeartbeatPeriod*9/10)
	}, testutil.WaitMedium, testutil.IntervalMedium)
}

// TestPGCoordinatorDual_Mainline tests with 2 coordinators, one agent connected to each, and 2 clients per agent.
//
//	            +---------+
//	agent1 ---> | coord1  | <--- client11 (coord 1, agent 1)
//	            |         |
//	            |         | <--- client12 (coord 1, agent 2)
//	            +---------+
//	            +---------+
//	agent2 ---> | coord2  | <--- client21 (coord 2, agent 1)
//	            |         |
//	            |         | <--- client22 (coord2, agent 2)
//	            +---------+
func TestPGCoordinatorDual_Mainline(t *testing.T) {
	t.Parallel()
	if !dbtestutil.WillUsePostgres() {
		t.Skip("test only with postgres")
	}
	store, ps := dbtestutil.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitSuperLong)
	defer cancel()
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	coord1, err := tailnet.NewPGCoord(ctx, logger.Named("coord1"), ps, store)
	require.NoError(t, err)
	defer coord1.Close()
	coord2, err := tailnet.NewPGCoord(ctx, logger.Named("coord2"), ps, store)
	require.NoError(t, err)
	defer coord2.Close()

	agent1 := newTestAgent(t, coord1, "agent1")
	defer agent1.close()
	agent2 := newTestAgent(t, coord2, "agent2")
	defer agent2.close()

	client11 := newTestClient(t, coord1, agent1.id)
	defer client11.close()
	client12 := newTestClient(t, coord1, agent2.id)
	defer client12.close()
	client21 := newTestClient(t, coord2, agent1.id)
	defer client21.close()
	client22 := newTestClient(t, coord2, agent2.id)
	defer client22.close()

	client11.sendNode(&agpl.Node{PreferredDERP: 11})
	assertEventuallyHasDERPs(ctx, t, agent1, 11)

	client21.sendNode(&agpl.Node{PreferredDERP: 21})
	assertEventuallyHasDERPs(ctx, t, agent1, 21, 11)

	client22.sendNode(&agpl.Node{PreferredDERP: 22})
	assertEventuallyHasDERPs(ctx, t, agent2, 22)

	agent2.sendNode(&agpl.Node{PreferredDERP: 2})
	assertEventuallyHasDERPs(ctx, t, client22, 2)
	assertEventuallyHasDERPs(ctx, t, client12, 2)

	client12.sendNode(&agpl.Node{PreferredDERP: 12})
	assertEventuallyHasDERPs(ctx, t, agent2, 12, 22)

	agent1.sendNode(&agpl.Node{PreferredDERP: 1})
	assertEventuallyHasDERPs(ctx, t, client21, 1)
	assertEventuallyHasDERPs(ctx, t, client11, 1)

	// let's close coord2
	err = coord2.Close()
	require.NoError(t, err)

	// this closes agent2, client22, client21
	err = agent2.recvErr(ctx, t)
	require.ErrorIs(t, err, io.EOF)
	err = client22.recvErr(ctx, t)
	require.ErrorIs(t, err, io.EOF)
	err = client21.recvErr(ctx, t)
	require.ErrorIs(t, err, io.EOF)

	// agent1 will see an update that drops client21.
	// In this case the update is superfluous because client11's node hasn't changed, and agents don't deprogram clients
	// from the dataplane even if they are missing.  Suppressing this kind of update would require the coordinator to
	// store all the data its sent to each connection, so we don't bother.
	assertEventuallyHasDERPs(ctx, t, agent1, 11)

	// note that although agent2 is disconnected, client12 does NOT get an update because we suppress empty updates.
	// (Its easy to tell these are superfluous.)

	assertEventuallyNoAgents(ctx, t, store, agent2.id)

	// Close coord1
	err = coord1.Close()
	require.NoError(t, err)
	// this closes agent1, client12, client11
	err = agent1.recvErr(ctx, t)
	require.ErrorIs(t, err, io.EOF)
	err = client12.recvErr(ctx, t)
	require.ErrorIs(t, err, io.EOF)
	err = client11.recvErr(ctx, t)
	require.ErrorIs(t, err, io.EOF)

	// wait for all connections to close
	err = agent1.close()
	require.NoError(t, err)
	agent1.waitForClose(ctx, t)

	err = agent2.close()
	require.NoError(t, err)
	agent2.waitForClose(ctx, t)

	err = client11.close()
	require.NoError(t, err)
	client11.waitForClose(ctx, t)

	err = client12.close()
	require.NoError(t, err)
	client12.waitForClose(ctx, t)

	err = client21.close()
	require.NoError(t, err)
	client21.waitForClose(ctx, t)

	err = client22.close()
	require.NoError(t, err)
	client22.waitForClose(ctx, t)

	assertEventuallyNoAgents(ctx, t, store, agent1.id)
	assertEventuallyNoClientsForAgent(ctx, t, store, agent1.id)
	assertEventuallyNoClientsForAgent(ctx, t, store, agent2.id)
}

// TestPGCoordinator_MultiAgent tests when a single agent connects to multiple coordinators.
// We use two agent connections, but they share the same AgentID.  This could happen due to a reconnection,
// or an infrastructure problem where an old workspace is not fully cleaned up before a new one started.
//
//	            +---------+
//	agent1 ---> | coord1  |
//	            +---------+
//	            +---------+
//	agent2 ---> | coord2  |
//	            +---------+
//	            +---------+
//	            | coord3  | <--- client
//	            +---------+
func TestPGCoordinator_MultiAgent(t *testing.T) {
	t.Parallel()
	if !dbtestutil.WillUsePostgres() {
		t.Skip("test only with postgres")
	}
	store, ps := dbtestutil.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitSuperLong)
	defer cancel()
	logger := slogtest.Make(t, nil).Leveled(slog.LevelDebug)
	coord1, err := tailnet.NewPGCoord(ctx, logger.Named("coord1"), ps, store)
	require.NoError(t, err)
	defer coord1.Close()
	coord2, err := tailnet.NewPGCoord(ctx, logger.Named("coord2"), ps, store)
	require.NoError(t, err)
	defer coord2.Close()
	coord3, err := tailnet.NewPGCoord(ctx, logger.Named("coord3"), ps, store)
	require.NoError(t, err)
	defer coord3.Close()

	agent1 := newTestAgent(t, coord1, "agent1")
	defer agent1.close()
	agent2 := newTestAgent(t, coord2, "agent2", agent1.id)
	defer agent2.close()

	client := newTestClient(t, coord3, agent1.id)
	defer client.close()

	client.sendNode(&agpl.Node{PreferredDERP: 3})
	assertEventuallyHasDERPs(ctx, t, agent1, 3)
	assertEventuallyHasDERPs(ctx, t, agent2, 3)

	agent1.sendNode(&agpl.Node{PreferredDERP: 1})
	assertEventuallyHasDERPs(ctx, t, client, 1)

	// agent2's update overrides agent1 because it is newer
	agent2.sendNode(&agpl.Node{PreferredDERP: 2})
	assertEventuallyHasDERPs(ctx, t, client, 2)

	// agent2 disconnects, and we should revert back to agent1
	err = agent2.close()
	require.NoError(t, err)
	err = agent2.recvErr(ctx, t)
	require.ErrorIs(t, err, io.ErrClosedPipe)
	agent2.waitForClose(ctx, t)
	assertEventuallyHasDERPs(ctx, t, client, 1)

	agent1.sendNode(&agpl.Node{PreferredDERP: 11})
	assertEventuallyHasDERPs(ctx, t, client, 11)

	client.sendNode(&agpl.Node{PreferredDERP: 31})
	assertEventuallyHasDERPs(ctx, t, agent1, 31)

	err = agent1.close()
	require.NoError(t, err)
	err = agent1.recvErr(ctx, t)
	require.ErrorIs(t, err, io.ErrClosedPipe)
	agent1.waitForClose(ctx, t)

	err = client.close()
	require.NoError(t, err)
	err = client.recvErr(ctx, t)
	require.ErrorIs(t, err, io.ErrClosedPipe)
	client.waitForClose(ctx, t)

	assertEventuallyNoClientsForAgent(ctx, t, store, agent1.id)
	assertEventuallyNoAgents(ctx, t, store, agent1.id)
}

func TestPGCoordinator_Unhealthy(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitSuperLong)
	defer cancel()
	ctrl := gomock.NewController(t)
	mStore := dbmock.NewMockStore(ctrl)
	ps := pubsub.NewInMemory()
	logger := slogtest.Make(t, &slogtest.Options{IgnoreErrors: true}).Leveled(slog.LevelDebug)

	calls := make(chan struct{})
	threeMissed := mStore.EXPECT().UpsertTailnetCoordinator(gomock.Any(), gomock.Any()).
		Times(3).
		Do(func(_ context.Context, _ uuid.UUID) { <-calls }).
		Return(database.TailnetCoordinator{}, xerrors.New("test disconnect"))
	mStore.EXPECT().UpsertTailnetCoordinator(gomock.Any(), gomock.Any()).
		MinTimes(1).
		After(threeMissed).
		Do(func(_ context.Context, _ uuid.UUID) { <-calls }).
		Return(database.TailnetCoordinator{}, nil)
	// extra calls we don't particularly care about for this test
	mStore.EXPECT().CleanTailnetCoordinators(gomock.Any()).AnyTimes().Return(nil)
	mStore.EXPECT().GetTailnetClientsForAgent(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	mStore.EXPECT().DeleteTailnetAgent(gomock.Any(), gomock.Any()).
		AnyTimes().Return(database.DeleteTailnetAgentRow{}, nil)
	mStore.EXPECT().DeleteCoordinator(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

	uut, err := tailnet.NewPGCoord(ctx, logger, ps, mStore)
	require.NoError(t, err)
	defer func() {
		err := uut.Close()
		require.NoError(t, err)
	}()
	agent1 := newTestAgent(t, uut, "agent1")
	defer agent1.close()
	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			t.Fatal("timeout")
		case calls <- struct{}{}:
			// OK
		}
	}
	// connected agent should be disconnected
	agent1.waitForClose(ctx, t)

	// new agent should immediately disconnect
	agent2 := newTestAgent(t, uut, "agent2")
	defer agent2.close()
	agent2.waitForClose(ctx, t)

	// next heartbeats succeed, so we are healthy
	for i := 0; i < 2; i++ {
		select {
		case <-ctx.Done():
			t.Fatal("timeout")
		case calls <- struct{}{}:
			// OK
		}
	}
	agent3 := newTestAgent(t, uut, "agent3")
	defer agent3.close()
	select {
	case <-agent3.closeChan:
		t.Fatal("agent conn closed after we are healthy")
	case <-time.After(time.Second):
		// OK
	}
}

type testConn struct {
	ws, serverWS net.Conn
	nodeChan     chan []*agpl.Node
	sendNode     func(node *agpl.Node)
	errChan      <-chan error
	id           uuid.UUID
	closeChan    chan struct{}
}

func newTestConn(ids []uuid.UUID) *testConn {
	a := &testConn{}
	a.ws, a.serverWS = net.Pipe()
	a.nodeChan = make(chan []*agpl.Node)
	a.sendNode, a.errChan = agpl.ServeCoordinator(a.ws, func(nodes []*agpl.Node) error {
		a.nodeChan <- nodes
		return nil
	})
	if len(ids) > 1 {
		panic("too many")
	}
	if len(ids) == 1 {
		a.id = ids[0]
	} else {
		a.id = uuid.New()
	}
	a.closeChan = make(chan struct{})
	return a
}

func newTestAgent(t *testing.T, coord agpl.Coordinator, name string, id ...uuid.UUID) *testConn {
	a := newTestConn(id)
	go func() {
		err := coord.ServeAgent(a.serverWS, a.id, name)
		assert.NoError(t, err)
		close(a.closeChan)
	}()
	return a
}

func (c *testConn) close() error {
	return c.ws.Close()
}

func (c *testConn) recvNodes(ctx context.Context, t *testing.T) []*agpl.Node {
	t.Helper()
	select {
	case <-ctx.Done():
		t.Fatalf("testConn id %s: timeout receiving nodes ", c.id)
		return nil
	case nodes := <-c.nodeChan:
		return nodes
	}
}

func (c *testConn) recvErr(ctx context.Context, t *testing.T) error {
	t.Helper()
	select {
	case <-ctx.Done():
		t.Fatal("timeout receiving error")
		return ctx.Err()
	case err := <-c.errChan:
		return err
	}
}

func (c *testConn) waitForClose(ctx context.Context, t *testing.T) {
	t.Helper()
	select {
	case <-ctx.Done():
		t.Fatal("timeout waiting for connection to close")
		return
	case <-c.closeChan:
		return
	}
}

func newTestClient(t *testing.T, coord agpl.Coordinator, agentID uuid.UUID, id ...uuid.UUID) *testConn {
	c := newTestConn(id)
	go func() {
		err := coord.ServeClient(c.serverWS, c.id, agentID)
		assert.NoError(t, err)
		close(c.closeChan)
	}()
	return c
}

func assertEventuallyHasDERPs(ctx context.Context, t *testing.T, c *testConn, expected ...int) {
	t.Helper()
	for {
		nodes := c.recvNodes(ctx, t)
		if len(nodes) != len(expected) {
			t.Logf("expected %d, got %d nodes", len(expected), len(nodes))
			continue
		}

		derps := make([]int, 0, len(nodes))
		for _, n := range nodes {
			derps = append(derps, n.PreferredDERP)
		}
		for _, e := range expected {
			if !slices.Contains(derps, e) {
				t.Logf("expected DERP %d to be in %v", e, derps)
				continue
			}
		}
		return
	}
}

func assertEventuallyNoAgents(ctx context.Context, t *testing.T, store database.Store, agentID uuid.UUID) {
	assert.Eventually(t, func() bool {
		agents, err := store.GetTailnetAgents(ctx, agentID)
		if xerrors.Is(err, sql.ErrNoRows) {
			return true
		}
		if err != nil {
			t.Fatal(err)
		}
		return len(agents) == 0
	}, testutil.WaitShort, testutil.IntervalFast)
}

func assertEventuallyNoClientsForAgent(ctx context.Context, t *testing.T, store database.Store, agentID uuid.UUID) {
	assert.Eventually(t, func() bool {
		clients, err := store.GetTailnetClientsForAgent(ctx, agentID)
		if xerrors.Is(err, sql.ErrNoRows) {
			return true
		}
		if err != nil {
			t.Fatal(err)
		}
		return len(clients) == 0
	}, testutil.WaitShort, testutil.IntervalFast)
}

type fakeCoordinator struct {
	ctx   context.Context
	t     *testing.T
	store database.Store
	id    uuid.UUID
}

func (c *fakeCoordinator) heartbeat() {
	c.t.Helper()
	_, err := c.store.UpsertTailnetCoordinator(c.ctx, c.id)
	require.NoError(c.t, err)
}

func (c *fakeCoordinator) agentNode(agentID uuid.UUID, node *agpl.Node) {
	c.t.Helper()
	nodeRaw, err := json.Marshal(node)
	require.NoError(c.t, err)
	_, err = c.store.UpsertTailnetAgent(c.ctx, database.UpsertTailnetAgentParams{
		ID:            agentID,
		CoordinatorID: c.id,
		Node:          nodeRaw,
	})
	require.NoError(c.t, err)
}
