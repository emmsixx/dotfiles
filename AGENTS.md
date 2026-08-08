# Repository Guidelines

## Overview

This repository contains a Go CLI and Stow-managed development configuration for macOS and Linux. The CLI installs tools, links configuration, manages secrets and authentication, and supports desktop and headless-server profiles.

## Repository Layout

- `cmd/dotfiles/`: CLI entry point.
- `internal/app/`: command parsing and orchestration.
- `internal/setup/`: setup groups, installers, and Stow linking.
- `internal/secrets/`: local secret-file management.
- `internal/codexhome/`: Codex account isolation.
- `internal/ui/`: terminal UI helpers.
- `.config/`, `.claude/`, `.pi/`, and `.scripts/`: managed user configuration.
- `setup.sh`: minimal POSIX shell bootstrapper for the released CLI.

## Development Commands

Run these before finishing a Go change:

```sh
gofmt -w <changed-go-files>
go vet ./...
go test ./...
go build ./cmd/dotfiles
```

Use a focused `go test` invocation while iterating, then run the full suite.

## Conventions

- Use the Go version declared in `go.mod` and keep Go code formatted with `gofmt`.
- Keep bootstrap scripts POSIX `sh`; use four-space indentation and avoid Bash-only syntax.
- Preserve support for both macOS and Linux. Do not assume Homebrew is available on Linux or that every Linux host is interactive.
- Route external command execution through the existing injected runner where practical so behavior remains testable.
- When changing CLI commands or flags, update help text, Zsh completion, README examples, and tests together.
- When adding repository-only top-level files, exclude them in both `.stow-local-ignore` and `internal/setup/link.go` so setup does not link them into `$HOME`.
- Do not commit credentials, provider state, generated skills, build artifacts, or machine-specific runtime data. Follow `.gitignore` and `.secrets.example`.
- Do not overwrite unrelated working-tree changes.

## Testing and Safety

- Add or update tests for behavior changes and regressions.
- Use temporary repository and home directories in setup/linking tests.
- Do not test setup or linking against the real home directory.
- Keep setup operations idempotent and preserve unrelated user files and links.
