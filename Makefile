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
	printf "a b\nc d\n" | ./$(BINARY) -c | grep -qxF '[["a","b"],["c","d"]]' && echo "PASS: columns (whitespace default)" || (echo "FAIL: columns (whitespace default)"; exit 1)
	printf "a,b\nc,d\n" | ./$(BINARY) -c -d , | grep -qxF '[["a","b"],["c","d"]]' && echo "PASS: custom delimiter" || (echo "FAIL: custom delimiter"; exit 1)
	printf "x\ny\n" | ./$(BINARY) -p | grep -q '"x"' && echo "PASS: pretty-print" || (echo "FAIL: pretty-print"; exit 1)
	# smart typing: pure integers become JSON numbers
	printf "1234\n5678\n9012\n" | ./$(BINARY) | grep -qxF '[1234,5678,9012]' && echo "PASS: numbers typed" || (echo "FAIL: numbers typed"; exit 1)
	# /etc/hosts-style: IPs are not numbers, stay strings
	printf "127.0.0.1 localhost\n192.168.1.1 gateway\n" | ./$(BINARY) -c | grep -qxF '[["127.0.0.1","localhost"],["192.168.1.1","gateway"]]' && echo "PASS: hosts file columns" || (echo "FAIL: hosts file columns"; exit 1)
	# /etc/passwd colon fields: uid becomes a number
	printf "root:0:root\nnobody:65534:nobody\n" | ./$(BINARY) -c -d : | grep -qxF '[["root",0,"root"],["nobody",65534,"nobody"]]' && echo "PASS: colon-separated fields" || (echo "FAIL: colon-separated fields"; exit 1)
	# df output: sizes like 50G stay strings
	printf "/dev/sda1 ext4 50G 20G 30G\n/dev/sdb1 xfs 100G 40G 60G\n" | ./$(BINARY) -c | grep -qxF '[["/dev/sda1","ext4","50G","20G","30G"],["/dev/sdb1","xfs","100G","40G","60G"]]' && echo "PASS: df-style rows" || (echo "FAIL: df-style rows"; exit 1)
	# env vars KEY=VALUE
	printf "HOME=/root\nPATH=/usr/bin\n" | ./$(BINARY) -c -d = | grep -qxF '[["HOME","/root"],["PATH","/usr/bin"]]' && echo "PASS: env var pairs" || (echo "FAIL: env var pairs"; exit 1)
	# empty input
	printf "" | ./$(BINARY) | grep -qxF '[]' && echo "PASS: empty input" || (echo "FAIL: empty input"; exit 1)
	# syslog: extract ERRORs into a JSON array
	printf "Jan 1 00:01:01 host sshd: Accepted\nJan 1 00:01:02 host kernel: ERROR disk failure\nJan 1 00:01:03 host cron: ERROR job failed\n" \
		| grep ERROR | ./$(BINARY) \
		| grep -qF '"Jan 1 00:01:02 host kernel: ERROR disk failure"' && echo "PASS: syslog error extraction" || (echo "FAIL: syslog error extraction"; exit 1)
	# ss/netstat: tab-separated connection rows
	printf "tcp\t0.0.0.0:22\t10.0.0.5:54321\tESTABLISHED\ntcp\t0.0.0.0:80\t10.0.0.9:43210\tESTABLISHED\n" \
		| ./$(BINARY) -c -d $$'\t' \
		| grep -qxF '[["tcp","0.0.0.0:22","10.0.0.5:54321","ESTABLISHED"],["tcp","0.0.0.0:80","10.0.0.9:43210","ESTABLISHED"]]' \
		&& echo "PASS: netstat tab-separated" || (echo "FAIL: netstat tab-separated"; exit 1)
	# ip route: pipe-separated, metric becomes a number
	printf "10.0.0.0/8|192.168.1.1|eth0|100\n0.0.0.0/0|192.168.1.254|eth0|0\n" \
		| ./$(BINARY) -c -d '|' \
		| grep -qxF '[["10.0.0.0/8","192.168.1.1","eth0",100],["0.0.0.0/0","192.168.1.254","eth0",0]]' \
		&& echo "PASS: ip route table" || (echo "FAIL: ip route table"; exit 1)
	# docker ps: tab-separated container id, image, status
	printf "a1b2c3d4\tnginx:latest\tUp 2 hours\nb5c6d7e8\tredis:7\tUp 5 days\n" \
		| ./$(BINARY) -c -d $$'\t' \
		| grep -qxF '[["a1b2c3d4","nginx:latest","Up 2 hours"],["b5c6d7e8","redis:7","Up 5 days"]]' \
		&& echo "PASS: docker ps columns" || (echo "FAIL: docker ps columns"; exit 1)
	# systemctl: failed unit:state pairs
	printf "sshd.service:active\nnginx.service:failed\nmysql.service:failed\n" \
		| grep failed | ./$(BINARY) -c -d : \
		| grep -qxF '[["nginx.service","failed"],["mysql.service","failed"]]' \
		&& echo "PASS: failed systemctl units" || (echo "FAIL: failed systemctl units"; exit 1)
	# /proc/meminfo: collect keys as a string array
	printf "MemTotal: 16384 kB\nMemFree: 8192 kB\nSwapTotal: 4096 kB\n" \
		| awk '{print $$1}' | ./$(BINARY) \
		| grep -qxF '["MemTotal:","MemFree:","SwapTotal:"]' \
		&& echo "PASS: meminfo key extraction" || (echo "FAIL: meminfo key extraction"; exit 1)
	# crontab: lines collected verbatim
	printf "0 2 * * * /usr/bin/backup.sh\n30 6 * * 1 /usr/bin/report.sh\n" \
		| ./$(BINARY) \
		| grep -qF '"0 2 * * * /usr/bin/backup.sh"' \
		&& echo "PASS: crontab audit" || (echo "FAIL: crontab audit"; exit 1)
	# key-value: env vars, PORT becomes a number
	out=$$(printf "HOST=db.internal\nPORT=5432\nDB=myapp\n" | ./$(BINARY) -k -d =); \
	echo "$$out" | grep -qF '"HOST":"db.internal"' && \
	echo "$$out" | grep -qF '"PORT":5432' && \
	echo "$$out" | grep -qF '"DB":"myapp"' && \
	echo "PASS: kv env vars" || (echo "FAIL: kv env vars"; exit 1)
	# key-value: meminfo, value keeps its unit (string)
	printf "MemTotal: 16384 kB\nMemFree: 8192 kB\n" \
		| ./$(BINARY) -k \
		| grep -qF '"MemTotal:":"16384 kB"' \
		&& echo "PASS: kv meminfo" || (echo "FAIL: kv meminfo"; exit 1)
	# key-value: os-release, values with spaces stay strings
	out=$$(printf 'ID=ubuntu\nVERSION_ID=22.04\nPRETTY_NAME=Ubuntu 22.04 LTS\n' | ./$(BINARY) -k -d =); \
	echo "$$out" | grep -qF '"ID":"ubuntu"' && \
	echo "$$out" | grep -qF '"PRETTY_NAME":"Ubuntu 22.04 LTS"' && \
	echo "PASS: kv os-release" || (echo "FAIL: kv os-release"; exit 1)
	# key-value: tab-separated service + PID (number)
	out=$$(printf "nginx\t1234\npostgres\t5678\n" | ./$(BINARY) -k -d $$'\t'); \
	echo "$$out" | grep -qF '"nginx":1234' && \
	echo "$$out" | grep -qF '"postgres":5678' && \
	echo "PASS: kv tab service-pid" || (echo "FAIL: kv tab service-pid"; exit 1)
	# objects via -n names
	printf "a x\nb y\n" | ./$(BINARY) -n k,v | grep -qxF '[{"k":"a","v":"x"},{"k":"b","v":"y"}]' && echo "PASS: objects -n" || (echo "FAIL: objects -n"; exit 1)
	# objects via -H header (whitespace default), PID typed
	printf "NAME PID\nnginx 12\n" | ./$(BINARY) -H | grep -qxF '[{"NAME":"nginx","PID":12}]' && echo "PASS: objects -H" || (echo "FAIL: objects -H"; exit 1)
	# --raw keeps everything a string
	printf "x 200\n" | ./$(BINARY) -k --raw | grep -qxF '{"x":"200"}' && echo "PASS: --raw" || (echo "FAIL: --raw"; exit 1)
	# round-trip typing: 007 stays string, 200 becomes number
	printf "n 007\nm 200\n" | ./$(BINARY) -k | grep -qxF '{"m":200,"n":"007"}' && echo "PASS: round-trip typing" || (echo "FAIL: round-trip typing"; exit 1)
	# jsonl: one value per line
	out=$$(printf "a\nb\n" | ./$(BINARY) -l); \
	[ "$$(printf '%s\n' "$$out" | wc -l | tr -d ' ')" = "2" ] && \
	printf '%s\n' "$$out" | grep -qxF '"a"' && \
	printf '%s\n' "$$out" | grep -qxF '"b"' && \
	echo "PASS: jsonl" || (echo "FAIL: jsonl"; exit 1)
	# whitespace split collapses runs, leading 1 typed
	printf "  1   systemd   0.0\n" | ./$(BINARY) -c | grep -qxF '[[1,"systemd","0.0"]]' && echo "PASS: whitespace split" || (echo "FAIL: whitespace split"; exit 1)
	# field selection with a range
	printf "a:b:c:d\n" | ./$(BINARY) -c -d : -f 1-2,4 | grep -qxF '[["a","b","d"]]' && echo "PASS: field range" || (echo "FAIL: field range"; exit 1)
	# csv: quoted field with separator stays intact, 42 typed
	printf '"Smith, John",42\n' | ./$(BINARY) --csv | grep -qxF '[["Smith, John",42]]' && echo "PASS: csv quoted" || (echo "FAIL: csv quoted"; exit 1)
	# tsv
	printf 'a\tb\n' | ./$(BINARY) --tsv | grep -qxF '[["a","b"]]' && echo "PASS: tsv" || (echo "FAIL: tsv"; exit 1)
	# csv auto-detect from .csv extension + header → objects
	printf 'svc,cost\nEC2,42.5\n' > /tmp/je_costs.csv; \
	./$(BINARY) -H /tmp/je_costs.csv | grep -qxF '[{"cost":42.5,"svc":"EC2"}]' && echo "PASS: csv auto-detect" || (echo "FAIL: csv auto-detect"; exit 1); \
	rm -f /tmp/je_costs.csv
	# NUL-delimited input
	printf 'a b\0c d\0' | ./$(BINARY) -0 | grep -qxF '["a b","c d"]' && echo "PASS: nul-delimited" || (echo "FAIL: nul-delimited"; exit 1)
	# follow: streams one value per line
	printf 'one\ntwo\n' | ./$(BINARY) -F -l | grep -qxF '"two"' && echo "PASS: follow stream" || (echo "FAIL: follow stream"; exit 1)
	# wrap: --wrap adds host/timestamp/data
	printf 'a\n' | ./$(BINARY) --wrap | jq -e '.data == ["a"] and has("host") and has("timestamp")' > /dev/null && echo "PASS: wrap" || (echo "FAIL: wrap"; exit 1)
	# output to a file
	printf 'a\nb\n' | ./$(BINARY) -o /tmp/je_out.json; \
	grep -qxF '["a","b"]' /tmp/je_out.json && echo "PASS: -o output file" || (echo "FAIL: -o output file"; exit 1); \
	rm -f /tmp/je_out.json
	# terminal-aware: piped output is compact and uncolored
	printf 'a\n' | ./$(BINARY) | grep -qxF '["a"]' && echo "PASS: piped compact/no-color" || (echo "FAIL: piped compact/no-color"; exit 1)
	# --color=always forces ANSI even when piped
	printf 'a\n' | ./$(BINARY) --color=always | grep -q $$'\x1b\[' && echo "PASS: color always" || (echo "FAIL: color always"; exit 1)
	# version
	./$(BINARY) -V | grep -qF 'json_encode' && echo "PASS: version" || (echo "FAIL: version"; exit 1)
	# invalid flag value exits non-zero
	printf 'a\n' | ./$(BINARY) --color=weird > /dev/null 2>&1 && (echo "FAIL: bad flag exit"; exit 1) || echo "PASS: bad flag exit"
	# loki batch: raw lines → valid Loki push payload
	out=$$(printf '10.0.0.1 - - [01/Jun/2025:12:00:00 +0000] "GET /health HTTP/1.1" 200 5\n10.0.0.2 - - [01/Jun/2025:12:00:01 +0000] "POST /api HTTP/1.1" 201 42\n' \
		| ./$(BINARY) \
		| jq -c --arg job nginx --arg host testhost \
		    '{streams:[{stream:{job:$$job,host:$$host},values:[.[]|[(now*1e9|tostring),.]] }]}'); \
	echo "$$out" | jq -e '.streams[0].stream.job == "nginx"' > /dev/null && \
	echo "$$out" | jq -e '.streams[0].values | length == 2' > /dev/null && \
	echo "$$out" | jq -e '.streams[0].values[0][1] | contains("GET /health")' > /dev/null && \
	echo "PASS: loki batch payload" || (echo "FAIL: loki batch payload"; exit 1)
	# loki structured: parsed fields → per-line label strings
	out=$$(printf '10.0.0.1 - - [01/Jun/2025:12:00:00 +0000] "GET /health HTTP/1.1" 200 5\n10.0.0.2 - - [01/Jun/2025:12:00:01 +0000] "POST /api HTTP/1.1" 404 12\n' | awk '{print $$1"|"$$7"|"$$9}' | ./$(BINARY) -c -d '|' | jq -c --arg host testhost '[.[]|{ip:.[0],path:.[1],status:.[2]}]|{streams:[{stream:{job:"nginx",host:$$host},values:[.[]|[(now*1e9|tostring),("ip="+.ip+" path="+.path+" status="+(.status|tostring))]]}]}'); \
	echo "$$out" | jq -e '.streams[0].values | length == 2' > /dev/null && \
	echo "$$out" | jq -e '.streams[0].values[0][1] | startswith("ip=10.0.0.1")' > /dev/null && \
	echo "$$out" | jq -e '.streams[0].values[1][1] | contains("status=404")' > /dev/null && \
	echo "PASS: loki structured payload" || (echo "FAIL: loki structured payload"; exit 1)
	# loki single line: real-time path → single-value stream
	out=$$(printf '10.0.0.1 - - [01/Jun/2025:12:00:00 +0000] "DELETE /item/9 HTTP/1.1" 204 0\n' | ./$(BINARY) | jq -c --arg host testhost '{streams:[{stream:{job:"nginx",host:$$host},values:[[(now*1e9|tostring),.[0]]]}]}'); \
	echo "$$out" | jq -e '.streams[0].values | length == 1' > /dev/null && \
	echo "$$out" | jq -e '.streams[0].values[0][1] | contains("DELETE")' > /dev/null && \
	echo "PASS: loki single-line payload" || (echo "FAIL: loki single-line payload"; exit 1)

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean

# Regenerate the README demo GIF from demo/demo.tape.
# Needs: vhs, gifsicle (brew install vhs gifsicle). VHS drives a localhost ttyd,
# so it must run outside a network sandbox. Scenes 4-6 hit a live kubectl cluster
# and podman, so those tools must be reachable for an identical render.
demo: build
	vhs demo/demo.tape
	gifsicle -O3 --lossy=30 -o demo/json_encode.gif demo/json_encode.gif

clean:
	rm -f $(BINARY) coverage.out
	rm -rf dist

.PHONY: build test vet clean functional-test release snapshot demo
