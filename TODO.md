# Roadmap

## v0.6.0

- [x] Print playback errors to stderr with the requested sound name.
- [x] Warn or fail clearly on unknown sounds and options instead of silently playing `success`.
- [x] Explain that `tuta debug` requires a debug build when invoked from a release build.
- [x] Add root CLI tests for help, version, sound dispatch, errors, export flags, and debug dispatch.
- [x] Require tests, linting, vetting, and vulnerability checks to pass before publishing releases.
- [x] Define and document CLI exit codes and best-effort playback behavior.

- [x] Add `tuta list`, including machine-readable `--json` output.
- [x] Centralize sound metadata and generate help text from that source of truth.
- [x] Add `tuta preview` for discovering sounds individually or in sequence.
- [x] Add global playback volume control without exposing low-level synthesis parameters.

## v0.7.0

- [ ] Add structured output for sound checks and FLAC exports.
- [ ] Add safe export overwrite behavior with `--dry-run` and `--force`.
- [ ] Add package-manager distribution, starting with Homebrew if macOS demand warrants it.
- [ ] Add context-aware or cancellable library playback.
- [ ] Add custom sound extensibility only if downstream consumers require it.

## Out of scope for now

- Background daemons or services
- Cloud synchronization
- Large configuration or profile systems
- Plugin infrastructure
- Arbitrary synthesis configuration
- Automatic network updates
- A heavy CLI framework without demonstrated need
