package onionbuffer

import (
	"archive/zip"
	"bufio"
	"bytes"
	"io"
	"mime/multipart"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// OnionBuffer struct
type OnionBuffer struct {
	sync.RWMutex
	Name          string
	Bytes         []byte
	Checksum      string
	Encrypted     bool
	Downloads     int64
	DownloadLimit int64
	Expire        bool
	ExpiresAt     time.Time
}

// Destroy is mostly used to destroy temporary OnionBuffer objects after they
// have been copied to the store or to remove an individual OnionBuffer
// from the store.
func (b *OnionBuffer) Destroy() error {
	var err error
	buffer := bytes.NewBuffer(b.Bytes)
	zWriter := zip.NewWriter(buffer)
	reader := bufio.NewReader(bytes.NewReader(b.Bytes))
	chunk := make([]byte, 1)
	bufFile, err := zWriter.Create(b.Name)
	if err != nil {
		return err
	}

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

	if err := b.Munlock(); err != nil {
		return err
	}

	return nil
}

// IsExpired is used to check if an OnionBuffer is expired or not.
func (b *OnionBuffer) IsExpired() bool {
	b.RLock()
	defer b.RUnlock()
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
	defer b.Unlock()
	t, err := time.ParseDuration(expiration)
	if err != nil {
		return err
	}
	b.Expire = true
	b.ExpiresAt = time.Now().Add(t)
	return nil
}

func WriteFilesToBuffers(w *zip.Writer, files chan *multipart.FileHeader, wg *sync.WaitGroup) error {
	for {
		select {
		case fileHeader := <-files:
			file, err := fileHeader.Open() // Open uploaded file
			if err != nil {
				return err
			}

			zBuffer, err := w.Create(fileHeader.Filename) // Create file in zip with same name
			if err != nil {
				return err
			}
			if err := writeBytesByChunk(file, zBuffer, 256); err != nil { // Write file in chunks to zBuffer
				return err
			}
			// Flush zipwriter to write compressed bytes to buffer
			// before moving onto the next file
			if err := w.Flush(); err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
			wg.Done()
		}
	}
}

func writeBytesByChunk(file io.Reader, bufWriter io.Writer, chunkSize int64) error {
	var count int
	var err error
	reader := bufio.NewReader(file) // Read uploaded file
	chunk, err := Allocate(int(chunkSize))
	if err != nil {
		return err
	}
	for {
		if count, err = reader.Read(chunk); err != nil { // Read the specific chunk of uploaded file
			break
		}
		if _, err := bufWriter.Write(chunk[:count]); err != nil {
			return err // Write the specific chunk to the new zip entry
		}
		// Advise the kernel not to dump. Ignore failure.
		// Unable to reference unix.MADV_DONTDUMP, raw value is 0x10 per:
		// https://godoc.org/golang.org/x/sys/unix
		unix.Madvise(chunk, 0x10)
		if err := unix.Mlock(chunk); err != nil { // Lock memory allotted to chunk from being used in SWAP
			return err
		}
	}
	if err != io.EOF { // If not EOF, return the err
		return err
	} else { // if EOF, do not return an error
		err = nil
	}
	return nil
}

func (b *OnionBuffer) Mlock() error {
	if runtime.GOOS != "windows" {
		// Advise the kernel not to dump. Ignore failure.
		// Unable to reference unix.MADV_DONTDUMP, raw value is 0x10 per:
		// https://godoc.org/golang.org/x/sys/unix
		// TODO: causes error
		unix.Madvise(b.Bytes, 0x10)
		if err := unix.Mlock(b.Bytes); err != nil { // Lock memory allotted to chunk from being used in SWAP
			return err
		}
	} else {
		// Do windows stuff
	}
	return nil
}

func (b *OnionBuffer) Munlock() error {
	if runtime.GOOS != "windows" {
		if err := unix.Munlock(b.Bytes); err != nil { // Unlock memory allotted to chunk to be used for SWAP
			return err
		}
		if err := Unallocate(b.Bytes); err != nil {
			// TODO: causes error
			return err
		}
	} else {
		// Do windows stuff
	}
	return nil
}

func Allocate(length int) ([]byte, error) {
	b, err := unix.Mmap(-1, 0, length, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANONYMOUS)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func Unallocate(bytes []byte) error {
	if err := unix.Munmap(bytes); err != nil {
		// TODO: causes error, change back to err eventually
		return err
	}
	return nil
}

//func UploadMR(mr *multipart.Reader, writer io.Writer) error {
//	var count int
//	var err error
//	var part *multipart.Part
//	chunk, err := Allocate(4096)
//	if err != nil {
//		return err
//	}
//
//	for {
//		part, err = mr.NextPart()
//		if err == io.EOF {
//			// err is io.EOF, files upload completed
//			break
//		}
//		if err != nil {
//			// A normal error occurred
//			return err
//		}
//
//		for {
//			count, err = part.Read(chunk)
//			if err == io.EOF {
//				break
//			}
//			if err != nil {
//				return err
//			}
//			if _, err := writer.Write(chunk[:count]); err != nil {
//				return err
//			}
//		}
//	}
//	return nil
//}
