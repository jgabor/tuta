package alert

import (
	"bytes"
	"encoding/binary"
	"slices"
	"time"

	"github.com/hajimehoshi/oto/v2"
)

// Play plays a named notification sound. Unknown names fall back to "success"
// (same behavior as the CLI). Returns an error if audio init or playback fails.
func Play(name string) error {
	var mono []float32

	if name == "timeout" {
		mono = generateSweep(329.63, 233.08, 0.50, "triangle", 0.12)
	} else {
		tones, ok := sounds[name]
		if !ok {
			tones = sounds["success"]
		}

		gap := make([]float32, int(sampleRate*0.02))
		for _, t := range tones {
			mono = append(mono, generateTone(t)...)
			mono = append(mono, gap...)
		}
	}

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
