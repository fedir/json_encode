// Copyright 2018 Fedir RYKHTIK. All rights reserved.
// Use of this source code is governed by the GNU GPL 3.0
// license that can be found in the LICENSE file.
package main

import (
	"bytes"
	"strings"
	"testing"
)

// cli runs the program end-to-end with in-memory streams (never a TTY, so the
// output is compact and uncolored — deterministic for assertions).
func cli(args []string, stdin string) (stdout, stderr string, code int) {
	var out, errb bytes.Buffer
	code = realMain(args, strings.NewReader(stdin), &out, &errb)
	return out.String(), errb.String(), code
}

func TestModes(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		stdin string
		want  string
	}{
		{"lines", nil, "a\nb\nc\n", `["a","b","c"]` + "\n"},
		{"lines typed", nil, "1\n2\n3\n", `[1,2,3]` + "\n"},
		{"columns whitespace default", []string{"-c"}, "  1   systemd  0.0\n", `[[1,"systemd","0.0"]]` + "\n"},
		{"columns delimiter", []string{"-c", "-d", ","}, "a,b\nc,d\n", `[["a","b"],["c","d"]]` + "\n"},
		{"names objects", []string{"-n", "k,v"}, "a 1\n", `[{"k":"a","v":1}]` + "\n"},
		{"header", []string{"-H"}, "NAME PID\nnginx 12\n", `[{"NAME":"nginx","PID":12}]` + "\n"},
		{"kv", []string{"-k"}, "host db\nport 5432\n", `{"host":"db","port":5432}` + "\n"},
		{"kv roundtrip keeps leading zero", []string{"-k"}, "id 007\nstatus 200\n", `{"id":"007","status":200}` + "\n"},
		{"raw keeps strings", []string{"-k", "--raw"}, "status 200\n", `{"status":"200"}` + "\n"},
		{"jsonl", []string{"-l"}, "a\nb\n", "\"a\"\n\"b\"\n"},
		{"fields range", []string{"-c", "-d", ":", "-f", "1-2,4"}, "a:b:c:d\n", `[["a","b","d"]]` + "\n"},
		{"csv quoted", []string{"--csv"}, "\"Smith, John\",42\n", `[["Smith, John",42]]` + "\n"},
		{"null delimited", []string{"-0"}, "a b\x00c d\x00", `["a b","c d"]` + "\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, errb, code := cli(tc.args, tc.stdin)
			if code != 0 {
				t.Fatalf("exit %d, stderr=%q", code, errb)
			}
			if out != tc.want {
				t.Errorf("got %q, want %q", out, tc.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	out, _, code := cli([]string{"--wrap"}, "a\n")
	if code != 0 {
		t.Fatal(code)
	}
	for _, sub := range []string{`"data":["a"]`, `"host":`, `"timestamp":`} {
		if !strings.Contains(out, sub) {
			t.Errorf("missing %q in %q", sub, out)
		}
	}
}

func TestColor(t *testing.T) {
	// piped (buffer, not a TTY) → no ANSI by default
	if out, _, _ := cli(nil, "a\n"); strings.Contains(out, "\x1b[") {
		t.Errorf("unexpected ANSI in piped output: %q", out)
	}
	// --color=always forces ANSI even when piped
	if out, _, _ := cli([]string{"--color=always"}, "a\n"); !strings.Contains(out, "\x1b[") {
		t.Errorf("expected ANSI with --color=always: %q", out)
	}
}

func TestPretty(t *testing.T) {
	out, _, code := cli([]string{"-p"}, "a\nb\n")
	if code != 0 {
		t.Fatal(code)
	}
	if !strings.Contains(out, "[\n  \"a\",\n  \"b\"\n]") {
		t.Errorf("not pretty: %q", out)
	}
}

func TestVersion(t *testing.T) {
	out, _, code := cli([]string{"--version"}, "")
	if code != 0 || !strings.HasPrefix(out, "json_encode ") {
		t.Errorf("version: code=%d out=%q", code, out)
	}
}

func TestErrors(t *testing.T) {
	cases := [][]string{
		{"--color=weird"},
		{"-p", "--compact"},
		{"-c", "-f", "0"},
		{"--bogus"},
	}
	for _, args := range cases {
		if _, errb, code := cli(args, "a\n"); code == 0 || errb == "" {
			t.Errorf("args %v: expected non-zero exit and stderr", args)
		}
	}
}

func TestCoerceRoundTrip(t *testing.T) {
	cases := map[string]interface{}{
		"200":   int64(200),
		"3.14":  3.14,
		"true":  true,
		"false": false,
		"null":  nil,
		"007":   "007",   // leading zero
		"1.50":  "1.50",  // non-canonical float
		"+5":    "+5",    // sign prefix
		"1.2.3": "1.2.3", // version
		"08:00": "08:00", // time
		"":      "",      // empty
		"NaN":   "NaN",   // not a JSON number
		"1_000": "1_000", // underscore
		"0xFF":  "0xFF",  // hex
	}
	for in, want := range cases {
		if got := coerce(in, false); got != want {
			t.Errorf("coerce(%q) = %v (%T), want %v (%T)", in, got, got, want, want)
		}
	}
	if got := coerce("200", true); got != "200" {
		t.Errorf("raw coerce should keep string, got %v", got)
	}
}
