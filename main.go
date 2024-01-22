package main

import (
	"demo-chatbot/scenes"
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
)

func main() {
	bot := chatbot.NewBot("{INSTANCE}", "{TOKEN}")

	bot.SetStartScene(scenes.StartScene{})

	bot.StartReceivingNotifications()
}
