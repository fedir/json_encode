// Copyright 2018 Fedir RYKHTIK. All rights reserved.
// Use of this source code is governed by the GNU GPL 3.0
// license that can be found in the LICENSE file.
package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

var (
	useSimpleColumns = flag.Bool("sc", false, "Simple columns: split each line into an array of fields")
	useKeyValue      = flag.Bool("kv", false, "Key-value mode: first field becomes key, remainder becomes value")
	separator        = flag.String("s", " ", "Separator for splitting lines into fields")
	printPretty      = flag.Bool("p", false, "Pretty-print the JSON output")
	columnNames      = flag.String("cols", "", "Comma-separated column names; emit an array of objects")
	useHeader        = flag.Bool("header", false, "Use the first input line as the object keys")
	inferTypes       = flag.Bool("t", false, "Infer numbers, booleans and null instead of strings")
	ndjson           = flag.Bool("nd", false, "Emit newline-delimited JSON (one value per line)")
	streamMode       = flag.Bool("stream", false, "Stream line-by-line; emit each line as it arrives (great with tail -f)")
	useWhitespace    = flag.Bool("w", false, "Split on runs of whitespace, awk-style (ignores -s)")
	fieldSpec        = flag.String("f", "", "Select fields by 1-based index, e.g. 1,3,6")
	useCSV           = flag.Bool("csv", false, "Parse input as CSV, honouring quoted fields")
	useTSV           = flag.Bool("tsv", false, "Parse input as TSV (tab-separated), honouring quoted fields")
	nulDelim         = flag.Bool("0", false, "Split input on NUL bytes instead of newlines (pairs with find -print0)")
	wrapObject       = flag.Bool("o", false, "Wrap the output in an object with host, timestamp and data fields")
	showVersion      = flag.Bool("version", false, "Print version and exit")
	showVersionShort = flag.Bool("v", false, "Print version and exit (alias for -version)")
)

func init() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `json_encode — turn shell output into JSON, one pipe away.

Usage: json_encode [flags] [file ...]

With no file arguments, input is read from stdin.

Flags:
`)
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, `
Examples:
  seq 1 5 | json_encode                          ["1","2","3","4","5"]
  printf 'a 1\nb 2\n' | json_encode -sc          [["a","1"],["b","2"]]
  ps -eo pid,comm | json_encode -sc -w -header   [{"PID":"1","COMMAND":"systemd"},...]
  printf 'x 200\n' | json_encode -kv -t          {"x":200}
  printf 'a\nb\n'  | json_encode -nd             "a"<newline>"b"
  tail -f app.log | json_encode -stream -nd      stream one JSON line per log line
`)
	}
}

type encodeMode int

const (
	modeLines encodeMode = iota
	modeColumns
	modeObjects
	modeKeyValue
)

// GetInputData gets data from stdin. Retained for backward compatibility.
func GetInputData() string {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	return string(data)
}

// readInput reads from the named file arguments, or stdin when none are given.
func readInput() string {
	files := flag.Args()
	if len(files) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal("Cannot read stdin: ", err)
		}
		return string(data)
	}
	var sb strings.Builder
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			log.Fatalf("Cannot read %s: %v", f, err)
		}
		sb.Write(b)
	}
	return sb.String()
}

// ConvertInputToLines converts single input into separated elements.
func ConvertInputToLines(inputString string) []string {
	inputLines := make([]string, 0)
	lines := strings.Split(inputString, "\n")
	for _, line := range lines {
		if line != "" {
			inputLines = append(inputLines, line)
		}
	}
	return inputLines
}

// splitRecords splits raw input into non-empty records, on NUL or newline.
func splitRecords(input string) []string {
	sep := "\n"
	if *nulDelim {
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

// ConvertLinesToTable converts each line into a slice of fields.
func ConvertLinesToTable(inputLines []string) [][]string {
	linesWithColumns := make([][]string, 0)
	for _, inputLine := range inputLines {
		if inputLine != "" {
			linesWithColumns = append(linesWithColumns, strings.Split(inputLine, *separator))
		}
	}
	return linesWithColumns
}

// ConvertLinesToKeyValue maps the first field to the remainder. Retained for
// backward compatibility; the live path uses splitKV/coerce.
func ConvertLinesToKeyValue(inputLines []string) map[string]string {
	result := make(map[string]string, len(inputLines))
	for _, line := range inputLines {
		parts := strings.SplitN(line, *separator, 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		} else {
			result[parts[0]] = ""
		}
	}
	return result
}

// ConvertToJSON converts a value into JSON, honouring -p.
func ConvertToJSON(v interface{}) []byte {
	var err error
	var JSON []byte
	if *printPretty {
		JSON, err = json.MarshalIndent(v, "", "  ")
	} else {
		JSON, err = json.Marshal(v)
	}
	if err != nil {
		log.Fatal("Cannot encode to JSON ", err)
	}
	return JSON
}

// coerce optionally turns a string into a number, bool or null when -t is set.
func coerce(s string) interface{} {
	if !*inferTypes {
		return s
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true":
		return true
	case "false":
		return false
	case "null", "nil", "~":
		return nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsInf(f, 0) && !math.IsNaN(f) {
		return f
	}
	return s
}

// splitFields splits one record into fields, honouring -w and -s.
func splitFields(line string) []string {
	if *useWhitespace {
		return strings.Fields(line)
	}
	return strings.Split(line, *separator)
}

// splitKV splits one record into key and value (value is the remainder).
func splitKV(line string) (string, string) {
	if *useWhitespace {
		trimmed := strings.TrimLeft(line, " \t\r\f\v")
		i := strings.IndexFunc(trimmed, unicode.IsSpace)
		if i < 0 {
			return trimmed, ""
		}
		return trimmed[:i], strings.TrimLeft(trimmed[i:], " \t\r\f\v")
	}
	parts := strings.SplitN(line, *separator, 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

// parseFieldSpec parses "1,3,6" into a list of 1-based indices.
func parseFieldSpec(spec string) []int {
	if spec == "" {
		return nil
	}
	parts := strings.Split(spec, ",")
	idx := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			log.Fatalf("Invalid -f field index %q: %v", p, err)
		}
		idx = append(idx, n)
	}
	return idx
}

// selectFields keeps only the requested 1-based field indices, in order.
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

// readCSVRows parses the whole input as CSV/TSV into rows of fields.
func readCSVRows(input string) [][]string {
	r := csv.NewReader(strings.NewReader(input))
	r.FieldsPerRecord = -1
	if *useTSV {
		r.Comma = '\t'
	}
	rows, err := r.ReadAll()
	if err != nil {
		log.Fatal("Cannot parse CSV ", err)
	}
	return rows
}

func columnKey(names []string, i int) string {
	if i < len(names) && names[i] != "" {
		return names[i]
	}
	return "col" + strconv.Itoa(i+1)
}

func rowToObject(row []string, names []string) map[string]interface{} {
	m := make(map[string]interface{}, len(row))
	for i, f := range row {
		m[columnKey(names, i)] = coerce(f)
	}
	return m
}

func rowToArray(row []string) []interface{} {
	out := make([]interface{}, 0, len(row))
	for _, f := range row {
		out = append(out, coerce(f))
	}
	return out
}

func buildLines(records []string) []interface{} {
	out := make([]interface{}, 0, len(records))
	for _, r := range records {
		out = append(out, coerce(r))
	}
	return out
}

func buildTable(rows [][]string) []interface{} {
	out := make([]interface{}, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToArray(row))
	}
	return out
}

func buildObjects(rows [][]string, names []string) []interface{} {
	out := make([]interface{}, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToObject(row, names))
	}
	return out
}

func buildKeyValue(records []string) map[string]interface{} {
	m := make(map[string]interface{}, len(records))
	for _, line := range records {
		k, v := splitKV(line)
		m[k] = coerce(v)
	}
	return m
}

func rowsToKeyValue(rows [][]string) map[string]interface{} {
	m := make(map[string]interface{}, len(rows))
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		val := ""
		if len(row) > 1 {
			val = strings.Join(row[1:], " ")
		}
		m[row[0]] = coerce(val)
	}
	return m
}

func resolveMode() encodeMode {
	switch {
	case *useKeyValue:
		return modeKeyValue
	case *columnNames != "" || *useHeader:
		return modeObjects
	case *useSimpleColumns || *useCSV || *useTSV:
		return modeColumns
	default:
		return modeLines
	}
}

func columnNamesFromFlag() []string {
	if *columnNames == "" {
		return nil
	}
	return strings.Split(*columnNames, ",")
}

// output writes the final value, honouring -o (wrap) and -nd (ndjson).
func output(v interface{}) {
	if *wrapObject {
		host, _ := os.Hostname()
		v = map[string]interface{}{
			"host":      host,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"data":      v,
		}
	}
	if *ndjson {
		if arr, ok := v.([]interface{}); ok {
			w := bufio.NewWriter(os.Stdout)
			defer w.Flush()
			for _, e := range arr {
				b, err := json.Marshal(e)
				if err != nil {
					log.Fatal("Cannot encode to JSON ", err)
				}
				w.Write(b)
				w.WriteByte('\n')
			}
			return
		}
	}
	fmt.Fprintf(os.Stdout, "%s\n", ConvertToJSON(v))
}

// runBatch reads all input and emits a single JSON document (or ndjson).
func runBatch() {
	input := readInput()
	mode := resolveMode()
	names := columnNamesFromFlag()
	idx := parseFieldSpec(*fieldSpec)

	if *useCSV || *useTSV {
		rows := readCSVRows(input)
		for i := range rows {
			rows[i] = selectFields(rows[i], idx)
		}
		if *useHeader && len(rows) > 0 {
			names = rows[0]
			rows = rows[1:]
		}
		switch mode {
		case modeKeyValue:
			output(rowsToKeyValue(rows))
		case modeObjects:
			output(buildObjects(rows, names))
		default:
			output(buildTable(rows))
		}
		return
	}

	records := splitRecords(input)

	if *useHeader && mode == modeObjects && names == nil && len(records) > 0 {
		names = selectFields(splitFields(records[0]), idx)
		records = records[1:]
	}

	switch mode {
	case modeKeyValue:
		output(buildKeyValue(records))
	case modeObjects:
		rows := make([][]string, 0, len(records))
		for _, r := range records {
			rows = append(rows, selectFields(splitFields(r), idx))
		}
		output(buildObjects(rows, names))
	case modeColumns:
		rows := make([][]string, 0, len(records))
		for _, r := range records {
			rows = append(rows, selectFields(splitFields(r), idx))
		}
		output(buildTable(rows))
	default:
		output(buildLines(records))
	}
}

// runStream reads stdin line by line and emits one JSON value per line as it
// arrives, so it works with tail -f and other never-ending pipes.
func runStream() {
	mode := resolveMode()
	names := columnNamesFromFlag()
	idx := parseFieldSpec(*fieldSpec)
	headerPending := *useHeader && names == nil

	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if line == "" {
			continue
		}
		if headerPending {
			names = selectFields(splitFields(line), idx)
			headerPending = false
			continue
		}
		var v interface{}
		switch mode {
		case modeKeyValue:
			k, val := splitKV(line)
			v = map[string]interface{}{k: coerce(val)}
		case modeObjects:
			v = rowToObject(selectFields(splitFields(line), idx), names)
		case modeColumns:
			v = rowToArray(selectFields(splitFields(line), idx))
		default:
			v = coerce(line)
		}
		b, err := json.Marshal(v)
		if err != nil {
			log.Fatal("Cannot encode to JSON ", err)
		}
		w.Write(b)
		w.WriteByte('\n')
		w.Flush()
	}
	if err := sc.Err(); err != nil {
		log.Fatal("Cannot read stdin: ", err)
	}
}

func main() {
	flag.Parse()

	if *showVersion || *showVersionShort {
		fmt.Fprintf(os.Stdout, "json_encode %s\n", version)
		return
	}

	if *streamMode {
		runStream()
		return
	}

	runBatch()
}
