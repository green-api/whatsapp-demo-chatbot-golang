package main

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/scenes"
)

func main() {
	bot := chatbot.NewBot("{INSTANCE}", "{TOKEN}")

	bot.SetStartScene(scenes.StartScene{})

	bot.StartReceivingNotifications()
}
