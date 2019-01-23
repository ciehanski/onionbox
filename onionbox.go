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

type Onionbox struct {
	Logger      *log.Logger
	TorVersion3 bool
	OnionSvc    *tor.OnionService
	Debug       bool
	FileStore   []*onion_file.OnionFile
}

//func init() {
//	// Create files dir
//	filesDir, _ := os.Getwd()
//	filesPath := path.Join(filesDir, "files")
//	if _, err := os.Stat(filesPath); os.IsNotExist(err) {
//		if err := os.Mkdir(filesPath, 0755); err != nil {
//			os.Exit(1)
//		}
//	}
//}

func main() {
	ob := Onionbox{Logger: log.New(os.Stdout, "[onionbox] ", log.LstdFlags)}
	// Init flags
	flag.BoolVar(&ob.Debug, "debug", false, "run in debug mode")
	flag.BoolVar(&ob.TorVersion3, "torv3", true, "use version 3 of the Tor circuit")
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
	ob.OnionSvc = onionSvc

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
		ob.destroy(0)
	}()
}

func (ob *Onionbox) router(w http.ResponseWriter, r *http.Request) {
	// Set download url regex
	downloadURLreg := regexp.MustCompile(`((?:[a-z][a-z]+))`)

	if r.URL.Path == "/" {
		ob.upload(w, r)
	} else if matches := downloadURLreg.FindStringSubmatch(r.URL.Path); matches != nil {
		ob.download(w, r)
	}
}

func (ob *Onionbox) upload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Create CSRF
		hasher := md5.New()
		_, err := io.WriteString(hasher, strconv.FormatInt(time.Now().Unix(), 10))
		if err != nil {
			ob.logf("%v", err)
			http.Error(w, "CSRF Error.", http.StatusInternalServerError)
		}
		csrf := fmt.Sprintf("%x", hasher.Sum(nil))
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
		if err := r.ParseMultipartForm(512 << 20); err != nil {
			ob.logf("Error parsing files from form: %v", err)
			http.Error(w, "Error parsing files.", http.StatusInternalServerError)
		}
		// Loop through all files in the form
		filesBuffer := new(bytes.Buffer)
		zWriter := zip.NewWriter(filesBuffer)
		files := r.MultipartForm.File["files"]
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
			reader := bufio.NewReader(file)
			var count int
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

			// TODO: comeback. Would prefer to use bufFile.Write and loop through each chunk of file instead
			// TODO: of using io.Copy. Need benchmarks
			// Copy contents from uploaded file into file in zip
			//_, err = io.Copy(bufFile, file)
			//if err != nil {
			//	ob.logf("Error copying uploaded file into zip: %v", err)
			//	http.Error(w, "Error uploading files.", http.StatusInternalServerError)
			//}
		}
		// Close zipwriter
		if err := zWriter.Close(); err != nil {
			ob.logf("Error closing zip writer: %v", err)
		}

		// Create random zip name
		zipFile := strings.ToLower(randomdata.SillyName())
		of := &onion_file.OnionFile{Name: zipFile, CreatedAt: time.Now()}

		// If password option was enabled
		if r.FormValue("password_enabled") == "on" {
			pass := r.FormValue("password")
			of.Bytes = onion_file.Encrypt(filesBuffer.Bytes(), pass)
			// Write the zipped file to the disk
			//if err := ioutil.WriteFile("./files/"+zipFile+".zip", encryptedBytes, 0777); err != nil {
			//	ob.logf("Error writing encrypted zip to disk: %v", err)
			//	http.Error(w, "Error writing encrypted zip to disk.", http.StatusInternalServerError)
			//}
			of.Encrypted = true
		} else {
			of.Bytes = filesBuffer.Bytes()
			// Write the zipped file to the disk
			//if err := ioutil.WriteFile("./files/"+zipFile+".zip", filesBuffer.Bytes(), 0777); err != nil {
			//	ob.logf("Error writing zip to disk: %v", err)
			//	http.Error(w, "Error writing zip to disk.", http.StatusInternalServerError)
			//}
		}
		// If limit downloads was enabled
		if r.FormValue("limit_downloads") == "on" {
			// TODO: implement
			form := r.FormValue("download_limit")
			limit, _ := strconv.Atoi(form)
			of.DownloadLimit = limit
		}
		// if expiration was enabled
		if r.FormValue("expire") == "on" {
			// TODO: implement
			of.ExpiresAfter = time.Hour * 1
		}

		// Append onion file to filestore
		ob.FileStore = append(ob.FileStore, of)

		// Redirect to the files download page
		_, err := w.Write([]byte(fmt.Sprintf("Files uploaded. Please share this link with your recipients: http://%s.onion/%s",
			ob.OnionSvc.ID, of.Name)))
		if err != nil {
			ob.logf("Error writing to client: %v", err)
			http.Error(w, "Error writing to client.", http.StatusInternalServerError)
		}
	default:
		http.Error(w, "Invalid HTTP Method.", http.StatusMethodNotAllowed)
	}
}

func (ob *Onionbox) download(w http.ResponseWriter, r *http.Request) {
	//file := r.URL.Path[1:] + ".zip"
	switch r.Method {
	case http.MethodGet:
		for _, of := range ob.FileStore {
			if of.Name == r.URL.Path[1:] {
				if of.Encrypted {
					// Create CSRF
					hasher := md5.New()
					_, err := io.WriteString(hasher, strconv.FormatInt(time.Now().Unix(), 10))
					if err != nil {
						ob.logf("%v", err)
						http.Error(w, "CSRF Error.", http.StatusInternalServerError)
					}
					csrf := fmt.Sprintf("%x", hasher.Sum(nil))
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
						of.Downloads += 1
						w.Header().Set("Content-Type", "application/zip")
						w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", of.Name))
						w.Write(of.Bytes)
						// Parse template
						//t, err := template.ParseFiles("./templates/download.gtpl")
						//if err != nil {
						//	ob.logf("Error loading template: %v", err)
						//	http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
						//}
						//// Execute template
						//if err := t.Execute(w, fmt.Sprintf("/files/%s", file)); err != nil {
						//	ob.logf("Error executing template: %v", err)
						//	http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
						//}
					} else {
						http.Error(w, "Download Limit Reached.", http.StatusUnauthorized)
					}
				}
			} else {
				http.Error(w, "Unable to find requested file.", http.StatusInternalServerError)
			}
		}
	case http.MethodPost:
		//zipBytes, err := ioutil.ReadFile("./files/"+file)
		//if err != nil {
		//	ob.logf("Error reading zip file: %v", err)
		//}
		for _, of := range ob.FileStore {
			if of.Name == r.URL.Path[1:] {
				if of.Downloads <= of.DownloadLimit {
					pass := r.FormValue("password")
					decryptedBytes := onion_file.Decrypt(of.Bytes, pass)
					of.Downloads += 1
					w.Header().Set("Content-Type", "application/zip")
					w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", of.Name))
					w.Write(decryptedBytes)
					// Write the zipped file to the disk
					//if err := ioutil.WriteFile("./files/"+file, decryptedBytes, 0777); err != nil {
					//	ob.logf("Error decrypting zip file: %v", err)
					//}
				} else {
					http.Error(w, "Download Limit Reached.", http.StatusUnauthorized)
				}
			}
		}
	default:
		http.Error(w, "Invalid HTTP Method.", http.StatusMethodNotAllowed)
	}
}

// After download limit or expiration of all files, kill the server and assets.
func (ob *Onionbox) destroy(exitCode int) {
	ob.FileStore = []*onion_file.OnionFile{}
	if err := os.RemoveAll("./files"); err != nil {
		ob.logf("Error removing files dir: %v", err)
	}
	os.Exit(exitCode)
}

func (ob *Onionbox) logf(format string, args ...interface{}) {
	if ob.Debug {
		ob.Logger.Printf(format, args...)
	}
}
