package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
	// Available if you need it!
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func benDecodeString(s string) (int, int, error) {
	var firstColonIndex int

	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	lengthStr := s[:firstColonIndex]

	lenth, err := strconv.Atoi(lengthStr)
	if err != nil {
		return 0, 0, err
	}

	return firstColonIndex + 1, firstColonIndex + 1 + lenth, nil
}

func benDecodeInt(s string) (int, int, error) {
	var endIndex int

	for i := 0; i < len(s); i++ {
		if s[i] == 'e' {
			endIndex = i
			break
		}
	}

	return 1, endIndex, nil
}

func decodeBencode(bencodedString string) (interface{}, error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		firstColonIndex, endIndex, _ := benDecodeString(bencodedString)
		return bencodedString[firstColonIndex:endIndex], nil
	} else if rune(bencodedString[0]) == 'i' {
		firstColonIndex, endIndex, _ := benDecodeInt(bencodedString)
		return strconv.Atoi(bencodedString[firstColonIndex:endIndex])
	} else if rune(bencodedString[0]) == 'l' {
		var encodeList = make([]interface{}, 0)
		for i := 1; i < len(bencodedString); i++ {
			if unicode.IsDigit(rune(bencodedString[i])) {
				firstColonIndex, endIndex, _ := benDecodeString(bencodedString[i:])
				encodeList = append(encodeList, bencodedString[i+firstColonIndex:i+endIndex])
				i = i + endIndex - 1
			} else if rune(bencodedString[i]) == 'i' {
				firstColonIndex, endIndex, _ := benDecodeInt(bencodedString[i:])
				decoded, _ := strconv.Atoi(bencodedString[i+firstColonIndex : i+endIndex])
				encodeList = append(encodeList, decoded)
				i = i + endIndex - 1
			}
		}
		return encodeList, nil
	} else {
		return nil, fmt.Errorf("unsupported bencode type")
	}
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Println("Logs from your program will appear here!")

	command := os.Args[1]

	if command == "decode" {
		// Uncomment this block to pass the first stage

		bencodedValue := os.Args[2]

		decoded, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
