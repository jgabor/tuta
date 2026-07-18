package alert

import (
	"strings"
	"testing"
)

func TestPlayAtVolumeRejectsOutOfRangePercentage(t *testing.T) {
	for _, percent := range []int{-1, 101} {
		if err := PlayAtVolume("success", percent); err == nil || !strings.Contains(err.Error(), "0 to 100") {
			t.Fatalf("PlayAtVolume(_, %d) error = %v, want range error", percent, err)
		}
	}
}
