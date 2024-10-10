package main

import (
	"fmt"
	"sort"
	"strconv"
)

type BencodeDecoder struct{}

func NewBencodeDecoder() *BencodeDecoder {
	return &BencodeDecoder{}
}

func (bd *BencodeDecoder) decodeString(s string) (string, int, error) {
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

func (bd *BencodeDecoder) decodeInt(s string) (int64, int, error) {
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

func (bd *BencodeDecoder) decodeList(s string) ([]interface{}, int, error) {
	var list = make([]interface{}, 0)
	i := 1
	for i < len(s) && s[i] != 'e' {
		item, n, err := bd.Decode(s[i:])
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

func (bd *BencodeDecoder) decodeDict(s string) (map[string]interface{}, int, error) {
	dict := make(map[string]interface{})
	i := 1
	var keys []string

	for i < len(s) && s[i] != 'e' {
		key, n, err := bd.decodeString(s[i:])
		if err != nil {
			return nil, 0, fmt.Errorf("invalid dictionary key: %v", err)
		}
		i += n

		value, n, err := bd.Decode(s[i:])
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

func (bd *BencodeDecoder) Decode(s string) (interface{}, int, error) {
	if len(s) == 0 {
		return nil, 0, fmt.Errorf("empty input")
	}

	switch {
	case s[0] >= '0' && s[0] <= '9':
		return bd.decodeString(s)
	case s[0] == 'i':
		return bd.decodeInt(s)
	case s[0] == 'l':
		return bd.decodeList(s)
	case s[0] == 'd':
		return bd.decodeDict(s)
	default:
		return nil, 0, fmt.Errorf("unsupported bencode type")
	}
}
