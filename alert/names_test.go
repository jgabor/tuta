package alert

import (
	"slices"
	"testing"
)

func TestNames(t *testing.T) {
	names := Names()
	if len(names) != len(sounds)+1 {
		t.Fatalf("Names() len = %d, want %d", len(names), len(sounds)+1)
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
