package calls

import (
	"testing"
	"time"
)

func TestCallConfigDefaultsAndOverrides(t *testing.T) {
	for _, name := range []string{
		"CALL_RING_TIMEOUT_SECONDS", "CALL_INCOMING_RING_TIMEOUT_SECONDS", "CALL_TALK_TIMEOUT_SECONDS",
		"CALL_SHUTDOWN_TIMEOUT_SECONDS", "CALL_REALTIME_MODEL", "CALL_REALTIME_VOICE",
	} {
		t.Setenv(name, "")
	}

	got, err := LoadConfig()
	want := Config{30 * time.Second, time.Minute, 3 * time.Minute, 30 * time.Second, "gpt-realtime-2.1", "marin"}

	if err != nil || got != want {
		t.Fatalf("config = %+v, %v; want %+v", got, err, want)
	}

	t.Setenv("CALL_INCOMING_RING_TIMEOUT_SECONDS", "75")
	t.Setenv("CALL_REALTIME_VOICE", "other")
	got, err = LoadConfig()

	if err != nil || got.IncomingRing != 75*time.Second || got.RealtimeVoice != "other" {
		t.Fatalf("config overrides = %+v, %v", got, err)
	}

	t.Setenv("CALL_TALK_TIMEOUT_SECONDS", "0")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("zero talk timeout was accepted")
	}
}
