package onionbuffer

import "testing"

func TestDecrypt(t *testing.T) {
	secretMessage := []byte("This is a secret message")
	password := "hunter2"
	encryptedBytes, _ := Encrypt(secretMessage, password)
	decryptedBytes, err := Decrypt(encryptedBytes, password)
	if err != nil {
		t.Error(err)
	}
	if string(decryptedBytes) != string(secretMessage) {
		t.Error("Decrypted bytes were expected to match the original message")
	}
}

func BenchmarkDecrypt(b *testing.B) {
	secretMessage := []byte("This is a secret message")
	password := "hunter2"
	encryptedBytes, _ := Encrypt(secretMessage, password)
	for n := 0; n < b.N; n++ {
		Decrypt(encryptedBytes, password)
	}
}
