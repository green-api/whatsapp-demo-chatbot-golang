package calls

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	sdk "github.com/green-api/whatsapp-chatbot-golang/calls"
)

type Config struct {
	Ring, IncomingRing, Talk, Shutdown time.Duration
	RealtimeModel, RealtimeVoice       string
}

func LoadConfig() (Config, error) {
	config := Config{RealtimeModel: "gpt-realtime-2.1", RealtimeVoice: "marin"}

	for _, item := range []struct {
		name    string
		value   *time.Duration
		seconds int
	}{
		{"CALL_RING_TIMEOUT_SECONDS", &config.Ring, 30},
		{"CALL_INCOMING_RING_TIMEOUT_SECONDS", &config.IncomingRing, 60},
		{"CALL_TALK_TIMEOUT_SECONDS", &config.Talk, 180},
		{"CALL_SHUTDOWN_TIMEOUT_SECONDS", &config.Shutdown, 30},
	} {
		seconds := item.seconds

		if raw := strings.TrimSpace(os.Getenv(item.name)); raw != "" {
			var err error
			seconds, err = strconv.Atoi(raw)

			if err != nil || seconds <= 0 || int64(seconds) > int64((1<<63-1)/time.Second) {
				return Config{}, fmt.Errorf("%s must be a positive timeout in seconds", item.name)
			}
		}

		*item.value = time.Duration(seconds) * time.Second
	}

	if model := strings.TrimSpace(os.Getenv("CALL_REALTIME_MODEL")); model != "" {
		config.RealtimeModel = model
	}

	if voice := strings.TrimSpace(os.Getenv("CALL_REALTIME_VOICE")); voice != "" {
		config.RealtimeVoice = voice
	}

	return config, nil
}

func (c Config) NewService(apiURL, instanceID, token, openAIKey string) (*sdk.WhatsAppCallService, error) {
	options := sdk.DefaultCallOptions()
	options.RingSeconds = c.Ring.Seconds()
	options.TalkSeconds = c.Talk.Seconds()
	options.ShutdownSeconds = c.Shutdown.Seconds()

	return sdk.NewWhatsAppCallService(sdk.ProviderConfig{
		APIURL: apiURL, IDInstance: instanceID, APITokenInstance: token, OpenAIAPIKey: openAIKey,
		RealtimeModel: c.RealtimeModel, RealtimeVoice: c.RealtimeVoice,
	}, options)
}
