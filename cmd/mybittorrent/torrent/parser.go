package torrent

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/bencode"
)

type TorrentInfo struct {
	AnnounceURL string
	Length      int64
	InfoHash    string
	PieceLength int64
	Pieces      []string
}

type TorrentParser struct {
	decoder *bencode.BencodeDecoder
	encoder *bencode.BencodeEncoder
}

func NewTorrentParser() *TorrentParser {
	return &TorrentParser{
		decoder: bencode.NewBencodeDecoder(),
		encoder: bencode.NewBencodeEncoder(),
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

	pieces, ok := info["pieces"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid pieces")
	}

	var piece_list []string
	for i := 0; i < len(pieces); i += 20 {
		piece_list = append(piece_list, hex.EncodeToString([]byte(pieces[i:i+20])))
	}

	pieceLength, ok := info["piece length"].(int64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid piece length")
	}
	return &TorrentInfo{
		AnnounceURL: announce,
		Length:      length,
		InfoHash:    tp.calculateInfoHash(info),
		PieceLength: pieceLength,
		Pieces:      piece_list,
	}, nil
}

func (tp *TorrentParser) calculateInfoHash(info interface{}) string {
	bencode := tp.encoder.Encode(info)
	sha1 := sha1.New()
	sha1.Write([]byte(bencode))
	return hex.EncodeToString(sha1.Sum(nil))
}
