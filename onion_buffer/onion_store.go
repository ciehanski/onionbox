package onion_buffer

import (
	"syscall"
)

type OnionStore struct {
	BufferFiles []*OnionBuffer
}

func (store *OnionStore) Add(of *OnionBuffer) error {
	of.Lock()
	// TODO: Below append causes nil pointer error for some reason
	store.BufferFiles = append(store.BufferFiles, of)
	if err := syscall.Mlock(of.Bytes); err != nil {
		return err
	}
	of.Unlock()
	return nil
}

func (store *OnionStore) Get(filename string) *OnionBuffer {
	for _, f := range store.BufferFiles {
		f.Lock()
		if f.Name == filename {
			return f
		}
		f.Unlock()
	}
	return nil
}

func (store *OnionStore) Delete(of *OnionBuffer) error {
	for i, f := range store.BufferFiles {
		f.Lock()
		if f.Name == of.Name {
			if err := f.Destroy(); err != nil {
				return err
			}
			store.BufferFiles = append(store.BufferFiles[:i], store.BufferFiles[i+1:]...)
			if err := syscall.Munlock(f.Bytes); err != nil {
				return err
			}
		}
		f.Unlock()
	}
	return nil
}

func (store *OnionStore) Exists(filename string) bool {
	for _, f := range store.BufferFiles {
		f.Lock()
		if f.Name == filename {
			return true
		}
		f.Unlock()
	}
	return false
}

func (store *OnionStore) DestroyAll() error {
	for i, f := range store.BufferFiles {
		f.Lock()
		if err := f.Destroy(); err != nil {
			return err
		}
		store.BufferFiles = append(store.BufferFiles[:i], store.BufferFiles[i+1:]...)
		if err := syscall.Munlock(f.Bytes); err != nil {
			return err
		}
		f.Unlock()
	}
	//runtime.GC()
	return nil
}

func DeleteExpiredBuffers() {
	// TODO: implement go routine that always checks each onion file
	//  if its expired. If so, destroy it.
}

func NewStore() *OnionStore {
	store := &OnionStore{
		BufferFiles: make([]*OnionBuffer, 0),
	}
	return store
}
