// Copyright 2018 Fedir RYKHTIK. All rights reserved.
// Use of this source code is governed by the GNU GPL 3.0
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"os"
	"testing"
)

func TestSeqToJSON(t *testing.T) {
	inputRawData := []byte{0x31, 0xa, 0x32, 0xa, 0x33, 0xa, 0x34, 0xa, 0x35, 0xa, 0x36, 0xa, 0x37, 0xa, 0x38, 0xa, 0x39, 0xa, 0x31, 0x30, 0xa}
	JSON := string(ConvertToJSON(ConvertInputToLines(string(inputRawData))))
	expectedOutput := "[\"1\",\"2\",\"3\",\"4\",\"5\",\"6\",\"7\",\"8\",\"9\",\"10\"]"
	if expectedOutput != JSON {
		t.Fail()
	}
}

func TestStringsToJson(t *testing.T) {
	inputData := "1 2 3\n4 5 6"
	JSON := string(ConvertToJSON(ConvertInputToLines(inputData)))
	expectedOutput := "[\"1 2 3\",\"4 5 6\"]"
	if expectedOutput != JSON {
		t.Fail()
	}
}

func TestStringsToTable(t *testing.T) {
	inputData := "A B C\nD E F"
	JSON := string(ConvertToJSON(ConvertLinesToTable(ConvertInputToLines(inputData))))
	expectedOutput := "[[\"A\",\"B\",\"C\"],[\"D\",\"E\",\"F\"]]"
	if expectedOutput != JSON {
		t.Fail()
	}
}

func TestConvertInputToLines(t *testing.T) {
	inputData := "1\n2\n3"
	convertedData := fmt.Sprintf("%v", ConvertInputToLines(inputData))
	expectedOutput := fmt.Sprintf("%v", []string{"1", "2", "3"})
	if expectedOutput != convertedData {
		t.Fail()
	}
}

func TestMainProgram(t *testing.T) {
	main()
}

func TestMainProgramWithSimpleColumns(t *testing.T) {
	os.Args = []string{"-sc"}
	main()
}
