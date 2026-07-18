package main

import (
	"bytes"
	"encoding/json"
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
	r := &runner{
		stdout: out,
		stderr: errs,
		play:   func(name string, _ int) error { return play(name) },
		debug:  debug,
	}
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

func TestRunList(t *testing.T) {
	fp := &fakePlay{}
	r, out, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"list"}); code != exitOK {
		t.Fatalf("run(list): exit %d, want %d", code, exitOK)
	}
	for _, sound := range alert.Sounds() {
		if !strings.Contains(out.String(), sound.Name) || !strings.Contains(out.String(), sound.Description) {
			t.Fatalf("list output missing metadata for %q:\n%s", sound.Name, out.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("list wrote to stderr: %q", stderr.String())
	}
	if fp.last != "" {
		t.Fatalf("list triggered playback with %q", fp.last)
	}
}

func TestRunListJSON(t *testing.T) {
	fp := &fakePlay{}
	r, out, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"list", "--json"}); code != exitOK {
		t.Fatalf("run(list --json): exit %d, want %d", code, exitOK)
	}
	var got []alert.Sound
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("list --json returned invalid JSON: %v\n%s", err, out.String())
	}
	want := alert.Sounds()
	if len(got) != len(want) {
		t.Fatalf("JSON sound count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("JSON sound %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("list --json wrote to stderr: %q", stderr.String())
	}
	if fp.last != "" {
		t.Fatalf("list --json triggered playback with %q", fp.last)
	}
}

func TestRunListRejectsArguments(t *testing.T) {
	for _, args := range [][]string{{"list", "extra"}, {"list", "--bogus"}, {"list", "--json", "extra"}} {
		fp := &fakePlay{}
		r, _, stderr := newRunner(fp.play, nil)
		if code := r.run(args); code != exitUsage {
			t.Fatalf("run(%v): exit %d, want %d", args, code, exitUsage)
		}
		if stderr.Len() == 0 {
			t.Fatalf("run(%v) did not explain usage error", args)
		}
		if fp.last != "" {
			t.Fatalf("run(%v) triggered playback with %q", args, fp.last)
		}
	}
}

func TestRunPreviewAll(t *testing.T) {
	var played []string
	r, out, stderr := newRunner(func(name string) error {
		played = append(played, name)
		return nil
	}, nil)
	if code := r.run([]string{"preview"}); code != exitOK {
		t.Fatalf("run(preview): exit %d, want %d", code, exitOK)
	}
	want := alert.Names()
	if strings.Join(played, "\n") != strings.Join(want, "\n") {
		t.Fatalf("preview played %v, want %v", played, want)
	}
	if got := strings.Fields(out.String()); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("preview output names = %v, want %v", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("preview wrote to stderr: %q", stderr.String())
	}
}

func TestRunPreviewSequence(t *testing.T) {
	var played []string
	r, out, _ := newRunner(func(name string) error {
		played = append(played, name)
		return nil
	}, nil)
	want := []string{"success", "warning", "ready"}
	if code := r.run(append([]string{"preview"}, want...)); code != exitOK {
		t.Fatalf("run(preview sequence): exit %d, want %d", code, exitOK)
	}
	if strings.Join(played, "\n") != strings.Join(want, "\n") {
		t.Fatalf("preview played %v, want %v", played, want)
	}
	if got := strings.Fields(out.String()); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("preview output names = %v, want %v", got, want)
	}
}

func TestRunPreviewValidatesSequenceBeforePlayback(t *testing.T) {
	fp := &fakePlay{}
	r, out, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"preview", "success", "notasound"}); code != exitUsage {
		t.Fatalf("preview unknown sound: exit %d, want %d", code, exitUsage)
	}
	if fp.last != "" || out.Len() != 0 {
		t.Fatalf("invalid preview partially ran: played %q, output %q", fp.last, out.String())
	}
	if got := stderr.String(); !strings.Contains(got, `unknown sound "notasound"`) {
		t.Fatalf("stderr missing unknown-sound message: %q", got)
	}
}

func TestRunPreviewRejectsOption(t *testing.T) {
	fp := &fakePlay{}
	r, _, stderr := newRunner(fp.play, nil)
	if code := r.run([]string{"preview", "--bogus"}); code != exitUsage {
		t.Fatalf("preview unknown option: exit %d, want %d", code, exitUsage)
	}
	if fp.last != "" {
		t.Fatalf("preview option triggered playback with %q", fp.last)
	}
	if got := stderr.String(); !strings.Contains(got, `unknown preview option "--bogus"`) {
		t.Fatalf("stderr missing unknown-option message: %q", got)
	}
}

func TestRunPreviewPlaybackError(t *testing.T) {
	played := 0
	r, out, stderr := newRunner(func(string) error {
		played++
		return os.ErrNotExist
	}, nil)
	if code := r.run([]string{"preview", "error", "success"}); code != exitFail {
		t.Fatalf("preview playback error: exit %d, want %d", code, exitFail)
	}
	if played != 1 {
		t.Fatalf("preview continued after playback error: calls = %d, want 1", played)
	}
	if got := out.String(); got != "error\n" {
		t.Fatalf("preview output = %q, want %q", got, "error\\n")
	}
	if got := stderr.String(); !strings.Contains(got, `previewing "error"`) {
		t.Fatalf("stderr must name failed preview sound: %q", got)
	}
}

func TestHelpUsesSoundMetadata(t *testing.T) {
	fp := &fakePlay{}
	r, out, _ := newRunner(fp.play, nil)
	if code := r.run([]string{"--help"}); code != exitOK {
		t.Fatalf("run(--help): exit %d, want %d", code, exitOK)
	}
	for _, sound := range alert.Sounds() {
		if !strings.Contains(out.String(), sound.Name) || !strings.Contains(out.String(), sound.Description) {
			t.Fatalf("help missing generated metadata for %q:\n%s", sound.Name, out.String())
		}
	}
}

func TestRunGlobalVolume(t *testing.T) {
	for _, args := range [][]string{{"--volume", "35", "warning"}, {"--volume=35", "warning"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var gotName string
			var gotVolume int
			r := &runner{
				stdout: io.Discard,
				stderr: io.Discard,
				play: func(name string, volume int) error {
					gotName, gotVolume = name, volume
					return nil
				},
			}
			if code := r.run(args); code != exitOK {
				t.Fatalf("run(%v): exit %d, want %d", args, code, exitOK)
			}
			if gotName != "warning" || gotVolume != 35 {
				t.Fatalf("play(%q, %d), want play(%q, %d)", gotName, gotVolume, "warning", 35)
			}
		})
	}
}

func TestRunGlobalVolumeAppliesToDefaultFallbackAndPreview(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantNames []string
	}{
		{"default", []string{"--volume", "0"}, []string{"success"}},
		{"unknown fallback", []string{"--volume", "20", "notasound"}, []string{"success"}},
		{"preview", []string{"--volume", "75", "preview", "info", "ready"}, []string{"info", "ready"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var names []string
			var volumes []int
			r := &runner{
				stdout: io.Discard,
				stderr: io.Discard,
				play: func(name string, volume int) error {
					names = append(names, name)
					volumes = append(volumes, volume)
					return nil
				},
			}
			if code := r.run(tc.args); code != exitOK {
				t.Fatalf("run(%v): exit %d, want %d", tc.args, code, exitOK)
			}
			if strings.Join(names, "\n") != strings.Join(tc.wantNames, "\n") {
				t.Fatalf("played %v, want %v", names, tc.wantNames)
			}
			for _, volume := range volumes {
				want := map[string]int{"default": 0, "unknown fallback": 20, "preview": 75}[tc.name]
				if volume != want {
					t.Fatalf("volume = %d, want %d", volume, want)
				}
			}
		})
	}
}

func TestRunGlobalVolumeValidation(t *testing.T) {
	tests := [][]string{
		{"--volume"},
		{"--volume", "loud", "success"},
		{"--volume", "-1", "success"},
		{"--volume=101", "success"},
		{"--volume=1.5", "success"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			fp := &fakePlay{}
			r, _, stderr := newRunner(fp.play, nil)
			if code := r.run(args); code != exitUsage {
				t.Fatalf("run(%v): exit %d, want %d", args, code, exitUsage)
			}
			if fp.last != "" {
				t.Fatalf("invalid volume triggered playback with %q", fp.last)
			}
			if !strings.Contains(stderr.String(), "volume") || !strings.Contains(stderr.String(), "0 to 100") {
				t.Fatalf("stderr missing volume range guidance: %q", stderr.String())
			}
		})
	}
}

func TestRunGlobalVolumeRejectsNonPlaybackCommands(t *testing.T) {
	for _, command := range []string{"list", "export", "debug"} {
		t.Run(command, func(t *testing.T) {
			fp := &fakePlay{}
			r, _, stderr := newRunner(fp.play, nil)
			if code := r.run([]string{"--volume", "50", command}); code != exitUsage {
				t.Fatalf("volume with %s: exit %d, want %d", command, code, exitUsage)
			}
			if fp.last != "" {
				t.Fatalf("volume with %s triggered playback with %q", command, fp.last)
			}
			if !strings.Contains(stderr.String(), "applies only to sound playback and preview") {
				t.Fatalf("stderr missing command scope guidance: %q", stderr.String())
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
