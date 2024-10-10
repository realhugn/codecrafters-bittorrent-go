package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"unicode"
	// Available if you need it!
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func benDecodeString(s string) (string, int, error) {
	colonIndex := -1
	for i, ch := range s {
		if ch == ':' {
			colonIndex = i
			break
		}
	}
	if colonIndex == -1 {
		return "", 0, fmt.Errorf("invalid string format")
	}

	length, err := strconv.Atoi(s[:colonIndex])
	if err != nil {
		return "", 0, fmt.Errorf("invalid string length: %v", err)
	}

	endIndex := colonIndex + 1 + length
	if endIndex > len(s) {
		return "", 0, fmt.Errorf("string length exceeds input")
	}

	return s[colonIndex+1 : endIndex], endIndex, nil
}

func benDecodeInt(s string) (int64, int, error) {
	if s[0] != 'i' {
		return 0, 0, fmt.Errorf("invalid integer format")
	}

	endIndex := -1
	for i, ch := range s[1:] {
		if ch == 'e' {
			endIndex = i + 1
			break
		}
	}
	if endIndex == -1 {
		return 0, 0, fmt.Errorf("invalid integer format: no ending 'e'")
	}

	num, err := strconv.ParseInt(s[1:endIndex], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid integer: %v", err)
	}

	return num, endIndex + 1, nil
}

func decodeDict(s string) (map[string]interface{}, int, error) {
	dict := make(map[string]interface{})
	i := 1
	var keys []string

	for i < len(s) && s[i] != 'e' {
		key, n, err := benDecodeString(s[i:])
		if err != nil {
			return nil, 0, fmt.Errorf("invalid dictionary key: %v", err)
		}
		i += n

		value, n, err := decodeBencode(s[i:])
		if err != nil {
			return nil, 0, fmt.Errorf("invalid dictionary value: %v", err)
		}
		i += n

		dict[key] = value
		keys = append(keys, key)
	}

	if i >= len(s) || s[i] != 'e' {
		return nil, 0, fmt.Errorf("invalid dictionary: no ending 'e'")
	}

	// Check if keys are sorted
	if !sort.StringsAreSorted(keys) {
		return nil, 0, fmt.Errorf("invalid dictionary: keys are not sorted")
	}

	return dict, i + 1, nil
}

func benDecodeList(s string) ([]interface{}, int, error) {
	var list = make([]interface{}, 0)
	i := 1
	for i < len(s) && s[i] != 'e' {
		item, n, err := decodeBencode(s[i:])
		if err != nil {
			return nil, 0, err
		}
		list = append(list, item)
		i += n
	}
	if i >= len(s) || s[i] != 'e' {
		return nil, 0, fmt.Errorf("invalid list: no ending 'e'")
	}
	return list, i + 1, nil
}

func decodeBencode(s string) (interface{}, int, error) {
	if len(s) == 0 {
		return nil, 0, fmt.Errorf("empty input")
	}

	switch {
	case unicode.IsDigit(rune(s[0])):
		return benDecodeString(s)
	case s[0] == 'i':
		return benDecodeInt(s)
	case s[0] == 'l':
		return benDecodeList(s)
	case s[0] == 'd':
		return decodeDict(s)
	default:
		return nil, 0, fmt.Errorf("unsupported bencode type")
	}
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Println("Logs from your program will appear here!")

	command := os.Args[1]

	if command == "decode" {
		// Uncomment this block to pass the first stage

		bencodedValue := os.Args[2]

		decoded, _, err := decodeBencode(bencodedValue)
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
