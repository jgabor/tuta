package alert

import (
	"slices"
	"testing"
)

func TestNames(t *testing.T) {
	names := Names()
	if len(names) != len(soundDefinitions) {
		t.Fatalf("Names() len = %d, want %d", len(names), len(soundDefinitions))
	}
	if !slices.IsSorted(names) {
		t.Fatalf("Names() not sorted: %v", names)
	}
	for _, want := range []string{"success", "timeout"} {
		if !slices.Contains(names, want) {
			t.Fatalf("Names() missing %q: %v", want, names)
		}
	}
}

func TestSounds(t *testing.T) {
	catalog := Sounds()
	names := Names()
	if len(catalog) != len(names) {
		t.Fatalf("Sounds() len = %d, want %d", len(catalog), len(names))
	}
	for i, sound := range catalog {
		if sound.Name != names[i] {
			t.Fatalf("Sounds()[%d].Name = %q, want %q", i, sound.Name, names[i])
		}
		if sound.Description == "" || sound.Waveform == "" {
			t.Fatalf("incomplete metadata for %q: %+v", sound.Name, sound)
		}
	}

	catalog[0].Name = "mutated"
	if Sounds()[0].Name == "mutated" {
		t.Fatal("Sounds returned mutable internal metadata")
	}
}
