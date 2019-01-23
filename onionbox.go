package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
	"onionbox/onion_buffer"
	"onionbox/templates"
)

type onionbox struct {
	debug       bool
	logger      *log.Logger
	store       *onion_buffer.OnionStore
	maxMemory   int64
	torVersion3 bool
	onionURL    string
	chunkSize   int
}

func main() {
	// Create onionbox instance that stores config
	ob := onionbox{
		logger: log.New(os.Stdout, "[onionbox] ", log.LstdFlags),
		store:  onion_buffer.NewStore(),
	}
	// Init flags
	flag.BoolVar(&ob.debug, "debug", false, "run in debug mode")
	flag.BoolVar(&ob.torVersion3, "torv3", true, "use version 3 of the Tor circuit")
	flag.Int64Var(&ob.maxMemory, "mem", 128, "max memory allotted for handling file buffers")
	flag.IntVar(&ob.chunkSize, "chunk", 1024, "size of chunks for buffer I/O")
	// Parse flags
	flag.Parse()

	// Start tor
	ob.logf("Starting and registering onion service, please wait...")
	t, err := tor.Start(nil, &tor.StartConf{
		ProcessCreator: libtor.Creator,
		DebugWriter:    os.Stderr,
	})
	if err != nil {
		ob.logf("Failed to start Tor: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := t.Close(); err != nil {
			ob.logf("Error closing connection to Tor: %v", err)
			os.Exit(1)
		}
	}()

	// Wait at most a few minutes to publish the service
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Create an onion service to listen on any port but show as 80
	onionSvc, err := t.Listen(ctx, &tor.ListenConf{RemotePorts: []int{80}, Version3: ob.torVersion3})
	if err != nil {
		ob.logf("Failed to create onion service: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := onionSvc.Close(); err != nil {
			ob.logf("Error closing connection to onion service: %v", err)
			os.Exit(1)
		}
	}()

	ob.onionURL = onionSvc.ID
	ob.logf("Please open a Tor capable browser and navigate to http://%v.onion\n", onionSvc.ID)

	// Init routes
	http.HandleFunc("/", ob.router)
	// Init serving
	srv := &http.Server{
		IdleTimeout:  time.Second * 60,
		ReadTimeout:  time.Second * 60,
		WriteTimeout: time.Second * 60,
		Handler:      nil,
	}
	// Begin serving
	go ob.logger.Fatal(srv.Serve(onionSvc))
	// Proper srv shutdown when program ends
	defer func() {
		if err := srv.Shutdown(context.Background()); err != nil {
			ob.logf("Error shutting down onionbox srv: %v", err)
			os.Exit(1)
		}
	}()
}

func (ob *onionbox) router(w http.ResponseWriter, r *http.Request) {
	// Set download url regex
	downloadURLreg := regexp.MustCompile(`((?:[a-z][a-z]+))`)
	if r.URL.Path == "/" {
		ob.upload(w, r)
	} else if matches := downloadURLreg.FindStringSubmatch(r.URL.Path); matches != nil {
		if ob.store != nil {
			if ob.store.Exists(r.URL.Path[1:]) {
				r.Header.Set("filename", r.URL.Path[1:])
				ob.download(w, r)
			}
		}
		http.Error(w, "File not found", http.StatusNotFound)
	} else {
		http.Error(w, "404 page not found", http.StatusNotFound)
	}
}

func (ob *onionbox) upload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		csrf, err := createCSRF()
		if err != nil {
			ob.logf("Error creating CSRF token: %v", err)
			http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
		}
		// Parse template
		t, err := template.New("upload").Parse(templates.UploadHTML)
		if err != nil {
			ob.logf("Error loading template: %v", err)
			http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
		}
		// Execute template
		if err := t.Execute(w, csrf); err != nil {
			ob.logf("Error executing template: %v", err)
			http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
		}
	case http.MethodPost:
		// Parse file(s) from form
		if err := r.ParseMultipartForm(ob.maxMemory << 20); err != nil {
			ob.logf("Error parsing files from form: %v", err)
			http.Error(w, "Error parsing files.", http.StatusInternalServerError)
		}
		// Create buffer for session in-memory zip file
		zipBuffer := new(bytes.Buffer)
		// Lock memory allotted to zipBuffer from being used in SWAP
		if err := syscall.Mlock(zipBuffer.Bytes()); err != nil {
			ob.logf("Error mlocking allotted memory for zipBuffer: %v", err)
		}
		zWriter := zip.NewWriter(zipBuffer)
		files := r.MultipartForm.File["files"]
		// Loop through all files in the form
		for _, fileHeader := range files {
			// Open uploaded file
			file, err := fileHeader.Open()
			if err != nil {
				ob.logf("Error opening file %s: %v", fileHeader.Filename, err)
				http.Error(w, "Error uploading files.", http.StatusInternalServerError)
			}
			// Create file in zip with same name
			bufFile, err := zWriter.Create(fileHeader.Filename)
			if err != nil {
				ob.logf("Error creating new file in zip: %v", err)
				http.Error(w, "Error uploading files.", http.StatusInternalServerError)
			}
			// Read uploaded file
			var count int
			reader := bufio.NewReader(file)
			chunk := make([]byte, ob.chunkSize)
			// Lock memory allotted to chunk from being used in SWAP
			if err := syscall.Mlock(chunk); err != nil {
				ob.logf("Error mlocking allotted memory for chunk: %v", err)
			}
			for {
				if count, err = reader.Read(chunk); err != nil {
					break
				}
				_, err := bufFile.Write(chunk[:count])
				if err != nil {
					ob.logf("Error writing file to zip: %v", err)
					http.Error(w, "Error writing file to zip", http.StatusInternalServerError)
				}
			}
			if err != io.EOF {
				ob.logf("Error reading uploaded file: %v", err)
				http.Error(w, "Error reading uploaded file.", http.StatusInternalServerError)
			} else {
				err = nil
			}
			// Flush zipwriter to write compressed bytes to buffer
			if err := zWriter.Flush(); err != nil {
				ob.logf("Error flushing zip writer: %v", err)
			}
		}
		// Close zipwriter
		if err := zWriter.Close(); err != nil {
			ob.logf("Error closing zip writer: %v", err)
		}
		// Create random zip name
		zipFileName := strings.ToLower(randomdata.SillyName())
		// Create OnionBuffer object
		oFile := &onion_buffer.OnionBuffer{Name: zipFileName, CreatedAt: time.Now()}
		// If password option was enabled
		if r.FormValue("password_enabled") == "on" {
			var err error
			pass := r.FormValue("password")
			oFile.Bytes, err = onion_buffer.Encrypt(zipBuffer.Bytes(), pass)
			if err != nil {
				ob.logf("Error encrypting buffer: %v", err)
				http.Error(w, "Error encrypting buffer.", http.StatusInternalServerError)
			}
			// Lock memory allotted to oFile from being used in SWAP
			if err := syscall.Mlock(oFile.Bytes); err != nil {
				ob.logf("Error mlocking allotted memory for oFile: %v", err)
			}
			oFile.Encrypted = true
			chksm, err := oFile.GetChecksum()
			if err != nil {
				ob.logf("Error getting checksum: %v", err)
				http.Error(w, "Error getting checksum.", http.StatusInternalServerError)
			}
			oFile.Checksum = chksm
		} else {
			oFile.Bytes = zipBuffer.Bytes()
			chksm, err := oFile.GetChecksum()
			if err != nil {
				ob.logf("Error getting checksum: %v", err)
				http.Error(w, "Error getting checksum.", http.StatusInternalServerError)
			}
			oFile.Checksum = chksm
		}
		// If limit downloads was enabled
		if r.FormValue("limit_downloads") == "on" {
			form := r.FormValue("download_limit")
			limit, err := strconv.Atoi(form)
			if err != nil {
				ob.logf("Error converting duration string into time.Duration: %v", err)
				http.Error(w, "Error getting expiration time.", http.StatusInternalServerError)
			}
			oFile.DownloadLimit = limit
		}
		// if expiration was enabled
		if r.FormValue("expire") == "on" {
			expiration := r.FormValue("expiration_time")
			t, err := time.ParseDuration(expiration)
			if err != nil {
				ob.logf("Error parsing expiration time: %v", err)
				http.Error(w, "Error parsing expiration time.", http.StatusInternalServerError)
			}
			oFile.ExpiresAt = oFile.CreatedAt.Add(t)
		}
		// Append onion file to filestore
		if err := ob.store.Add(oFile); err != nil {
			ob.logf("Error adding file to store: %v", err)
			http.Error(w, "Error adding file to store.", http.StatusInternalServerError)
		}
		// Set temp oFile var to nil
		if err := oFile.Destroy(); err != nil {
			ob.logf("Error destroying temporary var for %s", oFile.Name)
		}
		// Write the zip's URL to client for sharing
		_, err := w.Write([]byte(fmt.Sprintf("Files uploaded. Please share this link with your recipients: http://%s.onion/%s",
			ob.onionURL, oFile.Name)))
		if err != nil {
			ob.logf("Error writing to client: %v", err)
			http.Error(w, "Error writing to client.", http.StatusInternalServerError)
		}
	default:
		http.Error(w, "Invalid HTTP Method.", http.StatusMethodNotAllowed)
	}
}

func (ob *onionbox) download(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		of := ob.store.Get(r.Header.Get("filename"))
		if of == nil {
			http.Error(w, "Nil file", http.StatusInternalServerError)
		}
		if of.Encrypted {
			csrf, err := createCSRF()
			if err != nil {
				ob.logf("Error creating CSRF token: %v", err)
				http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
			}
			// Parse template
			t, err := template.New("download_encrypted").Parse(templates.DownloadHTML)
			if err != nil {
				ob.logf("Error loading template: %v", err)
				http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
			}
			// Execute template
			if err := t.Execute(w, csrf); err != nil {
				ob.logf("Error executing template: %v", err)
				http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
			}
		} else {
			if of.DownloadLimit > 0 && of.Downloads >= of.DownloadLimit {
				if err := ob.store.Delete(of); err != nil {
					ob.logf("Error deleting onion file from store: %v", err)
				}
				ob.logf("Download limit reached for %s", of.Name)
				http.Error(w, "Download limit reached.", http.StatusUnauthorized)
			}
			// Check expiration
			if of.IsExpired() {
				if err := of.Destroy(); err != nil {
					ob.logf("Error destroying buffer %s: %v", of.Name, err)
				}
				http.Error(w, "Download link has expired", http.StatusUnauthorized)
			}
			// Validate checksum
			chksmValid, err := of.ValidateChecksum()
			if err != nil {
				ob.logf("Error validating checksum: %v", err)
				http.Error(w, "Error validating checksum.", http.StatusInternalServerError)
			}
			if !chksmValid {
				ob.logf("Invalid checksum for file %s", of.Name)
				http.Error(w, "Invalid checksum.", http.StatusInternalServerError)
			}
			// Increment files download count
			of.Downloads++
			// Set headers for browser to initiate download
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", of.Name))
			// Write the zip bytes to the response for download
			_, err = w.Write(of.Bytes)
			if err != nil {
				ob.logf("Error writing to client: %v", err)
				http.Error(w, "Error writing to client.", http.StatusInternalServerError)
			}
		}
	// If file was password protected
	case http.MethodPost:
		of := ob.store.Get(r.Header.Get("filename"))
		if of == nil {
			http.Error(w, "Nil file", http.StatusInternalServerError)
		}
		if of.DownloadLimit > 0 && of.Downloads >= of.DownloadLimit {
			if err := ob.store.Delete(of); err != nil {
				ob.logf("Error deleting onion file from store: %v", err)
			}
			ob.logf("Download limit reached for %s", of.Name)
			http.Error(w, "Download limit reached.", http.StatusUnauthorized)
		}
		// Check expiration
		if of.IsExpired() {
			if err := of.Destroy(); err != nil {
				ob.logf("Error destroying buffer %s: %v", of.Name, err)
			}
			http.Error(w, "Download link has expired", http.StatusUnauthorized)
		}
		// Validate checksum
		chksmValid, err := of.ValidateChecksum()
		if err != nil {
			ob.logf("Error validating checksum: %v", err)
			http.Error(w, "Error validating checksum.", http.StatusInternalServerError)
		}
		if !chksmValid {
			ob.logf("Invalid checksum for file %s", of.Name)
			http.Error(w, "Invalid checksum.", http.StatusInternalServerError)
		}
		// Get password and decrypt zip for download
		pass := r.FormValue("password")
		decryptedBytes, err := onion_buffer.Decrypt(of.Bytes, pass)
		if err != nil {
			ob.logf("Error decrypting buffer: %v", err)
			http.Error(w, "Error decrypting buffer.", http.StatusInternalServerError)
		}
		// Lock memory allotted to decryptedBytes from being used in SWAP
		if err := syscall.Mlock(decryptedBytes); err != nil {
			ob.logf("Error mlocking allotted memory for decryptedBytes: %v", err)
		}
		// Increment files download count
		of.Downloads++
		// Set headers for browser to initiate download
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", of.Name))
		// Write the zip bytes to the response for download
		_, err = w.Write(decryptedBytes)
		if err != nil {
			ob.logf("Error writing to client: %v", err)
			http.Error(w, "Error writing to client.", http.StatusInternalServerError)
		}
	default:
		http.Error(w, "Invalid HTTP Method.", http.StatusMethodNotAllowed)
	}
}

func createCSRF() (string, error) {
	hasher := md5.New()
	_, err := io.WriteString(hasher, strconv.FormatInt(time.Now().Unix(), 10))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func (ob *onionbox) logf(format string, args ...interface{}) {
	if ob.debug {
		ob.logger.Printf(format, args...)
	}
}

func (ob *onionbox) destroy() {
	if err := ob.store.DestroyAll(); err != nil {
		ob.logf("Error destroying all buffers from store: %v", err)
	}
}
