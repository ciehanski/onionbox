package onion_buffer

import (
	"runtime"
	"syscall"
	"time"
)

type OnionStore struct {
	BufferFiles []*OnionBuffer
}

// Used to create a nil store.
func NewStore() *OnionStore {
	return &OnionStore{BufferFiles: make([]*OnionBuffer, 0)}
}

func (s *OnionStore) Add(oBuffer *OnionBuffer) error {
	oBuffer.Lock()
	s.BufferFiles = append(s.BufferFiles, oBuffer)
	if err := syscall.Mlock(oBuffer.Bytes); err != nil {
		return err
	}
	oBuffer.Unlock()
	return nil
}

func (s *OnionStore) Get(bufName string) *OnionBuffer {
	for _, f := range s.BufferFiles {
		if f.Name == bufName {
			return f
		}
	}
	return nil
}

func (s *OnionStore) Destroy(oBuffer *OnionBuffer) error {
	for i, f := range s.BufferFiles {
		if f.Name == oBuffer.Name {
			if err := f.Destroy(); err != nil {
				return err
			}
			// Remove from s
			f.Lock()
			s.BufferFiles = append(s.BufferFiles[:i], s.BufferFiles[i+1:]...)
			// Free niled allotted memory for SWAP usage
			if err := syscall.Munlock(f.Bytes); err != nil {
				return err
			}
			f.Unlock()
		}
	}
	return nil
}

func (s *OnionStore) Exists(bufName string) bool {
	for _, f := range s.BufferFiles {
		if f.Name == bufName {
			return true
		}
	}
	return false
}

func (s *OnionStore) DestroyAll() error {
	if s != nil {
		for i, f := range s.BufferFiles {
			if err := f.Destroy(); err != nil {
				return err
			}
			f.Lock()
			s.BufferFiles = append(s.BufferFiles[:i], s.BufferFiles[i+1:]...)
			if err := syscall.Munlock(f.Bytes); err != nil {
				return err
			}
			f.Unlock()
		}
		// TODO: needs further testing. DestroyAll should only be
		//  used when killing the application.
		runtime.GC()
	}
	return nil
}

// DestroyExpiredBuffers will indefinitely loop through the store and destroy
// expired OnionBuffers.
func (s *OnionStore) DestroyExpiredBuffers() error {
	for {
		select {
		case <-time.After(time.Second):
			if s != nil {
				for _, f := range s.BufferFiles {
					if f.Expire && f.ExpiresAt.Equal(time.Now()) || f.ExpiresAt.Before(time.Now()) {
						if err := s.Destroy(f); err != nil {
							return err
						}
					}
				}
			}
		}
	}
}
