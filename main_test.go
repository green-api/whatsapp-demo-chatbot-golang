package main

import (
	"errors"
	"strings"
	"testing"

	greenapi "github.com/green-api/whatsapp-api-client-golang-v2"
	"github.com/green-api/whatsapp-demo-chatbot-golang/registry"
	"github.com/green-api/whatsapp-demo-chatbot-golang/scenes"
)

type fakeAccount func(...greenapi.SetSettingsOption) (*greenapi.APIResponse, error)

func (f fakeAccount) SetSettings(options ...greenapi.SetSettingsOption) (*greenapi.APIResponse, error) {
	return f(options...)
}

func TestConfigureAccountEnablesCallsAndPreservesSettings(t *testing.T) {
	account := fakeAccount(func(options ...greenapi.SetSettingsOption) (*greenapi.APIResponse, error) {
		var got greenapi.RequestSetSettings

		for _, apply := range options {
			if err := apply(&got); err != nil {
				return nil, err
			}
		}

		want := greenapi.RequestSetSettings{
			IncomingWebhook: "yes", IncomingCallWebhook: "yes", OutgoingWebhook: "no",
			StateWebhook: "no", OutgoingAPIMessageWebhook: "no", OutgoingMessageWebhook: "no",
			PollMessageWebhook: "yes", MarkIncomingMessagesReaded: "yes", KeepOnlineStatus: "yes",
		}

		if got != want {
			t.Errorf("settings = %+v; want %+v", got, want)
		}

		return &greenapi.APIResponse{StatusCode: 200}, nil
	})

	if err := configureAccount(account); err != nil {
		t.Fatal(err)
	}
}

func TestConfigureAccountFailsOnRequestErrorOrNonSuccessResponse(t *testing.T) {
	cause := errors.New("connection error for https://api.example/waInstance1000000001/setSettings/sensitive-token")
	api := greenapi.GreenAPI{APIURL: "https://api.example", IDInstance: "1000000001", APITokenInstance: "sensitive-token"}

	for _, tc := range []struct {
		name     string
		response *greenapi.APIResponse
		cause    error
		want     string
	}{
		{"transport", nil, cause, "connection error"},
		{"http", &greenapi.APIResponse{StatusCode: 400, Body: []byte("sensitive-token")}, nil, "HTTP 400"},
		{"empty", nil, nil, "empty response"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account := fakeAccount(func(...greenapi.SetSettingsOption) (*greenapi.APIResponse, error) {
				return tc.response, tc.cause
			})
			err := configureAccount(account)

			if err == nil || (tc.cause != nil && !errors.Is(err, tc.cause)) {
				t.Fatalf("unexpected settings error: %v", err)
			}

			got := settingsErrorText(err, api)

			if !strings.Contains(got, tc.want) || strings.Contains(got, api.APITokenInstance) ||
				strings.Contains(got, api.IDInstance) || strings.Contains(got, "api.example") {
				t.Fatalf("unexpected settings diagnostic: %s", got)
			}
		})
	}
}

func TestLoadCredentialsRequiresGreenAPI(t *testing.T) {
	for _, missing := range []string{"ID_INSTANCE", "AUTH_TOKEN"} {
		t.Run(missing, func(t *testing.T) {
			t.Setenv("ID_INSTANCE", "1000000001")
			t.Setenv("AUTH_TOKEN", "test")
			t.Setenv("OPENAI_API_KEY", "test")
			t.Setenv(missing, "")

			if _, err := loadCredentials(); err == nil {
				t.Fatalf("startup accepted empty %s", missing)
			}
		})
	}
}

func TestConfigureOpenAIWithoutKey(t *testing.T) {
	t.Setenv("ID_INSTANCE", "1000000001")
	t.Setenv("AUTH_TOKEN", "test")
	t.Setenv("OPENAI_API_KEY", "  ")
	t.Setenv("CALL_RING_TIMEOUT_SECONDS", "invalid")
	credentials, err := loadCredentials()

	if err != nil || credentials.openAIKey != "" {
		t.Fatalf("whitespace-only OpenAI key should be optional: %v", err)
	}

	handlers := &scenes.CallHandlers{}
	shutdown, err := configureOpenAI("https://api.example", credentials, handlers)

	if err != nil || handlers.Coordinator != nil || registry.GetGptHelper() != nil || shutdown != 0 {
		t.Fatalf("OpenAI features should be disabled: shutdown=%v, err=%v", shutdown, err)
	}
}
