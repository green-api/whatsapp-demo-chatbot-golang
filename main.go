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
		log.Fatalln("Error loading .env file")
	}

	util.GetConfig()

	bot := chatbot.NewBot(strconv.FormatInt(util.CloudConfig.InstanceId, 10), util.CloudConfig.Token)
	//bot := chatbot.NewBot("1101848919", "fe0453b47e1b403c8d88ce881291ea002292b3037ae045bcb2")
	//bot := chatbot.NewBot("7103861096", "faea159cf9774262894607cbb4843f009a3674f01e57415b95")

	logger := util.GetLogger()
	logger.WithField("marker", "Bot is inited").Debugln("Configuration data and environment loaded successfully")

	go func() {
		for err := range bot.ErrorChannel {
			if err != nil {
				logger.Debugln(err)
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
	logger.Debugln("Settings updated by bot")

	bot.SetStartScene(scenes.StartScene{})

	bot.StartReceivingNotifications()
}
