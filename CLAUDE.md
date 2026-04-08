# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

- Build: `make build` (or `make build-release` for a stripped, CGO-disabled release build)
- Install: `make install` (to `$GOPATH/bin`)
- Test: `make test` — single test: `go test ./pkg/gemini -run TestName`
- Lint / format: `make lint` (go vet), `make fmt`
- Pre-commit hooks are configured (go-fmt, go-vet, go-imports, golangci-lint) via `.pre-commit-config.yaml`.

Requires Go 1.25+. A Gemini API key is resolved from `NANOBANANA_GEMINI_API_KEY` or `GEMINI_API_KEY` (see `pkg/gemini/client.go` for the full precedence).

## Architecture

Single-binary Go CLI for Google's Gemini image API. Positioned as a lightweight alternative to Google's official CLI — no cgo, no runtime deps.

- `main.go` — version info injected via ldflags; delegates to `cmd.Execute()`.
- `cmd/` — Cobra commands, one file per subcommand: `generate`, `edit`, `restore`, `icon`, `pattern`, `story`, `diagram`. `cmd/root.go` wires them up.
- `pkg/gemini/client.go` — HTTP client for the Gemini API. Supports two models: `gemini-3-pro-image-preview` (Pro, default) and `gemini-3.1-flash-image-preview` (Frugal/Nano Banana 2). 13 aspect ratios. 5-minute HTTP timeout. The canonical multimodal entry point is `GenerateContentWithRefs(prompt, []RefImage, resolution, aspectRatio)`; `doGenerate` is the shared marshal→POST→parse path, and the older `GenerateContentWith{Image,Images,Resolution,FullOptions}` helpers are thin legacy shims that all funnel into `GenerateContentWithRefs`. New call sites should prefer `RefImage` (it preserves per-image MIME type) over the legacy `[]string` base64 slice (which hardcodes `image/png`).
- `pkg/gemini/config.go` — JSON config with per-talk and global defaults composition (used for presentation workflows). `LoadConfig` records the config file's directory on `ImageGenConfig.configDir`; `GetReferences()` uses it to resolve relative reference-image paths against the config file rather than CWD.
- `pkg/filehandler/` — base64 decoding and AI-suggested output filenames, with a truncated-prompt fallback.
- `pkg/metadata/png.go` — PNG metadata (optional prompt embedding) and JPEG→PNG conversion.

All subcommands share the same `gemini` client + `filehandler` + `metadata` pipeline. To add a new command, create a new `cmd/*.go` file, register it in `root.go`, and call into `pkg/gemini`.

## Release

GoReleaser builds multi-platform binaries (linux/darwin/windows × amd64/arm64).
