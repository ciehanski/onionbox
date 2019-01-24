package onion_buffer

import (
	"archive/zip"
	"bufio"
	"bytes"
	"io"
	"sync"
	"syscall"
	"time"
)

// OnionBuffer struct
type OnionBuffer struct {
	sync.Mutex
	Name             string
	Bytes            []byte
	Checksum         string
	Encrypted        bool
	Downloads        int
	DownloadLimit    int
	DownloadsLimited bool
	Expire           bool
	ExpiresAt        time.Time
}

func (oBuffer *OnionBuffer) Destroy() error {
	oBuffer.Lock()
	var err error
	buffer := bytes.NewBuffer(oBuffer.Bytes)
	zWriter := zip.NewWriter(buffer)
	reader := bufio.NewReader(bytes.NewReader(oBuffer.Bytes))
	chunk := make([]byte, 1)
	// Lock memory allotted to chunk from being used in SWAP
	if err := syscall.Mlock(chunk); err != nil {
		return err
	}
	bufFile, _ := zWriter.Create(oBuffer.Name)
	for {
		if _, err = reader.Read(chunk); err != nil {
			break
		}
		_, err := bufFile.Write([]byte("0"))
		if err != nil {
			return err
		}
	}
	if err != io.EOF {
		return err
	} else {
		err = nil
	}
	if err := syscall.Munlock(oBuffer.Bytes); err != nil {
		return err
	}
	oBuffer.Unlock()
	return nil
}

func (oBuffer *OnionBuffer) IsExpired() bool {
	if oBuffer.Expire {
		if oBuffer.ExpiresAt.After(time.Now()) {
			return false
		}
		return true
	}
	return false
}

func (oBuffer *OnionBuffer) SetExpiration(expiration string) error {
	t, err := time.ParseDuration(expiration)
	if err != nil {
		return err
	}
	oBuffer.Expire = true
	oBuffer.ExpiresAt = time.Now().Add(t)
	return nil
}
