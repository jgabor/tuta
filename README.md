# tuta — Tiny Utility for Tone Alerts

A tiny CLI tool that plays synthesized notification tones for scripts and build pipelines. Each sound is designed around music theory principles — interval quality, pitch contour, waveform choice — so they're immediately recognizable without being annoying.

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

## Usage

```sh
tuta [sound]
tuta --help
tuta --version
```

Available sounds: `success`, `error`, `warning`, `info`, `complete`, `increase`, `decrease`, `notify`, `progress`, `confirm`, `cancel`, `ready`, `timeout`

Defaults to `success` if no argument is given or the argument is unrecognized.

## Sounds

| Sound    | Character                      | Waveform |
| -------- | ------------------------------ | -------- |
| success  | ascending C major arpeggio     | sine     |
| error    | low descending buzz (D3 → A2)  | square   |
| warning  | two radar-style pings at C6    | sine     |
| info     | single neutral tone            | sine     |
| complete | ascending triad (B4 → D5 → G5) | triangle |
| increase | ascending major triad (C4 → E4 → G4) | sine     |
| decrease | descending minor triad (G4 → Eb4 → C4) | sine     |
| notify   | two bright pings at A5         | sine     |
| progress | three even pulses at E4        | triangle |
| confirm  | ascending perfect fifth (C5 → G5) | sine     |
| cancel   | descending minor second (B4 → Bb3) | triangle |
| ready    | single sustained tone at C5    | sine     |
| timeout  | descending square buzz (E4 → C4) | square   |

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

The `error` sound descends from D3 to A2. The `complete` sound ascends B4 → D5 → G5 with a sustained final note.

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
