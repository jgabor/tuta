package alert

import "math"

const sampleRate = 44100

// Sound describes a built-in notification sound.
type Sound struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Waveform    string `json:"waveform"`
}

type tone struct {
	frequency float64
	duration  float64
	volume    float64
}

type sweep struct {
	fromFreq float64
	toFreq   float64
	duration float64
	volume   float64
}

type soundDefinition struct {
	metadata Sound
	tones    []tone
	sweep    *sweep
}

// soundDefinitions is the source of truth for built-in names, user-facing
// metadata, and synthesis parameters. Keep it sorted by name so Sounds and
// Names have stable output without depending on map iteration order.
var soundDefinitions = []soundDefinition{
	{
		metadata: Sound{"cancel", "single tone (B4)", "triangle"},
		tones:    []tone{{493.88, 0.18, 0.30}},
	},
	{
		metadata: Sound{"complete", "ascending F major triad (F4-A4-C5)", "triangle"},
		tones:    []tone{{349.23, 0.10, 0.18}, {440.00, 0.10, 0.25}, {523.25, 0.40, 0.35}},
	},
	{
		metadata: Sound{"confirm", "ascending perfect fifth (C5-G5)", "sine"},
		tones:    []tone{{523.25, 0.08, 0.2}, {783.99, 0.35, 0.25}},
	},
	{
		metadata: Sound{"decrease", "descending minor triad, fading (G4-Eb4-C4)", "triangle"},
		tones:    []tone{{392.00, 0.08, 0.35}, {311.13, 0.08, 0.25}, {261.63, 0.15, 0.15}},
	},
	{
		metadata: Sound{"error", "descending tritone buzz (D3-Ab2)", "square"},
		tones:    []tone{{146.83, 0.10, 0.16}, {103.83, 0.30, 0.19}},
	},
	{
		metadata: Sound{"increase", "ascending major triad (C4-E4-G4)", "sine"},
		tones:    []tone{{261.63, 0.08, 0.2}, {329.63, 0.08, 0.2}, {392.00, 0.15, 0.35}},
	},
	{
		metadata: Sound{"info", "short neutral blip", "sine"},
		tones:    []tone{{523.25, 0.10, 0.27}},
	},
	{
		metadata: Sound{"notify", "ascending minor third ping (A5-C6)", "sine"},
		tones:    []tone{{880.00, 0.10, 0.30}, {1046.50, 0.18, 0.35}},
	},
	{
		metadata: Sound{"progress", "ascending major triad (E4-G4-B4)", "triangle"},
		tones:    []tone{{329.63, 0.08, 0.29}, {392.00, 0.08, 0.29}, {493.88, 0.10, 0.39}},
	},
	{
		metadata: Sound{"ready", "ascending major third (C5-E5, triangle)", "triangle"},
		tones:    []tone{{523.25, 0.15, 0.29}, {659.25, 0.12, 0.36}},
	},
	{
		metadata: Sound{"success", "ascending C major arpeggio (default)", "sine"},
		tones:    []tone{{523.25, 0.08, 0.2}, {659.25, 0.08, 0.2}, {783.99, 0.15, 0.35}},
	},
	{
		metadata: Sound{"timeout", "descending frequency sweep (E4-Bb3)", "triangle"},
		sweep:    &sweep{329.63, 233.08, 0.50, 0.30},
	},
	{
		metadata: Sound{"warning", "three pings with major second tension", "triangle"},
		tones:    []tone{{1046.50, 0.10, 0.30}, {1174.66, 0.10, 0.30}, {1046.50, 0.18, 0.35}},
	},
}

func findSound(name string) *soundDefinition {
	for i := range soundDefinitions {
		if soundDefinitions[i].metadata.Name == name {
			return &soundDefinitions[i]
		}
	}
	return nil
}

func generateTone(t tone, waveform string) []float32 {
	samples := int(sampleRate * t.duration)
	wave := make([]float32, samples)
	for i := range wave {
		ts := float64(i) / sampleRate
		var s float64
		switch waveform {
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
