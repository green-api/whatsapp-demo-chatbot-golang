package calls

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	greenapi "github.com/green-api/whatsapp-api-client-golang-v2"
	sdk "github.com/green-api/whatsapp-chatbot-golang/calls"
)

func TestShutdownDrainsOrCancelsAllRecordingUploads(t *testing.T) {
	for _, expire := range []bool{false, true} {
		t.Run(fmt.Sprintf("expire=%t", expire), func(t *testing.T) {
			checkShutdownUploads(t, expire)
		})
	}
}

func checkShutdownUploads(t *testing.T, expire bool) {
	const count = maxConcurrentDeliveries + 1
	dir := t.TempDir()
	started, finished := make(chan struct{}, count), make(chan bool, count)
	release, stopping := make(chan struct{}), make(chan struct{})
	client := blockingUploadClient(started, finished, release)
	executor := fakeExecutor{stop: func() { close(stopping) }, execute: func(_ context.Context, s *sdk.CallSession, _ sdk.TransitionCallback) (*sdk.CallExecutionResult, error) {
		return sdk.NewCallExecutionResult(filepath.Join(dir, s.SenderID)), nil
	}}
	c := newTestCoordinator(t, executor, nil)
	c.delivery.Uploader = RecordingUploader{client}

	for i := range count {
		id := fmt.Sprint(i)

		if err := os.WriteFile(filepath.Join(dir, id), []byte("audio"), 0600); err != nil {
			t.Fatal(err)
		}

		enqueue(t, c, id, Created)
	}

	c.Start()

	for range count {
		receive(t, started)
	}

	done := make(chan struct{})
	budget := time.Minute

	if expire {
		budget = time.Millisecond
	}

	go func() { c.Shutdown(time.Second, budget); close(done) }()
	receive(t, stopping)

	if !expire {
		close(release)
	}

	receive(t, done)

	for i := range count {
		if receive(t, finished) == expire {
			t.Fatal("unexpected upload outcome")
		}

		if _, err := os.Stat(filepath.Join(dir, fmt.Sprint(i))); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("recording was not cleaned up", err)
		}
	}
}

func blockingUploadClient(started chan<- struct{}, finished chan<- bool, release <-chan struct{}) *DeliveryClient {
	client := NewDeliveryClient(greenapi.GreenAPI{MediaURL: "https://example.test"})
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		started <- struct{}{}

		select {
		case <-r.Context().Done():
			finished <- false
			return nil, r.Context().Err()
		case <-release:
			finished <- true
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"idMessage":"sent"}`))}, nil
		}
	})
	return client
}
