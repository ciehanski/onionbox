package onion_buffer

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"syscall"
)

func (b *OnionBuffer) GetChecksum() (string, error) {
	b.Lock()
	var count int
	var err error
	hash := md5.New()
	reader := bufio.NewReader(bytes.NewReader(b.Bytes))
	chunk := make([]byte, b.ChunkSize)
	// Lock memory allotted to chunk from being used in SWAP
	if err := syscall.Mlock(chunk); err != nil {
		return "", err
	}
	for {
		if count, err = reader.Read(chunk); err != nil {
			break
		}
		_, err := hash.Write(chunk[:count])
		if err != nil {
			return "", err
		}
	}
	if err != io.EOF {
		return "", err
	} else {
		err = nil
	}
	b.Unlock()
	hashInBytes := hash.Sum(nil)[:16]
	return hex.EncodeToString(hashInBytes), nil
}

func (b *OnionBuffer) ValidateChecksum() (bool, error) {
	chksm, err := b.GetChecksum()
	if err != nil {
		return false, err
	}
	if b.Checksum == chksm {
		return true, nil
	}
	return false, nil
}
