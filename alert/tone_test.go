package alert

import "testing"

func TestGenerateTone(t *testing.T) {
	wave := generateTone(tone{440.0, 0.1, "sine", 0.2})
	want := int(sampleRate * 0.1)
	if len(wave) != want {
		t.Fatalf("generateTone len = %d, want %d", len(wave), want)
	}
}

func TestGenerateSweep(t *testing.T) {
	wave := generateSweep(329.63, 233.08, 0.5, "triangle", 0.12)
	want := int(sampleRate * 0.5)
	if len(wave) != want {
		t.Fatalf("generateSweep len = %d, want %d", len(wave), want)
	}
}
