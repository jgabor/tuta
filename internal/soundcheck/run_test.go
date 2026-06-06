//go:build debug

package soundcheck

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jgabor/tuta/alert"
)

func TestRunAllExportedSounds(t *testing.T) {
	dir := t.TempDir()
	for _, name := range alert.Names() {
		path := filepath.Join(dir, name+".flac")
		if err := alert.ExportFLAC(path, name, alert.FLACOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	if code := Run([]string{"all", dir}); code != 0 {
		t.Fatalf("Run(all): exit %d, want 0", code)
	}
}

func TestUsageNonEmpty(t *testing.T) {
	if Usage() == "" {
		t.Fatal("Usage() returned empty string")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if code := Run([]string{"nope"}); code != 2 {
		t.Fatalf("Run(nope): exit %d, want 2", code)
	}
}

func TestRunMissingFLACDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "empty")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"volume", dir}); code != 2 {
		t.Fatalf("Run(volume empty dir): exit %d, want 2", code)
	}
}
