package calls

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	greenapi "github.com/green-api/whatsapp-api-client-golang-v2"
	sdk "github.com/green-api/whatsapp-chatbot-golang/calls"
)

const RecordingFilename = "call-recording.mp3"

type CallResultNotifier interface {
	NotifyResult(context.Context, *sdk.CallSession) error
}

type CallRecordingUploader interface {
	Send(ctx context.Context, chatID, path string) error
}

type CallDeliveryService struct {
	Notifier CallResultNotifier
	Uploader CallRecordingUploader
}

// Deliver handles the result once and owns the temporary recording.
func (d CallDeliveryService) Deliver(ctx context.Context, session *sdk.CallSession, result *sdk.CallExecutionResult) {
	if result != nil {
		defer func() {
			if err := result.Close(); err != nil {
				slog.Warn("Unable to remove recording", "session", session.SessionID)
			}
		}()

		if result.RecordingError() != nil {
			slog.Warn("Unable to export recording", "session", session.SessionID)
		}
	}

	if session.EndReason == sdk.CallEndReasonShutdown {
		return
	}

	if err := d.Notifier.NotifyResult(ctx, session); err != nil {
		slog.Warn("Unable to deliver call result", "session", session.SessionID)
		return
	}

	if result != nil && result.RecordingPath() != "" && d.Uploader != nil {
		if err := d.Uploader.Send(ctx, session.ChatID, result.RecordingPath()); err != nil {
			slog.Warn("Unable to upload recording", "session", session.SessionID)
		}
	}
}

type CallNotifier struct {
	Client *DeliveryClient
	Text   func(key, language string) string
}

func (n CallNotifier) NotifyResult(ctx context.Context, session *sdk.CallSession) error {
	if session.Direction == sdk.CallDirectionIncoming && session.AnsweredAt.IsZero() {
		return nil
	}

	key := "call_failed"

	switch session.EndReason {
	case sdk.CallEndReasonRemoteHangup:
		key = "call_completed"
	case sdk.CallEndReasonRemoteRejected, sdk.CallEndReasonRingTimeout, sdk.CallEndReasonIncomingExpired, sdk.CallEndReasonCallerLeft:
		key = "call_unreachable"
	case sdk.CallEndReasonTalkTimeout:
		key = "call_talk_timeout"
	}

	return n.Client.SendMessage(ctx, session.ChatID, n.Text(key, session.Language))
}

type RecordingUploader struct{ Client *DeliveryClient }

func (u RecordingUploader) Send(ctx context.Context, chatID, path string) error {
	return u.Client.Upload(ctx, chatID, path, RecordingFilename, "")
}

// DeliveryClient uses the same GREEN-API methods as the SDK. Its requests
// preserve @lid recipients and have the demo's text and upload deadlines.
type DeliveryClient struct {
	api  greenapi.GreenAPI
	http *http.Client
}

func NewDeliveryClient(api greenapi.GreenAPI) *DeliveryClient {
	return &DeliveryClient{api: api, http: &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (c *DeliveryClient) SendMessage(ctx context.Context, chatID, message string) error {
	body, err := json.Marshal(struct {
		ChatID  string `json:"chatId"`
		Message string `json:"message"`
	}{chatID, message})

	if err != nil {
		return err
	}

	return c.send(ctx, c.api.APIURL, "sendMessage", "application/json", bytes.NewReader(body), 20*time.Second)
}

// SendFileByURL delivers the menu after cancellation, including to @lid chats.
func (c *DeliveryClient) SendFileByURL(ctx context.Context, chatID, fileURL, name, caption string) error {
	body, err := json.Marshal(struct {
		ChatID   string `json:"chatId"`
		URL      string `json:"urlFile"`
		Filename string `json:"fileName"`
		Caption  string `json:"caption"`
	}{chatID, fileURL, name, caption})

	if err != nil {
		return err
	}

	return c.send(ctx, c.api.APIURL, "sendFileByUrl", "application/json", bytes.NewReader(body), 20*time.Second)
}

func (c *DeliveryClient) Upload(ctx context.Context, chatID, path, name, caption string) error {
	file, err := os.Open(path)

	if err != nil {
		return err
	}

	defer file.Close()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)

	for _, field := range [][2]string{{"chatId", chatID}, {"fileName", name}, {"caption", caption}} {
		if field[1] != "" {
			if err := form.WriteField(field[0], field[1]); err != nil {
				return err
			}
		}
	}

	if err := writeUploadFile(form, file, name); err != nil {
		return err
	}

	if err := form.Close(); err != nil {
		return err
	}

	return c.send(ctx, c.api.MediaURL, "sendFileByUpload", form.FormDataContentType(), &body, 30*time.Second)
}

func writeUploadFile(form *multipart.Writer, file io.Reader, name string) error {
	extension := filepath.Ext(name)
	contentType := mime.TypeByExtension(extension)

	if extension == ".mp3" {
		contentType = "audio/mpeg"
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": name}))
	header.Set("Content-Type", contentType)
	part, err := form.CreatePart(header)

	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	return err
}

func (c *DeliveryClient) send(parent context.Context, host, method, contentType string, body io.Reader, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	endpoint := strings.TrimRight(host, "/") + "/waInstance" + c.api.IDInstance + "/" + method + "/" + c.api.APITokenInstance
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)

	if err != nil {
		return errors.New("invalid delivery URL")
	}

	request.Header.Set("Content-Type", contentType)
	response, err := c.http.Do(request)

	if err != nil {
		return errors.New("call delivery request failed")
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("call delivery returned HTTP %d", response.StatusCode)
	}

	var reply struct {
		MessageID string `json:"idMessage"`
	}

	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&reply); err != nil || reply.MessageID == "" {
		return errors.New("call delivery response missing idMessage")
	}

	return nil
}
