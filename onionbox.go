package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
	"onionbox/onion_file"
)

const chunkSize = 1024

type onionbox struct {
	Debug       bool
	Logger      *log.Logger
	FileStore   []*onion_file.OnionFile
	MaxMemory   int64
	TorVersion3 bool
	OnionURL    string
}

func main() {
	// Create onionbox instance
	ob := onionbox{Logger: log.New(os.Stdout, "[onionbox] ", log.LstdFlags)}
	// Init flags
	flag.BoolVar(&ob.Debug, "debug", false, "run in debug mode")
	flag.BoolVar(&ob.TorVersion3, "torv3", true, "use version 3 of the Tor circuit")
	flag.Int64Var(&ob.MaxMemory, "mem", 128, "max memory allotted for handling file buffers")
	// Parse flags
	flag.Parse()

	// Start tor with some defaults + elevated verbosity
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
	onionSvc, err := t.Listen(ctx, &tor.ListenConf{RemotePorts: []int{80}, Version3: ob.TorVersion3})
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

	ob.OnionURL = onionSvc.ID
	ob.logf("Please open a Tor capable browser and navigate to http://%v.onion\n", onionSvc.ID)

	// Init routes
	http.HandleFunc("/", ob.router)
	// Init serving
	server := &http.Server{
		IdleTimeout:  time.Second * 60,
		ReadTimeout:  time.Second * 60,
		WriteTimeout: time.Second * 60,
		Handler:      nil,
	}
	// Begin serving
	go ob.Logger.Fatal(server.Serve(onionSvc))
	// Proper server shutdown when program ends
	defer func() {
		if err := server.Shutdown(context.Background()); err != nil {
			ob.logf("Error shutting down onionbox server: %v", err)
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
		ob.download(w, r)
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
		t, err := template.ParseFiles("./templates/upload.gtpl")
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
		if err := r.ParseMultipartForm(ob.MaxMemory << 20); err != nil {
			ob.logf("Error parsing files from form: %v", err)
			http.Error(w, "Error parsing files.", http.StatusInternalServerError)
		}
		// Create buffer for session in-memory zip file
		filesBuffer := new(bytes.Buffer)
		zWriter := zip.NewWriter(filesBuffer)
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
			chunk := make([]byte, chunkSize)
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
		}
		// Close zipwriter
		if err := zWriter.Close(); err != nil {
			ob.logf("Error closing zip writer: %v", err)
		}
		// Create random zip name
		zipFileName := strings.ToLower(randomdata.SillyName())
		// Create OnionFile object
		oFile := &onion_file.OnionFile{Name: zipFileName, CreatedAt: time.Now()}
		// If password option was enabled
		if r.FormValue("password_enabled") == "on" {
			pass := r.FormValue("password")
			encryptedBytes, err := onion_file.Encrypt(filesBuffer.Bytes(), pass)
			if err != nil {
				ob.logf("Error encrypting buffer: %v", err)
				http.Error(w, "Error encrypting buffer.", http.StatusInternalServerError)
			}
			oFile.Bytes = encryptedBytes
			oFile.Encrypted = true
		} else {
			oFile.Bytes = filesBuffer.Bytes()
		}
		// If limit downloads was enabled
		if r.FormValue("limit_downloads") == "on" {
			form := r.FormValue("download_limit")
			limit, _ := strconv.Atoi(form)
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
			oFile.ExpiresAfter = t
		}
		// Append onion file to filestore
		ob.FileStore = append(ob.FileStore, oFile)
		// Write the zip's URL to client for sharing
		_, err := w.Write([]byte(fmt.Sprintf("Files uploaded. Please share this link with your recipients: http://%s.onion/%s",
			ob.OnionURL, oFile.Name)))
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
		for _, of := range ob.FileStore {
			if of.Name == r.URL.Path[1:] {
				if of.Encrypted {
					csrf, err := createCSRF()
					if err != nil {
						ob.logf("Error creating CSRF token: %v", err)
						http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
					}
					// Parse template
					t, err := template.ParseFiles("./templates/download_encrypted.gtpl")
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
					if of.Downloads <= of.DownloadLimit {
						// Increment files download count
						of.Downloads++
						// Set headers for browser to initiate download
						w.Header().Set("Content-Type", "application/zip")
						w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", of.Name))
						// Write the zip bytes to the response for download
						_, err := w.Write(of.Bytes)
						if err != nil {
							ob.logf("Error writing to client: %v", err)
							http.Error(w, "Error writing to client.", http.StatusInternalServerError)
						}
					} else {
						http.Error(w, "Download Limit Reached.", http.StatusUnauthorized)
					}
				}
			} else {
				http.Error(w, "Unable to find requested file.", http.StatusInternalServerError)
			}
		}
	case http.MethodPost:
		for _, of := range ob.FileStore {
			if of.Name == r.URL.Path[1:] {
				if of.Downloads <= of.DownloadLimit {
					// Get password and decrypt zip for download
					pass := r.FormValue("password")
					decryptedBytes, err := onion_file.Decrypt(of.Bytes, pass)
					if err != nil {
						ob.logf("Error decrypting buffer: %v", err)
						http.Error(w, "Error decrypting buffer.", http.StatusInternalServerError)
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
				} else {
					http.Error(w, "Download Limit Reached.", http.StatusUnauthorized)
				}
			}
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
	if ob.Debug {
		ob.Logger.Printf(format, args...)
	}
}
