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

GNU-style: every flag has a short and a long form. Stdlib `flag` only, so `--long`
works but short-flag bundling (`-cp`) does not.

| Flag | Default | Description |
|------|---------|-------------|
| `-c, --columns` | off | Split each line into fields → array of arrays |
| `-n, --names A,B` | — | Split, emit array of objects with these keys |
| `-H, --header` | off | Split; first row supplies the object keys |
| `-k, --kv` | off | Object `{first field: remainder}` |
| `--csv` / `--tsv` | off | Parse as CSV/TSV (quoted fields honoured) |
| `-d, --delimiter STR` | whitespace | Field separator; default splits on runs of whitespace |
| `-f, --fields LIST` | — | Keep 1-based fields; ranges allowed, e.g. `1-3,7` |
| `-0, --null` | off | Read NUL-delimited input |
| `-p, --pretty` / `--compact` | auto | Force pretty / compact (auto = pretty on a TTY) |
| `--color MODE` | auto | `auto` / `always` / `never` (`--no-color` alias); auto = color on a TTY, honours `NO_COLOR` |
| `--raw` | off | Keep every value a string (disable type inference) |
| `-l, --jsonl` | off | Newline-delimited JSON (one value per line) |
| `-F, --follow` | off | Stream line-by-line (`tail -f`) |
| `-o, --output FILE` | stdout | Write to FILE |
| `--wrap` | off | Wrap output in `{host,timestamp,data}` |
| `-V, --version` / `-h, --help` | — | Version / help |

**Defaults that matter:** type inference is **on** (round-trip-safe; `--raw` to
disable), the split delimiter defaults to **whitespace runs**, and output is
**TTY-aware** (pretty+color interactively, compact when piped).

**Mode precedence:** `-k` → `-H`/`-n` (objects) → `-c`/`--csv`/`--tsv` (columns) →
lines. Input is stdin, or file arguments when given (`.csv`/`.tsv` auto-detected).

## Architecture

Single-package `main`, standard library only. All logic lives in `json_encode.go`.

Entry chain (testable — no globals, streams injected):
- `main()` → `realMain(args, stdin, stdout, stderr) int`
- `parseArgs(args)` builds a fresh `flag.NewFlagSet`, registering each flag twice
  (short+long) into a `config` struct; returns `flag.ErrHelp` for `-h`.
- `realMain` handles help/version/parse errors, then calls
  `run(cfg, stdin, stdout, stderr) int`.
- `run` does batch; `runStream` (`-F`) is the `bufio.Scanner` per-line path.

Pipeline helpers (most reused from v2):
- `readInput()` — stdin or file args; `detectFormat()` auto-detects `.csv`/`.tsv`
- `splitRecords()` — splits on `\n` or NUL (`-0`), drops empties, trims `\r`
- `config.splitFields()` / `config.splitKV()` — whitespace by default,
  `strings.Split` when `-d` given; `readCSVRows()` for CSV/TSV
- `selectFields()` / `parseFieldSpec()` — `-f` projection with range expansion
- `coerce(s, raw)` — round-trip-safe type inference (`FormatInt/FormatFloat == s`,
  exact `true/false/null`; rejects Inf/NaN, leading zeros, non-canonical floats)
- `resolveMode()` → `modeLines/Columns/Objects/KeyValue`
- `buildLines/buildTable/buildObjects/buildKeyValue` — shape builders (take a `coerceFn`)
- output: `emit()` applies `--wrap` and `-l`; `usePretty`/`useColor` resolve the
  TTY-aware defaults (`isTerminal` via `os.ModeCharDevice`); `writeValue` uses
  stdlib `json.Marshal`/`MarshalIndent` when color is off, else `encodeColored`
  (a small ANSI colorizer over our value types, keys sorted for determinism).

Tests are in `json_encode_test.go` (table-driven, via `realMain` with in-memory
streams) and the `Makefile` `functional-test` target (shell integration).

## Releasing

`.goreleaser.yaml` cross-compiles static binaries (linux/darwin/windows ×
amd64/arm64) and pushes a Homebrew formula to `fedir/homebrew-tap`. CI lives in
`.github/workflows/`: `ci.yml` (vet + unit + functional on push/PR) and
`release.yml` (GoReleaser on `v*` tags). The tap publish needs a
`HOMEBREW_TAP_TOKEN` repo secret.

## Testing strategy

- **Unit tests** (`json_encode_test.go`) — cover pure Go functions in isolation
- **Functional tests** (`make functional-test`) — shell pipelines that exercise the compiled binary end-to-end; grouped by topic: basic, sysadmin, key-value, v3 UX (objects, typing, jsonl, color, output), Loki. Expected outputs assume type inference is on (e.g. `200` is a number).
- **Loki tests** do not require a running Loki instance; they validate the JSON payload shape using `jq -e` assertions (structure, field values, array lengths). `curl` is never called in tests.
- `jq` must be available on the test machine (`brew install jq` / `apt install jq`)

