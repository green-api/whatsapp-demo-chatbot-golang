package main

import (
	"demo-chatbot/scenes"
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
)

func main() {
	bot := chatbot.NewBot("1101848919", "fe0453b47e1b403c8d88ce881291ea002292b3037ae045bcb2")

	bot.SetStartScene(scenes.StartScene{})

	bot.StartReceivingNotifications()
}
