# tuta: tiny utility for tone alerts

A tiny CLI tool that plays synthesized notification tones for scripts and build pipelines. Each sound is designed around music theory principles — interval quality, pitch contour, waveform choice — so they're immediately recognizable without being annoying.

<details>
<summary>For LLMs</summary>

Copy this into any coding agent's custom instructions, hooks, or command wrapper:

```
Install: go install github.com/jgabor/tuta@latest
From source: mage build → ./build/tuta; mage install → $GOBIN/tuta (debug); mage test
Never go build -o tuta . in repo root (see AGENTS.md)
Missing tuta at runtime → exit 0; never block the host.

Long agent shell commands (≥5000ms, excl. approval wait): play one sound on completion.
Wire via the host's native after-command lifecycle (hooks, plugins, wrappers) for shell/terminal tools.
Prefer callbacks that expose exit code and output; if exit code is unavailable, use output-only hooks.
One completion handler per command — no duplicate after-hooks for the same event.

Required data (field names vary by host):
  duration in milliseconds (excl. approval/user wait)
  exit code when available
  stdout and stderr when available, or a single combined output string

Sound selection (first match wins):
  1. Explicit status, result, sound, or notification → map to a sound name below
  2. Timeout flag or output mentions timed out / timeout / watchdog → timeout
  3. Output (stdout or stderr) mentions warn, warning, deprecated, partial, or caution → warning
  4. Exit code 0 → success
  5. Non-zero exit with any output → warning
  6. Non-zero exit, no stdout and no stderr → error
  7. Exit code missing → notify
On command failure/cancellation: timed out → timeout; user or system interrupt/denial → cancel

Lifecycle (optional, separate handlers, no 5s gate):
  Semantic intents — map each row to the closest native event your host exposes; omit unmapped rows.
  turn done → ready (debounce 2s) | aborted → cancel | failed → error
  compact → complete | ask → notify | confirm → confirm | milestone → progress
  increase, decrease, info → invoke tuta manually only

Fallbacks: unknown status → notify; bad sound name → success (warns on stderr); tuta error → retry success (handler still exit 0)
Exit codes: 0 ok, 1 playback/export failure (retry success once), 2 usage error (bad sound name is NOT 2 — it falls back)
Invoke: tuta <sound> (installed) or ./build/tuta <sound> (in-repo)  |  Go: alert.Play("<sound>")
Sounds: success, error, warning, info, complete, increase, decrease, notify, progress, confirm, cancel, ready, timeout
```

</details>

## Install

Pre-built binaries are available on the [releases page](../../releases).

To build from source ([mage](https://magefile.org/) required, Go 1.26+):

```sh
go install github.com/magefile/mage@latest
mage build                    # release → ./build/tuta
mage build -debug=true        # debug → ./build/tuta (includes tuta debug)
DEBUG=1 mage build            # same as -debug=true
mage install                  # debug → $(go env GOBIN)/tuta
```

| Command | Output | `tuta debug` |
| ------- | ------ | ------------ |
| `mage build` | `./build/tuta` | no |
| `mage build -debug=true` | `./build/tuta` | yes |
| `mage install` | `$(go env GOBIN)/tuta` | yes |
| `go install …@latest` | `$(go env GOBIN)/tuta` | no |

Ensure `$(go env GOBIN)` is on your `PATH` (often `~/.local/share/go/bin` with a default Go install).

Do not run `go build -o tuta .` in the repo root — that leaves a stray binary on your PATH. See [AGENTS.md](AGENTS.md).

Or install a tagged release directly with `go install`:

```sh
go install github.com/jgabor/tuta@latest
```

### Development

```sh
mage test
mage debug             # exports tmp/ if needed, then runs every debug check
```

`mage debug` runs via `go run` with the `debug` build tag, so no separate build or install step is required. It auto-exports `tmp/` if no FLAC files are present.

For granular control, build a debug binary and call the CLI directly:

```sh
mage build -debug=true
./build/tuta export -o tmp/
./build/tuta debug sounds all tmp/
```

Or install a debug binary to PATH: `mage install`, then `tuta export …` and `tuta debug sounds all tmp/`.

If `tuta debug` reports it requires a debug build instead of running checks, you have a release binary — rebuild with `-debug=true` or run `mage install`.

## Library

Import the `alert` package to play sounds from Go programs:

```go
import "github.com/jgabor/tuta/alert"

if err := alert.Play("error"); err != nil {
    // handle audio failure
}
```

List built-in sound names with `alert.Names()`.

Render synthesized mono PCM at 44.1 kHz with `alert.Render(name)` or export lossless FLAC with [go-flac](https://github.com/tphakala/go-flac):

```go
if err := alert.ExportFLAC("error.flac", "error", alert.FLACOptions{}); err != nil {
    // handle export failure
}
```

`FLACOptions` defaults to mono 16-bit at compression level 5. Use `Channels: 2` for stereo (L=R, same as playback) or `BitDepth: 24` for higher precision.

Library consumers use the same audio stack as the CLI ([oto](https://github.com/hajimehoshi/oto)); on Linux, building from source typically requires ALSA development headers (`libasound2-dev`), and hearing sounds at runtime requires ALSA. FLAC export is pure Go and does not require ALSA.

While this module is on v0.x, the exported API may evolve in minor releases. A future v2+ breaking change would use a `/v2` import path.

## Usage

In-repo dev: prefix with `./build/tuta`. After `mage install` or `go install`, use `tuta` from PATH.

```sh
tuta [sound]
tuta export [-o DIR] [-mono|-stereo] [-depth 16|24] [sound ...]
tuta debug sounds <cmd> [dir]   # debug builds only (mage build -debug=true or mage install)
tuta --help
tuta --version
```

Available sounds: `success`, `error`, `warning`, `info`, `complete`, `increase`, `decrease`, `notify`, `progress`, `confirm`, `cancel`, `ready`, `timeout`

With no argument, `tuta` plays `success` (the default). An unrecognized sound name is not silent: `tuta` warns on stderr and still plays `success` as a best-effort fallback (exit 0), so agent hooks keep their feedback cue. An unrecognized option (e.g. `-foo`) is a usage error: stderr message plus usage and exit 2.

### Export FLAC

Export synthesized sounds as FLAC files for offline analysis (spectrum, fingerprinting, etc.):

```sh
./build/tuta export -o tmp/              # all sounds → tmp/*.flac
./build/tuta export -o tmp/ success error
./build/tuta export -stereo -depth 24 -o tmp/ success
```

Output defaults to mono 16-bit FLAC at 44.1 kHz. The files contain the same synthesized audio that `tuta <sound>` would play (without speaker capture). `tmp/` is gitignored.

### Debug sounds

Run consistency checks across a directory of FLAC exports. Requires a debug build (`mage build -debug=true` or `mage install`); a release build instead prints an error explaining the debug-build requirement and exits 2 (it does not play `success`).

```sh
./build/tuta debug sounds all tmp/      # run every check
./build/tuta debug sounds volume tmp/   # individual checks: volume, duration, pitch, spectrum, distinct
```

The `all` run reports five sections:

**`volume`** — is anything oddly quiet or loud compared to the rest of the set?

| Column    | Meaning                                                              |
| --------- | -------------------------------------------------------------------- |
| `peak`    | Loudest single sample (dBFS). Closer to 0 = louder, more clip risk.  |
| `rms`     | Average loudness across the file (dBFS) — what your ear perceives.   |
| `crest`   | `peak − rms`. Small = steady tone, large = punchy/spiky.            |
| `Δ rms`   | Drift from the set's median. Sounds within the threshold feel balanced. |

**`duration`** — are sounds within their target length window?

- `duration`: actual length.
- `min` / `max`: acceptable range. Below `min` feels clipped; above `max` feels sluggish for a UI alert.

**`pitch`** — does each sound play the right notes?

- `detected notes`: how many distinct notes the analyzer heard.
- `note N Hz (want X Hz)`: detected frequency vs. target. "ok" means within tolerance — detection rounds to the nearest bin, so values are never exact.
- `contour`: sequence shape (ascending, descending, arpeggio, sweep). `timeout` is a continuous descending sweep, so it's checked at time points (10/30/50/70%/end) rather than as discrete notes.

**`spectrum`** — what *shape* of sound is it?

| Column     | Meaning                                                                         |
| ---------- | ------------------------------------------------------------------------------- |
| `f0`       | Fundamental frequency (the lowest, strongest pitch).                            |
| `fund%`    | Energy in the fundamental vs. harmonics. High = pure, low = rich/buzzy.          |
| `H3/H1`    | 3rd harmonic vs. fundamental. Triangle waves have a small 3rd; pure sines have none. |
| `centroid` | "Brightness" — average frequency weighted by energy. Higher = tinnier.          |
| `class`    | Inferred waveform (sine / triangle / square). Each sound is *supposed* to be a particular class. |

**`distinct`** — are any two sounds too similar to tell apart?

Computes a perceptual distance (mix of pitch, spectrum, and timing features) between every pair. The `Closest pair` line shows the tightest match; the run passes if it stays above the minimum-distance threshold (0.12). Pairs that drop below the floor are hard for users to tell apart.

A run ends with `PASS: all checks passed.` when every section is green.

## Exit codes

`tuta` uses a small, stable exit code scheme so callers (shells, CI, agent hooks) can react consistently:

| Code | Meaning                                                                                          |
| ---- | ------------------------------------------------------------------------------------------------ |
| 0    | Success: a sound played, help/version was shown, or export/checks completed.                    |
| 1    | Runtime failure: audio playback or FLAC export could not complete. Retry with `success` if the host needs a cue. |
| 2    | Usage error: unknown option/command, bad flag, missing value, or `tuta debug` from a release build (which lacks `debug`). |

### Best-effort playback

`tuta` is designed as a fire-and-forget cue for scripts, build pipelines, and agent hosts: it should never silently swallow intent or block the host.

- **Missing binary**: the host's hook aborts with exit 0 — never block on tuta.
- **No argument**: plays `success` (the default), exit 0.
- **Unknown sound** (e.g. a typo): warns on stderr naming the requested sound, then plays `success` as a fallback so the host still gets a feedback cue, exit 0.
- **Trailing arguments or options** (e.g. `tuta success extra`, `tuta success --bogus`): the play path takes exactly one sound — extras are a usage error, message + usage to stderr, exit 2. `--help`/`--version` likewise take no arguments.
- **Unknown option** (e.g. `-foo`): treated as a usage error, not a sound — message + usage to stderr, exit 2.
- **Playback failure**: prints the requested sound name and error to stderr, exit 1. Hosts that need a cue may retry once with `success`.
- **`tuta debug` in a release build**: explains that a debug build is required and how to get one, exit 2 (so CI never mistakes a release build for passing checks).

## Sounds

| Sound    | Character                                      | Waveform |
| -------- | ---------------------------------------------- | -------- |
| success  | ascending C major arpeggio                     | sine     |
| error    | descending tritone buzz (D3 → Ab2)             | square   |
| warning  | three pings with major second (C6 → D6)        | triangle |
| info     | short neutral blip at C5                       | sine     |
| complete | ascending F major triad (F4 → A4 → C5)         | triangle |
| increase | ascending major triad (C4 → E4 → G4)           | sine     |
| decrease | descending minor triad, fading (G4 → Eb4 → C4) | triangle |
| notify   | ascending minor third ping (A5 → C6)           | sine     |
| progress | ascending major triad (E4 → G4 → B4)           | triangle |
| confirm  | ascending perfect fifth (C5 → G5)              | sine     |
| cancel   | single tone (B4)                               | triangle |
| ready    | ascending major third (C5 → E5)                | triangle |
| timeout  | descending frequency sweep (E4 → Bb3)          | triangle |

---

## Sound design guide

### What makes a notification sound work

Every sound in tuta is built from the same small set of parameters: frequency, duration, waveform, and volume. The art is in combining them to match the emotional signal you want to send.

### 1. Interval quality → emotional valence

The relationship between notes determines whether a sound feels positive, negative, or neutral. Intervals with simple frequency ratios are consonant (pleasant, resolved); complex ratios are dissonant (tense, alarming).

| Interval      | Ratio | Character           | Use                 |
| ------------- | ----- | ------------------- | ------------------- |
| Perfect fifth | 3:2   | open, stable        | calm completion     |
| Major third   | 5:4   | bright, happy       | success             |
| Major triad   | —     | resolved, uplifting | strong positive cue |
| Major second  | 9:8   | mild tension        | warning             |
| Minor third   | 6:5   | melancholic         | soft alert          |
| Tritone       | 45:32 | maximum dissonance  | critical error      |

The `success` sound plays **C5 → E5 → G5**, a C major arpeggio. The intervals are a major third and a perfect fifth — both highly consonant, which is why it reads immediately as positive.

### 2. Contour → direction

- **Ascending** pitch signals completion, alertness, uplift
- **Descending** pitch signals failure, winding down, negativity
- **Flat / single tone** is neutral and informational

The `error` sound descends through a tritone from D3 to Ab2. The `complete` sound ascends F4 → A4 → C5 with a sustained final note.

### 3. Waveform → timbre

| Waveform | Character                        | Best for            |
| -------- | -------------------------------- | ------------------- |
| Sine     | pure, soft, no harmonics         | gentle / non-urgent |
| Triangle | warm, mild harmonics             | calm / ambient      |
| Square   | buzzy, harsh, cuts through noise | urgent / error      |

Square waves contain strong odd harmonics, which is why they feel sharp and attention-grabbing. Sine waves are the opposite — pure and unobtrusive.

### 4. Rhythm → character

- **Short notes** (≤ 0.1s): crisp, punchy
- **Longer final note**: sense of resolution and landing
- **Gaps between notes** (currently 20ms): separation and clarity

### 5. Dynamics → emphasis

Increasing the volume on the final note (as `success` does: 0.2 → 0.35) creates a crescendo that reinforces the sense of resolution. Flat volume across notes feels more mechanical.

### 6. Decay rate → texture

All tones use exponential decay: `exp(-3 * t / duration)`. Adjusting the constant changes texture:

| Constant | Feel                   |
| -------- | ---------------------- |
| 1–2      | sustained, organ-like  |
| 3        | bell / pluck (default) |
| 6+       | percussive, staccato   |

### Adding a new sound

1. Choose the emotional signal (positive? urgent? neutral?)
2. Pick a root note and interval(s) to match
3. Decide on direction (ascending or descending)
4. Choose a waveform appropriate to the urgency
5. Set durations — make the final note slightly longer for resolution
6. Nudge the final volume up slightly for emphasis

Example — a soft "thinking" pulse:

```go
"thinking": {
    {440.00, 0.08, "triangle", 0.12},  // A4
    {440.00, 0.08, "triangle", 0.12},  // A4 repeated
},
```

A repeated flat tone reads as "in progress" rather than resolved.

## License

MIT

## Author

Jonathan Gabor ([@jgabor](https://github.com/jgabor))
