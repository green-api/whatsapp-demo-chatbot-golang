package calls

import (
	"context"
	"encoding/binary"
	"math"
	"os/exec"
	"testing"
	"time"

	sdk "github.com/green-api/whatsapp-chatbot-golang/calls"
)

func TestCallRecordingMP3RoundTrip(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")

	if err != nil {
		t.Skip("FFmpeg required for the recording round trip")
	}

	options := sdk.DefaultRecordingOptions()
	options.Directory = t.TempDir()
	options.EncoderPath = ffmpeg
	recorder := sdk.NewCallRecorder(options)
	defer recorder.Discard()
	pcm := make([]byte, sdk.PCMRate*2)

	for sample := 0; sample < sdk.PCMRate; sample++ {
		value := int16(8000 * math.Sin(2*math.Pi*440*float64(sample)/sdk.PCMRate))
		binary.LittleEndian.PutUint16(pcm[2*sample:], uint16(value))
	}

	if err := recorder.Write(sdk.TrackCaller, sdk.PcmChunk{Data: pcm}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	path, err := recorder.Finish(ctx)

	if err != nil {
		t.Fatal(err)
	}

	decoded, err := exec.CommandContext(ctx, ffmpeg, "-v", "error", "-f", "mp3", "-i", path, "-ac", "1", "-ar", "8000", "-f", "s16le", "-").Output()

	if err != nil || len(decoded) < 15000 || len(decoded) > 18000 {
		t.Fatalf("decoded %d bytes of mono PCM: %v", len(decoded), err)
	}

	for i := 0; i+1 < len(decoded); i += 2 {
		if int16(binary.LittleEndian.Uint16(decoded[i:])) > 500 {
			return
		}
	}

	t.Fatal("decoded recording contains no audible test signal")
}
