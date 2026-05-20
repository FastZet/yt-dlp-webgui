package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fastzet/yt-dlp-webgui/internal/auth"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: go run ./cmd/hashpw <password>")
	}

	hash, err := auth.HashPassword(os.Args[1])
	if err != nil {
		log.Fatalf("hashing password: %v", err)
	}

	fmt.Println(hash)
}
