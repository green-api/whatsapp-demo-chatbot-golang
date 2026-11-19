package calls

import (
	"context"
	"errors"
	"testing"
	"time"

	sdk "github.com/green-api/whatsapp-chatbot-golang/calls"
)

const (
	alice = "10001@c.us"
	bob   = "10002@lid"
	carol = "10003@c.us"
)

type fakeExecutor struct {
	execute func(context.Context, *sdk.CallSession, sdk.TransitionCallback) (*sdk.CallExecutionResult, error)
	reject  func(incomingKey)
	ended   func(string) bool
	stop    func()
}

func (f fakeExecutor) Execute(ctx context.Context, s *sdk.CallSession, transition sdk.TransitionCallback) (*sdk.CallExecutionResult, error) {
	return f.execute(ctx, s, transition)
}

func (f fakeExecutor) RejectIncoming(_ context.Context, id, sender string) (bool, error) {
	if f.reject != nil {
		f.reject(incomingKey{sender, id})
	}

	return true, nil
}

func (f fakeExecutor) NotifyIncomingEnded(id string) bool {
	return f.ended != nil && f.ended(id)
}

func (f fakeExecutor) RequestStop() {
	if f.stop != nil {
		f.stop()
	}
}

type notifyFunc func(*sdk.CallSession) error

func (f notifyFunc) NotifyResult(_ context.Context, s *sdk.CallSession) error { return f(s) }

func newTestCoordinator(t *testing.T, executor fakeExecutor, notify notifyFunc) *CallCoordinator {
	t.Helper()

	if notify == nil {
		notify = func(*sdk.CallSession) error { return nil }
	}

	c := NewCallCoordinator(executor, CallDeliveryService{Notifier: notify}, map[string]string{"en": "hello"}, time.Minute)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := c.Stop(ctx); err != nil {
			t.Error(err)
		}
	})

	return c
}

func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()

	select {
	case value := <-ch:
		return value
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for call work")
		var zero T
		return zero
	}
}

func offer(sender, id string, at time.Time) IncomingCall {
	return IncomingCall{SenderID: sender, CallID: id, Status: "offer", Timestamp: at}
}

func enqueue(t *testing.T, c *CallCoordinator, sender string, want EnqueueResult) {
	t.Helper()

	if got, err := c.Enqueue(sender, sender, "en"); err != nil || got != want {
		t.Fatalf("enqueue %s = %v, %v; want %v", sender, got, err, want)
	}
}

func TestCallCoordinatorSharesFIFOAndContinuesAfterError(t *testing.T) {
	type execution struct {
		sender string
		state  sdk.CallState
	}
	executed := make(chan execution, 3)
	c := newTestCoordinator(t, fakeExecutor{execute: func(_ context.Context, s *sdk.CallSession, transition sdk.TransitionCallback) (*sdk.CallExecutionResult, error) {
		executed <- execution{s.SenderID, s.State}

		if s.SenderID == alice {
			return nil, errors.New("fake executor failure")
		}

		transition(s, sdk.CallEventRemoteIdle)
		return nil, nil
	}}, nil)

	enqueue(t, c, alice, Created)
	c.HandleIncomingCall(offer(bob, "call-1", time.Now()), "en")
	enqueue(t, c, bob, AlreadyExists)
	enqueue(t, c, carol, Created)
	c.Start()

	for _, want := range []execution{{alice, sdk.CallStateDialing}, {bob, sdk.CallStateRinging}, {carol, sdk.CallStateDialing}} {
		if got := receive(t, executed); got != want {
			t.Fatalf("execution = %+v; want %+v", got, want)
		}
	}
}

func TestCallCoordinatorRetainsSenderThroughDelivery(t *testing.T) {
	executing, delivering := make(chan string, 2), make(chan string, 2)
	finishCall, finishDelivery := make(chan struct{}), make(chan struct{})
	c := newTestCoordinator(t, fakeExecutor{
		stop: func() {
			select {
			case <-finishCall:
			default:
				close(finishCall)
			}
		},
		execute: func(_ context.Context, s *sdk.CallSession, transition sdk.TransitionCallback) (*sdk.CallExecutionResult, error) {
			transition(s, sdk.CallEventRemoteAccepted)
			transition(s, sdk.CallEventBridgeReady)
			executing <- s.SenderID
			<-finishCall
			transition(s, sdk.CallEventRemoteIdle)
			return nil, nil
		},
	}, func(s *sdk.CallSession) error {
		delivering <- s.SenderID
		<-finishDelivery
		return nil
	})
	defer close(finishDelivery)

	enqueue(t, c, alice, Created)
	c.Start()

	if receive(t, executing) != alice {
		t.Fatal("first call belongs to the wrong sender")
	}

	if got := c.HandleCallMessage(alice, " 0 "); !got.Handled || got.MessageKey != "" {
		t.Fatalf("0 should be consumed silently during a connected call: %+v", got)
	}

	if got := c.HandleCallMessage(alice, "menu"); got.MessageKey != "call_in_progress" || !got.Handled {
		t.Fatalf("in-progress action = %+v", got)
	}

	close(finishCall)

	if receive(t, delivering) != alice || c.HandleCallMessage(alice, "0").MessageKey != "call_processing" {
		t.Fatal("0 should report delivery processing")
	}

	enqueue(t, c, alice, AlreadyExists)
	enqueue(t, c, bob, Created)

	if receive(t, executing) != bob {
		t.Fatal("another sender should run while the result is being sent")
	}
}

func TestCallCoordinatorCancelsQueuedCallAndAllowsReentry(t *testing.T) {
	executing := make(chan string, 1)
	c := newTestCoordinator(t, fakeExecutor{execute: func(_ context.Context, s *sdk.CallSession, _ sdk.TransitionCallback) (*sdk.CallExecutionResult, error) {
		executing <- s.SenderID
		return nil, nil
	}}, nil)

	enqueue(t, c, alice, Created)

	if got := c.HandleCallMessage(alice, "repeat"); got.MessageKey != "call_queued" || got.ShowMenu {
		t.Fatalf("queued action = %+v", got)
	}

	if got := c.HandleCallMessage(alice, " 0 "); (got != CallChatAction{"call_cancelled", "en", true, true}) {
		t.Fatalf("cancel action = %+v", got)
	}

	if c.HandleCallMessage(alice, "0").Handled {
		t.Fatal("repeated cancellation should reach the scene")
	}

	enqueue(t, c, alice, Created)
	c.Start()

	if receive(t, executing) != alice {
		t.Fatal("replacement call did not execute")
	}
}

func TestStopRejectsQueuedIncomingAndClosesAdmission(t *testing.T) {
	rejected := make(chan incomingKey, 1)
	c := newTestCoordinator(t, fakeExecutor{reject: func(key incomingKey) { rejected <- key }}, nil)
	enqueue(t, c, alice, Created)
	c.HandleIncomingCall(offer(bob, "pending", time.Now()), "en")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := c.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	if got := receive(t, rejected); got != (incomingKey{bob, "pending"}) {
		t.Fatalf("wrong rejection: %+v", got)
	}

	if _, err := c.Enqueue(carol, carol, "en"); !errors.Is(err, errQueueUnavailable) {
		t.Fatalf("closed admission error = %v", err)
	}
}

func TestCallActivityRefreshesAndConsumesCompletion(t *testing.T) {
	delivered := make(chan struct{})
	c := newTestCoordinator(t, fakeExecutor{execute: func(_ context.Context, s *sdk.CallSession, transition sdk.TransitionCallback) (*sdk.CallExecutionResult, error) {
		transition(s, sdk.CallEventRemoteIdle)
		return nil, nil
	}}, func(*sdk.CallSession) error {
		close(delivered)
		return nil
	})

	before := time.Now()
	enqueue(t, c, alice, Created)

	if at := c.ActivityTimestamp(alice); at.Before(before) {
		t.Fatalf("queued activity = %v; want >= %v", at, before)
	}

	c.Start()
	receive(t, delivered)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := c.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	if at := c.ActivityTimestamp(alice); at.Before(before) {
		t.Fatalf("delivery activity = %v; want >= %v", at, before)
	}

	if !c.ActivityTimestamp(alice).IsZero() {
		t.Fatal("completion timestamp was not consumed")
	}
}
