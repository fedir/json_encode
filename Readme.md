# json_encode for shell

[![CI](https://github.com/fedir/json_encode/actions/workflows/ci.yml/badge.svg)](https://github.com/fedir/json_encode/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/fedir/json_encode)](https://goreportcard.com/report/github.com/fedir/json_encode)
[![GoDoc](https://godoc.org/github.com/fedir/json_encode?status.svg)](https://godoc.org/github.com/fedir/json_encode)
[![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

Turn any shell output into JSON — one pipe away.

Useful for sysadmins, DevOps, and developers who need to feed shell data into APIs, monitoring systems, ELK, dashboards, or `jq` pipelines.

## Installation

**Go install** (requires Go 1.16+):

```bash
go install github.com/fedir/json_encode@latest
```

**Homebrew:**

```bash
brew install fedir/tap/json_encode
```

**Pre-built binaries** — grab one for your OS/arch from the
[releases page](https://github.com/fedir/json_encode/releases).

**From source:**

```bash
git clone https://github.com/fedir/json_encode.git
cd json_encode
make build
cp json_encode /usr/local/bin/
```

## Arguments

| Flag | Default | Description |
|------|---------|-------------|
| `-s` | `" "` | Separator used to split each line into fields |
| `-sc` | off | Column mode — split each line into an array of fields |
| `-kv` | off | Key-value mode — first field becomes the key, remainder becomes the value |
| `-cols` | — | Comma-separated column names — emit an **array of objects** |
| `-header` | off | Use the first input line as the object keys |
| `-t` | off | Infer numbers, booleans and `null` instead of quoting everything as strings |
| `-w` | off | Split on **runs of whitespace** (awk-style) — ignores `-s` |
| `-f` | — | Select fields by 1-based index, e.g. `-f 1,3,6` |
| `-csv` / `-tsv` | off | Parse input as CSV/TSV, honouring quoted fields |
| `-0` | off | Split input on NUL bytes (pairs with `find -print0`) |
| `-nd` | off | Emit newline-delimited JSON (one value per line) |
| `-stream` | off | Stream line-by-line; emit each line as it arrives (works with `tail -f`) |
| `-o` | off | Wrap output in an object with `host`, `timestamp` and `data` |
| `-p` | off | Pretty-print the JSON output |
| `-version` | — | Print version and exit |

**Mode precedence:** `-kv` → `-cols`/`-header` (objects) → `-sc`/`-csv`/`-tsv` (columns) → lines.
Input comes from stdin, or from file arguments if given: `json_encode FILE...`.

## Basic usage

**Lines → JSON array**

```bash
seq 1 5 | json_encode
["1","2","3","4","5"]
```

**Columns → array of arrays** (`-sc`)

```bash
echo -e "alice 30\nbob 25" | json_encode -sc
[["alice","30"],["bob","25"]]
```

**Key-value → JSON object** (`-kv`)

```bash
echo -e "host db.internal\nport 5432" | json_encode -kv
{"host":"db.internal","port":"5432"}
```

**Named columns → array of objects** (`-cols` / `-header`)

```bash
echo -e "alice 30\nbob 25" | json_encode -sc -cols name,age
[{"age":"30","name":"alice"},{"age":"25","name":"bob"}]

# or let the data name itself from a header row (+ -w to handle padded columns)
ps -eo pid,comm | json_encode -header -w
[{"COMMAND":"systemd","PID":"1"},{"COMMAND":"sshd","PID":"512"},...]
```

**Real numbers and booleans** (`-t`)

```bash
echo -e "status 200\ncached true" | json_encode -kv -t
{"cached":true,"status":200}
```

**Newline-delimited JSON for log shippers** (`-nd`)

```bash
echo -e "a\nb\nc" | json_encode -nd
"a"
"b"
"c"
```

**Custom separator**

```bash
echo -e "a,b,c\nd,e,f" | json_encode -sc -s ,
[["a","b","c"],["d","e","f"]]
```

**Pretty-print**

```bash
seq 1 3 | json_encode -p
[
  "1",
  "2",
  "3"
]
```

## Advanced usage

### Collect failed systemd units for an alert

```bash
systemctl list-units --state=failed --no-legend \
  | awk '{print $1}' \
  | json_encode
["nginx.service","mysql.service"]
```

Post directly to a webhook:

```bash
curl -s -X POST https://hooks.slack.com/... \
  -H 'Content-Type: application/json' \
  -d "{\"text\": \"Failed units: $(systemctl list-units --state=failed --no-legend | awk '{print $1}' | json_encode)\"}"
```

### Parse /etc/passwd into structured rows

```bash
cut -d: -f1,3,6 /etc/passwd | json_encode -sc -s :
[["root","0","/root"],["nobody","65534","/nonexistent"],...]
```

### Snapshot running processes for a diff later

```bash
ps -eo pid,comm,pcpu --no-headers | tr -s ' ' | json_encode -sc
[["1","systemd","0.0"],["512","sshd","0.1"],...]
```

### Turn /proc/meminfo into a key-value object

```bash
grep -E 'MemTotal|MemFree|MemAvailable' /proc/meminfo \
  | awk '{print $1, $2}' \
  | json_encode -kv
{"MemAvailable:":"7502432","MemFree:":"1234","MemTotal:":"16384000"}
```

### Map running containers to their image

```bash
docker ps --format '{{.ID}}\t{{.Image}}' \
  | json_encode -kv -s $'\t'
{"a1b2c3d4":"nginx:latest","b5c6d7e8":"redis:7"}
```

### Export environment as a JSON object

```bash
env | json_encode -kv -s =
{"HOME":"/root","PATH":"/usr/bin:/bin","USER":"root",...}
```

### Active SSH sessions as a JSON array

```bash
who | awk '{print $1"@"$2}' | json_encode
["alice@pts/0","bob@pts/1"]
```

### Git log as structured records

```bash
git log --date=local --pretty=format:"%h|%an|%ad|%s" -n 3 \
  | json_encode -sc -s "|" -p
[
  ["a1b2c3d", "Alice", "Mon Jun 2 10:00:00 2025", "fix: handle timeout"],
  ["b2c3d4e", "Bob",   "Sun Jun 1 18:30:00 2025", "feat: add retry logic"]
]
```

### Filter error logs and ship to an API

```bash
journalctl -u nginx --since "1 hour ago" --no-pager \
  | grep -i error \
  | json_encode \
  | curl -s -X POST https://logs.example.com/ingest \
      -H 'Content-Type: application/json' -d @-
```

### Audit open ports

```bash
ss -tlnp | awk 'NR>1 {print $4, $6}' | json_encode -sc
[["0.0.0.0:22","users:(\"sshd\",pid=512)"],["0.0.0.0:80","users:(\"nginx\")"]]
```

### Check disk usage thresholds in a script

```bash
df -h --output=source,pcent | tail -n +2 | tr -d ' %' \
  | json_encode -kv \
  | jq 'to_entries[] | select(.value | tonumber > 80) | .key'
"/dev/sda1"
```

### Ship access.log to Loki

Loki's push API expects `{"streams":[{"stream":{labels},"values":[["timestamp_ns","line"],...]}]}`.

**Batch — send the last N lines on a schedule (e.g. from cron):**

```bash
tail -n 500 /var/log/nginx/access.log \
  | json_encode \
  | jq -c --arg job nginx --arg host "$(hostname)" \
      '{streams:[{stream:{job:$job,host:$host},
                  values:[.[] | [(now*1e9|tostring), .]]}]}' \
  | curl -s -X POST http://loki:3100/loki/api/v1/push \
        -H 'Content-Type: application/json' -d @-
```

**Structured — parse fields and attach them as Loki stream labels:**

nginx default log format: `IP - - [date] "METHOD path proto" status bytes`

```bash
tail -n 500 /var/log/nginx/access.log \
  | awk '{print $1"|"$7"|"$9}' \
  | json_encode -sc -s '|' \
  | jq -c --arg host "$(hostname)" \
      '[.[] | {ip:.[0], path:.[1], status:.[2]}] |
       {streams:[{stream:{job:"nginx",host:$host},
                  values:[.[] | [(now*1e9|tostring),
                                 ("ip="+.ip+" path="+.path+" status="+.status)]]}]}' \
  | curl -s -X POST http://loki:3100/loki/api/v1/push \
        -H 'Content-Type: application/json' -d @-
```

**Tail in real time — ship each new line as it arrives:**

`-stream` emits one JSON value per line the moment it appears (no `while read`
loop, no waiting for EOF):

```bash
tail -f /var/log/nginx/access.log \
  | json_encode -stream -nd \
  | while IFS= read -r line; do
      printf '%s' "$line" \
        | jq -c --arg host "$(hostname)" \
            '{streams:[{stream:{job:"nginx",host:$host},
                        values:[[(now*1e9|tostring), .]]}]}' \
        | curl -s -X POST http://loki:3100/loki/api/v1/push \
              -H 'Content-Type: application/json' -d @-
    done
```
