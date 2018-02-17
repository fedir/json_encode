# json_encode for shell

[![Build Status](https://travis-ci.org/fedir/json_encode.svg?branch=master)](https://travis-ci.org/fedir/json_encode)
[![Code Coverage](https://codecov.io/gh/fedir/json_encode/branch/master/graph/badge.svg)](https://codecov.io/gh/fedir/json_encode)
[![Go Report Card](https://goreportcard.com/badge/github.com/fedir/json_encode)](https://goreportcard.com/report/github.com/fedir/json_encode)
[![GoDoc](https://godoc.org/github.com/fedir/json_encode?status.svg)](https://godoc.org/github.com/fedir/json_encode)
[![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

JSON encoder for Shell is designed to transform data from different sources into JSON format for further processing.

Could be useful for syadmins, devops and developers for data collection, exchange and unification (ELK, custom dashobards).

## Installation

golang 1.9+ should be installed. Go environment [is very simple to install](https://golang.org/doc/install).

To compile :

    git clone https://github.com/fedir/json_encode.git
    cd json_encode
    go build

After You could put compiled `json_encode` in Your favourite bin folder.

### Options

    -s string     Separator for lines splitting (default " ")
    -sc           Enable simple columns, lines will be splitted by defined separator
    -p            Pretty-printing

## Usage examples

Simple string

    echo "Hello world" | ./json_encode
    ["Hello world"]

Simple generated array

    seq 1 10 | ./json_encode
    ["1","2","3","4","5","6","7","8","9","10"]

Simple array from file listing

    ls /usr/local/go/src/ | head -n 5 | ./json_encode
    ["Make.dist","all.bash","all.bat","all.rc","androidtest.bash"]

Apache logs

    tail -n 3 /var/log/apache2/access_log | ./json_encode
    ["127.0.0.1 - - [04/Nov/2017:00:40:54 +0100] \"GET / HTTP/1.1\" 304 -","127.0.0.1 - - [04/Nov/2017:00:57:57 +0100] \"-\" 408 -","127.0.0.1 - - [04/Nov/2017:00:57:57 +0100] \"-\" 408 -"]

Git log, transformed into an array of records with separated columns

    $ git log --date=local --pretty=format:"%h|%an|%ad|%s" -n 3 | ./json_encode -sc -s "|" -p
    [
        [
            "25696c2",
            "Fedir RYKHTIK",
            "Fri Feb 16 00:34:01 2018",
            "Advanced examples"
        ],
        [
            "6b50dbb",
            "Fedir RYKHTIK",
            "Fri Feb 16 00:21:31 2018",
            "Cleaning from comments for better readability"
        ],
        [
            "6221afb",
            "Fedir RYKHTIK",
            "Fri Feb 16 00:20:20 2018",
            "TestMainProgramWithSimpleColumns"
        ]
    ]
