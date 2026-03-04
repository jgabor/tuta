# tuta — Tiny Utility for Tone Alerts

A minimal CLI notification sound player. Plays short synthesized tones for use in scripts, build pipelines, or anywhere you want audible feedback.

## Install

Pre-built binaries are available on the [releases page](../../releases).

To build and install from source:

```sh
go build -o tuta .
sudo mv tuta /usr/local/bin/
```

## Usage

```sh
tuta [sound]
tuta --version
```

Available sounds: `success`, `error`, `warning`, `info`, `complete`

Defaults to `success` if no argument is given or the argument is unrecognized.

## Sounds

| Sound     | Character                          | Waveform  |
|-----------|------------------------------------|-----------|
| success   | ascending C major arpeggio         | sine      |
| error     | descending dissonant interval      | square    |
| warning   | unresolved major second            | triangle  |
| info      | single neutral tone                | sine      |
| complete  | ascending perfect fifth, sustained | triangle  |

---

## Sound design guide

### What makes a notification sound work

Every sound in tuta is built from the same small set of parameters: frequency, duration, waveform, and volume. The art is in combining them to match the emotional signal you want to send.

### 1. Interval quality → emotional valence

The relationship between notes determines whether a sound feels positive, negative, or neutral. Intervals with simple frequency ratios are consonant (pleasant, resolved); complex ratios are dissonant (tense, alarming).

| Interval       | Ratio | Character           | Use                  |
|----------------|-------|---------------------|----------------------|
| Perfect fifth  | 3:2   | open, stable        | calm completion      |
| Major third    | 5:4   | bright, happy       | success              |
| Major triad    | —     | resolved, uplifting | strong positive cue  |
| Major second   | 9:8   | mild tension        | warning              |
| Minor third    | 6:5   | melancholic         | soft alert           |
| Tritone        | 45:32 | maximum dissonance  | critical error       |

The `success` sound plays **C5 → E5 → G5**, a C major arpeggio. The intervals are a major third and a perfect fifth — both highly consonant, which is why it reads immediately as positive.

### 2. Contour → direction

- **Ascending** pitch signals completion, alertness, uplift
- **Descending** pitch signals failure, winding down, negativity
- **Flat / single tone** is neutral and informational

The `error` sound descends from G4 to E4. The `complete` sound ascends a perfect fifth (G4 → D5).

### 3. Waveform → timbre

| Waveform | Character                        | Best for              |
|----------|----------------------------------|-----------------------|
| Sine     | pure, soft, no harmonics         | gentle / non-urgent   |
| Triangle | warm, mild harmonics             | calm / ambient        |
| Square   | buzzy, harsh, cuts through noise | urgent / error        |

Square waves contain strong odd harmonics, which is why they feel sharp and attention-grabbing. Sine waves are the opposite — pure and unobtrusive.

### 4. Rhythm → character

- **Short notes** (≤ 0.1s): crisp, punchy
- **Longer final note**: sense of resolution and landing
- **Gaps between notes** (currently 20ms): separation and clarity

### 5. Dynamics → emphasis

Increasing the volume on the final note (as `success` does: 0.2 → 0.25) creates a crescendo that reinforces the sense of resolution. Flat volume across notes feels more mechanical.

### 6. Decay rate → texture

All tones use exponential decay: `exp(-3 * t / duration)`. Adjusting the constant changes texture:

| Constant | Feel                     |
|----------|--------------------------|
| 1–2      | sustained, organ-like    |
| 3        | bell / pluck (default)   |
| 6+       | percussive, staccato     |

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
