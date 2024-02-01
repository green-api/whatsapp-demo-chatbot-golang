package main

import (
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	"github.com/green-api/whatsapp-demo-chatbot-golang/scenes"
	"github.com/green-api/whatsapp-demo-chatbot-golang/util"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"log"
	"time"
)

type Formatter struct {
	Location  *time.Location
	Formatter *logrus.JSONFormatter
}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Time = entry.Time.In(f.Location)

	return f.Formatter.Format(entry)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	data := util.GetConfig()

	bot := chatbot.NewBot(data.InstanceId, data.Token)

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
