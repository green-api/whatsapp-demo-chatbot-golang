package main

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/scenes"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
	"log"
)

func main() {
	cloudConfig := util.GetConfig()

	bot := chatbot.NewBot(cloudConfig.InstanceId, cloudConfig.Token)

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
