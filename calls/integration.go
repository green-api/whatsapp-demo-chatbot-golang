package calls

import (
	"context"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"

	sdk "github.com/green-api/whatsapp-chatbot-golang/calls"
)

const sessionTimeout = 300 * time.Second

// IncomingCall holds the fields used by the demo from an incomingCall webhook.
type IncomingCall struct {
	SenderID, CallID, Status string
	Timestamp                time.Time
}

func ParseIncomingCall(body map[string]any) IncomingCall {
	sender, _ := body["from"].(string)
	id, _ := body["idMessage"].(string)
	status, _ := body["status"].(string)
	return IncomingCall{sender, id, status, parseTimestamp(body["timestamp"])}
}

func parseTimestamp(value any) time.Time {
	number, ok := value.(float64)

	if !ok {
		return time.Time{}
	}

	const minSeconds = 946684800
	const maxSeconds = 4102444800

	for _, scale := range []float64{1, 1e3, 1e6} {
		if number >= minSeconds*scale && number < maxSeconds*scale {
			seconds, fraction := math.Modf(number / scale)
			return time.Unix(int64(seconds), int64(fraction*1e9))
		}
	}

	return time.Time{}
}

// HandleIncomingCall admits a live offer or removes it when the caller leaves.
func (c *CallCoordinator) HandleIncomingCall(call IncomingCall, language string) {
	if call.CallID == "" || !(strings.HasSuffix(call.SenderID, "@c.us") || strings.HasSuffix(call.SenderID, "@lid")) {
		return
	}

	key := incomingKey{call.SenderID, call.CallID}

	switch call.Status {
	case "declined", "missed", "hungUp":
		c.incomingEnded(key)
	case "offer":
		c.enqueueOffer(call, key, language)
	}
}

func (c *CallCoordinator) enqueueOffer(call IncomingCall, key incomingKey, language string) {
	now := time.Now()
	remaining := c.incomingTimeout - max(0, now.Sub(call.Timestamp))

	if call.Timestamp.IsZero() || remaining <= 0 {
		c.rejectUnqueuedIncoming(key)
		return
	}

	session, err := sdk.NewIncomingCallSession(call.SenderID, call.CallID, language, now.Add(remaining))

	if err != nil {
		return
	}

	result, err := c.enqueue(session)

	if err != nil || result == AlreadyExists {
		c.rejectUnqueuedIncoming(key)
	}
}

func (c *CallCoordinator) incomingEnded(key incomingKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.rememberIncoming(key)
	entry := c.bySender[key.senderID]

	if entry == nil || entry.incoming != key {
		return
	}

	if entry.state == sdk.CallStateQueued {
		c.removeQueued(entry, sdk.CallEventCallerLeft)
		return
	}

	if entry.cancel != nil && !c.executor.NotifyIncomingEnded(key.callID) && entry.state == sdk.CallStateRinging {
		// Execute may still be entering the service. Cancel just this execution.
		entry.callerLeft = true
		entry.cancel()
	}
}

func (c *CallCoordinator) expireIncoming(entry *callEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed && c.bySender[entry.incoming.senderID] == entry && entry.state == sdk.CallStateQueued {
		c.removeQueued(entry, sdk.CallEventIncomingExpired)
		c.rejectInBackground(entry.incoming)
	}
}

func (c *CallCoordinator) rejectUnqueuedIncoming(key incomingKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := c.bySender[key.senderID]

	if c.closed || (entry != nil && entry.incoming == key) || c.endedIncoming[key].After(time.Now()) {
		return
	}

	c.rememberIncoming(key)
	c.rejectInBackground(key)
}

// rejectInBackground runs under the coordinator mutex. When all slots are
// occupied, it releases the mutex while applying backpressure to the caller.
func (c *CallCoordinator) rejectInBackground(key incomingKey) {
	c.background.Add(1)
	reject := func() {
		defer c.background.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if _, err := c.executor.RejectIncoming(ctx, key.callID, key.senderID); err != nil {
			slog.Warn("Unable to reject incoming call", "error", err)
		}
	}

	select {
	case c.rejectionSlots <- struct{}{}:
		go func() {
			defer func() { <-c.rejectionSlots }()
			reject()
		}()
	default:
		c.mu.Unlock()
		reject()
		c.mu.Lock()
	}
}

func (c *CallCoordinator) pruneEndedIncoming(now time.Time) {
	for key, deadline := range c.endedIncoming {
		if !deadline.After(now) {
			delete(c.endedIncoming, key)
		}
	}
}

func (c *CallCoordinator) rememberIncoming(key incomingKey) {
	if key.callID != "" {
		now := time.Now()
		c.pruneEndedIncoming(now)
		c.endedIncoming[key] = now.Add(c.incomingTimeout)
	}
}

func (c *CallCoordinator) popQueued(entry *callEntry) {
	if entry.timer != nil {
		entry.timer.Stop()
	}

	index := slices.Index(c.queue, entry)
	c.queue = slices.Delete(c.queue, index, index+1)
}

func (c *CallCoordinator) removeQueued(entry *callEntry, event sdk.CallEvent) {
	c.popQueued(entry)
	c.fsm.Apply(entry.session, event)
	entry.state = entry.session.State
	delete(c.bySender, entry.incoming.senderID)
	c.rememberIncoming(entry.incoming)
}

type CallChatAction struct {
	MessageKey, Language string
	ShowMenu, Handled    bool
}

// HandleCallMessage decides cancellation and status against the same queue state.
func (c *CallCoordinator) HandleCallMessage(senderID, text string) CallChatAction {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := c.bySender[senderID]

	if entry == nil {
		return CallChatAction{}
	}

	action := CallChatAction{Language: entry.language, Handled: true}
	isZero := strings.TrimSpace(text) == "0"

	switch entry.state {
	case sdk.CallStateQueued:
		action.MessageKey = "call_queued"

		if isZero && !c.closed {
			c.removeQueued(entry, sdk.CallEventCancelRequested)
			action.MessageKey = "call_cancelled"
			action.ShowMenu = true

			if entry.incoming.callID != "" {
				c.rejectInBackground(entry.incoming)
			}
		}
	case sdk.CallStateCancelled, sdk.CallStateRejected, sdk.CallStateRingTimeout,
		sdk.CallStateRemoteEnded, sdk.CallStateTalkTimeout, sdk.CallStateFailed:
		action.MessageKey = "call_processing"
	default:
		if !isZero {
			action.MessageKey = "call_in_progress"
		}
	}

	return action
}

// ActivityTimestamp consumes a completion timestamp once or refreshes a live call.
func (c *CallCoordinator) ActivityTimestamp(senderID string) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.bySender[senderID] != nil {
		return time.Now()
	}

	finished := c.finishedAt[senderID]
	delete(c.finishedAt, senderID)
	return finished
}
