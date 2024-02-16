package main

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/scenes"
	"log"
)

func main() {
	bot := chatbot.NewBot("1101848919", "fe0453b47e1b403c8d88ce881291ea002292b3037ae045bcb2")

	go func() {
		select {
		case err := <-bot.ErrorChannel:
			if err != nil {
				log.Println(err)
			}
		}
	}()

	_, err := bot.GreenAPI.Methods().Account().SetSettings(map[string]interface{}{
		"incomingWebhook":            "yes",
		"outgoingMessageWebhook":     "yes",
		"outgoingAPIMessageWebhook":  "yes",
		"pollMessageWebhook":         "yes",
		"markIncomingMessagesReaded": "yes",
	})
	if err != nil {
		bot.ErrorChannel <- err
	}
	log.Println("Settings updated by bot")

	bot.SetStartScene(scenes.StartScene{})

	bot.StartReceivingNotifications()
}
