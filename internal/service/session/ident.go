package session

import (
	"crypto/rand"
	"encoding/hex"
)

const (
	// codeAlphabet skips look-alike characters.
	codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	codeLength   = 4
)

func newID() string { return "p" + randomHex(4) }

func newToken() string { return randomHex(16) }

func randomHex(bytes int) string {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
