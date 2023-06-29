package tailnet

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"cdr.dev/slog"
)

type MultiAgentConn interface {
	UpdateSelf(node *Node) error
	SubscribeAgent(agentID uuid.UUID) error
	UnsubscribeAgent(agentID uuid.UUID) error
	NextUpdate(ctx context.Context) ([]*Node, bool)
	AgentIsLegacy(agentID uuid.UUID) bool
	Close() error
	IsClosed() bool
}

type MultiAgent struct {
	mu     sync.RWMutex
	closed bool

	ID     uuid.UUID
	Logger slog.Logger

	AgentIsLegacyFunc func(agentID uuid.UUID) bool
	OnSubscribe       func(enq Enqueueable, agent uuid.UUID) error
	OnUnsubscribe     func(enq Enqueueable, agent uuid.UUID) error
	OnNodeUpdate      func(id uuid.UUID, node *Node) error
	OnRemove          func(id uuid.UUID)

	updates    chan []*Node
	closeOnce  sync.Once
	start      int64
	lastWrite  int64
	overwrites int64
}

func (m *MultiAgent) Init() *MultiAgent {
	m.updates = make(chan []*Node, 128)
	m.start = time.Now().Unix()
	return m
}

func (m *MultiAgent) UniqueID() uuid.UUID {
	return m.ID
}

func (m *MultiAgent) AgentIsLegacy(agentID uuid.UUID) bool {
	return m.AgentIsLegacyFunc(agentID)
}

func (m *MultiAgent) UpdateSelf(node *Node) error {
	return m.OnNodeUpdate(m.ID, node)
}

func (m *MultiAgent) SubscribeAgent(agentID uuid.UUID) error {
	return m.OnSubscribe(m, agentID)
}

func (m *MultiAgent) UnsubscribeAgent(agentID uuid.UUID) error {
	return m.OnUnsubscribe(m, agentID)
}

func (m *MultiAgent) NextUpdate(ctx context.Context) ([]*Node, bool) {
	select {
	case <-ctx.Done():
		return nil, false

	case nodes := <-m.updates:
		return nodes, true
	}
}

func (m *MultiAgent) Enqueue(nodes []*Node) error {
	atomic.StoreInt64(&m.lastWrite, time.Now().Unix())

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil
	}

	select {
	case m.updates <- nodes:
		return nil
	default:
		return ErrWouldBlock
	}
}

func (m *MultiAgent) Name() string {
	return m.ID.String()
}

func (m *MultiAgent) Stats() (start int64, lastWrite int64) {
	return m.start, atomic.LoadInt64(&m.lastWrite)
}

func (m *MultiAgent) Overwrites() int64 {
	return m.overwrites
}

func (m *MultiAgent) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

func (m *MultiAgent) CoordinatorClose() error {
	m.mu.Lock()
	if !m.closed {
		m.closed = true
		close(m.updates)
	}
	m.mu.Unlock()
	return nil
}

func (m *MultiAgent) Close() error {
	_ = m.CoordinatorClose()
	m.closeOnce.Do(func() { m.OnRemove(m.ID) })
	return nil
}
