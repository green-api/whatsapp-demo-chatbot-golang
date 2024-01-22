package scenes

import (
	"demo-chatbot/util"
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
)

type StartScene struct {
}

func (s StartScene) Start(bot *chatbot.Bot) {
	bot.IncomingMessageHandler(func(message *chatbot.Notification) {
		message.AnswerWithText(util.GetString([]string{"select_language"}))

		message.ActivateNextScene(MainMenuScene{})
	})
}
