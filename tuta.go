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
	"increase": {
		{261.63, 0.08, "sine", 0.2},
		{329.63, 0.08, "sine", 0.2},
		{392.00, 0.15, "sine", 0.35},
	},
	"decrease": {
		{392.00, 0.08, "sine", 0.2},
		{311.13, 0.08, "sine", 0.2},
		{261.63, 0.15, "sine", 0.35},
	},
	"notify": {
		{880.00, 0.10, "sine", 0.3},
		{880.00, 0.18, "sine", 0.35},
	},
	"progress": {
		{329.63, 0.08, "triangle", 0.15},
		{329.63, 0.08, "triangle", 0.15},
		{329.63, 0.08, "triangle", 0.15},
	},
	"confirm": {
		{523.25, 0.08, "sine", 0.2},
		{783.99, 0.20, "sine", 0.35},
	},
	"cancel": {
		{493.88, 0.10, "triangle", 0.2},
		{233.08, 0.30, "triangle", 0.35},
	},
	"ready": {
		{523.25, 0.25, "sine", 0.25},
	},
	"timeout": {
		{329.63, 0.15, "square", 0.25},
		{261.63, 0.35, "square", 0.35},
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

func usage() {
	fmt.Printf(`tuta v%s — Tiny Utility for Tone Alerts
Author: Jonathan Gabor

Usage:
  tuta [sound]

Available sounds:
  success   ascending C major arpeggio (default)
  error     low descending buzz
  warning   two radar-style pings
  info      single neutral tone
  complete  ascending triad
  increase  ascending major triad (C4-E4-G4)
  decrease  descending minor triad (G4-Eb4-C4)
  notify    two bright high pings
  progress  three even pulses
  confirm   ascending perfect fifth
  cancel    descending minor second
  ready     single sustained tone
  timeout   descending square buzz

Options:
  -h, --help      show this help
  -v, --version   show version
`, version)
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println("tuta", version)
			return
		case "--help", "-h":
			usage()
			return
		}
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
