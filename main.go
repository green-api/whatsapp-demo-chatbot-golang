package main

import (
	greenapi "github.com/green-api/whatsapp-api-client-golang-v2"
	"log"
	"os"

	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	gptbot "github.com/green-api/whatsapp-chatgpt-go"
	"github.com/green-api/whatsapp-demo-chatbot-golang/registry"
	"github.com/green-api/whatsapp-demo-chatbot-golang/scenes"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	idInstance := os.Getenv("ID_INSTANCE")
	authToken := os.Getenv("AUTH_TOKEN")
	openaiToken := os.Getenv("OPENAI_API_KEY")

	if idInstance == "" || authToken == "" {
		log.Fatal("ID_INSTANCE and AUTH_TOKEN must be set in .env file or environment variables.")
	}
	if openaiToken == "" {
		log.Fatal("OPENAI_API_KEY must be set in .env file or environment variables for GPT functionality.")
	}

	baseBot := chatbot.NewBot(idInstance, authToken)

	gptConfig := gptbot.GPTBotConfig{
		IDInstance:       idInstance,
		APITokenInstance: authToken,
		OpenAIApiKey:     openaiToken,
		Model:            gptbot.ModelGPT4o,
		MaxHistoryLength: 10,
		SystemMessage:    "You are a helpful WhatsApp assistant.",
	}
	gptHelper := gptbot.NewWhatsappGptBot(gptConfig)

	registry.RegisterGptHelper(gptHelper)

	go func() {
		for err := range baseBot.ErrorChannel {
			if err != nil {
				log.Printf("ERROR: %v", err)
			}
		}
	}()

	_, err = baseBot.Account().SetSettings(
		greenapi.OptionalIncomingWebhook(true),
		greenapi.OptionalOutgoingWebhook(false),
		greenapi.OptionalStateWebhook(false),
		greenapi.OptionalOutgoingAPIMessageWebhook(false),
		greenapi.OptionalOutgoingMessageWebhook(false),
		greenapi.OptionalPollMessageWebhook(true),
		greenapi.OptionalMarkIncomingMessagesRead(true),
		greenapi.OptionalKeepOnlineStatus(true),
	)
	if err != nil {
		log.Fatalf("Failed to set Green API settings: %v", err)
	}

	baseBot.SetStartScene(scenes.StartScene{})

	log.Println("Starting Green API Demo Bot...")
	baseBot.StartReceivingNotifications()

	log.Println("Bot stopped.")
}
