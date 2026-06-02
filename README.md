# tuta: tiny utility for tone alerts

A tiny CLI tool that plays synthesized notification tones for scripts and build pipelines. Each sound is designed around music theory principles — interval quality, pitch contour, waveform choice — so they're immediately recognizable without being annoying.

<details>
<summary>For LLMs</summary>

Copy this into any coding agent's custom instructions, hooks, or command wrapper:

```
Install: go install github.com/jgabor/tuta@latest
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

Fallbacks: unknown status → notify; bad sound name → success; tuta error → retry success (handler still exit 0)
Invoke: tuta <sound>  |  Go: alert.Play("<sound>")
Sounds: success, error, warning, info, complete, increase, decrease, notify, progress, confirm, cancel, ready, timeout
```

</details>

## Install

Pre-built binaries are available on the [releases page](../../releases).

To build and install from source:

```sh
go build -o tuta .
sudo mv tuta /usr/local/bin/
```

Or install directly with `go install`:

```sh
go install github.com/jgabor/tuta@latest
```

## Library

Import the `alert` package to play sounds from Go programs:

```go
import "github.com/jgabor/tuta/alert"

if err := alert.Play("error"); err != nil {
    // handle audio failure
}
```

List built-in sound names with `alert.Names()`.

Library consumers use the same audio stack as the CLI ([oto](https://github.com/hajimehoshi/oto)); on Linux, building typically requires ALSA development headers (`libasound2-dev`).

While this module is on v0.x, the exported API may evolve in minor releases. A future v2+ breaking change would use a `/v2` import path.

## Usage

```sh
tuta [sound]
tuta --help
tuta --version
```

Available sounds: `success`, `error`, `warning`, `info`, `complete`, `increase`, `decrease`, `notify`, `progress`, `confirm`, `cancel`, `ready`, `timeout`

Defaults to `success` if no argument is given or the argument is unrecognized.

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
