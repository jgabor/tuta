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

// Build compiles the tuta binary into ./build/ and, if `upx` is in PATH,
// compresses it with `--best --lzma`. Safe to re-run: `go build -o` always
// rewrites the binary fresh before compression.
func Build() error {
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return err
	}
	bin := filepath.Join(buildDir, "tuta")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	// Remove any previous output so `go build` always writes a fresh,
	// unpacked binary. Without this, re-running after a UPX pack leaves
	// the packed binary in place (Go sees a cached match and skips the
	// write), which then trips AlreadyPackedException on the next pack.
	_ = os.Remove(bin)
	fmt.Printf("building %s (%s)\n", bin, version())
	cmd := exec.Command("go", "build", "-ldflags", ldflags(), "-o", bin, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	upx, err := exec.LookPath("upx")
	if err != nil {
		return nil
	}
	fmt.Printf("compressing %s with upx\n", bin)
	pack := exec.Command(upx, "--best", "--lzma", bin)
	pack.Stdout = os.Stdout
	pack.Stderr = os.Stderr
	if err := pack.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: upx failed (%v); leaving binary uncompressed\n", err)
	}
	return nil
}

// Install installs the tuta binary to $GOPATH/bin.
func Install() error {
	fmt.Printf("installing tuta (%s)\n", version())
	cmd := exec.Command("go", "install", "-ldflags", ldflags(), ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Test runs all tests with the race detector.
func Test() error {
	cmd := exec.Command("go", "test", "./...", "-race")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Clean removes the build directory.
func Clean() error {
	fmt.Println("cleaning build/")
	return os.RemoveAll(buildDir)
}
