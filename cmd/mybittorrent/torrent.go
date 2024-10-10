package main

import (
	"fmt"
	"os"
)

type TorrentInfo struct {
	AnnounceURL string
	Length      int64
}

type TorrentParser struct {
	decoder *BencodeDecoder
}

func NewTorrentParser() *TorrentParser {
	return &TorrentParser{
		decoder: NewBencodeDecoder(),
	}
}

func (tp *TorrentParser) ParseFile(filename string) (*TorrentInfo, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}
	data_str := string(data)
	decoded, _, err := tp.decoder.Decode(data_str)
	if err != nil {
		return nil, fmt.Errorf("error decoding bencode: %v", err)
	}

	torrentInfo, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid torrent file format")
	}

	announce, ok := torrentInfo["announce"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid announce URL")
	}

	info, ok := torrentInfo["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid info dictionary")
	}

	length, ok := info["length"].(int64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid file length")
	}

	return &TorrentInfo{
		AnnounceURL: announce,
		Length:      length,
	}, nil
}
