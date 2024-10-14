package torrent

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

const BLOCK_SIZE = 16 * 1024 // 16 KiB

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

func (tc *TorrentClient) Handshake(peer string, infoHash string) ([]byte, net.Conn, error) {
	conn, err := net.Dial("tcp", peer)
	if err != nil {
		return nil, nil, fmt.Errorf("error connecting to peer: %v", err)
	}
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
		return nil, nil, fmt.Errorf("error sending handshake message: %v", err)
	}

	response := make([]byte, 68)
	_, err = conn.Read(response)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading handshake response: %v", err)
	}

	if !strings.HasPrefix(string(response[1:20]), "BitTorrent protocol") {
		return nil, nil, fmt.Errorf("invalid handshake response")
	}

	reservedPeerId := response[48:68]
	return reservedPeerId, conn, nil
}

func (tc *TorrentClient) DownloadPiece(torrentInfo TorrentInfo, infoHash []byte, outputFile string, pieceNumber int) ([]byte, error) {
	peers, err := tc.GetPeers(&torrentInfo)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}

	if len(peers) == 0 {
		return nil, fmt.Errorf("no peers found")
	}

	peer := peers[0]
	peerAddr := fmt.Sprintf("%s:%d", peer.IP, peer.Port)

	// Send handshake message0
	_, conn, err := tc.Handshake(peerAddr, torrentInfo.InfoHash)
	if err != nil {
		fmt.Println("Error sending handshake message:", err)
		return nil, err
	}

	// Wait for bitfield message
	// The message id for bitfield is 5
	// The payload is a bitfield indicating which pieces the peer has
	bitFieldMsg, err := readMessage(conn)
	if err != nil {
		fmt.Println("Error reading bitfield message:", err)
		return nil, err
	}
	if bitFieldMsg.id != 5 {
		return nil, fmt.Errorf("unexpected message ID: %d", bitFieldMsg.id)
	}

	// Send interested message
	// The message id for interested is 2
	// The payload is empty
	err = sendMessage(conn, 2, nil)
	if err != nil {
		fmt.Println("Error sending interested message:", err)
		return nil, err
	}

	// Wait unchoke message
	// The message id for unchoke is 1
	// The payload is empty
	unchokeMsg, err := readMessage(conn)
	if err != nil {
		fmt.Println("Error reading piece message:", err)
		return nil, err
	}

	if unchokeMsg.id != 1 {
		return nil, fmt.Errorf("unexpected message ID: %d", unchokeMsg.id)
	}

	// Request piece
	pieceLength := torrentInfo.PieceLength
	if pieceNumber == len(torrentInfo.Pieces)-1 {
		pieceLength = torrentInfo.Length % torrentInfo.PieceLength
	}

	piece := make([]byte, pieceLength)
	// Break the piece into blocks of 16 KiB
	for offset := 0; offset < int(pieceLength); offset += BLOCK_SIZE {
		length := BLOCK_SIZE
		if offset+length > int(pieceLength) {
			length = int(pieceLength) - offset
		}

		// Send request message
		// payload is a 12-byte message with the following format:
		// - piece index (4 bytes)
		// - block offset (4 bytes)
		// - block length (4 bytes)
		payload := make([]byte, 12)
		binary.BigEndian.PutUint32(payload[0:4], uint32(pieceNumber))
		binary.BigEndian.PutUint32(payload[4:8], uint32(offset))
		binary.BigEndian.PutUint32(payload[8:12], uint32(length))
		sendMessage(conn, 6, payload)

		// Wait for piece message
		// The message id for piece is 7
		// The payload message with the following format:
		// - piece index (4 bytes)
		// - block offset (4 bytes)
		msg, err := readMessage(conn)
		if err != nil {
			fmt.Println("Error reading piece message:", err)
			return nil, err
		}

		if msg.id != 7 {
			return nil, fmt.Errorf("unexpected message ID: %d", msg.id)
		}
		begin := binary.BigEndian.Uint32(msg.payload[4:8])
		block := msg.payload[8:]
		copy(piece[begin:], block)
	}

	//verify piece
	hash := sha1.Sum(piece)
	expectedHash := torrentInfo.Pieces[pieceNumber]
	fmt.Println("Expected hash:", expectedHash)
	fmt.Println("Actual hash:", hex.EncodeToString(hash[:]))
	if hex.EncodeToString(hash[:]) != expectedHash {
		return nil, fmt.Errorf("piece hash mismatch")
	}

	err = os.WriteFile(outputFile, piece, 0644)
	if err != nil {
		fmt.Println("Error writing piece to file:", err)
		return nil, err
	}

	return piece, nil
}

type Message struct {
	length  uint32
	id      uint8
	payload []byte
}

func readMessage(conn net.Conn) (Message, error) {
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(conn, lengthBuf)
	if err != nil {
		fmt.Println("Error reading message length:", err)
		return Message{}, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length == 0 {
		fmt.Println("Received keep-alive message")
		return Message{length: 0}, nil
	}

	messageBuf := make([]byte, length)
	_, err = io.ReadFull(conn, messageBuf)
	if err != nil {
		fmt.Println("Error reading message:", err)
		return Message{}, err
	}

	return Message{
		length:  length,
		id:      messageBuf[0],
		payload: messageBuf[1:],
	}, nil
}

func sendMessage(conn net.Conn, id uint8, payload []byte) error {
	length := uint32(len(payload) + 1)
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = id
	copy(buf[5:], payload)

	_, err := conn.Write(buf)
	return err
}

func (tc *TorrentClient) Download(torrentFile string, outputDir string) error {
	parser := NewTorrentParser()
	info, err := parser.ParseFile(torrentFile)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	peers, err := tc.GetPeers(info)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	if len(peers) == 0 {
		return fmt.Errorf("no peers found")
	}

	numPeers := len(info.Pieces)
	fmt.Println("Number of pieces:", numPeers)
	pieces := make([][]byte, numPeers)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	for i := 0; i < numPeers; i++ {
		wg.Add(1)
		go func(i int, info *TorrentInfo) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			piece, err := tc.DownloadPiece(*info, []byte(info.InfoHash), outputDir, i)
			if err != nil {
				fmt.Println("Error downloading piece:", err)
				return
			}

			if piece != nil {
				pieces[i] = piece
				fmt.Printf("Piece %d downloaded successfully\n", i)
			} else {
				fmt.Println("Error downloading piece:", err)
			}
		}(i, info)
	}

	wg.Wait()

	file, err := os.Create(outputDir)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return err
	}
	defer file.Close()

	for _, piece := range pieces {
		_, err = file.Write(piece)
		if err != nil {
			fmt.Println("Error writing piece to file:", err)
			return err
		}
	}

	fmt.Println("File downloaded successfully")
	return nil
}
