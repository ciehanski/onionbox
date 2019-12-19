package onionstore

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/ciehanski/onionbox/onionbuffer"
)

func TestGet(t *testing.T) {
	os := NewStore()
	testFile, _ := ioutil.ReadFile("../../../tests/gopher.jpg")
	oBuf := onionbuffer.OnionBuffer{Name: "testing_get", Bytes: testFile}
	os.Add(oBuf)
	if getOBuf := os.Get("testing_get"); getOBuf == nil {
		t.Error("Onion buffer not expected to be nil")
	}
}

func TestExists(t *testing.T) {
	os := NewStore()
	testFile, _ := ioutil.ReadFile("../../../tests/gopher.jpg")
	oBuf := onionbuffer.OnionBuffer{Name: "testing_exists", Bytes: testFile}
	os.Add(oBuf)
	if !os.Exists("testing_exists") {
		t.Error("Onion buffer should exist")
	}
}

func TestAdd(t *testing.T) {
	os := NewStore()
	testFile, _ := ioutil.ReadFile("../../../tests/gopher.jpg")
	oBuf := onionbuffer.OnionBuffer{Name: "testing_add", Bytes: testFile}
	if err := os.Add(oBuf); err != nil {
		t.Error(err)
	}
}

func TestDestroy(t *testing.T) {
	os := NewStore()
	testFile, _ := ioutil.ReadFile("../../../tests/gopher.jpg")
	oBuf := onionbuffer.OnionBuffer{Name: "testing_destroy", Bytes: testFile}
	os.Add(oBuf)
	if err := os.Destroy(&oBuf); err != nil {
		if err.Error() != "invalid argument" {
			t.Error(err)
		}
	}
	if len(os.BufferFiles) != 0 {
		t.Errorf("Expected empty store, but got %v onionbuffer(s)", len(os.BufferFiles))
	}
}

func TestDestroyAll(t *testing.T) {
	os := NewStore()
	testFile, _ := ioutil.ReadFile("../../../tests/gopher.jpg")
	oBuf1 := onionbuffer.OnionBuffer{Name: "testing_destroyall1", Bytes: testFile}
	oBuf2 := onionbuffer.OnionBuffer{Name: "testing_destroyall2", Bytes: testFile}
	oBuf3 := onionbuffer.OnionBuffer{Name: "testing_destroyall3", Bytes: testFile}
	oBuf4 := onionbuffer.OnionBuffer{Name: "testing_destroyall4", Bytes: testFile}
	os.Add(oBuf1)
	os.Add(oBuf2)
	os.Add(oBuf3)
	os.Add(oBuf4)
	if err := os.DestroyAll(); err != nil {
		if err.Error() != "invalid argument" {
			t.Error(err)
		}
	}
	if len(os.BufferFiles) != 0 {
		t.Errorf("Expected empty store, but got %v onionbuffer(s)", len(os.BufferFiles))
	}
}

func BenchmarkAppendDestroy(t *testing.B) {
	s := NewStore()
	testFile, _ := ioutil.ReadFile("../../../tests/gopher.jpg")
	oBuf1 := onionbuffer.OnionBuffer{Name: "testing_appenddestory", Bytes: testFile}
	s.Add(oBuf1)
	for i, _ := range s.BufferFiles {
		s.BufferFiles = append(s.BufferFiles[:i], s.BufferFiles[i+1:]...)
	}
}

func BenchmarkNonAppendDestroy(t *testing.B) {
	s := NewStore()
	testFile, _ := ioutil.ReadFile("../../../tests/gopher.jpg")
	oBuf1 := onionbuffer.OnionBuffer{Name: "testing_nonappenddestory", Bytes: testFile}
	s.Add(oBuf1)
	for i, _ := range s.BufferFiles {
		s.BufferFiles[i] = s.BufferFiles[len(s.BufferFiles)-1]
		s.BufferFiles[len(s.BufferFiles)-1] = &onionbuffer.OnionBuffer{}
		s.BufferFiles = s.BufferFiles[:len(s.BufferFiles)-1]
	}
}

func TestDestroyExpiredBuffers(t *testing.T) {
	os := NewStore()
	testFile, _ := ioutil.ReadFile("../../../tests/gopher.jpg")
	oBuf := onionbuffer.OnionBuffer{Name: "testing_destroyexpired", Bytes: testFile, Expire: true, ExpiresAt: time.Now()}
	os.Add(oBuf)
	go func() {
		if err := os.DestroyExpiredBuffers(); err != nil {
			if err.Error() != "invalid argument" {
				t.Error(err)
			}
		}
	}()
	time.Sleep(time.Second * 10)
	if len(os.BufferFiles) != 0 {
		t.Errorf("Expected empty store, but got %v onionbuffer(s)", len(os.BufferFiles))
	}
}
