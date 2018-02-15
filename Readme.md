# json_encode

[![Build Status](https://travis-ci.org/fedir/json_encode.svg?branch=master)](https://travis-ci.org/fedir/json_encode) [![Go Report Card](https://goreportcard.com/badge/github.com/fedir/json_encode)](https://goreportcard.com/report/github.com/fedir/json_encode) [![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

json_encode package pipes input into JSON for further processing.

## Installation

golang 1.9+ should be installed. Go environment [is very simple to install](https://golang.org/doc/install).

To compile :

    git clone https://github.com/fedir/json_encode.git
    cd json_encode
    go build

After You could put compiled `json_encode` in Your favourite bin folder.

## Usage

    echo "Hello world" | ./json_encode

### Options

    -s string     Separator for lines splitting (default " ")
    -sc           Enable simple columns, lines will be splitted by defined separator

## Use cases

    echo "Hello world" | ./json_encode
    ["Hello world"]

    seq 1 10 | ./json_encode
    ["1","2","3","4","5","6","7","8","9","10"]

    ls /usr/local/go/src/ | ./json_encode
    ["Make.dist","all.bash","all.bat","all.rc","androidtest.bash","archive","bootstrap.bash","bufio","buildall.bash","builtin","bytes","clean.bash","clean.bat","clean.rc","cmd","cmp.bash","compress","container","context","crypto","database","debug","encoding","errors","expvar","flag","fmt","go","hash","html","image","index","internal","io","iostest.bash","log","make.bash","make.bat","make.rc","math","mime","naclmake.bash","nacltest.bash","net","os","path","plugin","race.bash","race.bat","reflect","regexp","run.bash","run.bat","run.rc","runtime","sort","strconv","strings","sync","syscall","testing","text","time","unicode","unsafe","vendor"]

    tail /var/log/apache2/access_log | ./json_encode
    ["127.0.0.1 - - [04/Nov/2017:00:40:50 +0100] \"GET / HTTP/1.1\" 304 -","127.0.0.1 - - [04/Nov/2017:00:40:51 +0100] \"GET / HTTP/1.1\" 304 -","127.0.0.1 - - [04/Nov/2017:00:40:51 +0100] \"GET / HTTP/1.1\" 304 -","127.0.0.1 - - [04/Nov/2017:00:40:52 +0100] \"GET / HTTP/1.1\" 304 -","127.0.0.1 - - [04/Nov/2017:00:40:52 +0100] \"GET / HTTP/1.1\" 304 -","127.0.0.1 - - [04/Nov/2017:00:40:53 +0100] \"GET / HTTP/1.1\" 304 -","127.0.0.1 - - [04/Nov/2017:00:40:5

    echo -e "A B C\nD E F" | ./json_encode
    ["A B C","D E F"]

    echo -e "A B C\nD E F" | ./json_encode -sc -s " "
    [["A","B","C"],["D","E","F"]]
