package onion_buffer

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"syscall"
)

const chunkSize = 1024

func (of *OnionBuffer) GetChecksum() (string, error) {
	hash := md5.New()
	var count int
	var err error
	reader := bufio.NewReader(bytes.NewReader(of.Bytes))
	chunk := make([]byte, chunkSize)
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
	hashInBytes := hash.Sum(nil)[:16]
	return hex.EncodeToString(hashInBytes), nil
}

func (of *OnionBuffer) ValidateChecksum() (bool, error) {
	chksm, err := of.GetChecksum()
	if err != nil {
		return false, err
	}
	if of.Checksum == chksm {
		return true, nil
	}
	return false, nil
}
