package main

import (
	"encoding/json"
	"fmt"
	"os"
	// Available if you need it!
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Println("Logs from your program will appear here!")

	command := os.Args[1]

	if command == "decode" {
		bencodeDecoder := NewBencodeDecoder()
		bencodedValue := os.Args[2]

		decoded, _, err := bencodeDecoder.Decode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else if command == "info" {
		torrentFile := os.Args[2]
		parser := NewTorrentParser()
		info, err := parser.ParseFile(torrentFile)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		fmt.Printf("Tracker URL: %s\n", info.AnnounceURL)
		fmt.Printf("Length: %d\n", info.Length)
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
