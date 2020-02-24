package onionbuffer

import (
	"archive/zip"
	"bytes"
	"crypto/subtle"
	"io"
	"io/ioutil"
	"testing"
	"time"
)

//func TestWriteFilesToBuffers(t *testing.T) {
//	ob := &onionbox.Onionbox{Store: NewStore()}
//	// Prepare a form that you will submit to that URL.
//	var b bytes.Buffer
//	var err error
//	w := multipart.NewWriter(&b)
//	values := map[string]io.Reader{
//		"file":  mustOpen("main.go"), // lets assume its this file
//	}
//	for key, r := range values {
//		var fw io.Writer
//		if x, ok := r.(io.Closer); ok {
//			defer x.Close()
//		}
//		// Add an image file
//		if x, ok := r.(*os.File); ok {
//			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
//				t.Error(err)
//			}
//		}
//		if _, err = io.Copy(fw, r); err != nil {
//			t.Error(err)
//		}
//	}
//	// Don't forget to close the multipart writer.
//	// If you don't close it, your request will be missing the terminating boundary.
//	w.Close()
//
//	// Now that you have a form, you can submit it to your handler.
//	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s.onion", ob.OnionURL), &b)
//	if err != nil {
//		return
//	}
//	// Don't forget to set the content type, this will contain the boundary.
//	req.Header.Set("Content-Type", w.FormDataContentType())
//	// Submit the request
//	var client *http.Client
//	res, err := client.Do(req)
//	if err != nil {
//		return
//	}
//	// Check the response
//	if res.StatusCode != http.StatusOK {
//		err = fmt.Errorf("bad status: %s", res.Status)
//	}
//
//	req.ParseMultipartForm(32 << 20)
//	files := req.MultipartForm.File["files"]
//
//	zb := new(bytes.Buffer)
//	zw := zip.NewWriter(zb)
//	defer zw.Close()
//	wg := new(sync.WaitGroup)
//	wg.Add(len(files))
//
//	go func() {
//		if err := WriteFilesToBuffer(zw, files, wg, 1024); err != nil {
//			t.Error(err)
//		}
//	}()
//}

//func BenchmarkWriteFilesToBuffers(b *testing.B) {
//	ob := &onionbox{Store: onion_buffer.NewStore()}
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

func TestDestroy(t *testing.T) {
	testFile, _ := ioutil.ReadFile("../tests/gopher.jpg")
	buf := OnionBuffer{Name: "testing_destroy", Bytes: testFile}
	err := buf.Destroy()
	if err != nil {
		if err.Error() != "invalid argument" {
			t.Error(err)
		}
	}
	if len(buf.Bytes) != 0 {
		t.Errorf("bytes not destroyed")
	}
	if buf.Name == "testing_destory" {
		t.Error("bytes not destroyed")
	}
}

func TestWriteBytesInChunks(t *testing.T) {
	testFile, _ := ioutil.ReadFile("../tests/gopher.jpg")
	reader := bytes.NewReader(testFile)
	zb := new(bytes.Buffer)
	zw := zip.NewWriter(zb)
	defer zw.Close()
	// Create file in zip
	zBuffer, _ := zw.Create("testzip1")

	if err := writeBytesByChunk(reader, zBuffer, 1024); err != nil {
		t.Error(err)
	}
}

func BenchmarkWriteBytesInChunks(b *testing.B) {
	testFile, _ := ioutil.ReadFile("../tests/gopher.jpg")
	reader := bytes.NewReader(testFile)
	zb := new(bytes.Buffer)
	zw := zip.NewWriter(zb)
	defer zw.Close()
	// Create file in zip with same name
	zBuffer, _ := zw.Create("testzip2")

	for n := 0; n < b.N; n++ {
		writeBytesByChunk(reader, zBuffer, 1)
	}
}

func BenchmarkSubtleTimeCopy(b *testing.B) {
	testFile, _ := ioutil.ReadFile("../tests/gopher.jpg")
	zb := make([]byte, len(testFile))
	for n := 0; n < b.N; n++ {
		subtle.ConstantTimeCopy(1, zb, testFile)
	}
}

func BenchmarkIOCopy(b *testing.B) {
	testFile, _ := ioutil.ReadFile("../tests/gopher.jpg")
	reader := bytes.NewReader(testFile)
	zb := new(bytes.Buffer)
	zw := zip.NewWriter(zb)
	defer zw.Close()
	// Create file in zip with same name
	zBuffer, _ := zw.Create("testzip2")

	for n := 0; n < b.N; n++ {
		io.Copy(zBuffer, reader)
	}
}

func TestIsExpired(t *testing.T) {
	ob := OnionBuffer{Name: "testing_expired", Bytes: new(bytes.Buffer).Bytes(), Expire: true, ExpiresAt: time.Now()}
	if !ob.IsExpired() {
		t.Error("Expected result was an expired onionbuffer")
	}
}

//func mustOpen(f string) *os.File {
//	r, err := os.Open(f)
//	if err != nil {
//		panic(err)
//	}
//	return r
//}
