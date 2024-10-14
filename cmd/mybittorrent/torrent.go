package main

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
)

type TorrentInfo struct {
	AnnounceURL string
	Length      int64
	InfoHash    string
	PieceLength int64
	Pieces      []string
}

type TorrentParser struct {
	decoder *BencodeDecoder
	encoder *BencodeEncoder
}

type TrackerResponse struct {
	Interval int64
	Peers    string
}

type Peer struct {
	IP   string
	Port uint16
}

type Params struct {
	InfoHash   string `json:"info_hash"`
	PeerID     string `json:"peer_id"`
	Port       int    `json:"port"`
	Uploaded   int    `json:"uploaded"`
	Downloaded int    `json:"downloaded"`
	Left       int    `json:"left"`
	Compact    int    `json:"compact"`
}

func NewTorrentParser() *TorrentParser {
	return &TorrentParser{
		decoder: NewBencodeDecoder(),
		encoder: NewBencodeEncoder(),
	}
}

func (tp *TorrentParser) GetPeers(torrent *TorrentInfo) ([]Peer, error) {
	infoHashBytes, _ := hex.DecodeString(torrent.InfoHash)

	params := url.Values{
		"info_hash":  []string{string(infoHashBytes)},
		"peer_id":    []string{"-TO0042-123456789012"},
		"port":       []string{"6881"},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"left":       []string{fmt.Sprintf("%d", torrent.Length)},
		"compact":    []string{"1"},
	}
	resp, err := http.Get(torrent.AnnounceURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("error sending request to tracker: %v", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var trackerResp TrackerResponse
	decoded, _, err := tp.decoder.Decode(string(body))
	if err != nil {
		return nil, fmt.Errorf("error decoding tracker response: %v", err)
	}

	trackerRespMap, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid tracker response format")
	}

	trackerResp = TrackerResponse{
		// Interval: int64(trackerRespMap["interval"].(int)),
		Peers: trackerRespMap["peers"].(string),
	}

	return tp.parsePeers(trackerResp.Peers)
}

func (tp *TorrentParser) parsePeers(peers string) ([]Peer, error) {
	var peerList []Peer
	for i := 0; i < len(peers); i += 6 {
		ip := net.IP(peers[i : i+4])
		port := binary.BigEndian.Uint16([]byte(peers[i+4 : i+6]))
		peerList = append(peerList, Peer{IP: ip.String(), Port: port})
	}
	return peerList, nil
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

func ParsePeers(peers string) {
	for i := 0; i < len(peers); i += 6 {
		ip := net.IP(peers[i : i+4])
		port := binary.BigEndian.Uint16([]byte(peers[i+4 : i+6]))
		fmt.Printf("Peer IP: %s, Port: %d\n", ip.String(), port)
	}
}
