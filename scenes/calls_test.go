package scenes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	sdk "github.com/green-api/whatsapp-chatbot-golang/calls"
	"github.com/green-api/whatsapp-demo-chatbot-golang/calls"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
)

const (
	menuUser  = "10001@lid"
	freshUser = "10002@c.us"
	gptUser   = "10003@c.us"
	groupChat = "10004@g.us"
	finalUser = "10005@c.us"
)

type idleExecutor struct{}

func (idleExecutor) Execute(context.Context, *sdk.CallSession, sdk.TransitionCallback) (*sdk.CallExecutionResult, error) {
	panic("the integration test leaves calls queued")
}

func (idleExecutor) RejectIncoming(context.Context, string, string) (bool, error) { return true, nil }
func (idleExecutor) NotifyIncomingEnded(string) bool                              { return false }
func (idleExecutor) RequestStop()                                                 {}

type stopOnLastEvent struct {
	chatbot.StateManager
	bot *chatbot.Bot
}

func (s stopOnLastEvent) GetCurrentScene(id string) chatbot.Scene {
	if id == finalUser {
		// The SDK stop method belongs to the receiver goroutine.
		s.bot.StopReceivingNotifications()
	}

	return s.StateManager.GetCurrentScene(id)
}

func TestCallHandlersThroughSDKScenes(t *testing.T) {
	t.Chdir("..")
	events := []map[string]any{
		textEvent(menuUser, menuUser, " 16 ", "quotedMessage"),
		textEvent(menuUser, menuUser, " 0 ", "quotedMessage"),
		textEvent(menuUser, groupChat, "16", "textMessage"),
		textEvent(menuUser, groupChat, "17", "textMessage"),
		incomingEvent(freshUser, "new", "offer"),
		incomingEvent(gptUser, "gpt", "offer"),
		textEvent(gptUser, gptUser, "hello", "textMessage"),
		incomingEvent(finalUser, "last", "hungUp"),
	}
	server, provider := startCallProvider(t, events)
	bot := chatbot.NewBot("1000000001", "test")
	defer close(bot.ErrorChannel)
	bot.APIURL, bot.MediaURL = server.URL, server.URL
	bot.CleanNotificationQueue = false
	bot.StateManager = stopOnLastEvent{StateManager: bot.StateManager, bot: bot}

	c := calls.NewCallCoordinator(idleExecutor{}, calls.CallDeliveryService{}, nil, time.Minute)
	defer stopTestCoordinator(t, c)
	handlers := &CallHandlers{Coordinator: c, Delivery: calls.NewDeliveryClient(bot.GreenAPI)}
	prepareCallScenes(bot, handlers)
	bot.StartReceivingNotifications()
	provider.check(t, []providerReply{
		{"sendMessage", menuUser, CallText("call_queued", "ru"), ""},
		{"sendMessage", menuUser, CallText("call_cancelled", "ru"), ""},
		{"sendFileByUrl", menuUser, strings.TrimSpace(CallText("menu", "ru")), "welcome.jpg"},
		{"sendMessage", groupChat, CallText("call_private_chat_only", "ru"), ""},
		{"sendFileByUpload", groupChat, CallText("call_incoming_instruction", "ru"), incomingCallImage},
		{"sendMessage", gptUser, CallText("call_queued", "ru"), ""},
	})

	if got := c.HandleCallMessage(freshUser, "hello"); got.MessageKey != "call_queued" || got.Language != "en" {
		t.Errorf("incoming call in fresh scene = %+v", got)
	}

	if _, ok := bot.GetCurrentScene(gptUser).(GptScene); !ok || bot.GetStateData(gptUser)["gptSessionJson"] != "history" {
		t.Fatal("queued call changed the GPT scene or history")
	}
}

func TestMenuThroughSDKScenesWithoutOpenAI(t *testing.T) {
	t.Chdir("..")
	events := []map[string]any{
		textEvent(gptUser, gptUser, "14", "textMessage"),
		textEvent(gptUser, gptUser, "16", "textMessage"),
		textEvent(gptUser, groupChat, "16", "textMessage"),
		textEvent(gptUser, gptUser, "17", "textMessage"),
		incomingEvent(gptUser, "ignored", "offer"),
		textEvent(gptUser, gptUser, "1", "textMessage"),
		incomingEvent(finalUser, "last", "hungUp"),
	}
	server, provider := startCallProvider(t, events)
	bot := chatbot.NewBot("1000000001", "test")
	defer close(bot.ErrorChannel)
	bot.APIURL, bot.MediaURL = server.URL, server.URL
	bot.CleanNotificationQueue = false
	bot.StateManager = stopOnLastEvent{StateManager: bot.StateManager, bot: bot}

	handlers := &CallHandlers{Delivery: calls.NewDeliveryClient(bot.GreenAPI)}
	prepareCallScenes(bot, handlers)
	bot.ActivateNextScene(gptUser, EndpointsScene{Calls: handlers})
	bot.UpdateStateData(gptUser, map[string]any{"last_touch_timestamp": time.Now()})
	bot.StartReceivingNotifications()

	provider.check(t, []providerReply{
		{"sendMessage", gptUser, CallText("chat_gpt_error", "ru"), ""},
		{"sendMessage", gptUser, CallText("call_failed", "ru"), ""},
		{"sendMessage", groupChat, CallText("call_failed", "ru"), ""},
		{"sendFileByUpload", gptUser, CallText("call_incoming_instruction", "ru"), incomingCallImage},
		{"sendMessage", gptUser, CallText("send_text_message", "ru") + util.GetString([]string{"links", "ru", "send_text_documentation"}), ""},
	})

	if _, ok := bot.GetCurrentScene(gptUser).(EndpointsScene); !ok {
		t.Fatal("the sender should remain in the menu")
	}
}

func TestFreshLIDChatThroughSDKScenes(t *testing.T) {
	t.Chdir("..")
	const chatID = "10006@lid"
	const senderName = "Test User"
	hello := textEvent(chatID, chatID, "Привет", "extendedTextMessage")
	selectLanguage := textEvent(chatID, chatID, "3", "textMessage")
	selectLanguage["senderData"].(map[string]any)["senderName"] = senderName
	server, provider := startCallProvider(t, []map[string]any{
		hello,
		selectLanguage,
		incomingEvent(finalUser, "last", "hungUp"),
	})
	bot := chatbot.NewBot("1000000001", "test")
	defer close(bot.ErrorChannel)
	bot.APIURL, bot.MediaURL = server.URL, server.URL
	bot.CleanNotificationQueue = false
	bot.StateManager = stopOnLastEvent{StateManager: bot.StateManager, bot: bot}
	bot.SetStartScene(StartScene{})

	bot.StartReceivingNotifications()

	welcome := CallText("welcome_message", "ru") + "*" + senderName + "*!\n" + CallText("menu", "ru")
	provider.check(t, []providerReply{
		{"sendMessage", chatID, util.GetString([]string{"specify_language"}), ""},
		{"sendFileByUrl", chatID, welcome, "welcome.jpg"},
	})
}

func prepareCallScenes(bot *chatbot.Bot, handlers *CallHandlers) {
	bot.SetStartScene(StartScene{Calls: handlers})

	for _, id := range []string{menuUser, groupChat, gptUser} {
		bot.Create(id)
		bot.UpdateStateData(id, map[string]any{"lang": "ru", "last_touch_timestamp": time.Now().Add(-10 * time.Second)})
		bot.ActivateNextScene(id, EndpointsScene{Calls: handlers})
	}

	bot.UpdateStateData(gptUser, map[string]any{"gptSessionJson": "history", "last_touch_timestamp": time.Now().Add(-time.Hour)})
	bot.ActivateNextScene(gptUser, GptScene{Calls: handlers})
}

func stopTestCoordinator(t *testing.T, c *calls.CallCoordinator) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := c.Stop(ctx); err != nil {
		t.Error(err)
	}
}

func textEvent(sender, chat, text, kind string) map[string]any {
	data := map[string]any{"typeMessage": kind, "textMessageData": map[string]any{"textMessage": text}}

	if kind == "quotedMessage" || kind == "extendedTextMessage" {
		data["extendedTextMessageData"] = map[string]any{"text": text}
	}

	return map[string]any{"typeWebhook": "incomingMessageReceived", "senderData": map[string]any{"sender": sender, "chatId": chat}, "messageData": data}
}

func incomingEvent(sender, id, status string) map[string]any {
	return map[string]any{"typeWebhook": "incomingCall", "from": sender, "idMessage": id, "status": status, "timestamp": time.Now().Unix()}
}

type providerReply struct{ method, chat, text, file string }

type callProvider struct {
	mu            sync.Mutex
	events        []map[string]any
	received      int
	deleted       int
	readsWithBody int
	sent          []providerReply
}

func startCallProvider(t *testing.T, events []map[string]any) (*httptest.Server, *callProvider) {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")

	if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
		t.Skipf("loopback listener unavailable: %v", err)
	}

	if err != nil {
		t.Fatal(err)
	}

	provider := &callProvider{events: events}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: http.HandlerFunc(provider.serveHTTP)}}
	server.Start()
	t.Cleanup(server.Close)
	return server, provider
}

func (p *callProvider) serveHTTP(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	method := strings.Split(r.URL.Path, "/")[2]

	switch method {
	case "receiveNotification":
		if body, _ := io.ReadAll(r.Body); len(body) != 0 {
			p.readsWithBody++
		}

		if p.received == len(p.events) {
			io.WriteString(w, "null")
			return
		}

		body := p.events[p.received]
		p.received++
		json.NewEncoder(w).Encode(map[string]any{"receiptId": p.received, "body": body})
	case "deleteNotification":
		p.deleted++
		io.WriteString(w, `{"result":true}`)
	case "sendFileByUpload":
		p.sent = append(p.sent, providerReply{method, r.FormValue("chatId"), r.FormValue("caption"), r.FormValue("fileName")})
		io.WriteString(w, `{"idMessage":"demo-upload"}`)
	default:
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		text := body["message"]

		if method == "sendFileByUrl" {
			text = body["caption"]
		}

		p.sent = append(p.sent, providerReply{method, body["chatId"], text, body["fileName"]})
		io.WriteString(w, `{"idMessage":"demo-message"}`)
	}
}

func (p *callProvider) check(t *testing.T, want []providerReply) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.readsWithBody != 0 {
		t.Errorf("%d receiveNotification requests had a body", p.readsWithBody)
	}

	if p.deleted != len(p.events) {
		t.Errorf("deleted %d notifications; want %d", p.deleted, len(p.events))
	}

	if !reflect.DeepEqual(p.sent, want) {
		t.Errorf("replies = %+v; want %+v", p.sent, want)
	}
}

func TestCallLocalesAndInstructionAsset(t *testing.T) {
	t.Chdir("..")

	for _, lang := range []string{"en", "ru", "kz", "es", "he"} {
		menu := CallText("menu", lang)

		if !strings.Contains(menu, "*16*") || !strings.Contains(menu, "*17*") {
			t.Errorf("menu %s is missing call options", lang)
		}

		for _, key := range []string{"call_queued", "call_in_progress", "call_processing", "call_cancelled", "call_unreachable",
			"call_failed", "call_completed", "call_talk_timeout", "call_private_chat_only", "call_incoming_instruction", "call_prompts"} {
			if CallText(key, lang) == "" {
				t.Errorf("empty %s/%s", key, lang)
			}
		}
	}

	if _, err := os.Stat("assets/" + incomingCallImage); err != nil {
		t.Fatal(err)
	}
}
