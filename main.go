package main

import (
	"fmt"
	"os"

	"github.com/jgabor/tuta/alert"
)

var version = "0.4.0"

func usage() {
	fmt.Printf(`tuta %s — Tiny Utility for Tone Alerts
Author: Jonathan Gabor

Usage:
  tuta [sound]

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
`, version)
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
