// Copyright 2018 Fedir RYKHTIK. All rights reserved.
// Use of this source code is governed by the GNU GPL 3.0
// license that can be found in the LICENSE file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

var (
	useSimpleColumns = flag.Bool("sc", false, "Enable simple columns, lines will be splitted by defined separator")
	useKeyValue      = flag.Bool("kv", false, "Key-value mode: first column becomes key, remainder becomes value")
	separator        = flag.String("s", " ", "Separator for lines splitting")
	printPretty      = flag.Bool("p", false, "Pretty-printing")
)

// GetInputData gets data from stdin
func GetInputData() string {
	var data []byte
	var err error
	data, err = io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	return string(data)
}

// ConvertInputToLines converts single input into separated elements
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

// ConvertLinesToTable converts single line into simple structure
func ConvertLinesToTable(inputLines []string) [][]string {
	linesWithColumns := make([][]string, 0)
	for _, inputLine := range inputLines {
		if inputLine != "" {
			linesWithColumns = append(linesWithColumns, strings.Split(inputLine, *separator))
		}
	}
	return linesWithColumns
}

// ConvertToJSON converts input lines into JSON
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

func main() {
	flag.Parse()
	JSON := make([]byte, 0)
	switch {
	case *useKeyValue:
		JSON = ConvertToJSON(ConvertLinesToKeyValue(ConvertInputToLines(GetInputData())))
	case *useSimpleColumns:
		JSON = ConvertToJSON(ConvertLinesToTable(ConvertInputToLines(GetInputData())))
	default:
		JSON = ConvertToJSON(ConvertInputToLines(GetInputData()))
	}

	fmt.Fprintf(os.Stdout, "%s\n", JSON)
}
