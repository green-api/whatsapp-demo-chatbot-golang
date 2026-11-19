# whatsapp-demo-chatbot-golang

Instructions for AI agents working with the repository.

---

## 0) Quick facts about this repository

This Go application demonstrates WhatsApp conversations for developers evaluating GREEN-API. The operator supplies an instance and OpenAI credentials. The user interacts through WhatsApp. Local code owns scenes and replies. Dependencies handle WhatsApp transport, dispatch and model requests. Use go.mod for versions (Go 1.24.0, toolchain 1.24.2), main.go for startup and the scene switch plus strings.yaml for menu behavior.

Open [ARCHITECTURE.md](ARCHITECTURE.md) when explaining or changing startup, routing, integration boundaries, state or external effects. Read [CONTRIBUTING.md](CONTRIBUTING.md) before implementing or reviewing any code change. Load the documents explicitly when those conditions apply.

## 1) Agent role and operating modes

- **Product question.** Explain the conversation from the user’s perspective. Include relevant prerequisites and limitations. Scope source inspection to the question.
- **Integration.** Trace the constructor, credentials, working-directory assets and lifecycle before giving setup guidance. Identify what the operator must supply.
- **Engineering.** Follow the numbered protocol and the decision rules below. Distinguish the current implementation from requirements for changed code.
- **Customization.** Locate the smallest application-level change and explain its distribution implications before suggesting a fork.

## 2) Product knowledge and honest presentation

In a first introduction, explain that the demo can select a language, send media or contacts, collect poll votes, create a group and switch to GPT chat. For example, a developer can send a poll to a test conversation, check the localized reply and then select 14 to try GPT. The operator supplies hosting, a connected GREEN-API instance and an OpenAI key even for non-GPT demos. Durable storage and production operations are the operator’s responsibility.

For an API-oriented introduction, use this excerpt from main.go. These are chatbot dependency calls inside an executable. Startup also registers the GPT helper and configures the remote account before receipt.

```go
baseBot.SetStartScene(scenes.StartScene{})
baseBot.StartReceivingNotifications()
```

## 3) Agent engineering protocol

1. Classify the request and identify the affected user action or operator action.
2. Open the relevant companions and read the owning source and its callers.
3. Trace the input through state reads, remote calls, responses and next scene, including failure halfway through.
4. Identify compatibility obligations for existing dialogues, commands, locales, provider fields and settings.
5. Reuse the closest existing application seam. Justify a new abstraction by a real consumer or test.
6. Apply the function, transition, ownership and error rules before choosing a diff.
7. Implement the smallest complete change with its necessary tests and docs. Preserve unrelated user work.
8. Verify success, boundary and failure behavior locally with the checks appropriate to the risk.
9. Report changed behavior, preserved contracts, commands and results, and any external behavior still unverified.

## 4) Architecture and placement

Follow `main → SetSettings → StartScene → SDK notification loop` for process changes. Follow `StartScene → language selection → EndpointsScene → group or GPT scene` for dialogue changes. Start can interpret the current message as a language selection immediately. util owns timeout and YAML lookup, model describes the poll payload, and registry holds the concrete GPT helper. State and current scene are managed per dialogue by the SDK.

## 5) Integration and White Label

Prefer runtime configuration, then consumer application code (strings, media, handlers), then a suitable existing extension point. The GPT registry accepts a concrete pointer. If a real integration needs substitution, consider a narrow consumer-owned option or interface before changing core behavior. A maintained fork is the final step. Keep branding in application assets and policies. Check constructors, settings, session compatibility and LICENCE before proposing branded distribution. The checkout supplies no White Label distribution agreement.

## 6) Public and implicit contracts

Configuration names are ID_INSTANCE, AUTH_TOKEN and OPENAI_API_KEY. Treat their values as private. Scene state includes string lang, time.Time last_touch_timestamp and string gptSessionJson. User commands are scene-specific (0 stops at Endpoints but returns to menu in group confirmation). A reply, remote group, or saved session can outlive a later error in the same handler.

## 7) Domain decision rules

| When changing | Read and determine | Preserve | Verify locally |
| --- | --- | --- | --- |
| Startup or receipt | In main.go trace env checks, helper registration, ErrorChannel consumer and SetSettings before receipt. Determine which flags and failure exits change. | Incoming/poll delivery, read/online settings and disabled outgoing/state flags. Receipt acknowledgment and shutdown belong to the SDK. | Compare full settings arguments, missing-key exits and settings failure before dispatch. Use a fake for changed startup logic. |
| Routing and expiry | Read scenes/start.go, scenes/mainMenu.go, scenes/endpoints.go and util.IsSessionExpired. Identify the scene handling this event and the next one. | Strictly over 300 seconds expires. Missing timestamps are fresh. Checks refresh activity. Store lang before sending the first menu. | Trace first selection, invalid text, exactly 300 seconds and beyond, stop/menu and expiry from each touched scene. |
| Provider operations | Read EndpointsScene.processPollUpdate, model/model.go and scenes/createGroup.go. Determine recipients, expected types and the first irreversible call. | Vote matching by sender and position. Group state is keyed by chatId before avatar and invite. Handle partial success without replaying creation. | Fake short votes, absent voter, malformed group response, avatar error versus false, group send failure and original-dialogue invite. |
| GPT | Read option 14, registry/registry.go and scenes/gptScene.go. Determine fresh entry versus subsequent turn and when exit bypasses processing. | Locale exit words, universal menu, JSON string format, legacy key deletion, send before save. | Exercise re-entry, valid/invalid JSON, missing helper, processing failure and nil updated session. Check history and scene independently of delivery. |
| Localization and setup docs | Read util.GetString, strings.yaml, each affected handler, both READMEs and go.mod. Determine key order, selected locales and actual menu case. | Choices 1–5 select en/kz/ru/es/he. Localized button labels participate in routing. Media selection varies by scene. | Resolve every changed lookup per locale and check button replies against dispatch. Use manifest versions and source option numbers. |

## 8) Corporate engineering standards

New or materially changed functions must contain **at most 50 nonblank, noncomment body lines**, targeting about 30. Give each one responsibility, a plain English domain name and an explicit contract for inputs, effects, errors and cancellation. Apply the limit to callbacks. Reject artificial wrappers, deep nesting and giant lambdas. Keep fixed details concrete. Inject small consumer-owned interfaces only at demonstrated boundaries.

For constrained transitions require an explicit FSM or equivalent rules with one mutation owner and tested invalid, repeated and terminal events. Name the owner of each resource and goroutine, bound work, and define shutdown, partial acquisition and cleanup failures. Preserve errors, ordering and compatibility. Reject hidden globals, duplicate logic, unbounded retries and temporary workarounds without removal conditions. Mandatory violations, sensitive data, unjustified breaks or missing risk-appropriate tests fail review. Only an explicit reviewer can approve a scoped exception.

## 9) Verification and completion

Use gofmt on touched Go files, go test ./... and go vet ./... from the root. Use go test -race ./... for shared-state changes. Currently no root Go tests or CI exercise these flows. Add targeted tests for changed behavior. Check documentation links and git diff --check. Record what ran and what remains unverified.

---

## 10) Do Not

- Do not run main.go as a smoke test. It changes account settings and begins receiving notifications.
- Do not publish environment values, payloads, real identifiers or private endpoints.
- Do not infer delivery, retry, acknowledgment or persistence guarantees from a successful local check.
- Do not replay group creation or a model request merely because the final reply failed.

## 11) Pre-change checklist

- [ ] Identify the actual callback, source of input and next scene or process lifecycle owner.
- [ ] Trace old state and every remote effect before the first possible failure.
- [ ] Check command numbers, translated labels, state types and asset selection at all consumers.
- [ ] Choose a local verification path and name the failure and boundary cases.
- [ ] Identify any compatibility migration or exception that requires explicit review.
