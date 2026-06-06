package alert

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/tphakala/go-flac/pcm"
)

// FLACOptions configures FLAC export. Zero values use defaults: mono, 16-bit,
// compression level 5.
type FLACOptions struct {
	Channels         int
	BitDepth         int
	CompressionLevel int
}

func (o FLACOptions) normalized() FLACOptions {
	if o.Channels == 0 {
		o.Channels = 1
	}
	if o.BitDepth == 0 {
		o.BitDepth = 16
	}
	if o.CompressionLevel == 0 {
		o.CompressionLevel = 5
	}
	return o
}

// WriteFLAC encodes the named sound as FLAC to w. w should be an io.WriteSeeker
// (for example *os.File) so STREAMINFO is finalized on Close.
func WriteFLAC(w io.WriteSeeker, name string, opts FLACOptions) error {
	opts = opts.normalized()
	if opts.Channels != 1 && opts.Channels != 2 {
		return fmt.Errorf("alert: FLAC channels must be 1 or 2, got %d", opts.Channels)
	}
	if opts.BitDepth != 16 && opts.BitDepth != 24 {
		return fmt.Errorf("alert: FLAC bit depth must be 16 or 24, got %d", opts.BitDepth)
	}

	mono := Render(name)
	pcmBytes := float32ToPCM(mono, opts.Channels, opts.BitDepth)

	enc, err := pcm.NewEncoder(w, pcm.Config{
		SampleRate:       sampleRate,
		BitDepth:         opts.BitDepth,
		Channels:         opts.Channels,
		CompressionLevel: opts.CompressionLevel,
	})
	if err != nil {
		return err
	}
	if _, err := enc.Write(pcmBytes); err != nil {
		return err
	}
	return enc.Close()
}

// ExportFLAC writes the named sound to path as a FLAC file.
func ExportFLAC(path, name string, opts FLACOptions) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := WriteFLAC(f, name, opts); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func float32ToPCM(mono []float32, channels, bitDepth int) []byte {
	if channels == 1 {
		return samplesToPCM(mono, bitDepth)
	}

	interleaved := make([]float32, len(mono)*2)
	for i, s := range mono {
		interleaved[i*2] = s
		interleaved[i*2+1] = s
	}
	return samplesToPCM(interleaved, bitDepth)
}

func samplesToPCM(samples []float32, bitDepth int) []byte {
	switch bitDepth {
	case 16:
		buf := make([]byte, len(samples)*2)
		for i, s := range samples {
			v := floatToInt16(s)
			binary.LittleEndian.PutUint16(buf[i*2:], uint16(v))
		}
		return buf
	case 24:
		buf := make([]byte, len(samples)*3)
		for i, s := range samples {
			v := floatToInt24(s)
			buf[i*3] = byte(v)
			buf[i*3+1] = byte(v >> 8)
			buf[i*3+2] = byte(v >> 16)
		}
		return buf
	default:
		return nil
	}
}

func floatToInt16(s float32) int16 {
	s = clampFloat32(s)
	return int16(s * 32767)
}

func floatToInt24(s float32) int32 {
	s = clampFloat32(s)
	return int32(s * 8388607)
}

func clampFloat32(s float32) float32 {
	if s > 1 {
		return 1
	}
	if s < -1 {
		return -1
	}
	if math.IsNaN(float64(s)) {
		return 0
	}
	return s
}
