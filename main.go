package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jgabor/tuta/alert"
)

var version = "0.6.0"

// Exit codes (documented in README.md → Exit codes).
const (
	exitOK    = 0 // success
	exitFail  = 1 // runtime failure (playback, export, soundcheck)
	exitUsage = 2 // usage error / unavailable-in-this-build
)

// debugCLI runs `tuta debug ...` when built with -tags debug. It is nil in
// release builds; run() then explains that a debug build is required. Splitting
// the hook this way keeps the dispatch logic build-tag independent and testable.
var (
	debugCLI   func(args []string) int
	debugUsage func() string
)

// main wires production dependencies into run() and exits with its code.
func main() {
	os.Exit((&runner{
		stdout: os.Stdout,
		stderr: os.Stderr,
		play:   alert.PlayAtVolume,
		debug:  debugCLI,
	}).run(os.Args[1:]))
}

// runner dispatches the CLI. All side effects go through the injected writers
// and the play/debug functions, so run() is unit-testable without an audio
// device or the debug build tag.
type runner struct {
	stdout io.Writer
	stderr io.Writer
	play   func(name string, volume int) error
	debug  func(args []string) int // nil in release builds
}

// run dispatches args and returns a process exit code (see Exit codes).
func (r *runner) run(args []string) int {
	args, volume, volumeSet, code := r.parseGlobalVolume(args)
	if code != exitOK {
		return code
	}
	if len(args) == 0 {
		return r.playSound("success", volume)
	}

	switch args[0] {
	case "--version", "-v":
		if len(args) > 1 {
			return r.usageError("unexpected argument %q after %q", args[1], args[0])
		}
		_, _ = fmt.Fprintf(r.stdout, "tuta %s\n", version)
		return exitOK
	case "--help", "-h":
		if len(args) > 1 {
			return r.usageError("unexpected argument %q after %q", args[1], args[0])
		}
		r.writeUsage(r.stdout)
		return exitOK
	case "list":
		if volumeSet {
			return r.volumeCommandError("list")
		}
		return r.runList(args[1:])
	case "preview":
		return r.runPreview(args[1:], volume)
	case "export":
		if volumeSet {
			return r.volumeCommandError("export")
		}
		return r.runExport(args[1:])
	case "debug":
		if volumeSet {
			return r.volumeCommandError("debug")
		}
		if r.debug == nil {
			_, _ = fmt.Fprint(r.stderr, debugUnavailableMessage())
			return exitUsage
		}
		return r.debug(args[1:])
	}

	// A leading dash here is not a recognized option (the cases above handle
	// the known flags), so it is an unknown option rather than a sound name.
	if strings.HasPrefix(args[0], "-") {
		return r.usageError("unknown option %q", args[0])
	}

	// Play path: exactly one sound name, no trailing arguments or options
	// (e.g. `tuta success extra` or `tuta success --bogus` are usage errors).
	if len(args) > 1 {
		return r.usageError("unexpected argument %q; usage: tuta [sound]", args[1])
	}

	return r.playSound(args[0], volume)
}

func (r *runner) parseGlobalVolume(args []string) ([]string, int, bool, int) {
	const defaultVolume = 100
	if len(args) == 0 || (args[0] != "--volume" && !strings.HasPrefix(args[0], "--volume=")) {
		return args, defaultVolume, false, exitOK
	}

	var value string
	if args[0] == "--volume" {
		if len(args) < 2 {
			return nil, 0, false, r.usageError("--volume requires a percentage from 0 to 100")
		}
		value = args[1]
		args = args[2:]
	} else {
		value = strings.TrimPrefix(args[0], "--volume=")
		args = args[1:]
	}

	volume, err := strconv.Atoi(value)
	if err != nil || volume < 0 || volume > 100 {
		return nil, 0, false, r.usageError("invalid volume %q; must be an integer percentage from 0 to 100", value)
	}
	return args, volume, true, exitOK
}

func (r *runner) volumeCommandError(command string) int {
	return r.usageError("--volume applies only to sound playback and preview, not %s", command)
}

// usageError prints a tuta-prefixed message and usage to stderr. Used for all
// dispatch-level usage errors (unknown option, unexpected/trailing argument).
func (r *runner) usageError(format string, args ...any) int {
	_, _ = fmt.Fprintf(r.stderr, "tuta: "+format+"\n", args...)
	r.writeUsage(r.stderr)
	return exitUsage
}

// playSound plays name, warning clearly on stderr when name is not a built-in
// sound and falling back to "success" so agent hosts still get a feedback cue
// (best-effort playback — see README). Playback errors print the requested
// sound name to stderr.
func (r *runner) playSound(name string, volume int) int {
	sound := name
	if !isKnownSound(name) {
		_, _ = fmt.Fprintf(r.stderr, "tuta: unknown sound %q; playing \"success\" as fallback\n", name)
		sound = "success"
	}
	if err := r.play(sound, volume); err != nil {
		_, _ = fmt.Fprintf(r.stderr, "tuta: error playing %q: %v\n", name, err)
		return exitFail
	}
	return exitOK
}

func (r *runner) runList(args []string) int {
	if hasHelpArg(args) {
		_, _ = fmt.Fprint(r.stdout, listUsage)
		return exitOK
	}

	jsonOutput := len(args) > 0 && args[0] == "--json"
	if jsonOutput {
		args = args[1:]
	}
	if len(args) > 0 {
		if strings.HasPrefix(args[0], "-") {
			return r.usageError("unknown list option %q", args[0])
		}
		return r.usageError("unexpected argument %q; usage: tuta list [--json]", args[0])
	}

	catalog := alert.Sounds()
	if jsonOutput {
		encoder := json.NewEncoder(r.stdout)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(catalog); err != nil {
			_, _ = fmt.Fprintf(r.stderr, "tuta: writing sound list: %v\n", err)
			return exitFail
		}
		return exitOK
	}

	for _, sound := range catalog {
		_, _ = fmt.Fprintf(r.stdout, "%-10s %s\n", sound.Name, sound.Description)
	}
	return exitOK
}

func (r *runner) runPreview(args []string, volume int) int {
	if hasHelpArg(args) {
		_, _ = fmt.Fprint(r.stdout, previewUsage)
		return exitOK
	}

	names := args
	if len(names) == 0 {
		names = alert.Names()
	}

	// Validate the complete sequence before playing anything so a typo cannot
	// leave the user with a partially played preview.
	for _, name := range names {
		if strings.HasPrefix(name, "-") {
			return r.usageError("unknown preview option %q", name)
		}
		if !isKnownSound(name) {
			return r.usageError("unknown sound %q; usage: tuta preview [sound ...]", name)
		}
	}

	for _, name := range names {
		_, _ = fmt.Fprintln(r.stdout, name)
		if err := r.play(name, volume); err != nil {
			_, _ = fmt.Fprintf(r.stderr, "tuta: error previewing %q: %v\n", name, err)
			return exitFail
		}
	}
	return exitOK
}

func (r *runner) runExport(args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	var flagOutput strings.Builder
	fs.SetOutput(&flagOutput)
	fs.Usage = func() {
		_, _ = fmt.Fprint(fs.Output(), exportUsage)
		fs.PrintDefaults()
		_, _ = fmt.Fprintln(fs.Output(), "  -h, --help\n    \tshow this help")
	}

	outDir := fs.String("o", "./tmp", "output directory")
	mono := fs.Bool("mono", true, "export mono FLAC")
	stereo := fs.Bool("stereo", false, "export stereo FLAC (L=R)")
	depth := fs.Int("depth", 16, "bit depth: 16 or 24")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprint(r.stdout, flagOutput.String())
			return exitOK
		}
		_, _ = fmt.Fprint(r.stderr, flagOutput.String())
		return exitUsage
	}
	if *stereo {
		*mono = false
	}
	if *depth != 16 && *depth != 24 {
		_, _ = fmt.Fprintf(r.stderr, "tuta: -depth must be 16 or 24, got %d\n", *depth)
		return exitUsage
	}

	channels := 1
	if !*mono {
		channels = 2
	}

	opts := alert.FLACOptions{
		Channels: channels,
		BitDepth: *depth,
	}

	names := fs.Args()
	if len(names) == 0 {
		names = alert.Names()
	}

	for _, name := range names {
		if !isKnownSound(name) {
			_, _ = fmt.Fprintf(r.stderr, "tuta: unknown sound %q\n", name)
			return exitUsage
		}
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		_, _ = fmt.Fprintf(r.stderr, "tuta: %v\n", err)
		return exitFail
	}

	for _, name := range names {
		path := filepath.Join(*outDir, name+".flac")
		if err := alert.ExportFLAC(path, name, opts); err != nil {
			_, _ = fmt.Fprintf(r.stderr, "tuta: exporting %q: %v\n", name, err)
			return exitFail
		}
		_, _ = fmt.Fprintln(r.stdout, path)
	}
	return exitOK
}

func hasHelpArg(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func (r *runner) writeUsage(w io.Writer) {
	extra := ""
	if debugUsage != nil {
		extra = debugUsage()
	}
	_, _ = fmt.Fprintf(w, usageTemplate, version, extra, soundUsage())
}

func soundUsage() string {
	var usage strings.Builder
	for _, sound := range alert.Sounds() {
		_, _ = fmt.Fprintf(&usage, "  %-9s %s\n", sound.Name, sound.Description)
	}
	return usage.String()
}

const listUsage = `tuta list — list built-in sounds without playing them

Usage:
  tuta list [--json]

Options:
  --json      print a stable, name-sorted JSON array
  -h, --help  show this help
`

const previewUsage = `tuta preview — play built-in sounds in sequence

Usage:
  tuta [--volume PERCENT] preview [sound ...]

With no sound names, preview plays every sound in name order. The complete
sequence is validated before playback begins.

Options:
  -h, --help  show this help

Global options (must precede preview):
  --volume PERCENT  playback volume from 0 to 100 (default: 100)
`

const exportUsage = `tuta export — export built-in sounds as FLAC files

Usage:
  tuta export [-o DIR] [-mono|-stereo] [-depth 16|24] [sound ...]

Options:
`

const usageTemplate = `tuta %s — Tiny Utility for Tone Alerts
Author: Jonathan Gabor

Usage:
  tuta [--volume PERCENT] [sound]
  tuta list [--json]
  tuta [--volume PERCENT] preview [sound ...]
  tuta export [-o DIR] [-mono|-stereo] [-depth 16|24] [sound ...]%s

Available sounds:
%s
Options:
  -h, --help        show this help
  -v, --version     show version
  --volume PERCENT  playback volume from 0 to 100 (default: 100)

Export options:
  -o DIR          output directory (default: ./tmp)
  -mono           export mono FLAC (default)
  -stereo         export stereo FLAC (L=R, same as playback)
  -depth N        bit depth: 16 or 24 (default: 16)

Exit codes (see README for full detail):
  0  success; an unknown play sound warns and falls back to success
  1  runtime failure (playback, export, soundcheck)
  2  usage error (unknown option/command, bad flag, export of an unknown
     sound; or tuta debug invoked from a release build)
`

// debugUnavailableMessage explains why `tuta debug` does not run in a release
// build and how to get a debug build. Defined here (not behind a build tag) so
// the release path is exercised by unit tests regardless of build tags.
func debugUnavailableMessage() string {
	return `tuta debug requires a debug build; release builds omit it to stay small.

Rebuild with one of:
  mage build -debug=true     # ./build/tuta (debug)
  DEBUG=1 mage build         # same as above
  mage install               # $(go env GOBIN)/tuta (debug)

Then run:
  ./build/tuta debug sounds all tmp/  # after mage build
  tuta debug sounds all tmp/          # after mage install
`
}

func isKnownSound(name string) bool {
	for _, n := range alert.Names() {
		if n == name {
			return true
		}
	}
	return false
}
