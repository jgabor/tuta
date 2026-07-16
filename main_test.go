package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jgabor/tuta/alert"
	"github.com/tphakala/go-flac/pcm"
)

// fakePlay records the requested sound name and returns a configured error.
// It replaces alert.Play so dispatch tests never touch an audio device.
type fakePlay struct {
	last string
	err  error
}

func (f *fakePlay) play(name string) error {
	f.last = name
	return f.err
}

func newRunner(play func(string) error, debug func([]string) int) (*runner, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errs := &bytes.Buffer{}
	r := &runner{stdout: out, stderr: errs, play: play, debug: debug}
	return r, out, errs
}

func TestRunVersionLongShort(t *testing.T) {
	for _, arg := range []string{"--version", "-v"} {
		t.Run(arg, func(t *testing.T) {
			fp := &fakePlay{}
			r, out, _ := newRunner(fp.play, nil)
			if code := r.run([]string{arg}); code != exitOK {
				t.Fatalf("run(%q): exit %d, want %d", arg, code, exitOK)
			}
			if got := out.String(); !strings.HasPrefix(got, "tuta ") || !strings.HasSuffix(got, "\n") {
				t.Fatalf("version output %q must be \"tuta <version>\\n\"", got)
			}
			if fp.last != "" {
				t.Fatalf("play was invoked with %q; expected no playback", fp.last)
			}
		})
	}
}

func TestRunHelpLongShort(t *testing.T) {
	for _, arg := range []string{"--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			fp := &fakePlay{}
			r, out, _ := newRunner(fp.play, nil)
			if code := r.run([]string{arg}); code != exitOK {
				t.Fatalf("run(%q): exit %d, want %d", arg, code, exitOK)
			}
			if !strings.Contains(out.String(), "Available sounds:") || !strings.Contains(out.String(), "Export options:") {
				t.Fatalf("help output missing sections:\n%s", out.String())
			}
			if fp.last != "" {
				t.Fatalf("play was invoked with %q; expected no playback", fp.last)
			}
		})
	}
}

func TestRunDefaultPlaysSuccess(t *testing.T) {
	fp := &fakePlay{}
	r, _, _ := newRunner(fp.play, nil)
	if code := r.run(nil); code != exitOK {
		t.Fatalf("run(no args): exit %d, want %d", code, exitOK)
	}
	if fp.last != "success" {
		t.Fatalf("default sound = %q, want \"success\"", fp.last)
	}
}

func TestRunKnownSoundDispatch(t *testing.T) {
	for _, name := range []string{"error", "warning", "info", "timeout", "success"} {
		t.Run(name, func(t *testing.T) {
			fp := &fakePlay{}
			r, _, stderr := newRunner(fp.play, nil)
			if code := r.run([]string{name}); code != exitOK {
				t.Fatalf("run(%q): exit %d, want %d", name, code, exitOK)
			}
			if fp.last != name {
				t.Fatalf("played %q, want %q", fp.last, name)
			}
			if stderr.Len() != 0 {
				t.Fatalf("known sound wrote to stderr: %q", stderr.String())
			}
		})
	}
}

func TestRunUnknownSoundWarnsAndFallsBack(t *testing.T) {
	fp := &fakePlay{}
	r, _, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"frobnicate"}); code != exitOK {
		t.Fatalf("run(unknown sound): exit %d, want %d (best-effort fallback)", code, exitOK)
	}
	if fp.last != "success" {
		t.Fatalf("fallback played %q, want \"success\"", fp.last)
	}
	if got := stderr.String(); !strings.Contains(got, `unknown sound "frobnicate"`) || !strings.Contains(got, "success") {
		t.Fatalf("stderr missing clear unknown-sound warning: %q", got)
	}
}

func TestRunUnknownOptionIsUsageError(t *testing.T) {
	fp := &fakePlay{}
	r, _, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"--bogus"}); code != exitUsage {
		t.Fatalf("run(--bogus): exit %d, want %d", code, exitUsage)
	}
	if fp.last != "" {
		t.Fatalf("play was invoked with %q; unknown option must not play", fp.last)
	}
	if got := stderr.String(); !strings.Contains(got, `unknown option "--bogus"`) {
		t.Fatalf("stderr missing unknown-option message: %q", got)
	}
}

func TestRunPlaybackErrorPrintsSoundName(t *testing.T) {
	fp := &fakePlay{err: os.ErrNotExist}
	r, _, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"error"}); code != exitFail {
		t.Fatalf("playback error: exit %d, want %d", code, exitFail)
	}
	got := stderr.String()
	if !strings.Contains(got, `playing "error"`) {
		t.Fatalf("stderr must name the requested sound on playback failure: %q", got)
	}
}

func TestRunExportUnknownSound(t *testing.T) {
	fp := &fakePlay{}
	r, _, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"export", "-o", t.TempDir(), "notasound"}); code != exitUsage {
		t.Fatalf("export unknown sound: exit %d, want %d", code, exitUsage)
	}
	if got := stderr.String(); !strings.Contains(got, `unknown sound "notasound"`) {
		t.Fatalf("stderr missing unknown-sound message: %q", got)
	}
	if fp.last != "" {
		t.Fatalf("export error must not trigger playback; got %q", fp.last)
	}
}

func TestRunExportBadDepth(t *testing.T) {
	fp := &fakePlay{}
	r, _, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"export", "-depth", "99", "-o", t.TempDir(), "success"}); code != exitUsage {
		t.Fatalf("export -depth 99: exit %d, want %d", code, exitUsage)
	}
	if got := stderr.String(); !strings.Contains(got, "-depth must be 16 or 24") {
		t.Fatalf("stderr missing depth validation message: %q", got)
	}
}

func TestRunExportBadFlag(t *testing.T) {
	fp := &fakePlay{}
	r, _, _ := newRunner(fp.play, nil)
	if code := r.run([]string{"export", "-nope"}); code != exitUsage {
		t.Fatalf("export -nope: exit %d, want %d", code, exitUsage)
	}
}

func TestRunExportWritesFLACAndPrintsPath(t *testing.T) {
	fp := &fakePlay{}
	dir := t.TempDir()
	r, out, _ := newRunner(fp.play, nil)
	if code := r.run([]string{"export", "-o", dir, "success"}); code != exitOK {
		t.Fatalf("export success: exit %d, want %d", code, exitOK)
	}
	path := filepath.Join(dir, "success.flac")
	if info, err := os.Stat(path); err != nil || info.Size() == 0 {
		t.Fatalf("export did not write non-empty FLAC at %s: %v", path, err)
	}
	if printed := strings.TrimSpace(out.String()); printed != path {
		t.Fatalf("stdout = %q, want %q", printed, path)
	}
	if fp.last != "" {
		t.Fatalf("export must not trigger playback; got %q", fp.last)
	}
}

func TestRunExportStereo24BitRoundTrip(t *testing.T) {
	fp := &fakePlay{}
	dir := t.TempDir()
	r, out, _ := newRunner(fp.play, nil)
	if code := r.run([]string{"export", "-stereo", "-depth", "24", "-o", dir, "error"}); code != exitOK {
		t.Fatalf("export -stereo -depth 24: exit %d, want %d", code, exitOK)
	}
	path := filepath.Join(dir, "error.flac")
	if printed := strings.TrimSpace(out.String()); printed != path {
		t.Fatalf("stdout = %q, want %q", printed, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	dec, err := pcm.NewDecoder(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode export: %v", err)
	}
	info := dec.Info()
	if info.Channels != 2 {
		t.Fatalf("channels = %d, want 2 (stereo)", info.Channels)
	}
	if info.BitDepth != 24 {
		t.Fatalf("bit depth = %d, want 24", info.BitDepth)
	}
	if info.SampleRate != alert.SampleRate() {
		t.Fatalf("sample rate = %d, want %d", info.SampleRate, alert.SampleRate())
	}
	if _, err := io.ReadAll(dec); err != nil {
		t.Fatalf("read PCM stream: %v", err)
	}
	if fp.last != "" {
		t.Fatalf("export must not trigger playback; got %q", fp.last)
	}
}

func TestRunPlayRejectsTrailingArg(t *testing.T) {
	fp := &fakePlay{}
	r, _, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"success", "extra"}); code != exitUsage {
		t.Fatalf("tuta success extra: exit %d, want %d", code, exitUsage)
	}
	if fp.last != "" {
		t.Fatalf("trailing arg must not trigger playback; got %q", fp.last)
	}
	if got := stderr.String(); !strings.Contains(got, `unexpected argument "extra"`) {
		t.Fatalf("stderr missing unexpected-argument message: %q", got)
	}
}

func TestRunPlayRejectsTrailingOption(t *testing.T) {
	fp := &fakePlay{}
	r, _, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"success", "--bogus"}); code != exitUsage {
		t.Fatalf("tuta success --bogus: exit %d, want %d", code, exitUsage)
	}
	if fp.last != "" {
		t.Fatalf("trailing option must not trigger playback; got %q", fp.last)
	}
	if got := stderr.String(); !strings.Contains(got, `unexpected argument "--bogus"`) {
		t.Fatalf("stderr missing unexpected-argument message: %q", got)
	}
}

func TestRunHelpRejectsTrailingArg(t *testing.T) {
	fp := &fakePlay{}
	r, _, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"--help", "extra"}); code != exitUsage {
		t.Fatalf("tuta --help extra: exit %d, want %d", code, exitUsage)
	}
	if fp.last != "" {
		t.Fatalf("trailing arg must not trigger playback; got %q", fp.last)
	}
	if got := stderr.String(); !strings.Contains(got, `unexpected argument "extra"`) {
		t.Fatalf("stderr missing unexpected-argument message: %q", got)
	}
}

func TestRunVersionRejectsTrailingArg(t *testing.T) {
	fp := &fakePlay{}
	r, _, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"--version", "extra"}); code != exitUsage {
		t.Fatalf("tuta --version extra: exit %d, want %d", code, exitUsage)
	}
	if fp.last != "" {
		t.Fatalf("trailing arg must not trigger playback; got %q", fp.last)
	}
	if got := stderr.String(); !strings.Contains(got, `unexpected argument "extra"`) {
		t.Fatalf("stderr missing unexpected-argument message: %q", got)
	}
}

func TestRunDebugUnavailableInReleaseBuild(t *testing.T) {
	fp := &fakePlay{}
	r, _, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"debug", "sounds", "all", "tmp/"}); code != exitUsage {
		t.Fatalf("debug in release build: exit %d, want %d", code, exitUsage)
	}
	if got := stderr.String(); !strings.Contains(got, "requires a debug build") || !strings.Contains(got, "mage build -debug=true") {
		t.Fatalf("stderr must explain the debug build requirement: %q", got)
	}
	if fp.last != "" {
		t.Fatalf("debug unavailability must not trigger playback; got %q", fp.last)
	}
}

func TestRunDebugDispatchesToHandler(t *testing.T) {
	fp := &fakePlay{}
	var got []string
	fakeDebug := func(args []string) int {
		got = args
		return exitOK
	}
	r, _, _ := newRunner(fp.play, fakeDebug)
	if code := r.run([]string{"debug", "sounds", "all", "tmp/"}); code != exitOK {
		t.Fatalf("debug dispatch: exit %d, want %d", code, exitOK)
	}
	if len(got) != 3 || got[0] != "sounds" || got[1] != "all" || got[2] != "tmp/" {
		t.Fatalf("debug handler received %v, want [sounds all tmp/]", got)
	}
	if fp.last != "" {
		t.Fatalf("debug dispatch must not trigger playback; got %q", fp.last)
	}
}

func TestIsKnownSound(t *testing.T) {
	if !isKnownSound("success") || !isKnownSound("timeout") {
		t.Fatal("expected known sounds to be recognized")
	}
	if isKnownSound("nope") || isKnownSound("") {
		t.Fatal("expected unknown/empty names to be unrecognized")
	}
}

func TestDebugUnavailableMessage(t *testing.T) {
	msg := debugUnavailableMessage()
	for _, want := range []string{"requires a debug build", "mage build -debug=true", "mage install", "tuta debug sounds all tmp/"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("debugUnavailableMessage missing %q:\n%s", want, msg)
		}
	}
}
