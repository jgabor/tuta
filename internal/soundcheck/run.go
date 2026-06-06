//go:build debug

// Package soundcheck validates tuta FLAC exports (volume, duration, pitch, spectrum, distinctiveness).
// It is only compiled when building tuta with -tags debug.
package soundcheck

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tphakala/go-flac/pcm"
)

const (
	fullScale16         = 32768.0
	sampleRateHz        = 44100 // matches alert.SampleRate()
	minDistinctDistance = 0.12
)

type contour int

const (
	contourAny contour = iota
	contourAscending
	contourDescending
	contourFlat
	contourSweepDown
)

type track struct {
	name    string
	path    string
	samples []float64
}

type segment struct {
	start int
	end   int
}

type soundSpec struct {
	name          string
	noteCount     int
	frequencies   []float64
	freqTolerance float64
	contour       contour
	waveform      string
	minDuration   float64
	maxDuration   float64
}

type spectralProfile struct {
	fundamentalHz  float64
	fundamentalPct float64
	h3Ratio        float64
	centroidHz     float64
	classification string
}

type soundFeatures struct {
	name string
	vec  []float64
}

var soundSpecs = []soundSpec{
	{name: "success", noteCount: 3, frequencies: []float64{523.25, 659.25, 783.99}, freqTolerance: 0.03, contour: contourAscending, waveform: "sine", minDuration: 0.32, maxDuration: 0.42},
	{name: "error", noteCount: 2, frequencies: []float64{146.83, 103.83}, freqTolerance: 0.03, contour: contourDescending, waveform: "square", minDuration: 0.40, maxDuration: 0.52},
	{name: "warning", noteCount: 3, frequencies: []float64{1046.50, 1174.66, 1046.50}, freqTolerance: 0.03, contour: contourAny, waveform: "triangle", minDuration: 0.38, maxDuration: 0.50},
	{name: "info", noteCount: 1, frequencies: []float64{523.25}, freqTolerance: 0.03, contour: contourFlat, waveform: "sine", minDuration: 0.08, maxDuration: 0.16},
	{name: "complete", noteCount: 3, frequencies: []float64{349.23, 440.00, 523.25}, freqTolerance: 0.03, contour: contourAscending, waveform: "triangle", minDuration: 0.58, maxDuration: 0.74},
	{name: "increase", noteCount: 3, frequencies: []float64{261.63, 329.63, 392.00}, freqTolerance: 0.03, contour: contourAscending, waveform: "sine", minDuration: 0.32, maxDuration: 0.42},
	{name: "decrease", noteCount: 3, frequencies: []float64{392.00, 311.13, 261.63}, freqTolerance: 0.035, contour: contourDescending, waveform: "triangle", minDuration: 0.32, maxDuration: 0.42},
	{name: "notify", noteCount: 2, frequencies: []float64{880.00, 1046.50}, freqTolerance: 0.03, contour: contourAscending, waveform: "sine", minDuration: 0.26, maxDuration: 0.38},
	{name: "progress", noteCount: 3, frequencies: []float64{329.63, 392.00, 493.88}, freqTolerance: 0.03, contour: contourAscending, waveform: "triangle", minDuration: 0.26, maxDuration: 0.38},
	{name: "confirm", noteCount: 2, frequencies: []float64{523.25, 783.99}, freqTolerance: 0.03, contour: contourAscending, waveform: "sine", minDuration: 0.40, maxDuration: 0.54},
	{name: "cancel", noteCount: 1, frequencies: []float64{493.88}, freqTolerance: 0.03, contour: contourFlat, waveform: "triangle", minDuration: 0.14, maxDuration: 0.26},
	{name: "ready", noteCount: 2, frequencies: []float64{523.25, 659.25}, freqTolerance: 0.03, contour: contourAscending, waveform: "triangle", minDuration: 0.24, maxDuration: 0.38},
	{name: "timeout", noteCount: 0, frequencies: []float64{329.63, 233.08}, freqTolerance: 0.08, contour: contourSweepDown, waveform: "triangle", minDuration: 0.46, maxDuration: 0.54},
}

// Usage documents the sounds check subcommands.
func Usage() string {
	return `tuta debug sounds — validate exported FLAC files

Usage:
  tuta debug sounds <command> [options] [dir]

Commands:
  volume     RMS/peak loudness consistency (default threshold: 3 dB)
  duration   total length within expected bounds
  pitch      note count, frequencies, and contour
  spectrum   waveform class (sine / triangle / square)
  distinct   pairwise feature distance (sounds must not blur together)
  all        run every check

Options:
  -t, --threshold N   volume outlier threshold in dB (volume/all only)
  -h, --help          show this help

Default directory: sounds/

Examples:
  tuta debug sounds all
  tuta debug sounds volume -t 2 sounds/
  tuta debug sounds pitch sounds/
`
}

// Run executes a sound-check subcommand. Returns a process exit code.
func Run(args []string) int {
	if len(args) == 0 {
		fmt.Print(Usage())
		return 2
	}

	cmd := args[0]
	args = args[1:]

	dir := "sounds"
	threshold := 3.0

	switch cmd {
	case "-h", "--help", "help":
		fmt.Print(Usage())
		return 0
	case "volume":
		for len(args) > 0 {
			switch args[0] {
			case "-t", "--threshold":
				if len(args) < 2 {
					return fail("missing value for %s", args[0])
				}
				if _, err := fmt.Sscanf(args[1], "%f", &threshold); err != nil || threshold < 0 {
					return fail("invalid threshold %q", args[1])
				}
				args = args[2:]
			default:
				dir = args[0]
				args = args[1:]
			}
		}
		tracks, err := loadTracks(dir)
		if err != nil {
			return fail("%v", err)
		}
		return runVolume(tracks, threshold)
	case "duration", "pitch", "spectrum", "distinct":
		if len(args) > 0 {
			dir = args[0]
		}
		tracks, err := loadTracks(dir)
		if err != nil {
			return fail("%v", err)
		}
		switch cmd {
		case "duration":
			return runDuration(tracks)
		case "pitch":
			return runPitch(tracks)
		case "spectrum":
			return runSpectrum(tracks)
		case "distinct":
			return runDistinct(tracks)
		}
	case "all":
		for len(args) > 0 {
			switch args[0] {
			case "-t", "--threshold":
				if len(args) < 2 {
					return fail("missing value for %s", args[0])
				}
				if _, err := fmt.Sscanf(args[1], "%f", &threshold); err != nil || threshold < 0 {
					return fail("invalid threshold %q", args[1])
				}
				args = args[2:]
			default:
				dir = args[0]
				args = args[1:]
			}
		}
		tracks, err := loadTracks(dir)
		if err != nil {
			return fail("%v", err)
		}
		checks := []struct {
			name string
			fn   func([]track) int
		}{
			{"volume", func(t []track) int { return runVolume(t, threshold) }},
			{"duration", runDuration},
			{"pitch", runPitch},
			{"spectrum", runSpectrum},
			{"distinct", runDistinct},
		}
		failed := 0
		for _, c := range checks {
			fmt.Printf("=== %s ===\n", c.name)
			if code := c.fn(tracks); code != 0 {
				failed++
			}
			fmt.Println()
		}
		if failed > 0 {
			fmt.Printf("FAIL: %d of %d checks failed\n", failed, len(checks))
			return 1
		}
		fmt.Println("PASS: all checks passed.")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		fmt.Print(Usage())
		return 2
	}
	return 0
}

func fail(format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "soundcheck: "+format+"\n", args...)
	return 2
}

func specByName(name string) (soundSpec, bool) {
	for _, s := range soundSpecs {
		if s.name == name {
			return s, true
		}
	}
	return soundSpec{}, false
}

func loadTracks(dir string) ([]track, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.flac"))
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no .flac files in %s (run: tuta export -o %s)", dir, dir)
	}
	sort.Strings(entries)

	var tracks []track
	for _, path := range entries {
		tr, err := loadTrack(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		tracks = append(tracks, tr)
	}
	return tracks, nil
}

func loadTrack(path string) (track, error) {
	f, err := os.Open(path)
	if err != nil {
		return track{}, err
	}
	defer f.Close()

	dec, err := pcm.NewDecoder(f)
	if err != nil {
		return track{}, err
	}

	raw, err := io.ReadAll(dec)
	if err != nil {
		return track{}, err
	}

	info := dec.Info()
	samples, err := decodePCM(raw, info.BitDepth, info.Channels)
	if err != nil {
		return track{}, err
	}
	if len(samples) == 0 {
		return track{}, fmt.Errorf("no samples")
	}

	name := strings.TrimSuffix(filepath.Base(path), ".flac")
	return track{name: name, path: path, samples: samples}, nil
}

func decodePCM(raw []byte, bitDepth, channels int) ([]float64, error) {
	if channels < 1 {
		return nil, fmt.Errorf("invalid channel count %d", channels)
	}

	frameSize := bitDepth / 8 * channels
	if frameSize == 0 || len(raw)%frameSize != 0 {
		return nil, fmt.Errorf("unexpected PCM size for %d-bit %dch", bitDepth, channels)
	}

	frames := len(raw) / frameSize
	out := make([]float64, frames)
	for i := 0; i < frames; i++ {
		var sum float64
		for ch := 0; ch < channels; ch++ {
			off := i*frameSize + ch*(bitDepth/8)
			v, err := readSample(raw[off:], bitDepth)
			if err != nil {
				return nil, err
			}
			sum += v
		}
		out[i] = sum / float64(channels)
	}
	return out, nil
}

func readSample(b []byte, bitDepth int) (float64, error) {
	switch bitDepth {
	case 16:
		if len(b) < 2 {
			return 0, fmt.Errorf("short 16-bit sample")
		}
		v := int16(uint16(b[0]) | uint16(b[1])<<8)
		return float64(v) / fullScale16, nil
	case 24:
		if len(b) < 3 {
			return 0, fmt.Errorf("short 24-bit sample")
		}
		v := int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16)
		if v&0x800000 != 0 {
			v |= ^0xffffff
		}
		return float64(v) / 8388608.0, nil
	default:
		return 0, fmt.Errorf("unsupported bit depth %d", bitDepth)
	}
}

func measure(samples []float64) (peak, rms float64) {
	var sumSq float64
	for _, s := range samples {
		a := math.Abs(s)
		if a > peak {
			peak = a
		}
		sumSq += s * s
	}
	if len(samples) == 0 {
		return 0, 0
	}
	rms = math.Sqrt(sumSq / float64(len(samples)))
	return peak, rms
}

func linearToDBFS(v float64) float64 {
	if v <= 0 {
		return -120
	}
	return 20 * math.Log10(v)
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func durationSec(samples []float64) float64 {
	return float64(len(samples)) / sampleRateHz
}

func findSegments(samples []float64) []segment {
	const win = sampleRateHz / 100
	if len(samples) < win {
		return nil
	}

	var rms []float64
	maxRMS := 0.0
	for i := 0; i+win <= len(samples); i += win / 2 {
		_, r := measure(samples[i : i+win])
		rms = append(rms, r)
		if r > maxRMS {
			maxRMS = r
		}
	}
	if maxRMS == 0 {
		return nil
	}

	threshold := maxRMS * 0.08
	var segs []segment
	inSeg := false
	startIdx := 0
	for i, r := range rms {
		sampleStart := i * (win / 2)
		if r >= threshold {
			if !inSeg {
				inSeg = true
				startIdx = sampleStart
			}
		} else if inSeg {
			end := sampleStart
			if end-startIdx >= sampleRateHz/33 {
				segs = append(segs, segment{start: startIdx, end: end})
			}
			inSeg = false
		}
	}
	if inSeg {
		end := len(samples)
		if end-startIdx >= sampleRateHz/33 {
			segs = append(segs, segment{start: startIdx, end: end})
		}
	}
	return segs
}

func estimatePitch(samples []float64) float64 {
	if len(samples) < sampleRateHz/50 {
		return 0
	}

	start := len(samples) / 5
	end := len(samples) - len(samples)/5
	if end <= start {
		start = 0
		end = len(samples)
	}
	seg := samples[start:end]

	n := nextPow2(len(seg))
	if n < 2048 {
		n = 2048
	}
	if n > 8192 {
		n = 8192
	}
	windowed := make([]float64, n)
	copyLen := min(len(seg), n)
	copy(windowed, seg[:copyLen])
	for i := 0; i < copyLen; i++ {
		windowed[i] *= hann(i, copyLen)
	}

	mag := fftMagnitude(windowed)
	binHz := float64(sampleRateHz) / float64(n)
	return peakFrequency(mag, binHz, 80, 2500)
}

func estimatePitchAt(samples []float64, startPct, endPct float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	start := int(float64(len(samples)) * startPct)
	end := int(float64(len(samples)) * endPct)
	if end <= start {
		end = start + 1
	}
	if end > len(samples) {
		end = len(samples)
	}
	return estimatePitch(samples[start:end])
}

func analyzeSpectrum(samples []float64) spectralProfile {
	if len(samples) == 0 {
		return spectralProfile{}
	}

	n := nextPow2(len(samples))
	if n > 16384 {
		n = 16384
	}
	copyLen := min(len(samples), n)
	windowed := make([]float64, n)
	copy(windowed, samples[:copyLen])
	for i := 0; i < copyLen; i++ {
		windowed[i] *= hann(i, copyLen)
	}

	mag := fftMagnitude(windowed)
	binHz := float64(sampleRateHz) / float64(n)

	f0 := peakFrequency(mag, binHz, 70, 2500)
	if f0 == 0 {
		return spectralProfile{classification: "unknown"}
	}

	total := 0.0
	fundamental := bandEnergy(mag, binHz, f0, 0.03)
	for i := 1; i < len(mag)/2; i++ {
		total += mag[i] * mag[i]
	}

	h3 := bandEnergy(mag, binHz, f0*3, 0.03)
	h3Ratio := 0.0
	if fundamental > 0 {
		h3Ratio = h3 / fundamental
	}

	fundPct := 0.0
	if total > 0 {
		fundPct = fundamental / total
	}

	return spectralProfile{
		fundamentalHz:  f0,
		fundamentalPct: fundPct,
		h3Ratio:        h3Ratio,
		centroidHz:     spectralCentroid(mag, binHz),
		classification: classifyWaveform(h3Ratio),
	}
}

func classifyWaveform(h3Ratio float64) string {
	switch {
	case h3Ratio < 0.008:
		return "sine"
	case h3Ratio < 0.10:
		return "triangle"
	default:
		return "square"
	}
}

func attackPortion(samples []float64) []float64 {
	n := sampleRateHz / 25
	if n > len(samples) {
		return samples
	}
	return samples[:n]
}

func hann(i, n int) float64 {
	if n <= 1 {
		return 1
	}
	return 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n-1))
}

func nextPow2(n int) int {
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

func fftMagnitude(samples []float64) []float64 {
	n := len(samples)
	re := make([]float64, n)
	im := make([]float64, n)
	copy(re, samples)
	fft(re, im)

	mag := make([]float64, n/2+1)
	for i := range mag {
		mag[i] = math.Hypot(re[i], im[i])
	}
	return mag
}

func fft(re, im []float64) {
	n := len(re)
	if n <= 1 {
		return
	}

	for i, j := 0, 0; i < n; i++ {
		if i < j {
			re[i], re[j] = re[j], re[i]
			im[i], im[j] = im[j], im[i]
		}
		m := n >> 1
		for ; m >= 1 && j >= m; m >>= 1 {
			j -= m
		}
		j += m
	}

	for size := 2; size <= n; size <<= 1 {
		angle := -2 * math.Pi / float64(size)
		wRe := math.Cos(angle)
		wIm := math.Sin(angle)
		for start := 0; start < n; start += size {
			curRe, curIm := 1.0, 0.0
			half := size / 2
			for k := 0; k < half; k++ {
				i := start + k
				j := start + k + half
				tRe := curRe*re[j] - curIm*im[j]
				tIm := curRe*im[j] + curIm*re[j]
				re[j] = re[i] - tRe
				im[j] = im[i] - tIm
				re[i] += tRe
				im[i] += tIm
				nextRe := curRe*wRe - curIm*wIm
				curIm = curRe*wIm + curIm*wRe
				curRe = nextRe
			}
		}
	}
}

func peakFrequency(mag []float64, binHz, minHz, maxHz float64) float64 {
	minBin := int(minHz / binHz)
	maxBin := int(maxHz / binHz)
	if maxBin >= len(mag) {
		maxBin = len(mag) - 1
	}
	if minBin < 1 {
		minBin = 1
	}

	bestBin := minBin
	best := 0.0
	for i := minBin; i <= maxBin; i++ {
		if mag[i] > best {
			best = mag[i]
			bestBin = i
		}
	}
	return float64(bestBin) * binHz
}

func bandEnergy(mag []float64, binHz, centerHz, pctWidth float64) float64 {
	width := centerHz * pctWidth
	lo := int((centerHz - width) / binHz)
	hi := int((centerHz + width) / binHz)
	if lo < 0 {
		lo = 0
	}
	if hi >= len(mag) {
		hi = len(mag) - 1
	}
	sum := 0.0
	for i := lo; i <= hi; i++ {
		sum += mag[i] * mag[i]
	}
	return sum
}

func spectralCentroid(mag []float64, binHz float64) float64 {
	var weighted, total float64
	for i, m := range mag {
		power := m * m
		f := float64(i) * binHz
		weighted += f * power
		total += power
	}
	if total == 0 {
		return 0
	}
	return weighted / total
}

func pitchContourOK(c contour, pitches []float64) bool {
	if len(pitches) < 2 {
		return c == contourFlat || c == contourAny || len(pitches) <= 1
	}
	asc, desc := 0, 0
	for i := 1; i < len(pitches); i++ {
		diff := pitches[i] - pitches[i-1]
		if math.Abs(diff) < 5 {
			continue
		}
		if diff > 0 {
			asc++
		} else {
			desc++
		}
	}
	switch c {
	case contourAscending:
		return asc > 0 && desc == 0
	case contourDescending:
		return desc > 0 && asc == 0
	case contourFlat:
		return asc == 0 && desc == 0
	case contourAny:
		return true
	default:
		return true
	}
}

func freqMatch(got, want, tolerance float64) bool {
	if want == 0 {
		return got == 0
	}
	return math.Abs(got-want)/want <= tolerance
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func runVolume(tracks []track, threshold float64) int {
	rmsValues := make([]float64, len(tracks))
	for i, tr := range tracks {
		_, rms := measure(tr.samples)
		rmsValues[i] = linearToDBFS(rms)
	}
	medianRMS := median(rmsValues)

	fmt.Printf("Volume (%d files, median RMS %.1f dBFS, threshold %.1f dB)\n\n", len(tracks), medianRMS, threshold)
	fmt.Printf("%-12s %8s %8s %8s %8s\n", "sound", "peak", "rms", "crest", "Δ rms")
	fmt.Printf("%-12s %8s %8s %8s %8s\n", "----", "----", "---", "-----", "-----")

	var outliers []string
	minRMS, maxRMS := math.Inf(1), math.Inf(-1)
	for _, tr := range tracks {
		peak, rms := measure(tr.samples)
		peakDB := linearToDBFS(peak)
		rmsDB := linearToDBFS(rms)
		crest := peakDB - rmsDB
		delta := rmsDB - medianRMS
		flag := ""
		if math.Abs(delta) > threshold {
			flag = " *"
			outliers = append(outliers, tr.name)
		}
		fmt.Printf("%-12s %7.1f %7.1f %7.1f %7.1f%s\n", tr.name, peakDB, rmsDB, crest, delta, flag)
		if rmsDB < minRMS {
			minRMS = rmsDB
		}
		if rmsDB > maxRMS {
			maxRMS = rmsDB
		}
	}

	fmt.Printf("\nRMS spread: %.1f dB (%.1f to %.1f dBFS)\n", maxRMS-minRMS, minRMS, maxRMS)
	if len(outliers) > 0 {
		fmt.Printf("FAIL: outliers (>%.1f dB from median): %s\n", threshold, strings.Join(outliers, ", "))
		return 1
	}
	fmt.Println("PASS: all sounds within volume threshold.")
	return 0
}

func runDuration(tracks []track) int {
	fmt.Printf("Duration (%d files)\n\n", len(tracks))
	fmt.Printf("%-12s %10s %10s %10s\n", "sound", "duration", "min", "max")
	fmt.Printf("%-12s %10s %10s %10s\n", "----", "--------", "---", "---")

	var failed []string
	for _, tr := range tracks {
		spec, ok := specByName(tr.name)
		if !ok {
			failed = append(failed, tr.name+": no spec")
			continue
		}
		dur := durationSec(tr.samples)
		flag := ""
		if dur < spec.minDuration || dur > spec.maxDuration {
			flag = " *"
			failed = append(failed, fmt.Sprintf("%s (%.3fs, want %.2f-%.2fs)", tr.name, dur, spec.minDuration, spec.maxDuration))
		}
		fmt.Printf("%-12s %9.3fs %9.2fs %9.2fs%s\n", tr.name, dur, spec.minDuration, spec.maxDuration, flag)
	}

	if len(failed) > 0 {
		fmt.Printf("\nFAIL: %s\n", strings.Join(failed, "; "))
		return 1
	}
	fmt.Println("\nPASS: all durations within expected bounds.")
	return 0
}

func runPitch(tracks []track) int {
	fmt.Printf("Pitch fingerprint (%d files)\n\n", len(tracks))

	var failed []string
	for _, tr := range tracks {
		spec, ok := specByName(tr.name)
		if !ok {
			failed = append(failed, tr.name+": no spec")
			continue
		}

		fmt.Printf("%s:\n", tr.name)
		if spec.contour == contourSweepDown {
			if fail := checkSweep(tr, spec); fail != "" {
				failed = append(failed, fail)
				fmt.Printf("  FAIL: %s\n", fail)
			} else {
				fmt.Println("  PASS: descending sweep")
			}
			continue
		}

		segs := findSegments(tr.samples)
		var pitches []float64
		for _, seg := range segs {
			p := estimatePitch(tr.samples[seg.start:seg.end])
			if p > 0 {
				pitches = append(pitches, p)
			}
		}

		fmt.Printf("  detected notes: %d (want %d)\n", len(pitches), spec.noteCount)
		if len(pitches) != spec.noteCount {
			failed = append(failed, fmt.Sprintf("%s: note count %d, want %d", tr.name, len(pitches), spec.noteCount))
		}

		for i := 0; i < len(pitches) && i < len(spec.frequencies); i++ {
			ok := freqMatch(pitches[i], spec.frequencies[i], spec.freqTolerance)
			mark := "ok"
			if !ok {
				mark = "FAIL"
				failed = append(failed, fmt.Sprintf("%s: note %d %.1f Hz, want %.1f Hz", tr.name, i+1, pitches[i], spec.frequencies[i]))
			}
			fmt.Printf("  note %d: %.1f Hz (want %.1f Hz) %s\n", i+1, pitches[i], spec.frequencies[i], mark)
		}

		if !pitchContourOK(spec.contour, pitches) {
			failed = append(failed, fmt.Sprintf("%s: contour mismatch", tr.name))
			fmt.Printf("  FAIL: contour\n")
		} else {
			fmt.Printf("  contour: ok\n")
		}
		fmt.Println()
	}

	if len(failed) > 0 {
		fmt.Printf("FAIL: %s\n", strings.Join(failed, "; "))
		return 1
	}
	fmt.Println("PASS: all pitch fingerprints match.")
	return 0
}

func checkSweep(tr track, spec soundSpec) string {
	startWant, endWant := spec.frequencies[0], spec.frequencies[1]

	const startPct = 0.10
	startWantAt := startWant + (endWant-startWant)*startPct
	startPitch := estimatePitchAt(tr.samples, startPct-0.03, startPct+0.03)
	if startPitch == 0 {
		return tr.name + ": could not estimate sweep start"
	}
	if !freqMatch(startPitch, startWantAt, spec.freqTolerance) {
		return fmt.Sprintf("%s: sweep at %.0f%% %.1f Hz, want %.1f Hz", tr.name, startPct*100, startPitch, startWantAt)
	}
	fmt.Printf("  at %.0f%%: %.1f Hz (want %.1f Hz) ok\n", startPct*100, startPitch, startWantAt)

	checkpoints := []float64{0.30, 0.50, 0.70}
	var pitches []float64
	prev := startPitch
	for _, pct := range checkpoints {
		got := estimatePitchAt(tr.samples, pct-0.04, pct+0.04)
		if got == 0 {
			return tr.name + ": could not estimate sweep pitch"
		}
		if got >= prev {
			return fmt.Sprintf("%s: sweep not descending at %.0f%% (%.1f Hz after %.1f Hz)", tr.name, pct*100, got, prev)
		}
		pitches = append(pitches, got)
		fmt.Printf("  at %.0f%%: %.1f Hz (descending) ok\n", pct*100, got)
		prev = got
	}

	last := pitches[len(pitches)-1]
	if last < endWant*(1-spec.freqTolerance) {
		fmt.Printf("  end region: %.1f Hz (target %.1f Hz) ok\n", last, endWant)
	} else {
		fmt.Printf("  end region: %.1f Hz still above target %.1f Hz (expected with decay)\n", last, endWant)
	}
	return ""
}

func runSpectrum(tracks []track) int {
	fmt.Printf("Spectrum / waveform class (%d files)\n\n", len(tracks))
	fmt.Printf("%-12s %8s %8s %8s %10s %8s\n", "sound", "f0", "fund%", "H3/H1", "centroid", "class")
	fmt.Printf("%-12s %8s %8s %8s %10s %8s\n", "----", "--", "-----", "-----", "--------", "-----")

	var failed []string
	for _, tr := range tracks {
		spec, ok := specByName(tr.name)
		if !ok {
			failed = append(failed, tr.name+": no spec")
			continue
		}

		segs := findSegments(tr.samples)
		var loudest []float64
		maxLen := 0
		for _, seg := range segs {
			if seg.end-seg.start > maxLen {
				maxLen = seg.end - seg.start
				loudest = tr.samples[seg.start:seg.end]
			}
		}
		if len(loudest) == 0 {
			loudest = tr.samples
		}

		prof := analyzeSpectrum(attackPortion(loudest))
		match := prof.classification == spec.waveform
		flag := "ok"
		if !match {
			flag = "FAIL"
			failed = append(failed, fmt.Sprintf("%s: class %s, want %s", tr.name, prof.classification, spec.waveform))
		}
		fmt.Printf("%-12s %7.0f %7.1f%% %7.3f %9.0f %7s %s\n",
			tr.name, prof.fundamentalHz, prof.fundamentalPct*100, prof.h3Ratio,
			prof.centroidHz, prof.classification, flag)
	}

	if len(failed) > 0 {
		fmt.Printf("\nFAIL: %s\n", strings.Join(failed, "; "))
		return 1
	}
	fmt.Println("\nPASS: all waveform classes match.")
	return 0
}

func runDistinct(tracks []track) int {
	fmt.Printf("Distinctiveness (%d files, min distance %.2f)\n\n", len(tracks), minDistinctDistance)

	features := make([]soundFeatures, len(tracks))
	for i, tr := range tracks {
		features[i] = soundFeatures{name: tr.name, vec: featureVector(tr)}
	}

	normalizeFeatures(features)

	var tooClose [][2]string
	minDist := math.MaxFloat64
	var minPair [2]string

	for i := 0; i < len(features); i++ {
		for j := i + 1; j < len(features); j++ {
			d := euclidean(features[i].vec, features[j].vec)
			if d < minDist {
				minDist = d
				minPair = [2]string{features[i].name, features[j].name}
			}
			if d < minDistinctDistance {
				tooClose = append(tooClose, [2]string{features[i].name, features[j].name})
				fmt.Printf("  CLOSE: %s ↔ %s (distance %.3f)\n", features[i].name, features[j].name, d)
			}
		}
	}

	fmt.Printf("\nClosest pair: %s ↔ %s (distance %.3f)\n", minPair[0], minPair[1], minDist)

	if len(tooClose) > 0 {
		var pairs []string
		for _, p := range tooClose {
			pairs = append(pairs, p[0]+"/"+p[1])
		}
		fmt.Printf("FAIL: pairs below threshold: %s\n", strings.Join(pairs, ", "))
		return 1
	}
	fmt.Println("PASS: all sounds sufficiently distinct.")
	return 0
}

func featureVector(tr track) []float64 {
	_, rms := measure(tr.samples)
	dur := durationSec(tr.samples)

	segs := findSegments(tr.samples)
	var pitches []float64
	for _, seg := range segs {
		p := estimatePitch(tr.samples[seg.start:seg.end])
		pitches = append(pitches, p/1000)
	}
	for len(pitches) < 4 {
		pitches = append(pitches, 0)
	}

	prof := analyzeSpectrum(tr.samples)
	mag := logSpectrum(tr.samples)

	vec := []float64{linearToDBFS(rms) / 60, dur, prof.centroidHz / 2000, prof.fundamentalPct, prof.h3Ratio}
	vec = append(vec, pitches[:4]...)
	vec = append(vec, mag...)
	return vec
}

func logSpectrum(samples []float64) []float64 {
	const bins = 16
	n := nextPow2(len(samples))
	if n > 8192 {
		n = 8192
	}
	windowed := make([]float64, n)
	copy(windowed, samples[:min(len(samples), n)])
	mag := fftMagnitude(windowed)
	binHz := float64(sampleRateHz) / float64(n)

	out := make([]float64, bins)
	step := (len(mag) / 2) / bins
	if step < 1 {
		step = 1
	}
	for i := 0; i < bins; i++ {
		start := i * step
		end := start + step
		if end > len(mag) {
			end = len(mag)
		}
		sum := 0.0
		for j := start; j < end; j++ {
			sum += mag[j]
		}
		out[i] = math.Log1p(sum) / math.Log1p(float64(step)*binHz)
	}
	return out
}

func normalizeFeatures(features []soundFeatures) {
	if len(features) == 0 {
		return
	}
	dims := len(features[0].vec)
	minV := make([]float64, dims)
	maxV := make([]float64, dims)
	for d := 0; d < dims; d++ {
		minV[d] = math.MaxFloat64
		maxV[d] = -math.MaxFloat64
	}
	for _, f := range features {
		for d, v := range f.vec {
			if v < minV[d] {
				minV[d] = v
			}
			if v > maxV[d] {
				maxV[d] = v
			}
		}
	}
	for i := range features {
		for d, v := range features[i].vec {
			span := maxV[d] - minV[d]
			if span == 0 {
				features[i].vec[d] = 0
			} else {
				features[i].vec[d] = (v - minV[d]) / span
			}
		}
	}
}

func euclidean(a, b []float64) float64 {
	n := min(len(a), len(b))
	sum := 0.0
	for i := 0; i < n; i++ {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum / float64(n))
}
