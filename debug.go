//go:build debug

package main

import (
	"fmt"
	"os"

	"github.com/jgabor/tuta/internal/soundcheck"
)

func init() {
	tryDebugCLI = func(args []string) (int, bool) {
		if len(args) == 0 || args[0] != "debug" {
			return 0, false
		}
		return runDebug(args[1:]), true
	}
	debugUsage = debugUsageSection
}

func runDebug(args []string) int {
	if len(args) == 0 {
		fmt.Print(debugUsageSection())
		return 2
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Print(debugUsageSection())
		return 0
	case "sounds":
		return soundcheck.Run(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "tuta: unknown debug command %q\n", args[0])
		fmt.Print(debugUsageSection())
		return 2
	}
}

func debugUsageSection() string {
	return `
Debug commands:
  tuta debug sounds <command> [options] [dir]

Sound check commands: volume, duration, pitch, spectrum, distinct, all
  tuta debug sounds all
  tuta debug sounds volume -t 2 tmp/
`
}
