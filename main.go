package main

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/scenes"
	"log"
)

func main() {
	const (
		// idInstance = '1101123456'
		// apiTokenInstance = 'abcdefghjklmn1234567890oprstuwxyz'
		idInstance       = "{INSTANCE}"
		apiTokenInstance = "{TOKEN}"
	)

	bot := chatbot.NewBot(idInstance, apiTokenInstance)

	if _, err := bot.GreenAPI.Methods().Account().SetSettings(map[string]interface{}{
		"incomingWebhook":           "yes",
		"outgoingMessageWebhook":    "yes",
		"outgoingAPIMessageWebhook": "yes",
	}); err != nil {
		log.Fatalln(err)
	}

	bot.SetStartScene(scenes.StartScene{})

	bot.StartReceivingNotifications()
}
