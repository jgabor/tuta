package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jgabor/tuta/alert"
)

var version = "0.5.0"

// Optional hooks registered from debug.go when built with -tags debug.
var (
	tryDebugCLI func(args []string) (int, bool)
	debugUsage  func() string
)

func usage() {
	extra := ""
	if debugUsage != nil {
		extra = debugUsage()
	}
	fmt.Printf(`tuta %s — Tiny Utility for Tone Alerts
Author: Jonathan Gabor

Usage:
  tuta [sound]
  tuta export [-o DIR] [-mono|-stereo] [-depth 16|24] [sound ...]%s

Available sounds:
  success   ascending C major arpeggio (default)
  error     descending tritone buzz (D3-Ab2)
  warning   three pings with major second tension
  info      short neutral blip
  complete  ascending F major triad (F4-A4-C5)
  increase  ascending major triad (C4-E4-G4)
  decrease  descending minor triad, fading (G4-Eb4-C4)
  notify    ascending minor third ping (A5-C6)
  progress  ascending major triad (E4-G4-B4)
  confirm   ascending perfect fifth (C5-G5)
  cancel    single tone (B4)
  ready     ascending major third (C5-E5, triangle)
  timeout   descending frequency sweep (E4-Bb3)

Options:
  -h, --help      show this help
  -v, --version   show version

Export options:
  -o DIR          output directory (default: ./tmp)
  -mono           export mono FLAC (default)
  -stereo         export stereo FLAC (L=R, same as playback)
  -depth N        bit depth: 16 or 24 (default: 16)
`, version, extra)
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println("tuta", version)
			return
		case "--help", "-h":
			usage()
			return
		case "export":
			if err := runExport(os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
		if tryDebugCLI != nil {
			if code, ok := tryDebugCLI(os.Args[1:]); ok {
				os.Exit(code)
			}
		}
	}
	soundType := "success"
	if len(os.Args) > 1 {
		for _, name := range alert.Names() {
			if os.Args[1] == name {
				soundType = os.Args[1]
				break
			}
		}
	}
	if err := alert.Play(soundType); err != nil {
		os.Exit(1)
	}
}

func runExport(args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	outDir := fs.String("o", "./tmp", "output directory")
	mono := fs.Bool("mono", true, "export mono FLAC")
	stereo := fs.Bool("stereo", false, "export stereo FLAC (L=R)")
	depth := fs.Int("depth", 16, "bit depth: 16 or 24")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *stereo {
		*mono = false
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

	known := make(map[string]struct{}, len(alert.Names()))
	for _, name := range alert.Names() {
		known[name] = struct{}{}
	}
	for _, name := range names {
		if _, ok := known[name]; !ok {
			return fmt.Errorf("unknown sound: %s", name)
		}
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}

	for _, name := range names {
		path := filepath.Join(*outDir, name+".flac")
		if err := alert.ExportFLAC(path, name, opts); err != nil {
			return err
		}
		fmt.Println(path)
	}
	return nil
}
