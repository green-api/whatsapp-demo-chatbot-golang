package main

import (
	"log"

	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/scenes"
	"github.com/joho/godotenv"
)

func main() {
	idInstance := "{idInstance}"
	authToken := "{authToken}"
	envFile, err := godotenv.Read("instance.env")
	if err == nil {
		if val, exists := envFile["idInstance"]; exists && len(val) > 0 {
			idInstance = val
		}
		if val, exists := envFile["authToken"]; exists && len(val) > 0 {
			authToken = val
		}
	}
	if idInstance == "{idInstance}" || authToken == "{authToken}" {
		log.Fatal("No idInstance or authToken set")
	}

	bot := chatbot.NewBot(idInstance, authToken)

	go func() {
		for err := range bot.ErrorChannel {
			if err != nil {
				log.Println(err)
			}
		}
	}()

	_, err = bot.GreenAPI.Methods().Account().SetSettings(map[string]interface{}{
		"incomingWebhook":            "yes",
		"outgoingMessageWebhook":     "yes",
		"outgoingAPIMessageWebhook":  "yes",
		"pollMessageWebhook":         "yes",
		"markIncomingMessagesReaded": "yes",
	})
	if err != nil {
		bot.ErrorChannel <- err
	}

	bot.SetStartScene(scenes.StartScene{})

	bot.StartReceivingNotifications()
}
