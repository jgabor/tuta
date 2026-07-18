package alert

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/hajimehoshi/oto/v2"
)

// SampleRate returns the synthesis sample rate in Hz.
func SampleRate() int {
	return sampleRate
}

// Render returns mono float32 PCM for the named sound. Unknown names fall back
// to "success" (same behavior as Play and the CLI).
func Render(name string) []float32 {
	definition := findSound(name)
	if definition == nil {
		definition = findSound("success")
	}
	if definition.sweep != nil {
		s := definition.sweep
		return generateSweep(s.fromFreq, s.toFreq, s.duration, definition.metadata.Waveform, s.volume)
	}

	gap := make([]float32, int(sampleRate*0.02))
	var mono []float32
	for _, t := range definition.tones {
		mono = append(mono, generateTone(t, definition.metadata.Waveform)...)
		mono = append(mono, gap...)
	}
	return mono
}

// Play plays a named notification sound. Unknown names fall back to "success"
// (same behavior as the CLI). Returns an error if audio init or playback fails.
func Play(name string) error {
	mono := Render(name)

	buf := &bytes.Buffer{}
	for _, s := range mono {
		_ = binary.Write(buf, binary.LittleEndian, s)
		_ = binary.Write(buf, binary.LittleEndian, s)
	}

	ctx, ready, err := oto.NewContext(sampleRate, 2, oto.FormatFloat32LE)
	if err != nil {
		return err
	}
	<-ready

	player := ctx.NewPlayer(buf)
	player.Play()
	duration := time.Duration(float64(len(mono))/sampleRate*float64(time.Second)) + 200*time.Millisecond
	time.Sleep(duration)
	_ = player.Close()
	return nil
}

// Sounds returns metadata for all built-in sounds in stable name order.
func Sounds() []Sound {
	catalog := make([]Sound, len(soundDefinitions))
	for i, definition := range soundDefinitions {
		catalog[i] = definition.metadata
	}
	return catalog
}

// Names returns all built-in sound names in stable sorted order.
func Names() []string {
	catalog := Sounds()
	names := make([]string, len(catalog))
	for i, sound := range catalog {
		names[i] = sound.Name
	}
	return names
}
