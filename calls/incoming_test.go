package calls

import (
	"context"
	"math"
	"testing"
	"time"

	sdk "github.com/green-api/whatsapp-chatbot-golang/calls"
)

func TestIncomingTimestampUnits(t *testing.T) {
	const seconds = 1_700_000_000.125

	for _, scale := range []float64{1, 1e3, 1e6} {
		call := ParseIncomingCall(map[string]any{"from": bob, "idMessage": "call", "status": "offer", "timestamp": seconds * scale})

		if call.Timestamp != time.Unix(1_700_000_000, 125_000_000) || call.SenderID != bob || call.CallID != "call" || call.Status != "offer" {
			t.Fatalf("timestamp scale %v: %+v", scale, call)
		}
	}

	for _, value := range []any{nil, "1700000000", true, math.NaN(), math.Inf(1), seconds * 1e9, float64(4102444800)} {
		if got := parseTimestamp(value); !got.IsZero() {
			t.Errorf("invalid %v parsed as %v", value, got)
		}
	}
}

func TestIncomingOfferRejectsStaleButPreservesDuplicateAndReplacement(t *testing.T) {
	rejected := make(chan incomingKey, 8)
	c := newTestCoordinator(t, fakeExecutor{reject: func(key incomingKey) { rejected <- key }}, nil)
	now := time.Now()
	c.HandleIncomingCall(offer(alice, "expired", now.Add(-time.Minute)), "en")
	c.HandleIncomingCall(offer(alice, "missing", time.Time{}), "en")

	c.HandleIncomingCall(offer(bob, "old", now), "ru")
	c.HandleIncomingCall(offer(bob, "old", time.Time{}), "ru")
	c.HandleIncomingCall(IncomingCall{SenderID: bob, CallID: "old", Status: "hungUp"}, "ru")
	c.HandleIncomingCall(offer(bob, "old", now), "ru")
	c.HandleIncomingCall(IncomingCall{SenderID: bob, CallID: "terminal-first", Status: "missed"}, "ru")
	c.HandleIncomingCall(offer(bob, "terminal-first", now), "ru")

	c.HandleIncomingCall(offer(bob, "new", now.Add(time.Minute)), "ru")
	c.HandleIncomingCall(IncomingCall{SenderID: bob, CallID: "old", Status: "declined"}, "ru")
	c.HandleIncomingCall(offer(bob, "busy", now), "ru")

	if got := c.HandleCallMessage(bob, "0"); got.MessageKey != "call_cancelled" || got.Language != "ru" {
		t.Fatalf("current call cancellation = %+v", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := c.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	close(rejected)
	want := map[incomingKey]bool{{alice, "expired"}: true, {alice, "missing"}: true, {bob, "busy"}: true, {bob, "new"}: true}

	for key := range rejected {
		if !want[key] {
			t.Errorf("unexpected rejection: %+v", key)
		}

		delete(want, key)
	}

	if len(want) != 0 {
		t.Fatalf("missing rejections: %v", want)
	}
}

func TestIncomingHangupBeforeSDKRegistersRunner(t *testing.T) {
	entering := make(chan struct{})
	gotReason := make(chan sdk.CallEndReason, 1)
	next := make(chan struct{}, 1)
	proceed := make(chan struct{})
	c := newTestCoordinator(t, fakeExecutor{execute: func(ctx context.Context, s *sdk.CallSession, transition sdk.TransitionCallback) (*sdk.CallExecutionResult, error) {
		if s.Direction == sdk.CallDirectionOutgoing {
			next <- struct{}{}
			return nil, nil
		}

		close(entering)
		<-proceed
		// Mirrors Execute returning before the SDK runner exists.
		return nil, ctx.Err()
	}}, func(s *sdk.CallSession) error {
		if s.Direction == sdk.CallDirectionIncoming {
			gotReason <- s.EndReason
		}

		return nil
	})
	defer close(proceed)
	c.HandleIncomingCall(offer(bob, "handoff", time.Now()), "en")
	enqueue(t, c, alice, Created)
	c.Start()
	receive(t, entering)

	c.HandleIncomingCall(IncomingCall{SenderID: bob, CallID: "handoff", Status: "hungUp"}, "en")
	c.mu.Lock()
	cancelled := c.bySender[bob].callerLeft
	c.mu.Unlock()

	if !cancelled {
		t.Fatal("hangup was lost between queue and SDK runner")
	}

	// Let the fake SDK return so the worker can process the next call.
	proceed <- struct{}{}

	if got := receive(t, gotReason); got != sdk.CallEndReasonCallerLeft {
		t.Fatalf("terminal reason = %s", got)
	}

	receive(t, next)
}

func TestExpiredQueueEntryDoesNotRemoveNewCall(t *testing.T) {
	rejected := make(chan incomingKey, 2)
	c := newTestCoordinator(t, fakeExecutor{reject: func(key incomingKey) { rejected <- key }}, nil)
	c.HandleIncomingCall(offer(bob, "old", time.Now()), "en")
	old := c.bySender[bob]
	c.expireIncoming(old)

	if got := receive(t, rejected); got.callID != "old" {
		t.Fatal(got)
	}

	c.HandleIncomingCall(offer(bob, "replacement", time.Now()), "en")
	c.expireIncoming(old)

	if got := c.HandleCallMessage(bob, "other"); got.MessageKey != "call_queued" {
		t.Fatalf("stale timer changed the replacement: %+v", got)
	}
}

func TestWorkerRechecksOfferDeadlineBeforeExecuting(t *testing.T) {
	rejected := make(chan incomingKey, 1)
	c := newTestCoordinator(t, fakeExecutor{reject: func(key incomingKey) { rejected <- key }}, nil)
	c.HandleIncomingCall(offer(alice, "expires", time.Now()), "en")
	// Simulate the deadline passing while the timer callback is delayed.
	c.mu.Lock()
	c.bySender[alice].session.IncomingDeadline = time.Now().Add(-time.Second)
	c.mu.Unlock()

	if !c.runNext() || c.HandleCallMessage(alice, "0").Handled {
		t.Fatal("expired call should be released before Execute")
	}

	if got := receive(t, rejected); got != (incomingKey{alice, "expires"}) {
		t.Fatalf("rejection = %+v", got)
	}
}

func TestAnsweredIncomingHangupEndsNormally(t *testing.T) {
	answered, ended := make(chan struct{}), make(chan string, 1)
	reason := make(chan sdk.CallEndReason, 1)
	c := newTestCoordinator(t, fakeExecutor{
		execute: func(_ context.Context, s *sdk.CallSession, transition sdk.TransitionCallback) (*sdk.CallExecutionResult, error) {
			transition(s, sdk.CallEventRemoteAccepted)
			close(answered)

			select {
			case <-ended:
				transition(s, sdk.CallEventRemoteIdle)
			case <-time.After(time.Second):
				return nil, context.DeadlineExceeded
			}

			return nil, nil
		},
		ended: func(id string) bool {
			ended <- id
			return true
		},
	}, func(s *sdk.CallSession) error {
		reason <- s.EndReason
		return nil
	})

	c.HandleIncomingCall(offer(alice, "current", time.Now()), "en")
	c.Start()
	receive(t, answered)
	c.HandleIncomingCall(IncomingCall{SenderID: alice, CallID: "old", Status: "hungUp"}, "en")
	c.HandleIncomingCall(IncomingCall{SenderID: alice, CallID: "current", Status: "hungUp"}, "en")

	if got := receive(t, reason); got != sdk.CallEndReasonRemoteHangup {
		t.Fatalf("end reason = %s, want remote hangup", got)
	}
}
