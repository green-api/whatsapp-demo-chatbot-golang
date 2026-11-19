# Architecture of whatsapp-demo-chatbot-golang

This Go executable lets developers exercise WhatsApp operations through a conversation. An operator runs it with a connected GREEN-API instance and can supply an OpenAI key to enable GPT and calls. GREEN-API handles transport, the chatbot dependency dispatches notifications, and local scenes choose replies. Deployment, durable storage and a downstream Go library API are outside the application’s scope. See [agent instructions](AGENTS.md) for change decisions and [contribution rules](CONTRIBUTING.md) for review and verification.

## Component ownership

| Source | Caller and responsibility | Consequence for changes |
| --- | --- | --- |
| `main.go`, `go.mod` | Process entry configures concrete clients, instance settings and start scene. Go 1.24.0, toolchain directive 1.24.2. | A settings change affects the connected account before dispatch begins. |
| `scenes/start.go`, `scenes/mainMenu.go` | SDK callbacks interpret language selection, set dialogue state and send the first menu. | Every downstream scene expects the selected locale. |
| `scenes/endpoints.go`, `model/model.go` | EndpointsScene dispatches commands. Its processPollUpdate decodes model.PollMessage and checks Vote.OptionVoters through util.ContainString. | Changing option order changes vote interpretation even when labels still render correctly. |
| `scenes/createGroup.go` | Endpoint option 11 hands off to a confirmation scene. | One callback writes state for two separate conversations. |
| `scenes/gptScene.go`, `registry/registry.go` | Endpoint option 14 and GPT callbacks obtain the helper registered by main. The scene encodes history. | Replacing the global concrete pointer at runtime would introduce shared mutable access without synchronization. |
| `util/util.go`, `util/ymlReader.go`, `strings.yaml` | Scenes share the activity check and GetString. Each lookup reads and decodes the YAML relative to the working directory. | Missing file, invalid YAML or absent key can panic during a reply. A scalar ends traversal immediately. |

## Startup and receipt lifecycle

1. `main` attempts to load `.env`. A missing file is tolerated, other load errors produce a warning. Empty `ID_INSTANCE` or `AUTH_TOKEN` terminates startup. A blank `OPENAI_API_KEY` allows messaging-only use.
2. It constructs the chatbot and delivery client. With an OpenAI key, it constructs the call coordinator and registers the GPT helper with GPT-4o, a system prompt and history limit 10. The bot constructor owns the ErrorChannel consumer.
3. Account.SetSettings enables incoming message, poll and call webhooks, incoming read marking and keep-online status. It disables outgoing, state, outgoing API message and outgoing message webhooks. Failure is fatal. Only then does main install StartScene, start the call worker if configured and begin receiving notifications.
4. The SDK owns receipt and dispatch. On SIGINT or SIGTERM, main closes call admission and waits for call work up to the configured shutdown timeout. On signal-driven shutdown, the SDK receive loop ends with the process. An in-flight receive, callback or acknowledgment may be interrupted. After call execution stops, accepted deliveries get a separate 50-second budget before cancellation and cleanup. Acknowledgment relative to callback failure, delivery order, redelivery, old-notification deletion, settings propagation time and graceful shutdown are **not verified**. These affect whether repeating a side effect is safe.

## Dialogue admission, timeout and dispatch

Scene Start methods register incoming handlers. Scene activation selects the handler for a subsequent event. Language and current scene belong to SDK-managed dialogue state. Transition rules are distributed across callbacks.

| Current scene | Input or condition | Observable action and destination |
| --- | --- | --- |
| Start | Expired | Send language prompt, activate MainMenu. |
| Start | Fresh or missing timestamp | Immediately call MainMenuScene.SendMainMenu with this input. |
| MainMenu or delegated selection | 1–5 | Store string `lang` as en/kz/ru/es/he respectively, send welcome image with sender name and menu, activate Endpoints. Invalid input sends specify_language without activating a scene. |
| MainMenu, Endpoints or GPT | Expired | Activate Start and send language prompt. |
| CreateGroup | Expired | Activate MainMenu and send language prompt. |
| Endpoints | Menu / stop | Menu resends menu media. Stop (including 0) sends farewell and activates Start. Empty text is silent, unknown nonempty text gets localized feedback. |

`IsSessionExpired` uses a strict interval greater than 300 seconds. Missing or non-time `last_touch_timestamp` returns false. Every check defers a timestamp update. Moving the check or invoking it twice can change which branch runs. Expiry redirects the conversation without an explicit history purge. The first menu assumes senderData.senderName exists and all downstream locale assertions expect a string.

Poll updates take precedence over command text. After body conversion, a decode error or fewer than three vote entries logs and returns. The first of positions 0–2 containing the sender determines the localized reply. An absent voter produces no reply. Other events enter the menu switch, including file sends, contact, location, avatar lookup, quoted messages and buttons. Localized reply-button text participates in dispatch. Some operations issue multiple sends and ignore errors, including sends before activating the next scene. There is no local atomic delivery or deduplication. Replaying a poll update can reply twice.

## Group confirmation and partial completion

Option 11 sends instructions and a bot contact before activating CreateGroup. Menu commands (including 0 here) return to Endpoints. Invalid input sends feedback and leaves confirmation active. With input 1 the scene proceeds in this order:

1. Create a group with the localized name and sender participant. On call error, send the error to ErrorChannel and return the original dialogue to Endpoints.
2. Decode the body, assert string chatId, create state for that ID, copy current dialogue data into it and activate Endpoints for the group. This copies more than the locale, so adding private state to the original dialogue also affects this path.
3. Call SetGroupPicture with `assets/Group_avatar.jpg`. On call error, emit the error and skip the group message. Otherwise the asserted boolean setGroupPicture selects one of two group messages. A group send error is emitted.
4. Send groupInviteLink to the original dialogue and activate Endpoints there. The direct send result is unchecked. Invalid JSON or unexpected field types can panic after remote creation because decoding errors are ignored and fields asserted. There is no compensation or retry. A retry starting from creation can create another group.

## GPT entry, reply and history

Option 14 checks the registry before activation. A missing helper sends chat_gpt_error and leaves the endpoint scene active. Successful entry sends an introduction and initializes a fresh session seeded with the helper’s system prompt, including on re-entry. Within GptScene, expiry and exit precede model processing. Exit words depend on locale, compare lowercased text without trimming, and include universal menu. Exit sends menu media then activates Endpoints. English media is used for en/es/he on GPT exit and endpoint menu refresh, but only for en on initial selection and group menu return.

For an ordinary turn, loadGptSessionFromState decodes the string gptSessionJson into the dependency’s GPTSessionData. Missing, wrongly typed, empty or malformed JSON triggers logged reinitialization. The helper receives the notification, session and context.Background(), without a local deadline or cancellation path. Helper absence or processing failure sends chat_gpt_error and returns. Success attempts SendText before saving the updated session. Delivery is unchecked, so history can advance despite a send failure. Conversely a nil session or serialization failure logs after the reply attempt.

Saving updates the dialogue map, deletes the legacy gptSession key and calls SetStateData. It does not convert legacy history. Format changes affect returning dialogues independently of model choice. Session durability across process restarts, concurrent turn ordering and the helper’s actual history retention are **not verified**.

## WhatsApp voice calls

- Endpoint 16 queues an outgoing call in a private chat. Endpoint 17 uploads the incoming-call illustration with localized instructions. Each scene registers the incoming webhook handler and runs the call message handler before regular text commands.
- [The coordinator](calls/coordinator.go) owns one FIFO for both directions. A sender keeps their slot through result delivery. During SDK execution, chat handlers read snapshots and send matching hangups to the SDK service. The worker rechecks the incoming deadline before execution.
- [Delivery](calls/delivery.go) sends the result before its MP3 attachment and closes the SDK result on every path. The coordinator runs up to four background deliveries and uses the worker when all four slots are occupied.
- The Go queue holds up to 256 waiting calls. Without an OpenAI key, option 16 sends `call_failed`, incoming calls are ignored and option 17 still sends the instruction. The upstream v1.1.0 receiver logs complete webhook payloads and may panic before scene dispatch if an incoming call lacks a string `from` field.

## Integration and evidence limits

Text, links, images, handlers and GPT configuration are application customization points. Constructors and notification types are concrete. The registry supplies a GPT helper pointer, with no provider replacement interface. The operator supplies runtime assets, credentials and hosting. The repository is licensed under the [MIT License](LICENCE). Technical editability alone establishes no branded distribution permission.

Local tests exercise call setup, queueing, webhook routing and result delivery with fake providers. Live WhatsApp and OpenAI interoperability, MP3 playback and account effects require a test instance.
