package calls

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	sdk "github.com/green-api/whatsapp-chatbot-golang/calls"
)

const (
	maxConcurrentDeliveries = 4
	maxQueuedCalls          = 256
)

type EnqueueResult uint8

const (
	Created EnqueueResult = iota
	AlreadyExists
)

var errQueueUnavailable = errors.New("call queue unavailable")

type callExecutor interface {
	Execute(context.Context, *sdk.CallSession, sdk.TransitionCallback) (*sdk.CallExecutionResult, error)
	RejectIncoming(context.Context, string, string) (bool, error)
	NotifyIncomingEnded(string) bool
	RequestStop()
}

type incomingKey struct{ senderID, callID string }

type callEntry struct {
	session    *sdk.CallSession
	state      sdk.CallState
	language   string
	timer      *time.Timer
	incoming   incomingKey
	cancel     context.CancelFunc
	callerLeft bool
}

// CallCoordinator owns the FIFO and reserves the sender until delivery finishes.
// While Execute runs, chat handlers use the entry's state and language snapshot.
type CallCoordinator struct {
	executions     sync.WaitGroup
	deliveryCtx    context.Context
	cancelDelivery context.CancelFunc

	mu                                sync.Mutex
	executor                          callExecutor
	delivery                          CallDeliveryService
	fsm                               *sdk.CallStateMachine
	prompts                           map[string]string
	incomingTimeout                   time.Duration
	queue                             []*callEntry
	bySender                          map[string]*callEntry
	endedIncoming                     map[incomingKey]time.Time
	finishedAt                        map[string]time.Time
	wake                              chan struct{}
	done                              chan struct{}
	deliverySlots                     chan struct{}
	rejectionSlots                    chan struct{}
	background                        sync.WaitGroup
	started, closed, shutdownPrepared bool
}

func NewCallCoordinator(executor callExecutor, delivery CallDeliveryService, prompts map[string]string, incomingTimeout time.Duration) *CallCoordinator {
	deliveryCtx, cancelDelivery := context.WithCancel(context.Background())
	return &CallCoordinator{deliveryCtx: deliveryCtx, cancelDelivery: cancelDelivery,
		executor: executor, delivery: delivery, fsm: sdk.NewCallStateMachine(),
		prompts: prompts, incomingTimeout: incomingTimeout,
		bySender: make(map[string]*callEntry), endedIncoming: make(map[incomingKey]time.Time),
		finishedAt: make(map[string]time.Time), wake: make(chan struct{}, 1), done: make(chan struct{}),
		deliverySlots:  make(chan struct{}, maxConcurrentDeliveries),
		rejectionSlots: make(chan struct{}, maxConcurrentDeliveries),
	}
}

// Enqueue reserves a sender and schedules an outgoing call.
func (c *CallCoordinator) Enqueue(senderID, chatID, language string) (EnqueueResult, error) {
	return c.enqueue(sdk.NewCallSession(senderID, chatID, language))
}

func (c *CallCoordinator) enqueue(session *sdk.CallSession) (EnqueueResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return AlreadyExists, errQueueUnavailable
	}

	key := incomingKey{session.SenderID, session.CallID}
	now := time.Now()
	c.pruneEndedIncoming(now)

	if c.bySender[session.SenderID] != nil || c.endedIncoming[key].After(now) {
		return AlreadyExists, nil
	}

	if key.callID != "" && !session.IncomingDeadline.After(now) {
		return AlreadyExists, nil
	}

	if len(c.queue) >= maxQueuedCalls {
		return AlreadyExists, errQueueUnavailable
	}

	prompt, ok := c.prompts[session.Language]

	if !ok {
		prompt, ok = c.prompts["en"]
	}

	if ok {
		session.Prompt = &prompt
	}

	c.fsm.Apply(session, sdk.CallEventEnqueued)
	entry := &callEntry{session: session, state: session.State, language: session.Language, incoming: key}
	c.bySender[session.SenderID] = entry
	c.queue = append(c.queue, entry)

	if key.callID != "" {
		entry.timer = time.AfterFunc(time.Until(session.IncomingDeadline), func() { c.expireIncoming(entry) })
	}

	c.signalWorker()
	return Created, nil
}

func (c *CallCoordinator) signalWorker() {
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *CallCoordinator) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started && !c.closed {
		c.started = true
		go c.workerLoop()
	}
}

// Stop closes admission once. The context bounds the caller's wait.
func (c *CallCoordinator) Stop(ctx context.Context) error {
	c.beginShutdown()

	select {
	case <-c.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *CallCoordinator) beginShutdown() {
	c.mu.Lock()

	if !c.closed {
		c.closed = true
		go c.shutdown()
	}

	c.mu.Unlock()

}

// Shutdown drains accepted results after the last execution, including delivery
// on the worker. After either deadline it cancels the corresponding I/O and joins.
func (c *CallCoordinator) Shutdown(callBudget, deliveryBudget time.Duration) {
	c.beginShutdown()
	ended := make(chan struct{})
	go func() {
		c.executions.Wait()
		close(ended)
	}()

	timer := time.NewTimer(callBudget)
	defer timer.Stop()

	select {
	case <-ended:
	case <-timer.C:
		c.mu.Lock()

		for _, entry := range c.bySender {
			if entry.cancel != nil {
				entry.cancel()
			}
		}

		c.mu.Unlock()
		<-ended
	}

	timer.Reset(deliveryBudget)

	select {
	case <-c.done:
	case <-timer.C:
		c.cancelDelivery()
		<-c.done
	}

	c.cancelDelivery()
}

func (c *CallCoordinator) shutdown() {
	c.executor.RequestStop()
	c.mu.Lock()
	var reject []incomingKey

	for len(c.queue) > 0 {
		entry := c.queue[0]
		c.removeQueued(entry, sdk.CallEventShutdownRequested)

		if entry.incoming.callID != "" {
			reject = append(reject, entry.incoming)
		}
	}

	for _, key := range reject {
		c.rejectInBackground(key)
	}

	started := c.started
	c.shutdownPrepared = true
	c.mu.Unlock()
	c.signalWorker()

	if !started {
		c.background.Wait()
		close(c.done)
	}
}

func (c *CallCoordinator) workerLoop() {
	for {
		<-c.wake

		for c.runNext() {
		}

		c.mu.Lock()
		// Shutdown drains the queue and admits its rejections before waking us.
		stopped := c.shutdownPrepared
		c.mu.Unlock()

		if stopped {
			c.background.Wait()
			close(c.done)
			return
		}
	}
}

func (c *CallCoordinator) runNext() bool {
	c.mu.Lock()

	if c.closed || len(c.queue) == 0 {
		c.mu.Unlock()
		return false
	}

	entry := c.queue[0]

	if entry.incoming.callID != "" && !entry.session.IncomingDeadline.After(time.Now()) {
		c.removeQueued(entry, sdk.CallEventIncomingExpired)
		c.rejectInBackground(entry.incoming)
		c.mu.Unlock()
		return true
	}

	c.popQueued(entry)
	ctx, cancel := context.WithCancel(context.Background())
	c.executions.Add(1)
	entry.cancel = cancel
	c.fsm.Apply(entry.session, sdk.CallEventDequeued)
	entry.state = entry.session.State
	c.mu.Unlock()

	result := c.runSession(ctx, entry)
	c.executions.Done()
	cancel()
	c.finalizeSession(entry, result)
	return true
}

func (c *CallCoordinator) runSession(ctx context.Context, entry *callEntry) *sdk.CallExecutionResult {
	session := entry.session
	transition := func(s *sdk.CallSession, event sdk.CallEvent, details ...sdk.TransitionDetails) sdk.StateTransition {
		c.mu.Lock()
		defer c.mu.Unlock()

		if entry.callerLeft && s.State == sdk.CallStateRinging {
			event = sdk.CallEventCallerLeft
			details = nil
		}

		transition := c.fsm.Apply(s, event, details...)
		entry.state = s.State
		return transition
	}

	var result *sdk.CallExecutionResult
	err := ctx.Err()

	if err == nil {
		result, err = c.executor.Execute(ctx, session, transition)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if !session.Finished() {
		event := sdk.CallEventInternalError

		if entry.callerLeft {
			event = sdk.CallEventCallerLeft
		} else if c.closed {
			event = sdk.CallEventShutdownRequested
		}

		code := "executor_finished_without_terminal_state"

		if err != nil {
			code = "executor_failed"
		}

		c.fsm.Apply(session, event, sdk.TransitionDetails{ErrorCode: &code})
	}

	if !entry.callerLeft && !c.closed && (err != nil || session.State == sdk.CallStateFailed) {
		slog.Warn("Call execution failed", "session", session.SessionID, "error_code", session.ErrorCode, "error", err)
	}

	entry.cancel = nil
	entry.state = session.State
	c.rememberIncoming(entry.incoming)
	return result
}

func (c *CallCoordinator) finalizeSession(entry *callEntry, result *sdk.CallExecutionResult) {
	select {
	case c.deliverySlots <- struct{}{}:
		c.background.Add(1)
		go func() {
			defer c.background.Done()
			defer func() { <-c.deliverySlots }()
			c.deliver(entry, result)
		}()
	default:
		c.deliver(entry, result)
	}
}

func (c *CallCoordinator) deliver(entry *callEntry, result *sdk.CallExecutionResult) {
	c.delivery.Deliver(c.deliveryCtx, entry.session, result)
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	for sender, at := range c.finishedAt {
		if now.Sub(at) > sessionTimeout {
			delete(c.finishedAt, sender)
		}
	}

	c.finishedAt[entry.incoming.senderID] = now
	delete(c.bySender, entry.incoming.senderID)
}
