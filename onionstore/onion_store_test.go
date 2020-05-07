package onionstore

import (
	"io/ioutil"
	"testing"

	"github.com/ciehanski/onionbox/onionbuffer"
)

func TestGet(t *testing.T) {
	os := NewStore()
	testFile, _ := ioutil.ReadFile("../tests/gopher.jpg")
	oBuf := onionbuffer.OnionBuffer{Name: "testing_get", Bytes: testFile}
	_ = os.Add(&oBuf)
	if _, err := os.Get("testing_get"); err != nil {
		t.Error("Onion buffer not expected to be nil")
	}
}

func TestExists(t *testing.T) {
	os := NewStore()
	testFile, _ := ioutil.ReadFile("../tests/gopher.jpg")
	oBuf := onionbuffer.OnionBuffer{Name: "testing_exists", Bytes: testFile}
	_ = os.Add(&oBuf)
	if !os.Exists("testing_exists") {
		t.Error("Onion buffer should exist")
	}
}

func TestAdd(t *testing.T) {
	os := NewStore()
	testFile, _ := ioutil.ReadFile("../tests/gopher.jpg")
	oBuf := onionbuffer.OnionBuffer{Name: "testing_add", Bytes: testFile}
	if err := os.Add(&oBuf); err != nil {
		t.Error(err)
	}
	if len(os.BufferFiles["testing_add"].Bytes) == 0 {
		t.Errorf("bytes not added to onionstore")
	}
	if len(os.BufferFiles["testing_add"].Bytes) != len(testFile) {
		t.Error("incorrect bytes added")
	}
	if os.BufferFiles["testing_add"].Name != "testing_add" {
		t.Error("incorrect onionbuffer name")
	}
}

func TestDestroy(t *testing.T) {
	os := NewStore()
	testFile, _ := ioutil.ReadFile("../tests/gopher.jpg")
	oBuf := onionbuffer.OnionBuffer{Name: "testing_destroy", Bytes: testFile}
	_ = os.Add(&oBuf)
	if err := os.Destroy(&oBuf); err != nil {
		if err.Error() != "invalid argument" {
			t.Error(err)
		}
	}
	if _, err := os.Get(oBuf.Name); err != nil {
		t.Error("should have failed to get after destroy")
	}
	if len(os.BufferFiles["testing_destroy"].Bytes) != 0 {
		t.Error("bytes not destroyed")
	}
}

func TestDestroyAll(t *testing.T) {
	os := NewStore()
	testFile, _ := ioutil.ReadFile("../tests/gopher.jpg")
	oBuf1 := onionbuffer.OnionBuffer{Name: "testing_destroyall1", Bytes: testFile}
	oBuf2 := onionbuffer.OnionBuffer{Name: "testing_destroyall2", Bytes: testFile}
	oBuf3 := onionbuffer.OnionBuffer{Name: "testing_destroyall3", Bytes: testFile}
	oBuf4 := onionbuffer.OnionBuffer{Name: "testing_destroyall4", Bytes: testFile}
	_ = os.Add(&oBuf1)
	_ = os.Add(&oBuf2)
	_ = os.Add(&oBuf3)
	_ = os.Add(&oBuf4)

	if err := os.DestroyAll(); err != nil {
		if err.Error() != "invalid argument" {
			t.Error(err)
		}
	}
	for i := range os.BufferFiles {
		if _, err := os.Get(i); err == nil {
			t.Error("should have failed to get after destroy")
		}
	}
}

//func TestDestroyExpiredBuffers(t *testing.T) {
//	os := NewStore()
//	testFile, _ := ioutil.ReadFile("../tests/gopher.jpg")
//	oBuf := onionbuffer.OnionBuffer{Name: "testing_destroyexpired", Bytes: testFile, Expire: true, ExpiresAt: time.Now()}
//	_ = os.Add(oBuf)
//	go func() {
//		if err := os.DestroyExpiredBuffers(); err != nil {
//			if err.Error() != "invalid argument" {
//				t.Error(err)
//			}
//		}
//	}()
//	time.Sleep(time.Second * 10)
//	if b := os.Get(oBuf.Name); b != nil {
//		t.Errorf("should have failed Get after destroy")
//	}
//}
