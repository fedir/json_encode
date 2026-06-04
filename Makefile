# Use bash so $'\t' (ANSI-C quoting) works; on Ubuntu CI /bin/sh is dash, which
# does not support it and would pass a literal "$\t" as the separator.
SHELL := /bin/bash

BINARY=json_encode
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test -race ./...

vet:
	go vet ./...

functional-test: build
	# basic
	echo "a b c" | ./$(BINARY) | grep -qxF '["a b c"]' && echo "PASS: lines mode" || (echo "FAIL: lines mode"; exit 1)
	printf "a b\nc d\n" | ./$(BINARY) -sc | grep -qxF '[["a","b"],["c","d"]]' && echo "PASS: simple columns mode" || (echo "FAIL: simple columns mode"; exit 1)
	printf "a,b\nc,d\n" | ./$(BINARY) -sc -s , | grep -qxF '[["a","b"],["c","d"]]' && echo "PASS: custom separator" || (echo "FAIL: custom separator"; exit 1)
	printf "x\ny\n" | ./$(BINARY) -p | grep -q '"x"' && echo "PASS: pretty-print" || (echo "FAIL: pretty-print"; exit 1)
	# process list: collect PIDs into a JSON array
	printf "1234\n5678\n9012\n" | ./$(BINARY) | grep -qxF '["1234","5678","9012"]' && echo "PASS: pid list" || (echo "FAIL: pid list"; exit 1)
	# /etc/hosts-style: IP + hostname rows split into columns
	printf "127.0.0.1 localhost\n192.168.1.1 gateway\n" | ./$(BINARY) -sc | grep -qxF '[["127.0.0.1","localhost"],["192.168.1.1","gateway"]]' && echo "PASS: hosts file columns" || (echo "FAIL: hosts file columns"; exit 1)
	# CSV-like: colon-separated /etc/passwd fields
	printf "root:0:root\nnobody:65534:nobody\n" | ./$(BINARY) -sc -s : | grep -qxF '[["root","0","root"],["nobody","65534","nobody"]]' && echo "PASS: colon-separated fields" || (echo "FAIL: colon-separated fields"; exit 1)
	# df output: filesystem usage rows
	printf "/dev/sda1 ext4 50G 20G 30G\n/dev/sdb1 xfs 100G 40G 60G\n" | ./$(BINARY) -sc | grep -qxF '[["/dev/sda1","ext4","50G","20G","30G"],["/dev/sdb1","xfs","100G","40G","60G"]]' && echo "PASS: df-style rows" || (echo "FAIL: df-style rows"; exit 1)
	# env vars: KEY=VALUE pairs split on =
	printf "HOME=/root\nPATH=/usr/bin\n" | ./$(BINARY) -sc -s = | grep -qxF '[["HOME","/root"],["PATH","/usr/bin"]]' && echo "PASS: env var pairs" || (echo "FAIL: env var pairs"; exit 1)
	# empty input: should produce an empty JSON array
	printf "" | ./$(BINARY) | grep -qxF '[]' && echo "PASS: empty input" || (echo "FAIL: empty input"; exit 1)
	# syslog: extract ERRORs from a mixed log into a JSON array for alerting
	printf "Jan 1 00:01:01 host sshd: Accepted\nJan 1 00:01:02 host kernel: ERROR disk failure\nJan 1 00:01:03 host cron: ERROR job failed\n" \
		| grep ERROR | ./$(BINARY) \
		| grep -qF '"Jan 1 00:01:02 host kernel: ERROR disk failure"' && echo "PASS: syslog error extraction" || (echo "FAIL: syslog error extraction"; exit 1)
	# ss/netstat: tab-separated connection rows — proto, local addr, peer addr, state
	printf "tcp\t0.0.0.0:22\t10.0.0.5:54321\tESTABLISHED\ntcp\t0.0.0.0:80\t10.0.0.9:43210\tESTABLISHED\n" \
		| ./$(BINARY) -sc -s $$'\t' \
		| grep -qxF '[["tcp","0.0.0.0:22","10.0.0.5:54321","ESTABLISHED"],["tcp","0.0.0.0:80","10.0.0.9:43210","ESTABLISHED"]]' \
		&& echo "PASS: netstat tab-separated" || (echo "FAIL: netstat tab-separated"; exit 1)
	# ip route: pipe-separated route table fields (dst, gw, dev, metric)
	printf "10.0.0.0/8|192.168.1.1|eth0|100\n0.0.0.0/0|192.168.1.254|eth0|0\n" \
		| ./$(BINARY) -sc -s '|' \
		| grep -qxF '[["10.0.0.0/8","192.168.1.1","eth0","100"],["0.0.0.0/0","192.168.1.254","eth0","0"]]' \
		&& echo "PASS: ip route table" || (echo "FAIL: ip route table"; exit 1)
	# docker ps: tab-separated container id, image, status
	printf "a1b2c3d4\tnginx:latest\tUp 2 hours\nb5c6d7e8\tredis:7\tUp 5 days\n" \
		| ./$(BINARY) -sc -s $$'\t' \
		| grep -qxF '[["a1b2c3d4","nginx:latest","Up 2 hours"],["b5c6d7e8","redis:7","Up 5 days"]]' \
		&& echo "PASS: docker ps columns" || (echo "FAIL: docker ps columns"; exit 1)
	# systemctl: service:state pairs — collect names of failed units
	printf "sshd.service:active\nnginx.service:failed\nmysql.service:failed\n" \
		| grep failed | ./$(BINARY) -sc -s : \
		| grep -qxF '[["nginx.service","failed"],["mysql.service","failed"]]' \
		&& echo "PASS: failed systemctl units" || (echo "FAIL: failed systemctl units"; exit 1)
	# /proc/meminfo style: key:value with spaces, collect keys as array
	printf "MemTotal: 16384 kB\nMemFree: 8192 kB\nSwapTotal: 4096 kB\n" \
		| awk '{print $$1}' | ./$(BINARY) \
		| grep -qxF '["MemTotal:","MemFree:","SwapTotal:"]' \
		&& echo "PASS: meminfo key extraction" || (echo "FAIL: meminfo key extraction"; exit 1)
	# crontab: lines collected verbatim as array for auditing
	printf "0 2 * * * /usr/bin/backup.sh\n30 6 * * 1 /usr/bin/report.sh\n" \
		| ./$(BINARY) \
		| grep -qF '"0 2 * * * /usr/bin/backup.sh"' \
		&& echo "PASS: crontab audit" || (echo "FAIL: crontab audit"; exit 1)
	# key-value: env vars KEY=VALUE → JSON object
	out=$$(printf "HOST=db.internal\nPORT=5432\nDB=myapp\n" | ./$(BINARY) -kv -s =); \
	echo "$$out" | grep -qF '"HOST":"db.internal"' && \
	echo "$$out" | grep -qF '"PORT":"5432"' && \
	echo "$$out" | grep -qF '"DB":"myapp"' && \
	echo "PASS: kv env vars" || (echo "FAIL: kv env vars"; exit 1)
	# key-value: /proc/meminfo — "MemTotal: 16384 kB" → {"MemTotal:": "16384 kB"}
	printf "MemTotal: 16384 kB\nMemFree: 8192 kB\n" \
		| ./$(BINARY) -kv \
		| grep -qF '"MemTotal:":"16384 kB"' \
		&& echo "PASS: kv meminfo" || (echo "FAIL: kv meminfo"; exit 1)
	# key-value: /etc/os-release style — PRETTY_NAME value contains spaces
	out=$$(printf 'ID=ubuntu\nVERSION_ID=22.04\nPRETTY_NAME=Ubuntu 22.04 LTS\n' | ./$(BINARY) -kv -s =); \
	echo "$$out" | grep -qF '"ID":"ubuntu"' && \
	echo "$$out" | grep -qF '"PRETTY_NAME":"Ubuntu 22.04 LTS"' && \
	echo "PASS: kv os-release" || (echo "FAIL: kv os-release"; exit 1)
	# key-value: tab-separated — service name + PID from hypothetical supervisor output
	out=$$(printf "nginx\t1234\npostgres\t5678\n" | ./$(BINARY) -kv -s $$'\t'); \
	echo "$$out" | grep -qF '"nginx":"1234"' && \
	echo "$$out" | grep -qF '"postgres":"5678"' && \
	echo "PASS: kv tab service-pid" || (echo "FAIL: kv tab service-pid"; exit 1)
	# loki batch: raw lines produce a valid Loki push payload (has streams, values, ns timestamp)
	out=$$(printf '10.0.0.1 - - [01/Jun/2025:12:00:00 +0000] "GET /health HTTP/1.1" 200 5\n10.0.0.2 - - [01/Jun/2025:12:00:01 +0000] "POST /api HTTP/1.1" 201 42\n' \
		| ./$(BINARY) \
		| jq -c --arg job nginx --arg host testhost \
		    '{streams:[{stream:{job:$$job,host:$$host},values:[.[]|[(now*1e9|tostring),.]] }]}'); \
	echo "$$out" | jq -e '.streams[0].stream.job == "nginx"' > /dev/null && \
	echo "$$out" | jq -e '.streams[0].values | length == 2' > /dev/null && \
	echo "$$out" | jq -e '.streams[0].values[0] | length == 2' > /dev/null && \
	echo "$$out" | jq -e '.streams[0].values[0][1] | contains("GET /health")' > /dev/null && \
	echo "PASS: loki batch payload" || (echo "FAIL: loki batch payload"; exit 1)
	# loki structured: parsed fields produce per-line label strings
	out=$$(printf '10.0.0.1 - - [01/Jun/2025:12:00:00 +0000] "GET /health HTTP/1.1" 200 5\n10.0.0.2 - - [01/Jun/2025:12:00:01 +0000] "POST /api HTTP/1.1" 404 12\n' | awk '{print $$1"|"$$7"|"$$9}' | ./$(BINARY) -sc -s '|' | jq -c --arg host testhost '[.[]|{ip:.[0],path:.[1],status:.[2]}]|{streams:[{stream:{job:"nginx",host:$$host},values:[.[]|[(now*1e9|tostring),("ip="+.ip+" path="+.path+" status="+.status)]]}]}'); \
	echo "$$out" | jq -e '.streams[0].values | length == 2' > /dev/null && \
	echo "$$out" | jq -e '.streams[0].values[0][1] | startswith("ip=10.0.0.1")' > /dev/null && \
	echo "$$out" | jq -e '.streams[0].values[1][1] | contains("status=404")' > /dev/null && \
	echo "PASS: loki structured payload" || (echo "FAIL: loki structured payload"; exit 1)
	# loki single line: real-time path produces a single-value stream
	out=$$(printf '10.0.0.1 - - [01/Jun/2025:12:00:00 +0000] "DELETE /item/9 HTTP/1.1" 204 0\n' | ./$(BINARY) | jq -c --arg host testhost '{streams:[{stream:{job:"nginx",host:$$host},values:[[(now*1e9|tostring),.[0]]]}]}'); \
	echo "$$out" | jq -e '.streams[0].values | length == 1' > /dev/null && \
	echo "$$out" | jq -e '.streams[0].values[0][1] | contains("DELETE")' > /dev/null && \
	echo "PASS: loki single-line payload" || (echo "FAIL: loki single-line payload"; exit 1)

	# objects: -cols names produce an array of objects
	printf "a 1\nb 2\n" | ./$(BINARY) -sc -cols k,v | grep -qxF '[{"k":"a","v":"1"},{"k":"b","v":"2"}]' && echo "PASS: objects -cols" || (echo "FAIL: objects -cols"; exit 1)
	# objects: -header consumes the first line as keys, -w handles padded columns
	printf "NAME PID\nnginx 12\n" | ./$(BINARY) -header -w | grep -qxF '[{"NAME":"nginx","PID":"12"}]' && echo "PASS: objects -header" || (echo "FAIL: objects -header"; exit 1)
	# type inference: numbers and booleans become real JSON scalars
	printf "x 200\nz true\n" | ./$(BINARY) -kv -t | grep -qxF '{"x":200,"z":true}' && echo "PASS: type inference" || (echo "FAIL: type inference"; exit 1)
	# ndjson: one JSON value per line, no enclosing array
	out=$$(printf "a\nb\n" | ./$(BINARY) -nd); \
	[ "$$(printf '%s\n' "$$out" | wc -l | tr -d ' ')" = "2" ] && \
	printf '%s\n' "$$out" | grep -qxF '"a"' && \
	printf '%s\n' "$$out" | grep -qxF '"b"' && \
	echo "PASS: ndjson" || (echo "FAIL: ndjson"; exit 1)
	# whitespace split: runs of spaces collapse like awk (no `tr -s` needed)
	printf "  1   systemd   0.0\n" | ./$(BINARY) -sc -w | grep -qxF '[["1","systemd","0.0"]]' && echo "PASS: whitespace split" || (echo "FAIL: whitespace split"; exit 1)
	# field selection: pick 1-based columns after splitting
	printf "root:0:root\n" | ./$(BINARY) -sc -s : -f 1,3 | grep -qxF '[["root","root"]]' && echo "PASS: field selection" || (echo "FAIL: field selection"; exit 1)
	# csv: quoted field containing the separator stays intact
	printf '"Smith, John",42\n' | ./$(BINARY) -csv | grep -qxF '[["Smith, John","42"]]' && echo "PASS: csv quoted" || (echo "FAIL: csv quoted"; exit 1)
	# tsv: tab-separated parsing
	printf 'a\tb\n' | ./$(BINARY) -tsv | grep -qxF '[["a","b"]]' && echo "PASS: tsv" || (echo "FAIL: tsv"; exit 1)
	# NUL-delimited input: pairs with find -print0, survives spaces
	printf 'a b\0c d\0' | ./$(BINARY) -0 | grep -qxF '["a b","c d"]' && echo "PASS: nul-delimited" || (echo "FAIL: nul-delimited"; exit 1)
	# streaming: emits one JSON value per line (works with tail -f)
	printf 'one\ntwo\n' | ./$(BINARY) -stream -nd | grep -qxF '"two"' && echo "PASS: stream ndjson" || (echo "FAIL: stream ndjson"; exit 1)
	# wrap: -o attaches host/timestamp metadata around the data
	printf 'a\n' | ./$(BINARY) -o | jq -e '.data == ["a"] and has("host") and has("timestamp")' > /dev/null && echo "PASS: wrap object" || (echo "FAIL: wrap object"; exit 1)
	# version: prints the embedded build version
	./$(BINARY) -version | grep -qF 'json_encode' && echo "PASS: version" || (echo "FAIL: version"; exit 1)

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -f $(BINARY) coverage.out
	rm -rf dist

.PHONY: build test vet clean functional-test release snapshot
