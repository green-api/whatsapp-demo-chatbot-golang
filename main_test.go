package main

import (
	"errors"
	"testing"

	greenapi "github.com/green-api/whatsapp-api-client-golang-v2"
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
	for _, account := range []fakeAccount{
		func(...greenapi.SetSettingsOption) (*greenapi.APIResponse, error) {
			return nil, errors.New("connection error")
		},
		func(...greenapi.SetSettingsOption) (*greenapi.APIResponse, error) {
			return &greenapi.APIResponse{StatusCode: 400}, nil
		},
	} {
		if err := configureAccount(account); err == nil {
			t.Fatal("failed settings request was accepted")
		}
	}
}

func TestLoadCredentialsRequiresEveryKey(t *testing.T) {
	for _, missing := range []string{"ID_INSTANCE", "AUTH_TOKEN", "OPENAI_API_KEY"} {
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
