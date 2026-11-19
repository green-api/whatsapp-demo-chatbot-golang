package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
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
	handlers := &scenes.CallHandlers{Delivery: calls.NewDeliveryClient(baseBot.GreenAPI)}
	shutdown, err := configureOpenAI(baseBot.APIURL, credentials, handlers)

	if err != nil {
		log.Fatal(err)
	}

	if err := configureAccount(baseBot.Account()); err != nil {
		log.Fatal(settingsErrorText(err, baseBot.GreenAPI))
	}

	baseBot.SetStartScene(scenes.StartScene{Calls: handlers})
	runBot(baseBot, handlers.Coordinator, shutdown)
}

func configureOpenAI(apiURL string, credentials botCredentials, handlers *scenes.CallHandlers) (time.Duration, error) {
	if credentials.openAIKey == "" {
		slog.Warn("OPENAI_API_KEY is not set; ChatGPT and calls are unavailable")
		return 0, nil
	}

	config, err := calls.LoadConfig()

	if err != nil {
		return 0, err
	}

	service, err := config.NewService(apiURL, credentials.instanceID, credentials.token, credentials.openAIKey)

	if err != nil {
		return 0, fmt.Errorf("configure call service: %w", err)
	}

	prompts := make(map[string]string)

	for _, lang := range []string{"en", "kz", "ru", "es", "he"} {
		prompts[lang] = scenes.CallText("call_prompts", lang)
	}

	handlers.Coordinator = calls.NewCallCoordinator(service, calls.CallDeliveryService{
		Notifier: calls.CallNotifier{Client: handlers.Delivery, Text: scenes.CallText},
		Uploader: calls.RecordingUploader{Client: handlers.Delivery},
	}, prompts, config.IncomingRing)

	registerGPT(credentials.instanceID, credentials.token, credentials.openAIKey)
	return config.Shutdown, nil
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
		return fmt.Errorf("set GREEN-API settings: %w", err)
	}

	if response == nil {
		return errors.New("set GREEN-API settings: empty response")
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("set GREEN-API settings: HTTP %d", response.StatusCode)
	}

	return nil
}

// The settings SDK may include the request endpoint in an opaque error.
func settingsErrorText(err error, api greenapi.GreenAPI) string {
	detail := err.Error()
	sensitive := []string{api.APIURL, api.IDInstance, api.APITokenInstance,
		url.PathEscape(api.APITokenInstance), url.QueryEscape(api.APITokenInstance)}

	if endpoint, err := url.Parse(api.APIURL); err == nil {
		sensitive = append(sensitive, endpoint.Hostname())
	}

	for _, value := range sensitive {
		if value != "" {
			detail = strings.ReplaceAll(detail, value, "[redacted]")
		}
	}

	return detail
}

func runBot(bot *chatbot.Bot, coordinator *calls.CallCoordinator, shutdown time.Duration) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if coordinator != nil {
		coordinator.Start()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		bot.StartReceivingNotifications()
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}

	// The SDK receiver runs until process exit when shutdown starts via a signal.
	if coordinator != nil {
		coordinator.Shutdown(shutdown, 50*time.Second)
	}
}
