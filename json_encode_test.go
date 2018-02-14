package main

import (
	"testing"
)

func TestSeqToJSON(t *testing.T) {
	inputRawData := []byte{0x31, 0xa, 0x32, 0xa, 0x33, 0xa, 0x34, 0xa, 0x35, 0xa, 0x36, 0xa, 0x37, 0xa, 0x38, 0xa, 0x39, 0xa, 0x31, 0x30, 0xa}
	JSON := string(ConvertSliceJSON(ConvertInputToLines(string(inputRawData))))
	expectedOutput := "[\"1\",\"2\",\"3\",\"4\",\"5\",\"6\",\"7\",\"8\",\"9\",\"10\"]"
	if expectedOutput != JSON {
		t.Fail()
	}
}

func TestStringsToJson(t *testing.T) {
	inputData := "1 2 3\n4 5 6"
	JSON := string(ConvertSliceJSON(ConvertInputToLines(inputData)))
	//fmt.Printf("%s", JSON)
	expectedOutput := "[\"1 2 3\",\"4 5 6\"]"
	if expectedOutput != JSON {
		t.Fail()
	}
}

func TestStringsToTable(t *testing.T) {
	inputData := "A B C\nD E F"
	JSON := string(ConvertTableJSON(ConvertLinesToTable(ConvertInputToLines(inputData))))
	//fmt.Printf("%s", JSON)
	expectedOutput := "[[\"A\",\"B\",\"C\"],[\"D\",\"E\",\"F\"]]"
	if expectedOutput != JSON {
		t.Fail()
	}
}
