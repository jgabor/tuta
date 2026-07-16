# Agent notes for tuta

## Build and run

**Always use mage.** Do not run `go build -o tuta .` in the repo root.

| Task | Command | Binary | Tags |
|------|---------|--------|------|
| Build (release) | `mage build` | `./build/tuta` | release |
| Build (debug) | `mage build -debug=true` or `DEBUG=1 mage build` | `./build/tuta` | debug |
| Install from source | `mage install` | `$GOBIN/tuta` | debug (always) |
| Test | `mage test` | — | — |
| Verify (gate) | `mage verify` | — | test+lint+vet+vuln |
| Clean | `mage clean` | removes `./build/` | — |

`mage verify` runs tests (race), golangci-lint, go vet, and govulncheck — the same gate the release workflow requires before publishing. All four steps cover debug-tagged code (`-tags debug`; golangci-lint reads `build-tags` from `.golangci.yml`). Cross-compile a specific target with `GOOS=<os> GOARCH=<arch> mage build` (the `.exe` suffix follows the target `GOOS`).

Consumer install (`go install github.com/jgabor/tuta@latest`) is **release** — no `tuta debug`.

Run the local binary as `./build/tuta`, never assume `./tuta` in the repo root exists or is current.

**Which binary am I running?** `which tuta`, `tuta -v`. After `mage install`, PATH `tuta` is debug. `./build/tuta` may be release unless you built with `-debug=true` or `DEBUG=1`.

Requires [mage](https://magefile.org/): `go install github.com/magefile/mage@latest`

Go 1.26+ required to build from source.

Version string comes from `git describe` via mage ldflags; `main.go` `version` is fallback only. Bump tag and fallback when releasing.

## Debug sound validation

After exporting FLACs, validate with a debug build:

```sh
mage debug                            # exports tmp/ if needed, then runs all checks
```

Or build a debug binary and call the CLI directly:

```sh
mage build -debug=true                # or: mage install && tuta ...
./build/tuta export -o tmp/
./build/tuta debug sounds all tmp/
```

Release builds (`mage build` without `-debug`) do not include `tuta debug`; invoking `tuta debug` on a release build prints an error explaining the debug-build requirement and exits 2 (it does not silently play `success`).

## Exit codes

CLI exit codes: `0` success; `1` runtime failure (playback/export); `2` usage error (unknown option/command, bad flag) or `tuta debug` from a release build. An **unknown sound name** is not exit 2 — `tuta` warns to stderr and plays `success` as a best-effort fallback (exit 0). See README → Exit codes.

## Export and analysis

```sh
./build/tuta export -o tmp/     # all sounds → tmp/*.flac
```

`tmp/` is generated output — do not commit unless explicitly requested.

## Library

Import path: `github.com/jgabor/tuta/alert`

Key APIs: `alert.Play`, `alert.Render`, `alert.ExportFLAC`, `alert.Names`

## Commits

Do not commit unless the user asks. Do not bump `main.go` `version` unless releasing.
