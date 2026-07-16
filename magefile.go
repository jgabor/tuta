//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var buildDir = filepath.Join(".", "build")

func ldflags() string {
	v := version()
	return fmt.Sprintf("-s -w -X main.version=%s", v)
}

func version() string {
	if v := os.Getenv("VERSION"); v != "" {
		return v
	}
	out, err := exec.Command("git", "describe", "--tags", "--always", "--dirty").Output()
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(out))
}

func wantDebug(flag *bool) bool {
	if flag != nil && *flag {
		return true
	}
	v := os.Getenv("DEBUG")
	return v == "1" || strings.EqualFold(v, "true")
}

func goArgs(debug bool) []string {
	args := []string{"-ldflags", ldflags()}
	if debug {
		args = append([]string{"-tags", "debug"}, args...)
	}
	return args
}

// compile builds or installs tuta. local writes ./build/tuta; otherwise go install to GOBIN.
func compile(local, debug bool) error {
	args := goArgs(debug)
	kind := "release"
	if debug {
		kind = "debug"
	}

	if local {
		if err := os.MkdirAll(buildDir, 0o755); err != nil {
			return err
		}
		// Use the target GOOS (GOOS env) for the binary name so cross-compiling
		// to Windows from any host correctly appends .exe; fall back to the
		// host runtime when GOOS is unset (normal local builds).
		targetGOOS := os.Getenv("GOOS")
		if targetGOOS == "" {
			targetGOOS = runtime.GOOS
		}
		bin := filepath.Join(buildDir, "tuta")
		if targetGOOS == "windows" {
			bin += ".exe"
		}
		_ = os.Remove(bin)
		fmt.Printf("building %s (%s, %s)\n", bin, kind, version())
		cmd := exec.Command("go", append([]string{"build", "-o", bin}, append(args, ".")...)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	fmt.Printf("installing tuta (%s, %s) → $(go env GOBIN)/tuta\n", kind, version())
	cmd := exec.Command("go", append([]string{"install"}, append(args, ".")...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Build compiles tuta into ./build/. Pass -debug=true or set DEBUG=1 for tuta debug commands.
func Build(debug *bool) error {
	return compile(true, wantDebug(debug))
}

// Install installs a debug tuta to go env GOBIN (source dev install).
func Install() error {
	return compile(false, true)
}

// Test runs all tests with the race detector (includes debug-tagged soundcheck tests).
func Test() error {
	cmd := exec.Command("go", "test", "-tags", "debug", "./...", "-race")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Verify runs the full pre-release quality gate: tests (race), golangci-lint,
// go vet, and govulncheck — all with the debug build tag so debug-tagged code
// (soundcheck, debug CLI) is covered. The GitHub release workflow gates
// publishing on this. It mirrors the lefthook pre-push gate but never mutates
// (no --fix). golangci-lint reads build-tags from .golangci.yml.
func Verify() error {
	steps := []struct {
		name string
		cmd  *exec.Cmd
	}{
		{"test", exec.Command("go", "test", "-tags", "debug", "./...", "-race")},
		{"lint", exec.Command("golangci-lint", "run", "./...")},
		{"vet", exec.Command("go", "vet", "-tags", "debug", "./...")},
		{"vuln", exec.Command("govulncheck", "-tags", "debug", "./...")},
	}
	for _, s := range steps {
		fmt.Printf("=== verify: %s ===\n", s.name)
		s.cmd.Stdout = os.Stdout
		s.cmd.Stderr = os.Stderr
		if err := s.cmd.Run(); err != nil {
			return fmt.Errorf("verify %s: %w", s.name, err)
		}
	}
	return nil
}

// Debug runs `tuta debug sounds all tmp/` via go run with the debug build
// tag, so no separate build or install step is required. If `tmp/` has no
// FLAC files, exports them first.
func Debug() error {
	const dir = "tmp/"
	if !hasFLACs(dir) {
		if err := runTuta("export", "-o", dir); err != nil {
			return err
		}
	}
	return runTuta("debug", "sounds", "all", dir)
}

func runTuta(args ...string) error {
	cmdArgs := append([]string{"run"}, goArgs(true)...)
	cmdArgs = append(cmdArgs, ".")
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func hasFLACs(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".flac") {
			return true
		}
	}
	return false
}

// Clean removes the build directory.
func Clean() error {
	fmt.Println("cleaning build/")
	return os.RemoveAll(buildDir)
}
