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

var inputLines []string

func getInputLines() []string {
	flag.Parse()
	var data []byte
	var err error
	data, err = ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line != "" {
			inputLines = append(inputLines, line)
		}
	}
	return inputLines
}

func printMapJSON(filenames []string) {
	pagesJSON, err := json.Marshal(filenames)
	if err != nil {
		log.Fatal("Cannot encode to JSON ", err)
	}
	fmt.Fprintf(os.Stdout, "%s\n", pagesJSON)
}

func main() {
	switch flag.NArg() {
	case 0:
		inputLines := getInputLines()
		printMapJSON(inputLines)
		break
	default:
		fmt.Printf("input must be from stdin\n")
		os.Exit(1)
	}
}
