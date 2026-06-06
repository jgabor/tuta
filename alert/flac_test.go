package alert

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/tphakala/go-flac/pcm"
)

func TestRenderSuccessLength(t *testing.T) {
	tones := sounds["success"]
	gapLen := int(sampleRate * 0.02)
	want := 0
	for _, tone := range tones {
		want += int(sampleRate*tone.duration) + gapLen
	}

	got := len(Render("success"))
	if got != want {
		t.Fatalf("Render(success) len = %d, want %d", got, want)
	}
}

func TestRenderTimeoutLength(t *testing.T) {
	want := int(sampleRate * 0.5)
	got := len(Render("timeout"))
	if got != want {
		t.Fatalf("Render(timeout) len = %d, want %d", got, want)
	}
}

func TestRenderUnknownFallsBackToSuccess(t *testing.T) {
	if len(Render("not-a-sound")) != len(Render("success")) {
		t.Fatal("unknown sound should fall back to success")
	}
}

func TestWriteFLACRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		sound    string
		channels int
		depth    int
	}{
		{"success mono 16", "success", 1, 16},
		{"timeout mono 16", "timeout", 1, 16},
		{"error stereo 24", "error", 2, 24},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mono := Render(tc.sound)
			wantSamples := len(mono)
			if tc.channels == 2 {
				wantSamples *= 2
			}

			path := t.TempDir() + "/" + tc.sound + ".flac"
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			opts := FLACOptions{Channels: tc.channels, BitDepth: tc.depth}
			if err := WriteFLAC(f, tc.sound, opts); err != nil {
				_ = f.Close()
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}

			flacData, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			dec, err := pcm.NewDecoder(bytes.NewReader(flacData))
			if err != nil {
				t.Fatal(err)
			}

			info := dec.Info()
			if info.SampleRate != sampleRate {
				t.Fatalf("sample rate = %d, want %d", info.SampleRate, sampleRate)
			}
			if info.Channels != tc.channels {
				t.Fatalf("channels = %d, want %d", info.Channels, tc.channels)
			}
			if info.BitDepth != tc.depth {
				t.Fatalf("bit depth = %d, want %d", info.BitDepth, tc.depth)
			}

			pcmOut, err := io.ReadAll(dec)
			if err != nil {
				t.Fatal(err)
			}

			bytesPerSample := tc.depth / 8
			gotSamples := len(pcmOut) / bytesPerSample / tc.channels
			if gotSamples != len(mono) {
				t.Fatalf("decoded frames = %d, want %d", gotSamples, len(mono))
			}

			_ = wantSamples
		})
	}
}

func TestExportFLACFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/success.flac"
	if err := ExportFLAC(path, "success", FLACOptions{}); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := pcm.NewDecoder(f)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Info().SampleRate != sampleRate {
		t.Fatalf("sample rate = %d, want %d", dec.Info().SampleRate, sampleRate)
	}
	_ = f.Close()
}
