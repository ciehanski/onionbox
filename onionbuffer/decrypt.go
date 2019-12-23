package onionbuffer

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
)

func Decrypt(data []byte, passphrase string) ([]byte, error) {
	s := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(s[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
