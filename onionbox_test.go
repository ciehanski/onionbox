package main

import (
	"testing"
)

//func TestWriteFilesToBuffers(t *testing.T) {
//	ob := &onionbox{store: onion_buffer.NewStore()}
//	zb := new(bytes.Buffer)
//	zw := zip.NewWriter(zb)
//	defer zw.Close()
//	var wg sync.WaitGroup
//	wg.Add(1)
//	uploadQueue := make(chan *multipart.FileHeader, 5)
//
//	testDir, _ := ioutil.ReadDir("tests")
//	for _, file := range testDir {
//		fh := &multipart.FileHeader{Filename: fmt.Sprintf("./tests/%s", file.Name())}
//		uploadQueue <- fh
//	}
//	mpb := new(bytes.Buffer)
//	mpw := multipart.NewWriter(mpb)
//	defer mpw.Close()
//	createdFile, _ := mpw.CreateFormFile("cities.txt", "./tests/cities.txt")
//
//
//	go func() {
//		if err := ob.writeFilesToBuffers(zw, uploadQueue, &wg); err != nil {
//			t.Error(err)
//		}
//	}()
//	wg.Wait()
//}

//func BenchmarkWriteFilesToBuffers(b *testing.B) {
//	ob := &onionbox{store: onion_buffer.NewStore()}
//	zb := new(bytes.Buffer)
//	zw := zip.NewWriter(zb)
//	defer zw.Close()
//	var wg sync.WaitGroup
//	wg.Add(0)
//	uploadQueue := make(chan *multipart.FileHeader, 10000)
//
//	testDir, _ := ioutil.ReadDir("tests")
//	for _, file := range testDir {
//		fh := &multipart.FileHeader{Filename: fmt.Sprintf("./tests/%s", file.Name())}
//		uploadQueue <- fh
//	}
//
//	for n := 0; n < b.N; n++ {
//		ob.writeFilesToBuffers(zw, uploadQueue, &wg)
//	}
//}

//func TestWriteBytesInChunks(t *testing.T) {
//	ob := &onionbox{store: onion_buffer.NewStore()}
//	testFile, _ := ioutil.ReadFile("./tests/cities.txt")
//	reader := bytes.NewReader(testFile)
//	zb := new(bytes.Buffer)
//	zw := zip.NewWriter(zb)
//	defer zw.Close()
//	// Create file in zip
//	zBuffer, _ := zw.Create("cities1")
//
//	if err := ob.writeBytesInChunks(reader, zBuffer); err != nil {
//		t.Error(err)
//	}
//}
//
//func BenchmarkWriteBytesInChunks(b *testing.B) {
//	ob := &onionbox{store: onion_buffer.NewStore()}
//	testFile, _ := ioutil.ReadFile("./tests/cities.txt")
//	reader := bytes.NewReader(testFile)
//	zb := new(bytes.Buffer)
//	zw := zip.NewWriter(zb)
//	defer zw.Close()
//	// Create file in zip with same name
//	zBuffer, err := zw.Create("cities2")
//	if err != nil {
//		b.Error(err)
//	}
//	for n := 0; n < b.N; n++ {
//		ob.writeBytesInChunks(reader, zBuffer)
//	}
//}

func TestCreateCSRF(t *testing.T) {
	_, err := createCSRF()
	if err != nil {
		t.Error(err)
	}
}
