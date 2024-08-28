package notifications_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/xerrors"

	"github.com/coder/quartz"

	"github.com/google/uuid"
	smtpmock "github.com/mocktools/go-smtp-mock/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/coder/serpent"

	"github.com/coder/coder/v2/coderd"
	"github.com/coder/coder/v2/coderd/coderdtest"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/database/dbtestutil"
	"github.com/coder/coder/v2/coderd/notifications"
	"github.com/coder/coder/v2/coderd/notifications/dispatch"
	"github.com/coder/coder/v2/coderd/notifications/render"
	"github.com/coder/coder/v2/coderd/notifications/types"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/coderd/runtimeconfig"
	"github.com/coder/coder/v2/coderd/util/ptr"
	"github.com/coder/coder/v2/coderd/util/syncmap"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/enterprise/coderd/coderdenttest"
	"github.com/coder/coder/v2/enterprise/coderd/license"
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

	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	_, _, api := coderdtest.NewWithAPI(t, nil)
	method := database.NotificationMethodSmtp
	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(database.NotificationMethodSmtp)).Notifications

	// GIVEN: a manager with standard config but a faked dispatch handler
	handler := &fakeHandler{}
	interceptor := &syncInterceptor{Store: api.Database}
	cfg.RetryInterval = serpent.Duration(time.Hour) // Ensure retries don't interfere with the test
	mgr, err := notifications.NewManager(cfg, coderd.NewRuntimeConfigStore(api.Database), interceptor, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))
	require.NoError(t, err)
	mgr.WithHandlers(map[database.NotificationMethod]notifications.Handler{method: handler})
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})
	enq, err := notifications.NewStoreEnqueuer(cfg, api.Database, defaultHelpers(), api.Logger.Named("enqueuer"), quartz.NewReal())
	require.NoError(t, err)

	member := createSampleOrgMember(t, api.Database)

	// WHEN: 2 messages are enqueued
	sid, err := enq.Enqueue(ctx, member.UserID, member.OrganizationID, notifications.TemplateWorkspaceDeleted, map[string]string{"type": "success"}, "test")
	require.NoError(t, err)
	fid, err := enq.Enqueue(ctx, member.UserID, member.OrganizationID, notifications.TemplateWorkspaceDeleted, map[string]string{"type": "failure"}, "test")
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
	}, testutil.WaitLong, testutil.IntervalFast)

	// THEN: we verify that the store contains notifications in their expected state
	success, err := api.Database.GetNotificationMessagesByStatus(ctx, database.GetNotificationMessagesByStatusParams{
		Status: database.NotificationMessageStatusSent,
		Limit:  10,
	})
	require.NoError(t, err)
	require.Len(t, success, 1)
	failed, err := api.Database.GetNotificationMessagesByStatus(ctx, database.GetNotificationMessagesByStatusParams{
		Status: database.NotificationMessageStatusTemporaryFailure,
		Limit:  10,
	})
	require.NoError(t, err)
	require.Len(t, failed, 1)
}

func TestSMTPDispatch(t *testing.T) {
	t.Parallel()

	// SETUP

	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	_, _, api := coderdtest.NewWithAPI(t, nil)

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
	const (
		from = "danny@coder.com"
		to   = "bob@coder.com"
	)
	method := database.NotificationMethodSmtp
	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(method), func(vals *codersdk.DeploymentValues) {
		require.NoError(t, vals.Notifications.SMTP.From.Set(from))
		vals.Notifications.SMTP.Hello = "localhost"
		vals.Notifications.SMTP.Smarthost = serpent.HostPort{Host: "localhost", Port: fmt.Sprintf("%d", mockSMTPSrv.PortNumber())}
	})
	handler := newDispatchInterceptor(dispatch.NewSMTPHandler(cfg.Notifications.SMTP, defaultHelpers(), api.Logger.Named("smtp")))
	mgr, err := notifications.NewManager(cfg.Notifications, coderd.NewRuntimeConfigStore(api.Database), api.Database, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))
	require.NoError(t, err)
	mgr.WithHandlers(map[database.NotificationMethod]notifications.Handler{method: handler})
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})
	enq, err := notifications.NewStoreEnqueuer(cfg.Notifications, api.Database, defaultHelpers(), api.Logger.Named("enqueuer"), quartz.NewReal())
	require.NoError(t, err)

	member := createSampleOrgMember(t, api.Database, func(u *database.User, _ *database.Organization) {
		u.Email = to
	})

	// WHEN: a message is enqueued
	msgID, err := enq.Enqueue(ctx, member.UserID, member.OrganizationID, notifications.TemplateWorkspaceDeleted, map[string]string{}, "test")
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
	require.Contains(t, msgs[0].MsgRequest(), fmt.Sprintf("To: %s", to))
	require.Contains(t, msgs[0].MsgRequest(), fmt.Sprintf("Message-Id: %s", msgID))
}

func TestWebhookDispatch(t *testing.T) {
	t.Parallel()

	// SETUP

	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	_, _, api := coderdtest.NewWithAPI(t, nil)

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
	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(database.NotificationMethodWebhook)).Notifications
	cfg.Webhook = codersdk.NotificationsWebhookConfig{
		Endpoint: *serpent.URLOf(endpoint),
	}
	mgr, err := notifications.NewManager(cfg, coderd.NewRuntimeConfigStore(api.Database), api.Database, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})
	enq, err := notifications.NewStoreEnqueuer(cfg, api.Database, defaultHelpers(), api.Logger.Named("enqueuer"), quartz.NewReal())
	require.NoError(t, err)

	const (
		email    = "bob@coder.com"
		name     = "Robert McBobbington"
		username = "bob"
	)

	member := createSampleOrgMember(t, api.Database, func(u *database.User, o *database.Organization) {
		u.Email = email
		u.Username = username
		u.Name = name
	})

	// WHEN: a notification is enqueued (including arbitrary labels)
	input := map[string]string{
		"a": "b",
		"c": "d",
	}
	msgID, err := enq.Enqueue(ctx, member.UserID, member.OrganizationID, notifications.TemplateWorkspaceDeleted, input, "test")
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
	require.Equal(t, payload.Payload.UserUsername, username)
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

	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	_, _, api := coderdtest.NewWithAPI(t, nil)

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
	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(method)).Notifications
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

	handler := newDispatchInterceptor(dispatch.NewWebhookHandler(cfg.Webhook, api.Logger.Named("webhook")))

	// Intercept calls to submit the buffered updates to the store.
	storeInterceptor := &syncInterceptor{Store: api.Database}

	// GIVEN: a notification manager whose updates will be intercepted
	mgr, err := notifications.NewManager(cfg, coderd.NewRuntimeConfigStore(api.Database), storeInterceptor, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))
	require.NoError(t, err)
	mgr.WithHandlers(map[database.NotificationMethod]notifications.Handler{method: handler})
	enq, err := notifications.NewStoreEnqueuer(cfg, api.Database, defaultHelpers(), api.Logger.Named("enqueuer"), quartz.NewReal())
	require.NoError(t, err)

	member := createSampleOrgMember(t, api.Database)

	// WHEN: a set of notifications are enqueued, which causes backpressure due to the batchSize which can be processed per fetch
	const totalMessages = 30
	for i := 0; i < totalMessages; i++ {
		_, err = enq.Enqueue(ctx, member.UserID, member.OrganizationID, notifications.TemplateWorkspaceDeleted, map[string]string{"i": fmt.Sprintf("%d", i)}, "test")
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
	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	_, _, api := coderdtest.NewWithAPI(t, nil)

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
	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(method)).Notifications
	cfg.Webhook = codersdk.NotificationsWebhookConfig{
		Endpoint: *serpent.URLOf(endpoint),
	}

	cfg.MaxSendAttempts = maxAttempts

	// Tune intervals low to speed up test.
	cfg.StoreSyncInterval = serpent.Duration(time.Millisecond * 100)
	cfg.RetryInterval = serpent.Duration(time.Second) // query uses second-precision
	cfg.FetchInterval = serpent.Duration(time.Millisecond * 100)

	handler := newDispatchInterceptor(dispatch.NewWebhookHandler(cfg.Webhook, api.Logger.Named("webhook")))

	// Intercept calls to submit the buffered updates to the store.
	storeInterceptor := &syncInterceptor{Store: api.Database}

	mgr, err := notifications.NewManager(cfg, coderd.NewRuntimeConfigStore(api.Database), storeInterceptor, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})
	mgr.WithHandlers(map[database.NotificationMethod]notifications.Handler{method: handler})
	enq, err := notifications.NewStoreEnqueuer(cfg, api.Database, defaultHelpers(), api.Logger.Named("enqueuer"), quartz.NewReal())
	require.NoError(t, err)

	member := createSampleOrgMember(t, api.Database)

	// WHEN: a few notifications are enqueued, which will all fail until their final retry (determined by the mock server)
	const msgCount = 5
	for i := 0; i < msgCount; i++ {
		_, err = enq.Enqueue(ctx, member.UserID, member.OrganizationID, notifications.TemplateWorkspaceDeleted, map[string]string{"i": fmt.Sprintf("%d", i)}, "test")
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

	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	_, _, api := coderdtest.NewWithAPI(t, nil)

	// GIVEN: a manager which has its updates intercepted and paused until measurements can be taken

	const (
		leasePeriod = time.Second
		msgCount    = 5
		method      = database.NotificationMethodSmtp
	)

	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(method)).Notifications
	// Set low lease period to speed up tests.
	cfg.LeasePeriod = serpent.Duration(leasePeriod)
	cfg.DispatchTimeout = serpent.Duration(leasePeriod - time.Millisecond)

	noopInterceptor := newNoopStoreSyncer(api.Database)

	// nolint:gocritic // Unit test.
	mgrCtx, cancelManagerCtx := context.WithCancel(dbauthz.AsSystemRestricted(context.Background()))
	t.Cleanup(cancelManagerCtx)

	mgr, err := notifications.NewManager(cfg, coderd.NewRuntimeConfigStore(api.Database), noopInterceptor, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))
	require.NoError(t, err)
	enq, err := notifications.NewStoreEnqueuer(cfg, api.Database, defaultHelpers(), api.Logger.Named("enqueuer"), quartz.NewReal())
	require.NoError(t, err)

	member := createSampleOrgMember(t, api.Database)

	// WHEN: a few notifications are enqueued which will all succeed
	var msgs []string
	for i := 0; i < msgCount; i++ {
		id, err := enq.Enqueue(ctx, member.UserID, member.OrganizationID, notifications.TemplateWorkspaceDeleted, map[string]string{"type": "success", "index": fmt.Sprintf("%d", i)}, "test")
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
	leased, err := api.Database.GetNotificationMessagesByStatus(ctx, database.GetNotificationMessagesByStatusParams{
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
	storeInterceptor := &syncInterceptor{Store: api.Database}
	handler := newDispatchInterceptor(&fakeHandler{})
	mgr, err = notifications.NewManager(cfg, coderd.NewRuntimeConfigStore(api.Database), storeInterceptor, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))
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
	leased, err = api.Database.GetNotificationMessagesByStatus(ctx, database.GetNotificationMessagesByStatusParams{
		Status: database.NotificationMessageStatusLeased,
		Limit:  msgCount,
	})
	require.NoError(t, err)
	require.Len(t, leased, 0)
}

// TestInvalidConfig validates that misconfigurations lead to errors.
func TestInvalidConfig(t *testing.T) {
	t.Parallel()

	_, _, api := coderdtest.NewWithAPI(t, nil)

	// GIVEN: invalid config with dispatch period <= lease period
	const (
		leasePeriod = time.Second
		method      = database.NotificationMethodSmtp
	)
	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(method)).Notifications
	cfg.LeasePeriod = serpent.Duration(leasePeriod)
	cfg.DispatchTimeout = serpent.Duration(leasePeriod)

	// WHEN: the manager is created with invalid config
	_, err := notifications.NewManager(cfg, coderd.NewRuntimeConfigStore(api.Database), api.Database, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))

	// THEN: the manager will fail to be created, citing invalid config as error
	require.ErrorIs(t, err, notifications.ErrInvalidDispatchTimeout)
}

func TestNotifierPaused(t *testing.T) {
	t.Parallel()

	// Setup.

	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	_, _, api := coderdtest.NewWithAPI(t, nil)

	// Prepare the test
	handler := &fakeHandler{}
	method := database.NotificationMethodSmtp
	member := createSampleOrgMember(t, api.Database)

	const fetchInterval = time.Millisecond * 100
	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(method)).Notifications
	cfg.FetchInterval = serpent.Duration(fetchInterval)
	mgr, err := notifications.NewManager(cfg, coderd.NewRuntimeConfigStore(api.Database), api.Database, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))
	require.NoError(t, err)
	mgr.WithHandlers(map[database.NotificationMethod]notifications.Handler{method: handler})
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})
	enq, err := notifications.NewStoreEnqueuer(cfg, api.Database, defaultHelpers(), api.Logger.Named("enqueuer"), quartz.NewReal())
	require.NoError(t, err)

	mgr.Run(ctx)

	// Pause the notifier.
	settingsJSON, err := json.Marshal(&codersdk.NotificationsSettings{NotifierPaused: true})
	require.NoError(t, err)
	err = api.Database.UpsertNotificationsSettings(ctx, string(settingsJSON))
	require.NoError(t, err)

	// Notifier is paused, enqueue the next message.
	sid, err := enq.Enqueue(ctx, member.UserID, member.OrganizationID, notifications.TemplateWorkspaceDeleted, map[string]string{"type": "success", "i": "1"}, "test")
	require.NoError(t, err)

	// Sleep for a few fetch intervals to be sure we aren't getting false-positives in the next step.
	// TODO: use quartz instead.
	time.Sleep(fetchInterval * 5)

	// Ensure we have a pending message and it's the expected one.
	require.Eventually(t, func() bool {
		pendingMessages, err := api.Database.GetNotificationMessagesByStatus(ctx, database.GetNotificationMessagesByStatusParams{
			Status: database.NotificationMessageStatusPending,
			Limit:  10,
		})
		assert.NoError(t, err)
		return len(pendingMessages) == 1 &&
			pendingMessages[0].ID.String() == sid.String()
	}, testutil.WaitShort, testutil.IntervalFast)

	// Unpause the notifier.
	settingsJSON, err = json.Marshal(&codersdk.NotificationsSettings{NotifierPaused: false})
	require.NoError(t, err)
	err = api.Database.UpsertNotificationsSettings(ctx, string(settingsJSON))
	require.NoError(t, err)

	// Notifier is running again, message should be dequeued.
	require.Eventually(t, func() bool {
		handler.mu.RLock()
		defer handler.mu.RUnlock()
		return slices.Contains(handler.succeeded, sid.String())
	}, testutil.WaitShort, testutil.IntervalFast)
}

//go:embed events.go
var events []byte

// enumerateAllTemplates gets all the template names from the coderd/notifications/events.go file.
// TODO(dannyk): use code-generation to create a list of all templates: https://github.com/coder/team-coconut/issues/36
func enumerateAllTemplates(t *testing.T) ([]string, error) {
	t.Helper()

	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, "", bytes.NewBuffer(events), parser.AllErrors)
	if err != nil {
		return nil, err
	}

	var out []string
	// Traverse the AST and extract variable names.
	ast.Inspect(node, func(n ast.Node) bool {
		// Check if the node is a declaration statement.
		if decl, ok := n.(*ast.GenDecl); ok && decl.Tok == token.VAR {
			for _, spec := range decl.Specs {
				// Type assert the spec to a ValueSpec.
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						out = append(out, name.String())
					}
				}
			}
		}
		return true
	})

	return out, nil
}

func TestNotificationTemplatesCanRender(t *testing.T) {
	t.Parallel()

	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres; it relies on the notification templates added by migrations in the database")
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		payload types.MessagePayload
	}{
		{
			name: "TemplateWorkspaceDeleted",
			id:   notifications.TemplateWorkspaceDeleted,
			payload: types.MessagePayload{
				UserName: "bobby",
				Labels: map[string]string{
					"name":      "bobby-workspace",
					"reason":    "autodeleted due to dormancy",
					"initiator": "autobuild",
				},
			},
		},
		{
			name: "TemplateWorkspaceAutobuildFailed",
			id:   notifications.TemplateWorkspaceAutobuildFailed,
			payload: types.MessagePayload{
				UserName: "bobby",
				Labels: map[string]string{
					"name":   "bobby-workspace",
					"reason": "autostart",
				},
			},
		},
		{
			name: "TemplateWorkspaceDormant",
			id:   notifications.TemplateWorkspaceDormant,
			payload: types.MessagePayload{
				UserName: "bobby",
				Labels: map[string]string{
					"name":           "bobby-workspace",
					"reason":         "breached the template's threshold for inactivity",
					"initiator":      "autobuild",
					"dormancyHours":  "24",
					"timeTilDormant": "24 hours",
				},
			},
		},
		{
			name: "TemplateWorkspaceAutoUpdated",
			id:   notifications.TemplateWorkspaceAutoUpdated,
			payload: types.MessagePayload{
				UserName: "bobby",
				Labels: map[string]string{
					"name":                     "bobby-workspace",
					"template_version_name":    "1.0",
					"template_version_message": "template now includes catnip",
				},
			},
		},
		{
			name: "TemplateWorkspaceMarkedForDeletion",
			id:   notifications.TemplateWorkspaceMarkedForDeletion,
			payload: types.MessagePayload{
				UserName: "bobby",
				Labels: map[string]string{
					"name":           "bobby-workspace",
					"reason":         "template updated to new dormancy policy",
					"dormancyHours":  "24",
					"timeTilDormant": "24 hours",
				},
			},
		},
		{
			name: "TemplateUserAccountCreated",
			id:   notifications.TemplateUserAccountCreated,
			payload: types.MessagePayload{
				UserName: "bobby",
				Labels: map[string]string{
					"created_account_name": "bobby",
				},
			},
		},
		{
			name: "TemplateUserAccountDeleted",
			id:   notifications.TemplateUserAccountDeleted,
			payload: types.MessagePayload{
				UserName: "bobby",
				Labels: map[string]string{
					"deleted_account_name": "bobby",
				},
			},
		},
		{
			name: "TemplateUserAccountSuspended",
			id:   notifications.TemplateUserAccountSuspended,
			payload: types.MessagePayload{
				UserName: "bobby",
				Labels: map[string]string{
					"suspended_account_name": "bobby",
				},
			},
		},
		{
			name: "TemplateUserAccountActivated",
			id:   notifications.TemplateUserAccountActivated,
			payload: types.MessagePayload{
				UserName: "bobby",
				Labels: map[string]string{
					"activated_account_name": "bobby",
				},
			},
		},
		{
			name: "TemplateYourAccountSuspended",
			id:   notifications.TemplateYourAccountSuspended,
			payload: types.MessagePayload{
				UserName: "bobby",
				Labels: map[string]string{
					"suspended_account_name": "bobby",
				},
			},
		},
		{
			name: "TemplateYourAccountActivated",
			id:   notifications.TemplateYourAccountActivated,
			payload: types.MessagePayload{
				UserName: "bobby",
				Labels: map[string]string{
					"activated_account_name": "bobby",
				},
			},
		},
		{
			name: "TemplateTemplateDeleted",
			id:   notifications.TemplateTemplateDeleted,
			payload: types.MessagePayload{
				UserName: "bobby",
				Labels: map[string]string{
					"name":      "bobby-template",
					"initiator": "rob",
				},
			},
		},
	}

	allTemplates, err := enumerateAllTemplates(t)
	require.NoError(t, err)
	for _, name := range allTemplates {
		var found bool
		for _, tc := range tests {
			if tc.name == name {
				found = true
			}
		}

		require.Truef(t, found, "could not find test case for %q", name)
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, _, sql := dbtestutil.NewDBWithSQLDB(t)

			var (
				titleTmpl string
				bodyTmpl  string
			)
			err := sql.
				QueryRow("SELECT title_template, body_template FROM notification_templates WHERE id = $1 LIMIT 1", tc.id).
				Scan(&titleTmpl, &bodyTmpl)
			require.NoError(t, err, "failed to query body template for template:", tc.id)

			title, err := render.GoTemplate(titleTmpl, tc.payload, nil)
			require.NotContainsf(t, title, render.NoValue, "template %q is missing a label value", tc.name)
			require.NoError(t, err, "failed to render notification title template")
			require.NotEmpty(t, title, "title should not be empty")

			body, err := render.GoTemplate(bodyTmpl, tc.payload, nil)
			require.NoError(t, err, "failed to render notification body template")
			require.NotEmpty(t, body, "body should not be empty")
		})
	}
}

// TestDisabledBeforeEnqueue ensures that notifications cannot be enqueued once a user has disabled that notification template
func TestDisabledBeforeEnqueue(t *testing.T) {
	t.Parallel()

	// SETUP
	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres; it is testing business-logic implemented in the database")
	}

	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	_, _, api := coderdtest.NewWithAPI(t, nil)

	// GIVEN: an enqueuer & a sample user
	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(database.NotificationMethodSmtp)).Notifications
	enq, err := notifications.NewStoreEnqueuer(cfg, api.Database, defaultHelpers(), api.Logger.Named("enqueuer"), quartz.NewReal())
	require.NoError(t, err)
	member := createSampleOrgMember(t, api.Database)

	// WHEN: the user has a preference set to not receive the "workspace deleted" notification
	templateID := notifications.TemplateWorkspaceDeleted
	n, err := api.Database.UpdateUserNotificationPreferences(ctx, database.UpdateUserNotificationPreferencesParams{
		UserID:                  member.UserID,
		NotificationTemplateIds: []uuid.UUID{templateID},
		Disableds:               []bool{true},
	})
	require.NoError(t, err, "failed to set preferences")
	require.EqualValues(t, 1, n, "unexpected number of affected rows")

	// THEN: enqueuing the "workspace deleted" notification should fail with an error
	_, err = enq.Enqueue(ctx, member.UserID, member.OrganizationID, templateID, map[string]string{}, "test")
	require.ErrorIs(t, err, notifications.ErrCannotEnqueueDisabledNotification, "enqueueing did not fail with expected error")
}

// TestDisabledAfterEnqueue ensures that notifications enqueued before a notification template was disabled will not be
// sent, and will instead be marked as "inhibited".
func TestDisabledAfterEnqueue(t *testing.T) {
	t.Parallel()

	// SETUP
	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres; it is testing business-logic implemented in the database")
	}

	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	_, _, api := coderdtest.NewWithAPI(t, nil)

	method := database.NotificationMethodSmtp
	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(method)).Notifications

	mgr, err := notifications.NewManager(cfg, coderd.NewRuntimeConfigStore(api.Database), api.Database, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})

	enq, err := notifications.NewStoreEnqueuer(cfg, api.Database, defaultHelpers(), api.Logger.Named("enqueuer"), quartz.NewReal())
	require.NoError(t, err)
	member := createSampleOrgMember(t, api.Database)

	// GIVEN: a notification is enqueued which has not (yet) been disabled
	templateID := notifications.TemplateWorkspaceDeleted
	msgID, err := enq.Enqueue(ctx, member.UserID, member.OrganizationID, templateID, map[string]string{}, "test")
	require.NoError(t, err)

	// Disable the notification template.
	n, err := api.Database.UpdateUserNotificationPreferences(ctx, database.UpdateUserNotificationPreferencesParams{
		UserID:                  member.UserID,
		NotificationTemplateIds: []uuid.UUID{templateID},
		Disableds:               []bool{true},
	})
	require.NoError(t, err, "failed to set preferences")
	require.EqualValues(t, 1, n, "unexpected number of affected rows")

	// WHEN: running the manager to trigger dequeueing of (now-disabled) messages
	mgr.Run(ctx)

	// THEN: the message should not be sent, and must be set to "inhibited"
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		m, err := api.Database.GetNotificationMessagesByStatus(ctx, database.GetNotificationMessagesByStatusParams{
			Status: database.NotificationMessageStatusInhibited,
			Limit:  10,
		})
		assert.NoError(ct, err)
		if assert.Equal(ct, len(m), 1) {
			assert.Equal(ct, m[0].ID.String(), msgID.String())
			assert.Contains(ct, m[0].StatusReason.String, "disabled by user")
		}
	}, testutil.WaitLong, testutil.IntervalFast, "did not find the expected inhibited message")
}

func TestCustomNotificationMethod(t *testing.T) {
	t.Parallel()

	// SETUP
	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres; it relies on business-logic only implemented in the database")
	}

	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	_, _, api := coderdtest.NewWithAPI(t, nil)

	received := make(chan uuid.UUID, 1)

	// SETUP:
	// Start mock server to simulate webhook endpoint.
	mockWebhookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload dispatch.WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)

		received <- payload.MsgID
		close(received)

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("noted."))
		require.NoError(t, err)
	}))
	defer mockWebhookSrv.Close()

	// Start mock SMTP server.
	mockSMTPSrv := smtpmock.New(smtpmock.ConfigurationAttr{
		LogToStdout:       false,
		LogServerActivity: true,
	})
	require.NoError(t, mockSMTPSrv.Start())
	t.Cleanup(func() {
		assert.NoError(t, mockSMTPSrv.Stop())
	})

	endpoint, err := url.Parse(mockWebhookSrv.URL)
	require.NoError(t, err)

	// GIVEN: a notification template which has a method explicitly set
	var (
		template      = notifications.TemplateWorkspaceDormant
		defaultMethod = database.NotificationMethodSmtp
		customMethod  = database.NotificationMethodWebhook
	)
	out, err := api.Database.UpdateNotificationTemplateMethodByID(ctx, database.UpdateNotificationTemplateMethodByIDParams{
		ID:     template,
		Method: database.NullNotificationMethod{NotificationMethod: customMethod, Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, customMethod, out.Method.NotificationMethod)

	// GIVEN: a manager configured with multiple dispatch methods
	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(defaultMethod), func(vals *codersdk.DeploymentValues) {
		require.NoError(t, vals.Notifications.SMTP.From.Set("danny@coder.com"))
		vals.Notifications.SMTP.Hello = "localhost"
		vals.Notifications.SMTP.Smarthost = serpent.HostPort{Host: "localhost", Port: fmt.Sprintf("%d", mockSMTPSrv.PortNumber())}
		vals.Notifications.Webhook = codersdk.NotificationsWebhookConfig{
			Endpoint: *serpent.URLOf(endpoint),
		}
	}).Notifications

	mgr, err := notifications.NewManager(cfg, coderd.NewRuntimeConfigStore(api.Database), api.Database, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mgr.Stop(ctx)
	})

	enq, err := notifications.NewStoreEnqueuer(cfg, api.Database, defaultHelpers(), api.Logger, quartz.NewReal())
	require.NoError(t, err)

	// WHEN: a notification of that template is enqueued, it should be delivered with the configured method - not the default.
	member := createSampleOrgMember(t, api.Database)
	msgID, err := enq.Enqueue(ctx, member.UserID, member.OrganizationID, template, map[string]string{}, "test")
	require.NoError(t, err)

	// THEN: the notification should be received by the custom dispatch method
	mgr.Run(ctx)

	receivedMsgID := testutil.RequireRecvCtx(ctx, t, received)
	require.Equal(t, msgID.String(), receivedMsgID.String())

	// Ensure no messages received by default method (SMTP):
	msgs := mockSMTPSrv.MessagesAndPurge()
	require.Len(t, msgs, 0)

	// Enqueue a notification which does not have a custom method set to ensure default works correctly.
	msgID, err = enq.Enqueue(ctx, member.UserID, member.OrganizationID, notifications.TemplateWorkspaceDeleted, map[string]string{}, "test")
	require.NoError(t, err)
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		msgs := mockSMTPSrv.MessagesAndPurge()
		if assert.Len(ct, msgs, 1) {
			assert.Contains(ct, msgs[0].MsgRequest(), fmt.Sprintf("Message-Id: %s", msgID))
		}
	}, testutil.WaitLong, testutil.IntervalFast)
}

func TestNotificationsTemplates(t *testing.T) {
	t.Parallel()

	// SETUP
	if !dbtestutil.WillUsePostgres() {
		// Notification system templates are only served from the database and not dbmem at this time.
		t.Skip("This test requires postgres; it relies on business-logic only implemented in the database")
	}

	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	api := coderdtest.New(t, createOpts(t))

	// GIVEN: the first user (owner) and a regular member
	firstUser := coderdtest.CreateFirstUser(t, api)
	memberClient, _ := coderdtest.CreateAnotherUser(t, api, firstUser.OrganizationID, rbac.RoleMember())

	// WHEN: requesting system notification templates as owner should work
	templates, err := api.GetSystemNotificationTemplates(ctx)
	require.NoError(t, err)
	require.True(t, len(templates) > 1)

	// WHEN: requesting system notification templates as member should work
	templates, err = memberClient.GetSystemNotificationTemplates(ctx)
	require.NoError(t, err)
	require.True(t, len(templates) > 1)
}

func createOpts(t *testing.T) *coderdtest.Options {
	t.Helper()

	dt := coderdtest.DeploymentValues(t)
	dt.Experiments = []string{string(codersdk.ExperimentNotifications)}
	return &coderdtest.Options{
		DeploymentValues: dt,
	}
}

// TestNotificationDuplicates validates that identical notifications cannot be sent on the same day.
func TestNotificationDuplicates(t *testing.T) {
	t.Parallel()

	// SETUP
	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres; it is testing the dedupe hash trigger in the database")
	}

	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	_, _, api := coderdtest.NewWithAPI(t, nil)

	method := database.NotificationMethodSmtp
	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(method)).Notifications

	mgr, err := notifications.NewManager(cfg, coderd.NewRuntimeConfigStore(api.Database), api.Database, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})

	// Set the time to a known value.
	mClock := quartz.NewMock(t)
	mClock.Set(time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC))

	enq, err := notifications.NewStoreEnqueuer(cfg, api.Database, defaultHelpers(), api.Logger.Named("enqueuer"), mClock)
	require.NoError(t, err)
	member := createSampleOrgMember(t, api.Database)

	// GIVEN: two notifications are enqueued with identical properties.
	_, err = enq.Enqueue(ctx, member.UserID, member.OrganizationID, notifications.TemplateWorkspaceDeleted, map[string]string{"initiator": "danny"}, "test", member.UserID)
	require.NoError(t, err)

	// WHEN: the second is enqueued, the enqueuer will reject the request.
	_, err = enq.Enqueue(ctx, member.UserID, member.OrganizationID, notifications.TemplateWorkspaceDeleted, map[string]string{"initiator": "danny"}, "test", member.UserID)
	require.ErrorIs(t, err, notifications.ErrDuplicate)

	// THEN: when the clock is advanced 24h, the notification will be accepted.
	// NOTE: the time is used in the dedupe hash, so by advancing 24h we're creating a distinct notification from the one
	// which was enqueued "yesterday".
	mClock.Advance(time.Hour * 24)
	_, err = enq.Enqueue(ctx, member.UserID, member.OrganizationID, notifications.TemplateWorkspaceDeleted, map[string]string{"initiator": "danny"}, "test", member.UserID)
	require.NoError(t, err)
}

func TestOrgLevelOverrides(t *testing.T) {
	t.Parallel()

	// SETUP

	// nolint:gocritic // Unit test.
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitSuperLong))
	_, _, api := coderdtest.NewWithAPI(t, nil)

	// start mock SMTP server
	mockSMTPSrv := smtpmock.New(smtpmock.ConfigurationAttr{
		LogToStdout:       false,
		LogServerActivity: true,
	})
	require.NoError(t, mockSMTPSrv.Start())
	t.Cleanup(func() {
		assert.NoError(t, mockSMTPSrv.Stop())
	})

	// initialize deployment values and new organization.
	vals := coderdtest.DeploymentValues(t)
	vals.Experiments = []string{string(codersdk.ExperimentMultiOrganization)}
	adminClient, _, _, _ := coderdenttest.NewWithAPI(t, &coderdenttest.Options{
		Options: &coderdtest.Options{DeploymentValues: vals},
		LicenseOptions: &coderdenttest.LicenseOptions{
			Features: license.Features{
				codersdk.FeatureMultipleOrganizations: 1,
			},
		},
	})
	altOrg := coderdenttest.CreateOrganization(t, adminClient, coderdenttest.CreateOrganizationOptions{})

	store := runtimeconfig.NewInMemoryStore()
	orgResolver := runtimeconfig.NewOrgResolver(altOrg.ID, runtimeconfig.NewStoreResolver(store))
	orgMutator := runtimeconfig.NewOrgMutator(altOrg.ID, runtimeconfig.NewStoreMutator(store))

	// GIVEN: an SMTP setup referencing a mock SMTP server
	const (
		from              = "danny@coder.com"
		orgOverriddenFrom = "bob@coder.com"
		to                = "rob@coder.com"
	)
	method := database.NotificationMethodSmtp
	cfg := coderdtest.DeploymentValues(t, defaultNotificationsMutator(method), func(vals *codersdk.DeploymentValues) {
		require.NoError(t, vals.Notifications.SMTP.From.Set(from))
		vals.Notifications.SMTP.Hello = "localhost"
		vals.Notifications.SMTP.Smarthost = serpent.HostPort{Host: "localhost", Port: fmt.Sprintf("%d", mockSMTPSrv.PortNumber())}
	}).Notifications

	// GIVEN:
	require.NoError(t, cfg.SMTP.From.Save(ctx, orgMutator, serpent.StringOf(ptr.Ref(orgOverriddenFrom))))
	val, err := cfg.SMTP.From.Coalesce(ctx, orgResolver)
	require.NoError(t, err)
	require.Equal(t, orgOverriddenFrom, val.String())

	handler := newDispatchInterceptor(dispatch.NewSMTPHandler(cfg.SMTP, defaultHelpers(), api.Logger.Named("smtp")))
	mgr, err := notifications.NewManager(cfg, store, api.Database, defaultHelpers(), createMetrics(), api.Logger.Named("manager"))
	require.NoError(t, err)
	mgr.WithHandlers(map[database.NotificationMethod]notifications.Handler{method: handler})
	t.Cleanup(func() {
		assert.NoError(t, mgr.Stop(ctx))
	})
	enq, err := notifications.NewStoreEnqueuer(cfg, api.Database, defaultHelpers(), api.Logger.Named("enqueuer"), quartz.NewReal())
	require.NoError(t, err)

	member := createSampleOrgMember(t, api.Database, func(u *database.User, o *database.Organization) {
		o.ID = altOrg.ID
		u.Email = to
	})

	// WHEN: a message is enqueued
	msgID, err := enq.Enqueue(ctx, member.UserID, altOrg.ID, notifications.TemplateWorkspaceDeleted, map[string]string{}, "test")
	require.NoError(t, err)

	mgr.Run(ctx)

	// THEN: wait until the dispatch interceptor validates that the messages were dispatched
	require.Eventually(t, func() bool {
		assert.Nil(t, handler.lastErr.Load())
		assert.True(t, handler.retryable.Load() == 0)
		return handler.sent.Load() == 1
	}, testutil.WaitLong, testutil.IntervalMedium)

	// THEN: we verify that the expected message was received by the mock SMTP server and delivered from the overridden address
	msgs := mockSMTPSrv.MessagesAndPurge()
	require.Len(t, msgs, 1)
	require.Contains(t, msgs[0].MsgRequest(), fmt.Sprintf("From: %s", orgOverriddenFrom))
	require.Contains(t, msgs[0].MsgRequest(), fmt.Sprintf("To: %s", to))
	require.Contains(t, msgs[0].MsgRequest(), fmt.Sprintf("Message-Id: %s", msgID))
}

type fakeHandler struct {
	mu                sync.RWMutex
	succeeded, failed []string
}

func (f *fakeHandler) Dispatcher(payload types.MessagePayload, _, _ string) (dispatch.DeliveryFunc, error) {
	return func(_ context.Context, cfgResolver runtimeconfig.Resolver, msgID uuid.UUID) (retryable bool, err error) {
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

// noopStoreSyncer pretends to perform store syncs, but does not; leading to messages being stuck in "leased" state.
type noopStoreSyncer struct {
	*acquireSignalingInterceptor
}

func newNoopStoreSyncer(db notifications.Store) *noopStoreSyncer {
	return &noopStoreSyncer{newAcquireSignalingInterceptor(db)}
}

func (*noopStoreSyncer) BulkMarkNotificationMessagesSent(_ context.Context, arg database.BulkMarkNotificationMessagesSentParams) (int64, error) {
	return int64(len(arg.IDs)), nil
}

func (*noopStoreSyncer) BulkMarkNotificationMessagesFailed(_ context.Context, arg database.BulkMarkNotificationMessagesFailedParams) (int64, error) {
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
