package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/hajimehoshi/oto/v2"
)

const sampleRate = 44100

var version = "0.1.0"

type tone struct {
	frequency float64
	duration  float64
	waveform  string
	volume    float64
}

var sounds = map[string][]tone{
	"success": {
		{523.25, 0.08, "sine", 0.2},
		{659.25, 0.08, "sine", 0.2},
		{783.99, 0.15, "sine", 0.35},
	},
	"error": {
		{150, 0.15, "square", 0.25},
		{110, 0.35, "square", 0.35},
	},
	"warning": {
		{1046.50, 0.12, "sine", 0.35},
		{1046.50, 0.20, "sine", 0.35},
	},
	"info": {
		{523.25, 0.15, "sine", 0.35},
	},
	"complete": {
		{493.88, 0.10, "triangle", 0.18},
		{587.33, 0.10, "triangle", 0.25},
		{783.99, 0.40, "triangle", 0.35},
	},
}

func generateTone(t tone) []float32 {
	samples := int(sampleRate * t.duration)
	wave := make([]float32, samples)
	for i := range wave {
		ts := float64(i) / sampleRate
		var s float64
		switch t.waveform {
		case "sine":
			s = math.Sin(2 * math.Pi * t.frequency * ts)
		case "square":
			s = math.Copysign(1, math.Sin(2*math.Pi*t.frequency*ts))
		case "triangle":
			p := ts*t.frequency - math.Floor(ts*t.frequency+0.5)
			s = 2*math.Abs(2*p) - 1
		default:
			s = math.Sin(2 * math.Pi * t.frequency * ts)
		}
		wave[i] = float32(s * math.Exp(-3*ts/t.duration) * t.volume)
	}
	return wave
}

func playNotification(soundType string) error {
	tones, ok := sounds[soundType]
	if !ok {
		tones = sounds["success"]
	}

	gap := make([]float32, int(sampleRate*0.02))
	var mono []float32
	for _, t := range tones {
		mono = append(mono, generateTone(t)...)
		mono = append(mono, gap...)
	}

	// Write as stereo (interleaved L/R channels)
	buf := &bytes.Buffer{}
	for _, s := range mono {
		binary.Write(buf, binary.LittleEndian, s)
		binary.Write(buf, binary.LittleEndian, s)
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
	player.Close()
	return nil
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("tuta", version)
		return
	}
	soundType := "success"
	if len(os.Args) > 1 {
		if _, ok := sounds[os.Args[1]]; ok {
			soundType = os.Args[1]
		}
	}
	if err := playNotification(soundType); err != nil {
		os.Exit(1)
	}
}
