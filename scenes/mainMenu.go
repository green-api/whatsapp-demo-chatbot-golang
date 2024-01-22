package scenes

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
)

type MainMenuScene struct {
}

func (s MainMenuScene) Start(bot *chatbot.Bot) {
	bot.IncomingMessageHandler(func(message *chatbot.Notification) {
		text, _ := message.Text()
		switch text {

		case "1":
			message.UpdateStateData(map[string]interface{}{"lang": "eng"})
			message.AnswerWithText(
				util.GetString([]string{"welcome_message", "eng"}) +
					message.Body["senderData"].(map[string]interface{})["senderName"].(string) +
					util.GetString([]string{"menu", "eng"}))

			message.ActivateNextScene(EndpointsScene{})

		case "2":
			message.UpdateStateData(map[string]interface{}{"lang": "ru"})
			message.AnswerWithText(
				util.GetString([]string{"welcome_message", "ru"}) +
					message.Body["senderData"].(map[string]interface{})["senderName"].(string) +
					util.GetString([]string{"menu", "ru"}))

			message.ActivateNextScene(EndpointsScene{})

		default:
			message.AnswerWithText(
				util.GetString([]string{"specify_language"}))
		}
	})
}
