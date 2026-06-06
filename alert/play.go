package alert

import (
	"bytes"
	"encoding/binary"
	"slices"
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
	if name == "timeout" {
		return generateSweep(329.63, 233.08, 0.50, "triangle", 0.30)
	}

	tones, ok := sounds[name]
	if !ok {
		tones = sounds["success"]
	}

	gap := make([]float32, int(sampleRate*0.02))
	var mono []float32
	for _, t := range tones {
		mono = append(mono, generateTone(t)...)
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

// Names returns all built-in sound names, including "timeout", in stable sorted order.
func Names() []string {
	names := make([]string, 0, len(sounds)+1)
	for name := range sounds {
		names = append(names, name)
	}
	names = append(names, "timeout")
	slices.Sort(names)
	return names
}
