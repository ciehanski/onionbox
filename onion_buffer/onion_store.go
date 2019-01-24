package onion_buffer

import (
	"log"
	"syscall"
	"time"
)

type OnionStore struct {
	BufferFiles []*OnionBuffer
}

func (store *OnionStore) Add(oBuffer *OnionBuffer) error {
	oBuffer.Lock()
	store.BufferFiles = append(store.BufferFiles, oBuffer)
	if err := syscall.Mlock(oBuffer.Bytes); err != nil {
		return err
	}
	oBuffer.Unlock()
	return nil
}

func (store *OnionStore) Get(bufName string) *OnionBuffer {
	for _, f := range store.BufferFiles {
		if f.Name == bufName {
			return f
		}
	}
	return nil
}

func (store *OnionStore) Delete(oBuffer *OnionBuffer) error {
	for i, f := range store.BufferFiles {
		if f.Name == oBuffer.Name {
			if err := f.Destroy(); err != nil {
				return err
			}
			// Remove from store
			f.Lock()
			store.BufferFiles = append(store.BufferFiles[:i], store.BufferFiles[i+1:]...)
			// Free niled allotted memory for SWAP usage
			if err := syscall.Munlock(f.Bytes); err != nil {
				return err
			}
			f.Unlock()
		}
	}
	return nil
}

func (store *OnionStore) Exists(bufName string) bool {
	for _, f := range store.BufferFiles {
		if f.Name == bufName {
			return true
		}
	}
	return false
}

func (store *OnionStore) DestroyAll() error {
	for i, f := range store.BufferFiles {
		if err := f.Destroy(); err != nil {
			return err
		}
		f.Lock()
		store.BufferFiles = append(store.BufferFiles[:i], store.BufferFiles[i+1:]...)
		if err := syscall.Munlock(f.Bytes); err != nil {
			return err
		}
		f.Unlock()
	}
	//runtime.GC()
	return nil
}

func NewStore() *OnionStore {
	return &OnionStore{BufferFiles: make([]*OnionBuffer, 0)}
}

func (store *OnionStore) DestroyExpiredBuffers() error {
	for {
		if store != nil {
			for _, f := range store.BufferFiles {
				if f.Expire {
					if f.ExpiresAt.Equal(time.Now()) || f.ExpiresAt.Before(time.Now()) {
						log.Printf("Destroyed expired buffer %s", f.Name)
						if err := f.Destroy(); err != nil {
							return err
						}
					}
				}
			}
		}
	}
}
