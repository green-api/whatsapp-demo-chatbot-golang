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

			case "3":
				message.UpdateStateData(map[string]interface{}{"lang": "he"})
				message.AnswerWithText(
					message.Body["senderData"].(map[string]interface{})["senderName"].(string) +
						util.GetString([]string{"welcome_message", "he"}) +
						util.GetString([]string{"menu", "he"}))

				message.ActivateNextScene(EndpointsScene{})

			case "4":
				message.UpdateStateData(map[string]interface{}{"lang": "es"})
				message.AnswerWithText(
					util.GetString([]string{"welcome_message", "es"}) +
						message.Body["senderData"].(map[string]interface{})["senderName"].(string) +
						util.GetString([]string{"menu", "es"}))

				message.ActivateNextScene(EndpointsScene{})

			case "5":
				message.UpdateStateData(map[string]interface{}{"lang": "ar"})
				message.AnswerWithText(
					util.GetString([]string{"welcome_message", "ar"}) +
						message.Body["senderData"].(map[string]interface{})["senderName"].(string) +
						util.GetString([]string{"menu", "ar"}))

				message.ActivateNextScene(EndpointsScene{})

			default:
				message.AnswerWithText(
					util.GetString([]string{"specify_language"}))
			}
		}
	})
}
