package notifications_test

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/exp/slices"
	"golang.org/x/xerrors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	smtpmock "github.com/mocktools/go-smtp-mock/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/coder/serpent"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbgen"
	"github.com/coder/coder/v2/coderd/database/dbtestutil"
	"github.com/coder/coder/v2/coderd/notifications"
	"github.com/coder/coder/v2/coderd/notifications/dispatch"
	"github.com/coder/coder/v2/coderd/notifications/types"
	"github.com/coder/coder/v2/coderd/util/syncmap"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// TestBasicNotificationRoundtrip enqueues a message to the store, waits for it to be acquired by a notifier,
// passes it off to a fake handler, and ensures the results are synchronized to the store.
func TestBasicNotificationRoundtrip(t *testing.T) {
	t.Parallel()

	// SETUP
	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres; it relies on business-logic only implemented in the database")
	}

	ctx, logger, db := setup(t)
	method := database.NotificationMethodSmtp

	// GIVEN: a manager with standard config but a faked dispatch handler
	handler := &fakeHandler{}
	interceptor := &bulkUpdateInterceptor{Store: db}
	cfg := defaultNotificationsConfig(method)
	cfg.RetryInterval = serpent.Duration(time.Hour) // Ensure retries don't interfere with the test
	mgr, err := notifications.NewManager(cfg, interceptor, logger.Named("manager"))
	require.NoError(t, err)
	mgr.WithHandlers(map[database.NotificationMethod]notifications.Handler{method: handler})
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})
	enq, err := notifications.NewStoreEnqueuer(cfg, db, defaultHelpers(), logger.Named("enqueuer"))
	require.NoError(t, err)

	user := createSampleUser(t, db)

	// WHEN: 2 messages are enqueued
	sid, err := enq.Enqueue(ctx, user.ID, notifications.TemplateWorkspaceDeleted, map[string]string{"type": "success"}, "test")
	require.NoError(t, err)
	fid, err := enq.Enqueue(ctx, user.ID, notifications.TemplateWorkspaceDeleted, map[string]string{"type": "failure"}, "test")
	require.NoError(t, err)

	mgr.Run(ctx)

	// THEN: we expect that the handler will have received the notifications for dispatch
	require.Eventually(t, func() bool {
		handler.mu.RLock()
		defer handler.mu.RUnlock()
		return slices.Contains(handler.succeeded, sid.String()) &&
			slices.Contains(handler.failed, fid.String())
	}, testutil.WaitLong, testutil.IntervalFast)

	// THEN: we expect the store to be called with the updates of the earlier dispatches
	require.Eventually(t, func() bool {
		return interceptor.sent.Load() == 1 &&
			interceptor.failed.Load() == 1
	}, testutil.WaitShort, testutil.IntervalFast)

	// THEN: we verify that the store contains notifications in their expected state
	success, err := db.GetNotificationMessagesByStatus(ctx, database.GetNotificationMessagesByStatusParams{
		Status: database.NotificationMessageStatusSent,
		Limit:  10,
	})
	require.NoError(t, err)
	require.Len(t, success, 1)
	failed, err := db.GetNotificationMessagesByStatus(ctx, database.GetNotificationMessagesByStatusParams{
		Status: database.NotificationMessageStatusTemporaryFailure,
		Limit:  10,
	})
	require.NoError(t, err)
	require.Len(t, failed, 1)
}

func TestSMTPDispatch(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx, logger, db := setupInMemory(t)

	// start mock SMTP server
	mockSMTPSrv := smtpmock.New(smtpmock.ConfigurationAttr{
		LogToStdout:       false,
		LogServerActivity: true,
	})
	require.NoError(t, mockSMTPSrv.Start())
	t.Cleanup(func() {
		assert.NoError(t, mockSMTPSrv.Stop())
	})

	// GIVEN: an SMTP setup referencing a mock SMTP server
	const from = "danny@coder.com"
	method := database.NotificationMethodSmtp
	cfg := defaultNotificationsConfig(method)
	cfg.SMTP = codersdk.NotificationsEmailConfig{
		From:      from,
		Smarthost: serpent.HostPort{Host: "localhost", Port: fmt.Sprintf("%d", mockSMTPSrv.PortNumber())},
		Hello:     "localhost",
	}
	handler := newDispatchInterceptor(dispatch.NewSMTPHandler(cfg.SMTP, logger.Named("smtp")))
	mgr, err := notifications.NewManager(cfg, db, logger.Named("manager"))
	require.NoError(t, err)
	mgr.WithHandlers(map[database.NotificationMethod]notifications.Handler{method: handler})
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})
	enq, err := notifications.NewStoreEnqueuer(cfg, db, defaultHelpers(), logger.Named("enqueuer"))
	require.NoError(t, err)

	user := createSampleUser(t, db)

	// WHEN: a message is enqueued
	msgID, err := enq.Enqueue(ctx, user.ID, notifications.TemplateWorkspaceDeleted, map[string]string{}, "test")
	require.NoError(t, err)

	mgr.Run(ctx)

	// THEN: wait until the dispatch interceptor validates that the messages were dispatched
	require.Eventually(t, func() bool {
		assert.Nil(t, handler.lastErr.Load())
		assert.True(t, handler.retryable.Load() == 0)
		return handler.sent.Load() == 1
	}, testutil.WaitLong, testutil.IntervalMedium)

	// THEN: we verify that the expected message was received by the mock SMTP server
	msgs := mockSMTPSrv.MessagesAndPurge()
	require.Len(t, msgs, 1)
	require.Contains(t, msgs[0].MsgRequest(), fmt.Sprintf("From: %s", from))
	require.Contains(t, msgs[0].MsgRequest(), fmt.Sprintf("To: %s", user.Email))
	require.Contains(t, msgs[0].MsgRequest(), fmt.Sprintf("Message-Id: %s", msgID))
}

func TestWebhookDispatch(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx, logger, db := setupInMemory(t)

	sent := make(chan dispatch.WebhookPayload, 1)
	// Mock server to simulate webhook endpoint.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload dispatch.WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("noted."))
		assert.NoError(t, err)
		sent <- payload
	}))
	defer server.Close()

	endpoint, err := url.Parse(server.URL)
	require.NoError(t, err)

	// GIVEN: a webhook setup referencing a mock HTTP server to receive the webhook
	cfg := defaultNotificationsConfig(database.NotificationMethodWebhook)
	cfg.Webhook = codersdk.NotificationsWebhookConfig{
		Endpoint: *serpent.URLOf(endpoint),
	}
	mgr, err := notifications.NewManager(cfg, db, logger.Named("manager"))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})
	enq, err := notifications.NewStoreEnqueuer(cfg, db, defaultHelpers(), logger.Named("enqueuer"))
	require.NoError(t, err)

	const (
		email = "bob@coder.com"
		name  = "Robert McBobbington"
	)
	user := dbgen.User(t, db, database.User{
		Email:    email,
		Username: "bob",
		Name:     name,
	})

	// WHEN: a notification is enqueued (including arbitrary labels)
	input := map[string]string{
		"a": "b",
		"c": "d",
	}
	msgID, err := enq.Enqueue(ctx, user.ID, notifications.TemplateWorkspaceDeleted, input, "test")
	require.NoError(t, err)

	mgr.Run(ctx)

	// THEN: the webhook is received by the mock server and has the expected contents
	payload := testutil.RequireRecvCtx(testutil.Context(t, testutil.WaitShort), t, sent)
	require.EqualValues(t, "1.0", payload.Version)
	require.Equal(t, *msgID, payload.MsgID)
	require.Equal(t, payload.Payload.Labels, input)
	require.Equal(t, payload.Payload.UserEmail, email)
	// UserName is coalesced from `name` and `username`; in this case `name` wins.
	// This is not strictly necessary for this test, but it's testing some side logic which is too small for its own test.
	require.Equal(t, payload.Payload.UserName, name)
	// Right now we don't have a way to query notification templates by ID in dbmem, and it's not necessary to add this
	// just to satisfy this test. We can safely assume that as long as this value is not empty that the given value was delivered.
	require.NotEmpty(t, payload.Payload.NotificationName)
}

// TestBackpressure validates that delays in processing the buffered updates will result in slowed dequeue rates.
// As a side-effect, this also tests the graceful shutdown and flushing of the buffers.
func TestBackpressure(t *testing.T) {
	t.Parallel()

	// SETUP
	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres; it relies on business-logic only implemented in the database")
	}

	ctx, logger, db := setup(t)

	// Mock server to simulate webhook endpoint.
	var received atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload dispatch.WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("noted."))
		assert.NoError(t, err)

		received.Add(1)
	}))
	defer server.Close()

	endpoint, err := url.Parse(server.URL)
	require.NoError(t, err)

	method := database.NotificationMethodWebhook
	cfg := defaultNotificationsConfig(method)
	cfg.Webhook = codersdk.NotificationsWebhookConfig{
		Endpoint: *serpent.URLOf(endpoint),
	}

	// Tune the queue to fetch often.
	const fetchInterval = time.Millisecond * 200
	const batchSize = 10
	cfg.FetchInterval = serpent.Duration(fetchInterval)
	cfg.LeaseCount = serpent.Int64(batchSize)

	// Shrink buffers down and increase flush interval to provoke backpressure.
	// Flush buffers every 5 fetch intervals.
	const syncInterval = time.Second
	cfg.StoreSyncInterval = serpent.Duration(syncInterval)
	cfg.StoreSyncBufferSize = serpent.Int64(2)

	handler := newDispatchInterceptor(dispatch.NewWebhookHandler(cfg.Webhook, logger.Named("webhook")))

	// Intercept calls to submit the buffered updates to the store.
	storeInterceptor := &bulkUpdateInterceptor{Store: db}

	// GIVEN: a notification manager whose updates will be intercepted
	mgr, err := notifications.NewManager(cfg, storeInterceptor, logger.Named("manager"))
	require.NoError(t, err)
	mgr.WithHandlers(map[database.NotificationMethod]notifications.Handler{method: handler})
	enq, err := notifications.NewStoreEnqueuer(cfg, db, defaultHelpers(), logger.Named("enqueuer"))
	require.NoError(t, err)

	user := createSampleUser(t, db)

	// WHEN: a set of notifications are enqueued, which causes backpressure due to the batchSize which can be processed per fetch
	const totalMessages = 30
	for i := 0; i < totalMessages; i++ {
		_, err = enq.Enqueue(ctx, user.ID, notifications.TemplateWorkspaceDeleted, map[string]string{"i": fmt.Sprintf("%d", i)}, "test")
		require.NoError(t, err)
	}

	// Start the notifier.
	mgr.Run(ctx)

	// THEN:

	// Wait for 3 fetch intervals, then check progress.
	time.Sleep(fetchInterval * 3)

	// We expect the notifier will have dispatched ONLY the initial batch of messages.
	// In other words, the notifier should have dispatched 3 batches by now, but because the buffered updates have not
	// been processed: there is backpressure.
	require.EqualValues(t, batchSize, handler.sent.Load()+handler.err.Load())
	// We expect that the store will have received NO updates.
	require.EqualValues(t, 0, storeInterceptor.sent.Load()+storeInterceptor.failed.Load())

	// However, when we Stop() the manager the backpressure will be relieved and the buffered updates will ALL be flushed,
	// since all the goroutines that were blocked (on writing updates to the buffer) will be unblocked and will complete.
	require.NoError(t, mgr.Stop(ctx))
	require.EqualValues(t, batchSize, storeInterceptor.sent.Load()+storeInterceptor.failed.Load())
}

func TestRetries(t *testing.T) {
	t.Parallel()

	// SETUP
	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres; it relies on business-logic only implemented in the database")
	}

	const maxAttempts = 3
	ctx, logger, db := setup(t)

	// GIVEN: a mock HTTP server which will receive webhooksand a map to track the dispatch attempts

	receivedMap := syncmap.New[uuid.UUID, int]()
	// Mock server to simulate webhook endpoint.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload dispatch.WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)

		count, _ := receivedMap.LoadOrStore(payload.MsgID, 0)
		count++
		receivedMap.Store(payload.MsgID, count)

		// Let the request succeed if this is its last attempt.
		if count == maxAttempts {
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte("noted."))
			assert.NoError(t, err)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte("retry again later..."))
		assert.NoError(t, err)
	}))
	defer server.Close()

	endpoint, err := url.Parse(server.URL)
	require.NoError(t, err)

	method := database.NotificationMethodWebhook
	cfg := defaultNotificationsConfig(method)
	cfg.Webhook = codersdk.NotificationsWebhookConfig{
		Endpoint: *serpent.URLOf(endpoint),
	}

	cfg.MaxSendAttempts = maxAttempts

	// Tune intervals low to speed up test.
	cfg.StoreSyncInterval = serpent.Duration(time.Millisecond * 100)
	cfg.RetryInterval = serpent.Duration(time.Second) // query uses second-precision
	cfg.FetchInterval = serpent.Duration(time.Millisecond * 100)

	handler := newDispatchInterceptor(dispatch.NewWebhookHandler(cfg.Webhook, logger.Named("webhook")))

	// Intercept calls to submit the buffered updates to the store.
	storeInterceptor := &bulkUpdateInterceptor{Store: db}

	mgr, err := notifications.NewManager(cfg, storeInterceptor, logger.Named("manager"))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})
	mgr.WithHandlers(map[database.NotificationMethod]notifications.Handler{method: handler})
	enq, err := notifications.NewStoreEnqueuer(cfg, db, defaultHelpers(), logger.Named("enqueuer"))
	require.NoError(t, err)

	user := createSampleUser(t, db)

	// WHEN: a few notifications are enqueued, which will all fail until their final retry (determined by the mock server)
	const msgCount = 5
	for i := 0; i < msgCount; i++ {
		_, err = enq.Enqueue(ctx, user.ID, notifications.TemplateWorkspaceDeleted, map[string]string{"i": fmt.Sprintf("%d", i)}, "test")
		require.NoError(t, err)
	}

	mgr.Run(ctx)

	// THEN: we expect to see all but the final attempts failing
	require.Eventually(t, func() bool {
		// We expect all messages to fail all attempts but the final;
		return storeInterceptor.failed.Load() == msgCount*(maxAttempts-1) &&
			// ...and succeed on the final attempt.
			storeInterceptor.sent.Load() == msgCount
	}, testutil.WaitLong, testutil.IntervalFast)
}

// TestExpiredLeaseIsRequeued validates that notification messages which are left in "leased" status will be requeued once their lease expires.
// "leased" is the status which messages are set to when they are acquired for processing, and this should not be a terminal
// state unless the Manager shuts down ungracefully; the Manager is responsible for updating these messages' statuses once
// they have been processed.
func TestExpiredLeaseIsRequeued(t *testing.T) {
	t.Parallel()

	// SETUP
	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres; it relies on business-logic only implemented in the database")
	}

	ctx, logger, db := setup(t)

	// GIVEN: a manager which has its updates intercepted and paused until measurements can be taken

	const (
		leasePeriod = time.Second
		msgCount    = 5
		method      = database.NotificationMethodSmtp
	)

	cfg := defaultNotificationsConfig(method)
	// Set low lease period to speed up tests.
	cfg.LeasePeriod = serpent.Duration(leasePeriod)
	cfg.DispatchTimeout = serpent.Duration(leasePeriod - time.Millisecond)

	noopInterceptor := newNoopBulkUpdater(db)

	mgrCtx, cancelManagerCtx := context.WithCancel(context.Background())
	t.Cleanup(cancelManagerCtx)

	mgr, err := notifications.NewManager(cfg, noopInterceptor, logger.Named("manager"))
	require.NoError(t, err)
	enq, err := notifications.NewStoreEnqueuer(cfg, db, defaultHelpers(), logger.Named("enqueuer"))
	require.NoError(t, err)

	user := createSampleUser(t, db)

	// WHEN: a few notifications are enqueued which will all succeed
	var msgs []string
	for i := 0; i < msgCount; i++ {
		id, err := enq.Enqueue(ctx, user.ID, notifications.TemplateWorkspaceDeleted, map[string]string{"type": "success"}, "test")
		require.NoError(t, err)
		msgs = append(msgs, id.String())
	}

	mgr.Run(mgrCtx)

	// THEN:

	// Wait for the messages to be acquired
	<-noopInterceptor.acquiredChan
	// Then cancel the context, forcing the notification manager to shutdown ungracefully (simulating a crash); leaving messages in "leased" status.
	cancelManagerCtx()

	// Fetch any messages currently in "leased" status, and verify that they're exactly the ones we enqueued.
	leased, err := db.GetNotificationMessagesByStatus(ctx, database.GetNotificationMessagesByStatusParams{
		Status: database.NotificationMessageStatusLeased,
		Limit:  msgCount,
	})
	require.NoError(t, err)

	var leasedIDs []string
	for _, msg := range leased {
		leasedIDs = append(leasedIDs, msg.ID.String())
	}

	sort.Strings(msgs)
	sort.Strings(leasedIDs)
	require.EqualValues(t, msgs, leasedIDs)

	// Wait out the lease period; all messages should be eligible to be re-acquired.
	time.Sleep(leasePeriod + time.Millisecond)

	// Start a new notification manager.
	// Intercept calls to submit the buffered updates to the store.
	storeInterceptor := &bulkUpdateInterceptor{Store: db}
	handler := newDispatchInterceptor(&fakeHandler{})
	mgr, err = notifications.NewManager(cfg, storeInterceptor, logger.Named("manager"))
	require.NoError(t, err)
	mgr.WithHandlers(map[database.NotificationMethod]notifications.Handler{method: handler})

	// Use regular context now.
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})
	mgr.Run(ctx)

	// Wait until all messages are sent & updates flushed to the database.
	require.Eventually(t, func() bool {
		return handler.sent.Load() == msgCount &&
			storeInterceptor.sent.Load() == msgCount
	}, testutil.WaitLong, testutil.IntervalFast)

	// Validate that no more messages are in "leased" status.
	leased, err = db.GetNotificationMessagesByStatus(ctx, database.GetNotificationMessagesByStatusParams{
		Status: database.NotificationMessageStatusLeased,
		Limit:  msgCount,
	})
	require.NoError(t, err)
	require.Len(t, leased, 0)
}

// TestInvalidConfig validates that misconfigurations lead to errors.
func TestInvalidConfig(t *testing.T) {
	t.Parallel()

	_, logger, db := setupInMemory(t)

	// GIVEN: invalid config with dispatch period <= lease period
	const (
		leasePeriod = time.Second
		method      = database.NotificationMethodSmtp
	)
	cfg := defaultNotificationsConfig(method)
	cfg.LeasePeriod = serpent.Duration(leasePeriod)
	cfg.DispatchTimeout = serpent.Duration(leasePeriod)

	// WHEN: the manager is created with invalid config
	_, err := notifications.NewManager(cfg, db, logger.Named("manager"))

	// THEN: the manager will fail to be created, citing invalid config as error
	require.ErrorIs(t, err, notifications.ErrInvalidDispatchTimeout)
}

type fakeHandler struct {
	mu sync.RWMutex

	succeeded []string
	failed    []string
}

func (f *fakeHandler) Dispatcher(payload types.MessagePayload, _, _ string) (dispatch.DeliveryFunc, error) {
	return func(ctx context.Context, msgID uuid.UUID) (retryable bool, err error) {
		f.mu.Lock()
		defer f.mu.Unlock()

		if payload.Labels["type"] == "success" {
			f.succeeded = append(f.succeeded, msgID.String())
			return false, nil
		}

		f.failed = append(f.failed, msgID.String())
		return true, xerrors.New("oops")
	}, nil
}

type dispatchInterceptor struct {
	handler notifications.Handler

	sent        atomic.Int32
	retryable   atomic.Int32
	unretryable atomic.Int32
	err         atomic.Int32
	lastErr     atomic.Value
}

func newDispatchInterceptor(h notifications.Handler) *dispatchInterceptor {
	return &dispatchInterceptor{
		handler: h,
	}
}

func (i *dispatchInterceptor) Dispatcher(payload types.MessagePayload, title, body string) (dispatch.DeliveryFunc, error) {
	return func(ctx context.Context, msgID uuid.UUID) (retryable bool, err error) {
		deliveryFn, err := i.handler.Dispatcher(payload, title, body)
		if err != nil {
			return false, err
		}

		retryable, err = deliveryFn(ctx, msgID)

		if err != nil {
			i.err.Add(1)
			i.lastErr.Store(err)
		}

		switch {
		case !retryable && err == nil:
			i.sent.Add(1)
		case retryable:
			i.retryable.Add(1)
		case !retryable && err != nil:
			i.unretryable.Add(1)
		}
		return retryable, err
	}, nil
}

// noopBulkUpdater pretends to perform bulk updates, but does not; leading to messages being stuck in "leased" state.
type noopBulkUpdater struct {
	*acquireSignalingInterceptor
}

func newNoopBulkUpdater(db notifications.Store) *noopBulkUpdater {
	return &noopBulkUpdater{newAcquireSignalingInterceptor(db)}
}

func (*noopBulkUpdater) BulkMarkNotificationMessagesSent(_ context.Context, arg database.BulkMarkNotificationMessagesSentParams) (int64, error) {
	return int64(len(arg.IDs)), nil
}

func (*noopBulkUpdater) BulkMarkNotificationMessagesFailed(_ context.Context, arg database.BulkMarkNotificationMessagesFailedParams) (int64, error) {
	return int64(len(arg.IDs)), nil
}

type acquireSignalingInterceptor struct {
	notifications.Store
	acquiredChan chan struct{}
}

func newAcquireSignalingInterceptor(db notifications.Store) *acquireSignalingInterceptor {
	return &acquireSignalingInterceptor{
		Store:        db,
		acquiredChan: make(chan struct{}, 1),
	}
}

func (n *acquireSignalingInterceptor) AcquireNotificationMessages(ctx context.Context, params database.AcquireNotificationMessagesParams) ([]database.AcquireNotificationMessagesRow, error) {
	messages, err := n.Store.AcquireNotificationMessages(ctx, params)
	n.acquiredChan <- struct{}{}
	return messages, err
}
