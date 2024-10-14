package torrent

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type TorrentClient struct{}

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

func NewTorrentClient() *TorrentClient {
	return &TorrentClient{}
}

func (tc *TorrentClient) GetPeers(torrent *TorrentInfo) ([]Peer, error) {
	tp := NewTorrentParser()
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

	return tc.parsePeers(trackerResp.Peers)
}

func (tc *TorrentClient) parsePeers(peers string) ([]Peer, error) {
	var peerList []Peer
	for i := 0; i < len(peers); i += 6 {
		ip := net.IP(peers[i : i+4])
		port := binary.BigEndian.Uint16([]byte(peers[i+4 : i+6]))
		peerList = append(peerList, Peer{IP: ip.String(), Port: port})
	}
	return peerList, nil
}

func (tc *TorrentClient) Handshake(peer string, infoHash string) ([]byte, error) {
	conn, err := net.Dial("tcp", peer)
	if err != nil {
		return nil, fmt.Errorf("error connecting to peer: %v", err)
	}
	defer conn.Close()
	// Construct handshake message
	pstrlen := byte(19)
	pstr := []byte("BitTorrent protocol")
	reserved := make([]byte, 8)
	peerId := make([]byte, 20)
	infoHashBytes, _ := hex.DecodeString(infoHash)

	rand.Read(peerId)

	handshake := append([]byte{pstrlen}, pstr...)
	handshake = append(handshake, reserved...)
	handshake = append(handshake, infoHashBytes...)
	handshake = append(handshake, peerId...)
	_, err = conn.Write(handshake)
	if err != nil {
		return nil, fmt.Errorf("error sending handshake message: %v", err)
	}

	response := make([]byte, 68)
	_, err = conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("error reading handshake response: %v", err)
	}

	if !strings.HasPrefix(string(response[1:20]), "BitTorrent protocol") {
		return nil, fmt.Errorf("invalid handshake response")
	}

	reservedPeerId := response[48:68]
	return reservedPeerId, nil
}
