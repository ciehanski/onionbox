package onion_file

import (
	"archive/zip"
	"bufio"
	"bytes"
	"io"
	"sync"
	"syscall"
	"time"
)

// OnionFile struct
type OnionFile struct {
	sync.Mutex
	Name             string
	Bytes            []byte
	Checksum         string
	Encrypted        bool
	Downloads        int
	DownloadLimit    int
	DownloadsLimited bool
	CreatedAt        time.Time
	ExpiresAt        time.Time
}

func (of *OnionFile) Destroy() error {
	var err error
	buffer := bytes.NewBuffer(of.Bytes)
	zWriter := zip.NewWriter(buffer)
	reader := bufio.NewReader(bytes.NewReader(of.Bytes))
	chunk := make([]byte, 1)
	// Lock memory allotted to chunk from being used in SWAP
	if err := syscall.Mlock(chunk); err != nil {
		return err
	}
	bufFile, _ := zWriter.Create(of.Name)
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
	if err := syscall.Munlock(of.Bytes); err != nil {
		return err
	}
	return nil
}

func (of *OnionFile) IsExpired() bool {
	if of.ExpiresAt.After(time.Now()) {
		return false
	}
	return true
}
