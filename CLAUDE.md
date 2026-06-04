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
make snapshot         # goreleaser local snapshot build (no publish)
make release          # goreleaser tagged release (CI/maintainer only)
make clean            # remove binary, coverage artifacts and dist/

# Run a single test
go test -race -run TestName .
```

The build embeds the version via `-ldflags "-X main.version=$(VERSION)"`, where
`VERSION` comes from `git describe`. `make build` keeps it; a plain `go build`
yields `dev`.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-s`  | `" "` | Separator for splitting lines into fields |
| `-sc` | off   | Column mode — split each line into an array of fields |
| `-kv` | off   | Key-value mode — first field is key, remainder is value |
| `-cols` | —   | Comma-separated column names → array of objects |
| `-header` | off | Use the first input line as object keys |
| `-t`  | off   | Infer numbers / booleans / null instead of strings |
| `-w`  | off   | Split on runs of whitespace (awk-style); ignores `-s` |
| `-f`  | —     | Select 1-based field indices, e.g. `1,3,6` |
| `-csv` / `-tsv` | off | Parse input as CSV/TSV (quoted fields honoured) |
| `-0`  | off   | Split input on NUL instead of newline |
| `-nd` | off   | Newline-delimited JSON (one value per line) |
| `-stream` | off | Stream line-by-line; emit per line (works with `tail -f`) |
| `-o`  | off   | Wrap output in `{host,timestamp,data}` |
| `-p`  | off   | Pretty-print output |
| `-version` / `-v` | — | Print version and exit |

**Mode precedence:** `-kv` → `-cols`/`-header` (objects) → `-sc`/`-csv`/`-tsv`
(columns) → lines. Input is stdin, or file arguments when given.

## Architecture

Single-package `main`, standard library only. All logic lives in `json_encode.go`.

Two top-level paths from `main()`:
- **batch** (`runBatch`) — reads all input, builds one value, emits via `output()`.
- **stream** (`runStream`, `-stream`) — `bufio.Scanner` loop, marshals and flushes
  one value per line so it works with never-ending pipes like `tail -f`.

Pipeline helpers:
- `readInput()` — stdin, or file args from `flag.Args()`
- `splitRecords()` — splits on `\n` or NUL (`-0`), drops empty records, trims `\r`
- `splitFields()` — `strings.Fields` (`-w`) or `strings.Split` on `-s`; `readCSVRows()`
  uses `encoding/csv` for `-csv`/`-tsv`
- `selectFields()` / `parseFieldSpec()` — `-f` 1-based projection
- `coerce()` — `-t` type inference (rejects Inf/NaN so JSON stays valid)
- `resolveMode()` — maps flags to `modeLines/Columns/Objects/KeyValue`
- `buildLines/buildTable/buildObjects/buildKeyValue` — shape builders
- `output()` — applies `-o` wrap and `-nd`, else `ConvertToJSON` (`-p` indent)

Legacy exported functions (`GetInputData`, `ConvertInputToLines`,
`ConvertLinesToTable`, `ConvertLinesToKeyValue`, `ConvertToJSON`) are retained
because the unit tests call them directly — keep them working.

Tests are in `json_encode_test.go` (unit) and `Makefile` `functional-test` target
(shell integration).

## Releasing

`.goreleaser.yaml` cross-compiles static binaries (linux/darwin/windows ×
amd64/arm64) and pushes a Homebrew formula to `fedir/homebrew-tap`. CI lives in
`.github/workflows/`: `ci.yml` (vet + unit + functional on push/PR) and
`release.yml` (GoReleaser on `v*` tags). The tap publish needs a
`HOMEBREW_TAP_TOKEN` repo secret.

## Testing strategy

- **Unit tests** (`json_encode_test.go`) — cover pure Go functions in isolation
- **Functional tests** (`make functional-test`) — shell pipelines that exercise the compiled binary end-to-end; grouped by topic: basic, sysadmin, key-value, Loki
- **Loki tests** do not require a running Loki instance; they validate the JSON payload shape using `jq -e` assertions (structure, field values, array lengths). `curl` is never called in tests.
- `jq` must be available on the test machine (`brew install jq` / `apt install jq`)

