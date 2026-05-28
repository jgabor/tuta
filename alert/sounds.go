package alert

import "math"

const sampleRate = 44100

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
		{146.83, 0.10, "square", 0.30},
		{103.83, 0.30, "square", 0.35},
	},
	"warning": {
		{1046.50, 0.10, "triangle", 0.30},
		{1174.66, 0.10, "triangle", 0.30},
		{1046.50, 0.18, "triangle", 0.35},
	},
	"info": {
		{523.25, 0.10, "sine", 0.40},
	},
	"complete": {
		{349.23, 0.10, "triangle", 0.18},
		{440.00, 0.10, "triangle", 0.25},
		{523.25, 0.40, "triangle", 0.35},
	},
	"increase": {
		{261.63, 0.08, "sine", 0.2},
		{329.63, 0.08, "sine", 0.2},
		{392.00, 0.15, "sine", 0.35},
	},
	"decrease": {
		{392.00, 0.08, "triangle", 0.35},
		{311.13, 0.08, "triangle", 0.25},
		{261.63, 0.15, "triangle", 0.15},
	},
	"notify": {
		{880.00, 0.10, "sine", 0.30},
		{1046.50, 0.18, "sine", 0.35},
	},
	"progress": {
		{329.63, 0.08, "triangle", 0.15},
		{392.00, 0.08, "triangle", 0.15},
		{493.88, 0.10, "triangle", 0.20},
	},
	"confirm": {
		{523.25, 0.08, "sine", 0.2},
		{783.99, 0.35, "sine", 0.25},
	},
	"cancel": {
		{493.88, 0.18, "triangle", 0.30},
	},
	"ready": {
		{523.25, 0.15, "triangle", 0.20},
		{659.25, 0.12, "triangle", 0.25},
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

func generateSweep(fromFreq, toFreq, duration float64, waveform string, volume float64) []float32 {
	samples := int(sampleRate * duration)
	wave := make([]float32, samples)
	for i := range wave {
		ts := float64(i) / sampleRate
		freq := fromFreq + (toFreq-fromFreq)*ts/duration
		var s float64
		switch waveform {
		case "sine":
			s = math.Sin(2 * math.Pi * freq * ts)
		case "square":
			s = math.Copysign(1, math.Sin(2*math.Pi*freq*ts))
		case "triangle":
			p := ts*freq - math.Floor(ts*freq+0.5)
			s = 2*math.Abs(2*p) - 1
		default:
			s = math.Sin(2 * math.Pi * freq * ts)
		}
		wave[i] = float32(s * math.Exp(-3*ts/duration) * volume)
	}
	return wave
}
