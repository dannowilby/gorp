package gorp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func GenerateHash(message json.RawMessage) (string, error) {
	hash := sha256.Sum256(message)
	return hex.EncodeToString(hash[:]), nil
}
