package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var inputLines = make([]string, 0)

// GetInputData gets data from stdin
func GetInputData() string {
	flag.Parse()
	var data []byte
	var err error
	data, err = ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	return string(data)
}

// ConvertInputToLines converts single input into separated elements
func ConvertInputToLines(inputString string) []string {
	lines := strings.Split(inputString, "\n")
	for _, line := range lines {
		if line != "" {
			inputLines = append(inputLines, line)
		}
	}
	return inputLines
}

// ConvertSliceJSON converts input lines into JSON
func ConvertSliceJSON(inputLines []string) []byte {
	JSON, err := json.Marshal(inputLines)
	if err != nil {
		log.Fatal("Cannot encode to JSON ", err)
	}
	return JSON
}

func main() {
	switch flag.NArg() {
	case 0:
		JSON := ConvertSliceJSON(ConvertInputToLines(GetInputData()))
		fmt.Fprintf(os.Stdout, "%s\n", JSON)
		break
	default:
		fmt.Printf("input must be from stdin\n")
		os.Exit(1)
	}
}
