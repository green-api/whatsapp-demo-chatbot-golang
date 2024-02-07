package main

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/scenes"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
	"github.com/joho/godotenv"
	"log"
	"strconv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	util.GetConfig()

	bot := chatbot.NewBot(strconv.FormatInt(util.CloudConfig.InstanceId, 10), util.CloudConfig.Token)

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
