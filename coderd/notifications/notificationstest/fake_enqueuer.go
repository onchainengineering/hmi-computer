package notificationstest

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/coderd/rbac/policy"
)

type FakeEnqueuer struct {
	authorizer rbac.Authorizer
	mu         sync.Mutex
	sent       []*FakeNotification
}

type FakeNotification struct {
	UserID, TemplateID uuid.UUID
	Labels             map[string]string
	Data               map[string]any
	CreatedBy          string
	Targets            []uuid.UUID
}

func (f *FakeEnqueuer) assertRBACNoLock(ctx context.Context) {
	if f.mu.TryLock() {
		panic("Developer error: do not call assertRBACNoLock outside of a mutex lock!")
	}

	// If we get here, we are locked.
	if f.authorizer == nil {
		f.authorizer = rbac.NewStrictCachingAuthorizer(prometheus.NewRegistry())
	}

	act, ok := dbauthz.ActorFromContext(ctx)
	if !ok {
		panic("Developer error: no actor in context, you may need to use dbauthz.AsNotifier(ctx)")
	}

	err := f.authorizer.Authorize(ctx, act, policy.ActionCreate, rbac.ResourceNotificationMessage)
	if err == nil {
		return
	}

	if rbac.IsUnauthorizedError(err) {
		panic("Developer error: not authorized to send notification msg. " +
			"Ensure that you are using dbauthz.AsXXX with an actor that has " +
			"policy.ActionCreate on rbac.ResourceNotificationMessage")
	}
	panic("Developer error: failed to check auth:" + err.Error())
}

func (f *FakeEnqueuer) Enqueue(ctx context.Context, userID, templateID uuid.UUID, labels map[string]string, createdBy string, targets ...uuid.UUID) (*uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.assertRBACNoLock(ctx)
	return f.enqueueWithDataNoLock(ctx, userID, templateID, labels, nil, createdBy, targets...)
}

func (f *FakeEnqueuer) EnqueueWithData(ctx context.Context, userID, templateID uuid.UUID, labels map[string]string, data map[string]any, createdBy string, targets ...uuid.UUID) (*uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.assertRBACNoLock(ctx)
	return f.enqueueWithDataNoLock(ctx, userID, templateID, labels, data, createdBy, targets...)
}

func (f *FakeEnqueuer) enqueueWithDataNoLock(_ context.Context, userID, templateID uuid.UUID, labels map[string]string, data map[string]any, createdBy string, targets ...uuid.UUID) (*uuid.UUID, error) {
	if f.mu.TryLock() {
		panic("Developer error: do not call enqueueWithDataNoLock outside of a mutex lock!")
	}

	f.sent = append(f.sent, &FakeNotification{
		UserID:     userID,
		TemplateID: templateID,
		Labels:     labels,
		Data:       data,
		CreatedBy:  createdBy,
		Targets:    targets,
	})

	id := uuid.New()
	return &id, nil
}

func (f *FakeEnqueuer) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sent = nil
}

func (f *FakeEnqueuer) Sent() []*FakeNotification {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*FakeNotification{}, f.sent...)
}
