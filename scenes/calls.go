package scenes

import (
	"context"
	"log/slog"
	"strings"
	"time"

	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/calls"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
)

const (
	outgoingCallOption = "16"
	incomingCallOption = "17"
	incomingCallImage  = "incoming_call.png"
)

type CallHandlers struct {
	Coordinator *calls.CallCoordinator
	Delivery    *calls.DeliveryClient
}

func (h *CallHandlers) register(bot *chatbot.Bot, next func(*chatbot.Notification)) {
	if h == nil {
		bot.IncomingMessageHandler(next)
		return
	}

	bot.IncomingCallHandler(h.handleIncomingCall)
	bot.IncomingMessageHandler(func(n *chatbot.Notification) {
		if !h.handleCallMessage(n) {
			next(n)
		}
	})
}

func (h *CallHandlers) handleIncomingCall(n *chatbot.Notification) {
	call := calls.ParseIncomingCall(n.Body)
	lang, _ := n.GetStateData()["lang"].(string)
	h.Coordinator.HandleIncomingCall(call, callLanguage(lang))
}

func callMessage(n *chatbot.Notification) (senderID, chatID, text string, isText bool) {
	sender, _ := n.Body["senderData"].(map[string]any)
	senderID, _ = sender["sender"].(string)
	chatID, _ = sender["chatId"].(string)
	data, _ := n.Body["messageData"].(map[string]any)

	var field, key string

	switch data["typeMessage"] {
	case "textMessage":
		field, key = "textMessageData", "textMessage"
	case "extendedTextMessage", "quotedMessage":
		field, key = "extendedTextMessageData", "text"
	default:
		return senderID, chatID, "", false
	}

	content, _ := data[field].(map[string]any)
	text, _ = content[key].(string)
	return senderID, chatID, text, true
}

func (h *CallHandlers) handleCallMessage(n *chatbot.Notification) bool {
	senderID, chatID, text, isText := callMessage(n)
	h.updateCallActivity(n, senderID)

	if !isText || chatID == "" {
		return false
	}

	action := h.Coordinator.HandleCallMessage(senderID, text)
	h.sendReply(chatID, action.MessageKey, action.Language)

	if action.ShowMenu {
		h.sendCallMenu(chatID, action.Language)
	}

	return action.Handled
}

func (h *CallHandlers) updateCallActivity(n *chatbot.Notification, senderID string) {
	if senderID == "" {
		return
	}

	stateID := n.StateId

	if strings.HasSuffix(stateID, "@g.us") {
		stateID = senderID
	}

	data := n.StateManager.GetStateData(stateID)

	if data == nil {
		return
	}

	at := h.Coordinator.ActivityTimestamp(senderID)
	old, _ := data["last_touch_timestamp"].(time.Time)

	if at.After(old) {
		n.StateManager.UpdateStateData(stateID, map[string]any{"last_touch_timestamp": at})
	}
}

func (h *CallHandlers) handleCallOption(n *chatbot.Notification, lang string) bool {
	if h == nil {
		return false
	}

	senderID, chatID, text, isText := callMessage(n)

	if !isText || senderID == "" || chatID == "" {
		return false
	}

	switch strings.TrimSpace(text) {
	case outgoingCallOption:
		h.sendReply(chatID, h.enqueueOutgoing(senderID, chatID, lang), lang)
	case incomingCallOption:
		caption := CallText("call_incoming_instruction", lang)

		if err := h.Delivery.Upload(context.Background(), chatID, "assets/"+incomingCallImage, incomingCallImage, caption); err != nil {
			slog.Warn("Unable to send incoming call instructions")
		}
	default:
		return false
	}

	return true
}

func (h *CallHandlers) enqueueOutgoing(senderID, chatID, lang string) string {
	if strings.HasSuffix(chatID, "@g.us") {
		return "call_private_chat_only"
	}

	result, err := h.Coordinator.Enqueue(senderID, chatID, lang)

	if err != nil {
		return "call_failed"
	}

	if result == calls.AlreadyExists {
		action := h.Coordinator.HandleCallMessage(senderID, outgoingCallOption)

		if action.MessageKey != "" {
			return action.MessageKey
		}
	}

	return "call_queued"
}

func (h *CallHandlers) sendReply(chatID, key, lang string) {
	if key != "" {
		if err := h.Delivery.SendMessage(context.Background(), chatID, CallText(key, lang)); err != nil {
			slog.Warn("Unable to send call status", "message", key)
		}
	}
}

func (h *CallHandlers) sendCallMenu(chatID, lang string) {
	asset := "welcome_en.jpg"

	if lang == "ru" {
		asset = "welcome_ru.jpg"
	}

	url := "https://raw.githubusercontent.com/green-api/whatsapp-demo-chatbot-golang/refs/heads/master/assets/" + asset
	caption := strings.TrimLeft(CallText("menu", lang), "\r\n\t ")

	if err := h.Delivery.SendFileByURL(context.Background(), chatID, url, "welcome.jpg", caption); err != nil {
		slog.Warn("Unable to send menu after call cancellation")
	}
}

func CallText(key, lang string) string {
	return util.GetString([]string{key, callLanguage(lang)})
}

func callLanguage(lang string) string {
	switch lang {
	case "en", "kz", "ru", "es", "he":
		return lang
	default:
		return "en"
	}
}
