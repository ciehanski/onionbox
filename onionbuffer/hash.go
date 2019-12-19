package onionbuffer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func createHash(key string) string {
	hasher := hmac.New(sha256.New, []byte(key))
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}
