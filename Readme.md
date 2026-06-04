# json_encode for shell

[![Build Status](https://travis-ci.org/fedir/json_encode.svg?branch=master)](https://travis-ci.org/fedir/json_encode)
[![Code Coverage](https://codecov.io/gh/fedir/json_encode/branch/master/graph/badge.svg)](https://codecov.io/gh/fedir/json_encode)
[![Go Report Card](https://goreportcard.com/badge/github.com/fedir/json_encode)](https://goreportcard.com/report/github.com/fedir/json_encode)
[![GoDoc](https://godoc.org/github.com/fedir/json_encode?status.svg)](https://godoc.org/github.com/fedir/json_encode)
[![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

Turn any shell output into JSON — one pipe away.

Useful for sysadmins, DevOps, and developers who need to feed shell data into APIs, monitoring systems, ELK, dashboards, or `jq` pipelines.

## Installation

```bash
git clone https://github.com/fedir/json_encode.git
cd json_encode
make build
# copy to your PATH
cp json_encode /usr/local/bin/
```

Requires Go 1.16+.

## Arguments

| Flag | Default | Description |
|------|---------|-------------|
| `-s` | `" "` | Separator used to split each line into columns |
| `-sc` | off | Column mode — split each line into an array of fields |
| `-kv` | off | Key-value mode — first field becomes the key, remainder becomes the value |
| `-p` | off | Pretty-print the JSON output |

Modes are mutually exclusive: `-kv` takes priority over `-sc`; without either, each line becomes a string element.

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
