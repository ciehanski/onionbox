package onionstore

import (
	"errors"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/ciehanski/onionbox/onionbuffer"
)

type OnionStore struct {
	sync.RWMutex
	BufferFiles map[string]*onionbuffer.OnionBuffer
}

// NewStore creates a nil onionstore.
func NewStore() *OnionStore {
	return &OnionStore{BufferFiles: make(map[string]*onionbuffer.OnionBuffer, 0)}
}

func (s *OnionStore) Add(b onionbuffer.OnionBuffer) error {
	b.Lock()
	defer b.Unlock()

	s.Lock()
	s.BufferFiles[b.Name] = &b
	s.Unlock()
	// Advise the kernel not to dump. Ignore failure.
	// Unable to reference unix.MADV_DONTDUMP, raw value is 0x10 per:
	// https://godoc.org/golang.org/x/sys/unix
	if runtime.GOOS != "windows" {
		_ = unix.Madvise(b.Bytes, 0x10)
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
	return s.BufferFiles[bufName]
}

func (s *OnionStore) Exists(bufName string) bool {
	s.RLock()
	defer s.RUnlock()
	_, exists := s.BufferFiles[bufName]
	return exists
}

func (s *OnionStore) Destroy(b *onionbuffer.OnionBuffer) error {
	s.Lock()
	defer s.Unlock()

	if err := b.Destroy(); err != nil {
		return err
	}
	// Remove from store
	delete(s.BufferFiles, b.Name)
	// Free niled allotted memory for SWAP usage
	if err := s.BufferFiles[b.Name].Munlock(); err != nil {
		return err
	}

	// Force garbage collection
	runtime.GC()

	return nil
}

func (s *OnionStore) DestroyAll() error {
	if len(s.BufferFiles) != 0 {
		for _, b := range s.BufferFiles {
			if err := s.Destroy(b); err != nil {
				return err
			}
		}
		return nil
	}
	return errors.New("store already empty")
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
							if err := s.Destroy(f); err != nil {
								return err
							}
						}
					}
				}
				s.RUnlock()
			}
		}
	}
}
