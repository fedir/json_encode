# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Git commits

Use simple conventional commit one-liners. No `Co-Authored-By` or other trailers.

```
feat: add markdown output format
fix: handle missing API key gracefully
refactor: extract pagination logic
```

## Commands

```bash
make build            # compile → ./json_encode binary
make test             # go test -race ./...
make vet              # go vet ./...
make functional-test  # build + run shell-level integration tests
make clean            # remove binary and coverage artifacts

# Run a single test
go test -race -run TestName .
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-s`  | `" "` | Separator for splitting lines into columns |
| `-sc` | off   | Column mode — split each line into an array of fields |
| `-kv` | off   | Key-value mode — first field is key, remainder is value |
| `-p`  | off   | Pretty-print output |

Modes: `-kv` takes priority over `-sc`; without either, each line becomes a string element.

## Architecture

Single-package `main` with no external dependencies. All logic lives in `json_encode.go`:

- `GetInputData()` — reads all of stdin
- `ConvertInputToLines()` — splits on `\n`, drops empty lines
- `ConvertLinesToTable()` — splits each line by `-s` separator
- `ConvertLinesToKeyValue()` — splits each line with `SplitN(..., 2)` so the separator only cuts once; first part is key, remainder is value
- `ConvertToJSON()` — marshals with optional indent via `-p`

Tests are in `json_encode_test.go` (unit) and `Makefile` `functional-test` target (shell integration).

## Testing strategy

- **Unit tests** (`json_encode_test.go`) — cover pure Go functions in isolation
- **Functional tests** (`make functional-test`) — shell pipelines that exercise the compiled binary end-to-end; grouped by topic: basic, sysadmin, key-value, Loki
- **Loki tests** do not require a running Loki instance; they validate the JSON payload shape using `jq -e` assertions (structure, field values, array lengths). `curl` is never called in tests.
- `jq` must be available on the test machine (`brew install jq` / `apt install jq`)

