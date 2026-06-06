# Agent notes for tuta

## Build and run

**Always use mage.** Do not run `go build -o tuta .` in the repo root.

| Task | Command | Binary | Tags |
|------|---------|--------|------|
| Build (release) | `mage build` | `./build/tuta` | release |
| Build (debug) | `mage build -debug=true` or `DEBUG=1 mage build` | `./build/tuta` | debug |
| Install from source | `mage install` | `$GOBIN/tuta` | debug (always) |
| Test | `mage test` | — | — |
| Clean | `mage clean` | removes `./build/` | — |

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

Release builds (`mage build` without `-debug`) do not include `tuta debug`; on release, `tuta debug` silently plays `success`.

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
