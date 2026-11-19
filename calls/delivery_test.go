package calls

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	greenapi "github.com/green-api/whatsapp-api-client-golang-v2"
	sdk "github.com/green-api/whatsapp-chatbot-golang/calls"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCallDeliveryTextThenRecordingAndCleanup(t *testing.T) {
	for _, tc := range []struct {
		name, failedMethod string
		want               []string
	}{
		{"success", "", []string{"sendMessage", "sendFileByUpload"}},
		{"text failure", "sendMessage", []string{"sendMessage"}},
		{"upload failure", "sendFileByUpload", []string{"sendMessage", "sendFileByUpload"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			capture := make(chan string, 3)
			client := NewDeliveryClient(greenapi.GreenAPI{APIURL: "https://api.example", MediaURL: "https://media.example", IDInstance: "1000000001", APITokenInstance: "test"})
			client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				method := strings.Split(r.URL.Path, "/")[2]
				capture <- method
				checkDeliveryRequest(t, r, method)

				w := httptest.NewRecorder()

				if method == tc.failedMethod {
					io.WriteString(w, "{}")
				} else {
					io.WriteString(w, `{"idMessage":"demo-id"}`)
				}

				return w.Result(), nil
			})
			session := sdk.NewCallSession(bob, bob, "ru")
			session.EndReason = sdk.CallEndReasonRemoteHangup
			path := filepath.Join(t.TempDir(), "temp.mp3")

			if err := os.WriteFile(path, []byte("test audio"), 0600); err != nil {
				t.Fatal(err)
			}

			CallDeliveryService{
				Notifier: CallNotifier{Client: client, Text: func(key, lang string) string { return key + "/" + lang }},
				Uploader: RecordingUploader{Client: client},
			}.Deliver(context.Background(), session, sdk.NewCallExecutionResult(path))

			var got []string
			close(capture)

			for method := range capture {
				got = append(got, method)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("delivery methods = %v; want %v", got, tc.want)
			}

			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("recording was not removed: %v", err)
			}
		})
	}
}

func checkDeliveryRequest(t *testing.T, r *http.Request, method string) {
	t.Helper()

	if method == "sendMessage" {
		var body map[string]string

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["chatId"] != bob || body["message"] != "call_completed/ru" {
			t.Errorf("result body = %v, %v", body, err)
		}

		return
	}

	file, info, err := r.FormFile("file")

	if err != nil {
		t.Error(err)
		return
	}

	defer file.Close()
	body, err := io.ReadAll(file)

	if err != nil || info.Filename != RecordingFilename || r.FormValue("fileName") != RecordingFilename ||
		r.FormValue("chatId") != bob || string(body) != "test audio" || info.Header.Get("Content-Type") != "audio/mpeg" {
		t.Errorf("unexpected recording: %#v / %q / %v", info, body, err)
	}
}

func TestUnansweredIncomingAndShutdownAreSilent(t *testing.T) {
	delivery := CallDeliveryService{
		// A nil client is never reached when the incoming caller was not answered.
		Notifier: CallNotifier{},
	}
	session, err := sdk.NewIncomingCallSession(bob, "missed", "en", time.Now().Add(time.Minute))

	if err != nil {
		t.Fatal(err)
	}

	session.EndReason = sdk.CallEndReasonCallerLeft
	delivery.Deliver(context.Background(), session, sdk.NewCallExecutionResult(""))

	outgoing := sdk.NewCallSession(alice, alice, "en")
	outgoing.EndReason = sdk.CallEndReasonShutdown
	delivery.Deliver(context.Background(), outgoing, sdk.NewCallExecutionResult(""))
}
