# Contributing to the WhatsApp demo chatbot

These review requirements apply to human and AI contributions. Existing handlers illustrate current behavior, including unchecked assertions and large callbacks. New and materially changed code must meet the rules below. For observed behavior see [architecture](ARCHITECTURE.md). Agents begin with [AGENTS.md](AGENTS.md).

## From request to review

1. Identify the user action and expected observable result. Trace its entry, state reads, sends, errors and subsequent scene. For startup, trace instance settings before notification receipt.
2. Record the compatibility surface and smallest sufficient change. Include necessary tests, translations and documentation. Reuse current code, configuration, standard library and dependencies where they fit.
3. Define success, malformed input, boundary and partial completion cases before editing. Choose local fakes for remote calls. Keep customer identifiers, message bodies, keys and private endpoints out of examples, fixtures and logs.
4. Implement within touched scope. Bring materially changed functions into compliance without sweeping adjacent legacy code. Run relevant checks and submit the results, behavioral differences, remaining uncertainty and any migration or exception for review.

## Function size and contracts

**Review rejects new or materially changed functions exceeding 50 nonblank, noncomment lines in the body. Target about 30.** Count body lines containing code, including code with trailing comments. Apply the rule to anonymous handlers too. Reject deep nesting, giant callbacks and meaningless forwarding wrappers used to evade the limit. Each function has one concrete reason to exist. Within five minutes a newcomer should understand its inputs, outputs, errors, effects, state changes and owner. This is a manual review gate.

Use common English and concrete Go domain names. A predicate answers a question. Validate functions describe a check that can return an error. Preserve existing public names unless a migration requires changing them. Make assumptions, cancellation limits and side effects visible in signatures and contract comments. Explain why a restriction matters.

### Decode a group identifier before using it

**Bad.** A single handle function ignores a JSON error, asserts chatId and starts creating dialogue state. A malformed success body then panics after the remote group already exists.

**Good.** Synthetic Go declarations below and in subsequent examples illustrate proposed patterns. Each block stands independently.

```go
func decodeGroupID(body []byte) (string, error) {
	var reply struct {
		ChatID string `json:"chatId"`
	}
	if err := json.Unmarshal(body, &reply); err != nil {
		return "", fmt.Errorf("decode group response: %w", err)
	}
	if reply.ChatID == "" {
		return "", errors.New("missing group chatId")
	}
	return reply.ChatID, nil
}
```

**Why.** The caller owns recovery after remote creation. A decoder can validate the response without making another request or writing state. **Tests.** Exercise a valid ID with extra fields, omitted/empty ID, wrong JSON type and truncated JSON. At the caller verify decode failure prevents state creation and a second create attempt. **Review.** Reject conflating an unusable response with proof that the remote creation failed.

## Dependency direction and extension

Business decisions depend on the capabilities they use. Define small interfaces near the consuming policy at a demonstrated integration or testing boundary. Inject the concrete adapter at setup and test the policy with a fake. Keep fixed local details concrete. Reject SDK payload construction inside unrelated high-level decisions, speculative factories, interfaces for every type and new hidden globals.

### Report an invite after group creation

**Bad.** A confirmation handler constructs its own client to send an invite. Testing the missing-invite branch can then require an instance, and a test cannot observe which dialogue received the link.

**Good.** The consumer below expects an already-localized prefix. The function adapter can capture the original notification’s send operation at the boundary. The fake records calls and injects failure.

```go
type inviteSender interface {
	SendInvite(text string) error
}

type inviteFunc func(string) error

func (send inviteFunc) SendInvite(text string) error { return send(text) }

func reportInvite(sender inviteSender, prefix, link string) error {
	if link == "" {
		return errors.New("group invite missing")
	}
	return sender.SendInvite(prefix + link)
}

type fakeSender struct {
	texts []string
	err   error
}

func (s *fakeSender) SendInvite(text string) error {
	s.texts = append(s.texts, text)
	return s.err
}
```

**Why.** At setup, wrap the delivery closure with inviteFunc and pass it to reportInvite. The real adapter binds the originating dialogue and translates the SDK send result into an error. The policy needs neither credentials nor SDK response bodies. **Tests.** Capture the exact localized message once, inject a send error and assert it reaches the caller, and expect zero calls for an empty link. Separately check adapter recipient binding with distinct group and direct-dialogue IDs. **Review.** Reject retrying group creation to recover from invite delivery failure.

## Explicit transitions

Where correctness depends on allowed states and events, centralize the rules in an explicit FSM or equivalent model. Give one owner authority to commit transitions. Specify invalid, repeated and terminal behavior and test it. Keep planning separate from I/O where feasible. A simple boolean alone needs no FSM. Current scene routing is distributed. Preserve it unless the change intentionally migrates it.

### A confirmation with duplicate input

**Bad.** Two callbacks each check a pending flag, send a create request and clear the flag afterward. Overlapping confirmations can both send.

**Good.** This synthetic policy reserves one attempt. A serialized dialogue owner must apply the result before issuing I/O. The owner emits one send only on entry into creating. A failed or ambiguous remote result requires reconciliation before any new attempt.

```go
type phase uint8
const (
	pending phase = iota
	creating
	closed
)

func nextPhase(current phase, event string) (phase, error) {
	switch {
	case current == pending && event == "confirm":
		return creating, nil
	case current == creating && event == "confirm":
		return current, nil
	case current == creating && event == "settle":
		return closed, nil
	case current == closed && event == "settle":
		return current, nil
	default:
		return current, fmt.Errorf("invalid event %q in phase %d", event, current)
	}
}
```

**Why.** Repeated confirmation while creating causes no new entry effect. Closed rejects confirmation. Local serialization alone gives no cross-restart delivery guarantee. **Tests.** Table-test each pair, including settle while pending, repeated settle and unknown events. Deliver concurrent confirmations through the owner and count one create request. **Review.** Reject a correct transition table paired with an unlocked read/plan/write sequence.

## Resource ownership, cancellation and errors

Name the owner of every goroutine, queue, timer, connection, file and stream. Specify acquisition, bounds, shutdown order and cleanup on partial acquisition. Higher layers use one lifecycle method. Define repeated Close or Stop behavior and surface cleanup failure. Propagate cancellation to work that supports it. Bound concurrency and queues and state who closes channels and waits for workers.

### Own a cancellable notification worker

**Bad.** A constructor opens a stream, returns an error on failed setup and leaks the stream. On success it launches a reader with no shutdown handle.

**Good.** This synthetic owner accepts an already-acquired stream and takes ownership on entry. Setup borrows it synchronously. Run must return when its context is cancelled. The stream is closed exactly once after use.

```go
type worker struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

func startWorker(parent context.Context, stream io.Closer,
	setup func() error, run func(context.Context) error) (*worker, error) {
	if err := setup(); err != nil {
		return nil, errors.Join(err, stream.Close())
	}
	ctx, cancel := context.WithCancel(parent)
	w := &worker{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(w.done)
		defer cancel()
		w.err = errors.Join(run(ctx), stream.Close())
	}()
	return w, nil
}

func (w *worker) Stop() error {
	w.cancel()
	<-w.done
	return w.err
}
```

**Why.** Completion publishes the stored result to all waiters. Concurrent or repeated Stop calls return the same result, including cancellation if run reports it. An uninterruptible read would violate the run contract and hang Stop. **Tests.** Count closes on setup failure, normal completion, run failure and parent cancellation. Inject a close error alongside a run error and check both with errors.Is. Use a deadline and concurrent Stop callers under the race detector. **Review.** Reject workers that claim cancellability while their I/O can block past shutdown.

Handle errors where the caller can act. Preserve useful wrapped causes and existing observability without logging payloads or credentials. For any retry specify bounded attempts, timeout, ordering, idempotency and behavior after ambiguous success. A prior send, group creation or GPT reply may already have happened. Reject swallowed errors, unbounded concurrency, duplicate logic and workarounds lacking scope, reason and removal conditions.

## Compatibility and customization

Preserve environment names, settings flags, menu numbers, button IDs and labels, locale keys, per-dialogue state types, expiry threshold, error behavior, ordering and session JSON. Check old data, partial failures and active dialogues. Intentional breaks require an explicit migration and compatibility tests. For branding, try configuration, application strings/media and scene code first. Use existing seams where they fit, then consider a narrow injected capability for a demonstrated need. Provider-wide changes or a maintained fork carry constructor, lifecycle and upgrade costs. Keep branding and business policy in the application. Review LICENCE and distribution rights before sharing a derivative.

## Verification by change

From the module root use the go.mod toolchain. Format touched Go files with gofmt. Run `go test ./...` and `go vet ./...`. For concurrency or shared state run `go test -race ./...` with relevant behavioral tests. There is currently no tracked Go suite or CI. A successful package check alone gives no evidence about notification delivery.

| Change | Required local evidence |
| --- | --- |
| Startup | Check every missing credential and settings failure without contacting an account. Assert settings and start order with a fake when changing orchestration. |
| Routing and localization | Table-test each affected input, fresh/expired/missing timestamps, exact timeout boundary, locale keys, next scene and reply. Check both READMEs for behavior or version drift. |
| Poll, group or button operation | Use synthetic provider bodies. Check malformed and short votes, repeated input, IDs and recipient distinction. Inject failures after each remote effect and record subsequent calls. |
| GPT | Exercise exit before processing, absent/corrupt and existing session JSON, fresh re-entry, helper failure, nil updated session and send/save order. Use a fake for model output. |
| Documentation | Resolve changed links, check snippets against actual symbols or label them synthetic, run git diff --check. |

Running main changes instance settings and consumes notifications. Routine verification uses local fakes. Live interoperability requires separate authorization, a dedicated test instance and credentials, and a record of actual checks.

## Review decision

Reject mandatory-rule violations, unapproved compatibility breaks, avoidable debt, unclear ownership, sensitive data exposure, unverifiable correctness or missing risk-appropriate tests. Review touched legacy functions against the same gates. Only an explicit reviewer may grant a narrowly scoped exception. Record its reason, affected code, risk, verification and removal condition. An agent cannot approve its own exception.
