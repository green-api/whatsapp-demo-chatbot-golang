package scenes

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
)

type MainMenuScene struct {
}

func (s MainMenuScene) Start(bot *chatbot.Bot) {
	bot.IncomingMessageHandler(func(message *chatbot.Notification) {
		if !util.IsSessionExpired(message) {
			text, _ := message.Text()
			switch text {
			case "1":
				s.sendMainMenu(message, "en")
			case "2":
				s.sendMainMenu(message, "ru")
			case "3":
				s.sendMainMenu(message, "kz")
			case "4":
				s.sendMainMenu(message, "he")
			case "5":
				s.sendMainMenu(message, "es")
			case "6":
				s.sendMainMenu(message, "ar")
			default:
				message.AnswerWithText(
					util.GetString([]string{"specify_language"}))
			}
		}
	})
}

func (s MainMenuScene) sendMainMenu(message *chatbot.Notification, lang string) {
	message.UpdateStateData(map[string]interface{}{"lang": lang})
	message.AnswerWithText(
		util.GetString([]string{"welcome_message", lang}) +
			message.Body["senderData"].(map[string]interface{})["senderName"].(string) + "\n" +
			util.GetString([]string{"menu", lang}))

	message.ActivateNextScene(EndpointsScene{})
}
