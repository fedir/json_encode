BINARY=json_encode

build:
	go build -o $(BINARY) .

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

clean:
	rm -f $(BINARY) coverage.out

.PHONY: build test vet clean functional-test
