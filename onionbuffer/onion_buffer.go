package onionbuffer

import (
	"archive/zip"
	"bufio"
	"io"
	"mime/multipart"
	"runtime"
	"sync"
	"syscall"
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
	b.Lock()
	defer b.Unlock()

	// Unlock bytes assigned to b so they can be reused for SWAP
	// since b is being deleted
	if err := b.Munlock(); err != nil {
		return err
	}

	// nil out onionbuffer
	b.Name = ""
	b.Bytes = nil
	b.Checksum = ""
	b.DownloadLimit = 0
	b.Downloads = 0
	b.Encrypted = false
	b.Expire = false
	b.ExpiresAt = time.Time{}

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

func WriteFilesToZip(w *zip.Writer, files chan *multipart.FileHeader, wg *sync.WaitGroup) error {
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
			if err := writeBytesByChunk(file, zBuffer, 1024); err != nil { // Write file in chunks to zBuffer
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
	// chunk, err := Allocate(int(chunkSize))
	chunk := make([]byte, chunkSize)
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
		_ = unix.Madvise(chunk, 0x10)
		if err := unix.Mlock(chunk); err != nil { // Lock memory allotted to chunk from being used in SWAP
			return err
		}
	}
	if err != io.EOF { // If not EOF, return the err
		return err
	}

	return nil
}

func (b *OnionBuffer) Mlock() error {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		// Advise the kernel not to dump. Ignore failure.
		// Unable to reference unix.MADV_DONTDUMP, raw value is 0x10 per:
		// https://godoc.org/golang.org/x/sys/unix
		if err := unix.Madvise(b.Bytes, 0x11); err != nil {
			return err
		}
		if err := unix.Mlock(b.Bytes); err != nil { // Lock memory allotted to chunk from being used in SWAP
			return err
		}
	} else {
		if err := syscall.Mlock(b.Bytes); err != nil {
			return err
		}
	}
	return nil
}

func (b *OnionBuffer) Munlock() error {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		if err := unix.Munlock(b.Bytes); err != nil { // Unlock memory allotted to chunk to be used for SWAP
			return err
		}
		//if err := Unallocate(b.Bytes); err != nil {
		//	// TODO: causes error
		//	return err
		//}
	} else {
		if err := syscall.Munlock(b.Bytes); err != nil {
			return err
		}
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
