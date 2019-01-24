package main

import (
	"archive/zip"
	"bytes"
	"mime/multipart"
	"sync"
	"testing"

	"onionbox/onion_buffer"
)

func TestWriteFilesToBuffers(t *testing.T) {
	ob := &onionbox{store: onion_buffer.NewStore()}
	zb := new(bytes.Buffer)
	zw := zip.NewWriter(zb)
	var wg sync.WaitGroup
	wg.Add(1)
	uploadQueue := make(chan *multipart.FileHeader, 100)
	for x := 0; x <= 5; x++ {
		uploadQueue <- new(multipart.FileHeader, 100000000)
	}
	go func() {
		if err := ob.writeFilesToBuffers(zw, uploadQueue, &wg); err != nil {
			t.Error(err)
		}
	}()
}

func BenchmarkWriteFilesToBuffers(b *testing.B) {
	ob := &onionbox{store: onion_buffer.NewStore()}
	zb := new(bytes.Buffer)
	zw := zip.NewWriter(zb)
	var wg sync.WaitGroup
	wg.Add(1)
	uploadQueue := make(chan *multipart.FileHeader, 100)
	for x := 0; x <= 5; x++ {
		uploadQueue <- new(multipart.FileHeader, 100000000)
	}
	for n := 0; n < b.N; n++ {
		ob.writeFilesToBuffers(zw, uploadQueue, &wg)
	}
}
