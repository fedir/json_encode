// Copyright 2018 Fedir RYKHTIK. All rights reserved.
// Use of this source code is governed by the GNU GPL 3.0
// license that can be found in the LICENSE file.
package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

// config holds every resolved command-line option.
type config struct {
	// modes
	columns bool
	names   string
	header  bool
	kv      bool
	csv     bool
	tsv     bool
	// splitting / input
	delimiter    string
	delimiterSet bool
	fields       string
	nullDelim    bool
	files        []string
	// output
	pretty  bool
	compact bool
	color   string
	noColor bool
	raw     bool
	jsonl   bool
	follow  bool
	output  string
	wrap    bool
	version bool
}

type encodeMode int

const (
	modeLines encodeMode = iota
	modeColumns
	modeObjects
	modeKeyValue
)

// ANSI colors for the syntax-highlighted output (jq-like).
const (
	colReset = "\x1b[0m"
	colKey   = "\x1b[34;1m" // bold blue
	colStr   = "\x1b[32m"   // green
	colBool  = "\x1b[33m"   // yellow
	colNull  = "\x1b[1;30m" // bright black
)

func usageText() string {
	return `json_encode ` + version + ` — turn shell output into JSON, one pipe away.

Usage: json_encode [flags] [file ...]

With no file arguments, input is read from stdin. On a terminal the output is
pretty-printed and colorized; when piped or redirected it is compact.

Modes (pick one; default = array of one string per line):
  -c, --columns        split each line into fields  → array of arrays
  -n, --names a,b,c    split, emit array of objects with these keys
  -H, --header         split, first row supplies the object keys
  -k, --kv             object {first field: remainder}
      --csv, --tsv     parse as CSV/TSV (quoted fields); with -H → objects

Splitting & input:
  -d, --delimiter STR  field separator (default: runs of whitespace, awk-style)
  -f, --fields LIST    keep 1-based fields, supports ranges, e.g. 1-3,7
  -0, --null           read NUL-delimited input (find -print0)

Output:
  -p, --pretty         force pretty (multi-line)
      --compact        force compact (single line)
      --color MODE     auto (default) | always | never
      --no-color       alias for --color=never
      --raw            keep every value a string (disable type inference)
  -l, --jsonl          newline-delimited JSON, one value per line
  -F, --follow         stream line-by-line as input arrives (tail -f)
  -o, --output FILE    write to FILE instead of stdout
      --wrap           wrap output in {host, timestamp, data}
  -V, --version        print version and exit
  -h, --help           show this help

Examples:
  seq 1 5 | json_encode                       [1,2,3,4,5]
  ps -eo pid,comm,pcpu | json_encode -H        [{"COMMAND":"systemd","PID":1,...},...]
  printf 'host db\nport 5432\n' | json_encode -k   {"host":"db","port":5432}
  tail -f app.log | json_encode -F -l          one JSON line per log line
`
}

// parseArgs builds a fresh FlagSet (so the function is re-entrant and testable)
// and registers every flag under both its short and long name.
func parseArgs(args []string) (config, error) {
	var c config
	fs := flag.NewFlagSet("json_encode", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {} // we print usageText() ourselves

	fs.BoolVar(&c.columns, "c", false, "")
	fs.BoolVar(&c.columns, "columns", false, "")
	fs.StringVar(&c.names, "n", "", "")
	fs.StringVar(&c.names, "names", "", "")
	fs.BoolVar(&c.header, "H", false, "")
	fs.BoolVar(&c.header, "header", false, "")
	fs.BoolVar(&c.kv, "k", false, "")
	fs.BoolVar(&c.kv, "kv", false, "")
	fs.BoolVar(&c.csv, "csv", false, "")
	fs.BoolVar(&c.tsv, "tsv", false, "")

	fs.StringVar(&c.delimiter, "d", "", "")
	fs.StringVar(&c.delimiter, "delimiter", "", "")
	fs.StringVar(&c.fields, "f", "", "")
	fs.StringVar(&c.fields, "fields", "", "")
	fs.BoolVar(&c.nullDelim, "0", false, "")
	fs.BoolVar(&c.nullDelim, "null", false, "")

	fs.BoolVar(&c.pretty, "p", false, "")
	fs.BoolVar(&c.pretty, "pretty", false, "")
	fs.BoolVar(&c.compact, "compact", false, "")
	fs.StringVar(&c.color, "color", "auto", "")
	fs.BoolVar(&c.noColor, "no-color", false, "")
	fs.BoolVar(&c.raw, "raw", false, "")
	fs.BoolVar(&c.jsonl, "l", false, "")
	fs.BoolVar(&c.jsonl, "jsonl", false, "")
	fs.BoolVar(&c.follow, "F", false, "")
	fs.BoolVar(&c.follow, "follow", false, "")
	fs.StringVar(&c.output, "o", "", "")
	fs.StringVar(&c.output, "output", "", "")
	fs.BoolVar(&c.wrap, "wrap", false, "")
	fs.BoolVar(&c.version, "V", false, "")
	fs.BoolVar(&c.version, "version", false, "")

	if err := fs.Parse(args); err != nil {
		return c, err
	}
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "d" || f.Name == "delimiter" {
			c.delimiterSet = true
		}
	})
	c.files = fs.Args()
	return c, nil
}

// --- input ---------------------------------------------------------------

func readInput(files []string, stdin io.Reader) (string, error) {
	if len(files) == 0 {
		b, err := io.ReadAll(stdin)
		return string(b), err
	}
	var sb strings.Builder
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		sb.Write(b)
	}
	return sb.String(), nil
}

func splitRecords(input string, null bool) []string {
	sep := "\n"
	if null {
		sep = "\x00"
	}
	parts := strings.Split(input, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimRight(p, "\r")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func readCSVRows(input string, tsv bool) ([][]string, error) {
	r := csv.NewReader(strings.NewReader(input))
	r.FieldsPerRecord = -1
	if tsv {
		r.Comma = '\t'
	}
	return r.ReadAll()
}

// --- splitting -----------------------------------------------------------

func (c config) splitFields(line string) []string {
	if !c.delimiterSet {
		return strings.Fields(line)
	}
	return strings.Split(line, c.delimiter)
}

func (c config) splitKV(line string) (string, string) {
	if !c.delimiterSet {
		trimmed := strings.TrimLeft(line, " \t\r\f\v")
		i := strings.IndexFunc(trimmed, unicode.IsSpace)
		if i < 0 {
			return trimmed, ""
		}
		return trimmed[:i], strings.TrimLeft(trimmed[i:], " \t\r\f\v")
	}
	parts := strings.SplitN(line, c.delimiter, 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

func parseFieldSpec(spec string) ([]int, error) {
	if spec == "" {
		return nil, nil
	}
	var idx []int
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			lo, err1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
			hi, err2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err1 != nil || err2 != nil || lo < 1 || hi < lo {
				return nil, fmt.Errorf("invalid -f range %q", part)
			}
			for i := lo; i <= hi; i++ {
				idx = append(idx, i)
			}
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("invalid -f field %q", part)
		}
		idx = append(idx, n)
	}
	return idx, nil
}

func selectFields(fields []string, idx []int) []string {
	if len(idx) == 0 {
		return fields
	}
	out := make([]string, 0, len(idx))
	for _, i := range idx {
		if i >= 1 && i <= len(fields) {
			out = append(out, fields[i-1])
		}
	}
	return out
}

// --- value building ------------------------------------------------------

// coerce turns a string into a number/bool/null when it round-trips exactly,
// so leading-zero IDs, versions, IPs and the like stay strings.
func coerce(s string, raw bool) interface{} {
	if raw {
		return s
	}
	switch s {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	if s == "" {
		return s
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil && strconv.FormatInt(i, 10) == s {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsInf(f, 0) && !math.IsNaN(f) {
		if strconv.FormatFloat(f, 'g', -1, 64) == s {
			return f
		}
	}
	return s
}

func columnKey(names []string, i int) string {
	if i < len(names) && names[i] != "" {
		return names[i]
	}
	return "col" + strconv.Itoa(i+1)
}

type coerceFn func(string) interface{}

func rowToObject(row, names []string, cf coerceFn) map[string]interface{} {
	m := make(map[string]interface{}, len(row))
	for i, f := range row {
		m[columnKey(names, i)] = cf(f)
	}
	return m
}

func rowToArray(row []string, cf coerceFn) []interface{} {
	out := make([]interface{}, 0, len(row))
	for _, f := range row {
		out = append(out, cf(f))
	}
	return out
}

func buildLines(records []string, cf coerceFn) []interface{} {
	out := make([]interface{}, 0, len(records))
	for _, r := range records {
		out = append(out, cf(r))
	}
	return out
}

func buildTable(rows [][]string, cf coerceFn) []interface{} {
	out := make([]interface{}, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToArray(row, cf))
	}
	return out
}

func buildObjects(rows [][]string, names []string, cf coerceFn) []interface{} {
	out := make([]interface{}, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToObject(row, names, cf))
	}
	return out
}

func buildKeyValue(records []string, cfg config, cf coerceFn) map[string]interface{} {
	m := make(map[string]interface{}, len(records))
	for _, line := range records {
		k, v := cfg.splitKV(line)
		m[k] = cf(v)
	}
	return m
}

func rowsToKeyValue(rows [][]string, cf coerceFn) map[string]interface{} {
	m := make(map[string]interface{}, len(rows))
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		val := ""
		if len(row) > 1 {
			val = strings.Join(row[1:], " ")
		}
		m[row[0]] = cf(val)
	}
	return m
}

func splitNames(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func resolveMode(cfg config, useCSV, useTSV bool) encodeMode {
	switch {
	case cfg.kv:
		return modeKeyValue
	case cfg.header || cfg.names != "":
		return modeObjects
	case cfg.columns || useCSV || useTSV:
		return modeColumns
	default:
		return modeLines
	}
}

// detectFormat resolves CSV/TSV, auto-detecting from a file extension unless an
// explicit format or delimiter was given.
func detectFormat(cfg config) (useCSV, useTSV bool) {
	if cfg.csv {
		return true, false
	}
	if cfg.tsv {
		return false, true
	}
	if !cfg.delimiterSet && len(cfg.files) > 0 {
		f := strings.ToLower(cfg.files[0])
		switch {
		case strings.HasSuffix(f, ".csv"):
			return true, false
		case strings.HasSuffix(f, ".tsv"), strings.HasSuffix(f, ".tab"):
			return false, true
		}
	}
	return false, false
}

func rowsOf(records []string, cfg config, idx []int) [][]string {
	rows := make([][]string, 0, len(records))
	for _, r := range records {
		rows = append(rows, selectFields(cfg.splitFields(r), idx))
	}
	return rows
}

// --- output --------------------------------------------------------------

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func useColor(cfg config, w io.Writer) bool {
	mode := cfg.color
	if cfg.noColor {
		mode = "never"
	}
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default: // auto
		if v, ok := os.LookupEnv("NO_COLOR"); ok && v != "" {
			return false
		}
		return isTerminal(w)
	}
}

func usePretty(cfg config, w io.Writer) bool {
	if cfg.compact {
		return false
	}
	if cfg.pretty {
		return true
	}
	return isTerminal(w)
}

func writeIndent(w *bufio.Writer, depth int) {
	for i := 0; i < depth; i++ {
		w.WriteString("  ")
	}
}

func sortedKeys(m map[string]interface{}) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// encodeColored writes v with ANSI colors, mirroring json.MarshalIndent layout.
func encodeColored(w *bufio.Writer, v interface{}, pretty bool, depth int) error {
	switch val := v.(type) {
	case nil:
		w.WriteString(colNull + "null" + colReset)
	case bool:
		w.WriteString(colBool + strconv.FormatBool(val) + colReset)
	case int64:
		w.WriteString(strconv.FormatInt(val, 10))
	case float64:
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}
		w.Write(b)
	case string:
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}
		w.WriteString(colStr + string(b) + colReset)
	case []interface{}:
		if len(val) == 0 {
			w.WriteString("[]")
			return nil
		}
		w.WriteByte('[')
		for i, e := range val {
			if pretty {
				w.WriteByte('\n')
				writeIndent(w, depth+1)
			}
			if err := encodeColored(w, e, pretty, depth+1); err != nil {
				return err
			}
			if i < len(val)-1 {
				w.WriteByte(',')
			}
		}
		if pretty {
			w.WriteByte('\n')
			writeIndent(w, depth)
		}
		w.WriteByte(']')
	case map[string]interface{}:
		keys := sortedKeys(val)
		if len(keys) == 0 {
			w.WriteString("{}")
			return nil
		}
		w.WriteByte('{')
		for i, k := range keys {
			if pretty {
				w.WriteByte('\n')
				writeIndent(w, depth+1)
			}
			kb, err := json.Marshal(k)
			if err != nil {
				return err
			}
			w.WriteString(colKey + string(kb) + colReset)
			w.WriteByte(':')
			if pretty {
				w.WriteByte(' ')
			}
			if err := encodeColored(w, val[k], pretty, depth+1); err != nil {
				return err
			}
			if i < len(keys)-1 {
				w.WriteByte(',')
			}
		}
		if pretty {
			w.WriteByte('\n')
			writeIndent(w, depth)
		}
		w.WriteByte('}')
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}
		w.Write(b)
	}
	return nil
}

func writeValue(w *bufio.Writer, v interface{}, color, pretty bool) error {
	if color {
		return encodeColored(w, v, pretty, 0)
	}
	var b []byte
	var err error
	if pretty {
		b, err = json.MarshalIndent(v, "", "  ")
	} else {
		b, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func emit(cfg config, w io.Writer, v interface{}) error {
	if cfg.wrap {
		host, _ := os.Hostname()
		v = map[string]interface{}{
			"host":      host,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"data":      v,
		}
	}
	color := useColor(cfg, w)
	pretty := usePretty(cfg, w)
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	if cfg.jsonl {
		if arr, ok := v.([]interface{}); ok {
			for _, e := range arr {
				if err := writeValue(bw, e, color, false); err != nil {
					return err
				}
				bw.WriteByte('\n')
			}
			return nil
		}
	}
	if err := writeValue(bw, v, color, pretty); err != nil {
		return err
	}
	return bw.WriteByte('\n')
}

// --- run -----------------------------------------------------------------

func fail(stderr io.Writer, code int, msg string) int {
	fmt.Fprintln(stderr, "json_encode: "+msg)
	return code
}

func run(cfg config, stdin io.Reader, stdout, stderr io.Writer) int {
	if cfg.pretty && cfg.compact {
		return fail(stderr, 2, "--pretty and --compact are mutually exclusive")
	}
	switch cfg.color {
	case "auto", "always", "never":
	default:
		return fail(stderr, 2, "--color must be auto, always or never")
	}
	idx, err := parseFieldSpec(cfg.fields)
	if err != nil {
		return fail(stderr, 2, err.Error())
	}

	out := stdout
	if cfg.output != "" {
		f, err := os.Create(cfg.output)
		if err != nil {
			return fail(stderr, 1, err.Error())
		}
		defer f.Close()
		out = f
	}
	cf := func(s string) interface{} { return coerce(s, cfg.raw) }

	if cfg.follow {
		return runStream(cfg, idx, cf, stdin, out, stderr)
	}

	input, err := readInput(cfg.files, stdin)
	if err != nil {
		return fail(stderr, 1, err.Error())
	}
	useCSV, useTSV := detectFormat(cfg)
	mode := resolveMode(cfg, useCSV, useTSV)
	names := splitNames(cfg.names)

	var value interface{}
	if useCSV || useTSV {
		rows, err := readCSVRows(input, useTSV)
		if err != nil {
			return fail(stderr, 1, err.Error())
		}
		for i := range rows {
			rows[i] = selectFields(rows[i], idx)
		}
		if cfg.header && len(rows) > 0 {
			names = rows[0]
			rows = rows[1:]
		}
		switch mode {
		case modeKeyValue:
			value = rowsToKeyValue(rows, cf)
		case modeObjects:
			value = buildObjects(rows, names, cf)
		default:
			value = buildTable(rows, cf)
		}
	} else {
		records := splitRecords(input, cfg.nullDelim)
		if cfg.header && mode == modeObjects && cfg.names == "" && len(records) > 0 {
			names = selectFields(cfg.splitFields(records[0]), idx)
			records = records[1:]
		}
		switch mode {
		case modeKeyValue:
			value = buildKeyValue(records, cfg, cf)
		case modeObjects:
			value = buildObjects(rowsOf(records, cfg, idx), names, cf)
		case modeColumns:
			value = buildTable(rowsOf(records, cfg, idx), cf)
		default:
			value = buildLines(records, cf)
		}
	}

	if err := emit(cfg, out, value); err != nil {
		return fail(stderr, 1, err.Error())
	}
	return 0
}

// runStream reads stdin line by line and emits one JSON value per line as it
// arrives, so it works with tail -f and other never-ending pipes.
func runStream(cfg config, idx []int, cf coerceFn, stdin io.Reader, out, stderr io.Writer) int {
	mode := resolveMode(cfg, false, false)
	names := splitNames(cfg.names)
	headerPending := cfg.header && cfg.names == ""
	color := useColor(cfg, out)

	sc := bufio.NewScanner(stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	bw := bufio.NewWriter(out)
	defer bw.Flush()

	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if line == "" {
			continue
		}
		if headerPending {
			names = selectFields(cfg.splitFields(line), idx)
			headerPending = false
			continue
		}
		var v interface{}
		switch mode {
		case modeKeyValue:
			k, val := cfg.splitKV(line)
			v = map[string]interface{}{k: cf(val)}
		case modeObjects:
			v = rowToObject(selectFields(cfg.splitFields(line), idx), names, cf)
		case modeColumns:
			v = rowToArray(selectFields(cfg.splitFields(line), idx), cf)
		default:
			v = cf(line)
		}
		if err := writeValue(bw, v, color, false); err != nil {
			return fail(stderr, 1, err.Error())
		}
		bw.WriteByte('\n')
		bw.Flush()
	}
	if err := sc.Err(); err != nil {
		return fail(stderr, 1, err.Error())
	}
	return 0
}

func realMain(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := parseArgs(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usageText())
			return 0
		}
		fmt.Fprintln(stderr, "json_encode: "+err.Error())
		fmt.Fprint(stderr, usageText())
		return 2
	}
	if cfg.version {
		fmt.Fprintf(stdout, "json_encode %s\n", version)
		return 0
	}
	return run(cfg, stdin, stdout, stderr)
}

func main() {
	os.Exit(realMain(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
