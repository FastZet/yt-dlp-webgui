#!/usr/bin/env sh
set -eu

if [ "${1:-}" = "" ]; then
  echo "Usage: $0 <password>" >&2
  exit 1
fi

PASSWORD="$1"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

cat > "$TMP_DIR/main.go" <<'EOF'
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fastzet/yt-dlp-webgui/internal/auth"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: go run main.go <password>")
	}

	hash, err := auth.HashPassword(os.Args[1])
	if err != nil {
		log.Fatalf("hashing password: %v", err)
	}

	fmt.Println(hash)
}
EOF

go run "$TMP_DIR/main.go" "$PASSWORD"
