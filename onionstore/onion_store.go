package onionstore

import (
	"crypto/subtle"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/ciehanski/onionbox/onionbuffer"
)

type OnionStore struct {
	sync.RWMutex
	BufferFiles []*onionbuffer.OnionBuffer
}

// NewStore creates a nil onionstore.
func NewStore() *OnionStore {
	return &OnionStore{BufferFiles: make([]*onionbuffer.OnionBuffer, 0)}
}

func (s *OnionStore) Add(b onionbuffer.OnionBuffer) error {
	b.Lock()
	defer b.Unlock()

	s.Lock()
	s.BufferFiles = append(s.BufferFiles, &b)
	s.Unlock()
	// Advise the kernel not to dump. Ignore failure.
	// Unable to reference unix.MADV_DONTDUMP, raw value is 0x10 per:
	// https://godoc.org/golang.org/x/sys/unix
	if runtime.GOOS != "windows" {
		unix.Madvise(b.Bytes, 0x10)
	}
	// Lock bytes from SWAP
	if err := b.Mlock(); err != nil {
		return err
	}
	return nil
}

func (s *OnionStore) Get(bufName string) *onionbuffer.OnionBuffer {
	s.RLock()
	defer s.RUnlock()
	for _, f := range s.BufferFiles {
		if subtle.ConstantTimeCompare([]byte(f.Name), []byte(bufName)) == 1 {
			return f
		}
	}
	return nil
}

func (s *OnionStore) Exists(bufName string) bool {
	s.RLock()
	defer s.RUnlock()
	for _, f := range s.BufferFiles {
		if subtle.ConstantTimeCompare([]byte(f.Name), []byte(bufName)) == 1 {
			return true
		}
	}
	return false
}

func (s *OnionStore) Destroy(b *onionbuffer.OnionBuffer) error {
	s.Lock()
	defer s.Unlock()
	for i, f := range s.BufferFiles {
		if subtle.ConstantTimeCompare([]byte(f.Name), []byte(b.Name)) == 1 {
			f.Lock()
			if err := b.Destroy(); err != nil {
				return err
			}
			// Remove from store
			s.BufferFiles = append(s.BufferFiles[:i], s.BufferFiles[i+1:]...)
			// Free niled allotted memory for SWAP usage
			if err := f.Munlock(); err != nil {
				return err
			}
			f.Unlock()
			// Force garbage collection
			// debug.FreeOSMemory()
			break
		}
	}
	return nil
}

func (s *OnionStore) DestroyAll() error {
	if s != nil {
		for i := len(s.BufferFiles) - 1; i >= 0; i-- {
			f := s.BufferFiles[i]
			if err := s.Destroy(f); err != nil {
				return err
			}
		}
		// Force garbage collection
		debug.FreeOSMemory()
	}
	return nil
}

// DestroyExpiredBuffers will indefinitely loop through the store and destroy
// expired OnionBuffers.
func (s *OnionStore) DestroyExpiredBuffers() error {
	for {
		select {
		case <-time.After(time.Second * 5):
			if s != nil {
				s.RLock()
				for _, f := range s.BufferFiles {
					if f.Expire {
						if f.ExpiresAt.Equal(time.Now()) || f.ExpiresAt.Before(time.Now()) {
							// TODO: debug, remove
							fmt.Println(f.Name)
							if err := s.Destroy(f); err != nil {
								return err
							}
							// Force garbage collection
							// debug.FreeOSMemory()
						}
					}
				}
				s.RUnlock()
			}
		}
	}
}
