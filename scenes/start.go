package scenes

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
)

type StartScene struct {
}

func (s StartScene) Start(bot *chatbot.Bot) {
	bot.IncomingMessageHandler(func(message *chatbot.Notification) {
		message.AnswerWithText(util.GetString([]string{"select_language"}))

		message.ActivateNextScene(MainMenuScene{})
	})
}
