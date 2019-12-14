package onionbuffer

import "testing"

func TestEncrypt(t *testing.T) {
	encryptedBytes, err := Encrypt([]byte("Top secret information"), "test")
	if err != nil {
		t.Error(err)
	}
	if string(encryptedBytes) == "Top secret information" {
		t.Error("bytes were not properly encrypted")
	}
}

func BenchmarkEncrypt(b *testing.B) {
	secretMessage := []byte("This is a secret message")
	password := "hunter2"
	for n := 0; n < b.N; n++ {
		Encrypt(secretMessage, password)
	}
}
