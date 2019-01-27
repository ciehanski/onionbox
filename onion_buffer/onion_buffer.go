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
	ChunkSize        int64
	Encrypted        bool
	Downloads        int64
	DownloadLimit    int64
	DownloadsLimited bool
	Expire           bool
	ExpiresAt        time.Time
}

// Destroy is mostly used to destroy temporary OnionBuffer objects after they
// have been copied to the store or to remove an individual OnionBuffer
// from the store.
func (b *OnionBuffer) Destroy() error {
	b.Lock()
	var err error
	buffer := bytes.NewBuffer(b.Bytes)
	zWriter := zip.NewWriter(buffer)
	reader := bufio.NewReader(bytes.NewReader(b.Bytes))
	chunk := make([]byte, 1)
	// Lock memory allotted to chunk from being used in SWAP
	if err := syscall.Mlock(chunk); err != nil {
		return err
	}
	bufFile, _ := zWriter.Create(b.Name)
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
	if err := syscall.Munlock(b.Bytes); err != nil {
		return err
	}
	b.Unlock()
	return nil
}

// IsExpired is used to check if an OnionBuffer is expired or not.
func (b *OnionBuffer) IsExpired() bool {
	if b.Expire {
		if b.ExpiresAt.After(time.Now()) {
			return false
		}
		return true
	}
	return false
}

// SetExpiration is used to set the expiration duration of the OnionBuffer.
func (b *OnionBuffer) SetExpiration(expiration string) error {
	b.Lock()
	t, err := time.ParseDuration(expiration)
	if err != nil {
		return err
	}
	b.Expire = true
	b.ExpiresAt = time.Now().Add(t)
	b.Unlock()
	return nil
}
