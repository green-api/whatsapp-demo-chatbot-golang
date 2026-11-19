package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	greenapi "github.com/green-api/whatsapp-api-client-golang-v2"
	chatbot "github.com/green-api/whatsapp-chatbot-golang"
	gptbot "github.com/green-api/whatsapp-chatgpt-go"
	"github.com/green-api/whatsapp-demo-chatbot-golang/calls"
	"github.com/green-api/whatsapp-demo-chatbot-golang/registry"
	"github.com/green-api/whatsapp-demo-chatbot-golang/scenes"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")

	if err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	credentials, err := loadCredentials()

	if err != nil {
		log.Fatal(err)
	}

	baseBot := chatbot.NewBot(credentials.instanceID, credentials.token)

	config, err := calls.LoadConfig()

	if err != nil {
		log.Fatal(err)
	}

	service, err := config.NewService(baseBot.APIURL, credentials.instanceID, credentials.token, credentials.openAIKey)

	if err != nil {
		log.Fatalf("Cannot configure call service: %v", err)
	}

	delivery := calls.NewDeliveryClient(baseBot.GreenAPI)
	prompts := make(map[string]string)

	for _, lang := range []string{"en", "kz", "ru", "es", "he"} {
		prompts[lang] = scenes.CallText("call_prompts", lang)
	}

	coordinator := calls.NewCallCoordinator(service, calls.CallDeliveryService{
		Notifier: calls.CallNotifier{Client: delivery, Text: scenes.CallText},
		Uploader: calls.RecordingUploader{Client: delivery},
	}, prompts, config.IncomingRing)

	registerGPT(credentials.instanceID, credentials.token, credentials.openAIKey)

	if err := configureAccount(baseBot.Account()); err != nil {
		log.Fatalf("Failed to set Green API settings: %v", err)
	}

	baseBot.SetStartScene(scenes.StartScene{Calls: &scenes.CallHandlers{Coordinator: coordinator, Delivery: delivery}})
	runBot(baseBot, coordinator, config.Shutdown)
}

type botCredentials struct{ instanceID, token, openAIKey string }

func loadCredentials() (botCredentials, error) {
	credentials := botCredentials{
		instanceID: os.Getenv("ID_INSTANCE"),
		token:      os.Getenv("AUTH_TOKEN"),
		openAIKey:  strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
	}

	if credentials.instanceID == "" || credentials.token == "" {
		return botCredentials{}, errors.New("ID_INSTANCE and AUTH_TOKEN must be set in .env file or environment variables")
	}

	if credentials.openAIKey == "" {
		return botCredentials{}, errors.New("OPENAI_API_KEY must be set in .env file or environment variables")
	}

	return credentials, nil
}

func registerGPT(idInstance, authToken, openaiToken string) {
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
}

func configureAccount(account interface {
	SetSettings(...greenapi.SetSettingsOption) (*greenapi.APIResponse, error)
}) error {
	response, err := account.SetSettings(
		greenapi.OptionalIncomingWebhook(true),
		greenapi.OptionalIncomingCallWebhook(true),
		greenapi.OptionalOutgoingWebhook(false),
		greenapi.OptionalStateWebhook(false),
		greenapi.OptionalOutgoingAPIMessageWebhook(false),
		greenapi.OptionalOutgoingMessageWebhook(false),
		greenapi.OptionalPollMessageWebhook(true),
		greenapi.OptionalMarkIncomingMessagesRead(true),
		greenapi.OptionalKeepOnlineStatus(true),
	)

	if err != nil {
		return errors.New("settings request failed")
	}

	if response == nil || response.StatusCode != 200 {
		return errors.New("settings request was not accepted")
	}

	return nil
}

func runBot(bot *chatbot.Bot, coordinator *calls.CallCoordinator, shutdown time.Duration) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	coordinator.Start()

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := bot.ReceiveNotifications(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Print("Notification receiver stopped with an error")
		}
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}

	coordinator.Shutdown(shutdown, 50*time.Second)
	<-done
}
