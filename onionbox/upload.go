package onionbox

import (
	"archive/zip"
	"bytes"
	"crypto/subtle"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/ciehanski/onionbox/onionbuffer"
	"github.com/ciehanski/onionbox/templates"

	"github.com/Pallinder/go-randomdata"
)

func (ob *Onionbox) upload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		csrf, err := createCSRF() // Create CSRF to inject into template
		if err != nil {
			ob.Logf("Error creating CSRF token: %v", err)
			http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
			return
		}

		// Set CSRF cookie
		http.SetCookie(w, &http.Cookie{
			Name:     cookieCSRF,
			Value:    csrf,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
		})

		t, err := template.New("upload").Parse(templates.UploadHTML) // Parse template
		if err != nil {
			ob.Logf("Error parsing template: %v", err)
			http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
			return
		}

		if err := t.Execute(w, csrf); err != nil { // Execute template
			ob.Logf("Error executing template: %v", err)
			http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
			return
		}
	case http.MethodPost:
		if err := r.ParseMultipartForm(32 << 20); err != nil { // Parse file(s) from form
			ob.Logf("Error parsing files from form: %v", err)
			http.Error(w, "Error parsing files.", http.StatusInternalServerError)
			return
		}

		// Check CSRF
		csrfForm := r.FormValue(formCSRF)
		csrfCookie, err := r.Cookie(cookieCSRF)
		if err != nil {
			ob.Logf("Error getting CSRF cookie: %v", err)
			http.Error(w, "Error getting CSRF.", http.StatusInternalServerError)
			return
		}
		if subtle.ConstantTimeCompare([]byte(csrfForm), []byte(csrfCookie.Value)) == 0 {
			ob.Logf("Form CSRF and Cookie CSRF values do not match")
			http.Error(w, "Invalid CSRF value.", http.StatusUnauthorized)
			return
		}

		files := r.MultipartForm.File["files"]

		uploadQueue := make(chan *multipart.FileHeader, len(files)) // A channel that we can queue upload requests on

		var fileSizes int64
		for _, fileHeader := range files { // Loop through files attached in form and offload to uploadQueue channel
			fileSizes += fileHeader.Size
			uploadQueue <- fileHeader
		}

		mmapBytes, err := onionbuffer.Allocate(int(fileSizes))
		if err != nil {
			ob.Logf("Error allocating bytes for zip: %v", err)
			http.Error(w, "Error allocating bytes.", http.StatusInternalServerError)
			return
		}
		zBuffer := bytes.NewBuffer(mmapBytes) // Create buffer for session's in-memory zip file
		zWriter := zip.NewWriter(zBuffer)     // Create new zip file

		wg := new(sync.WaitGroup) // Wait group for sync
		wg.Add(len(files))

		go func() { // Write all files in queue to memory
			if err := onionbuffer.WriteFilesToBuffer(zWriter, uploadQueue, wg); err != nil {
				ob.Logf("Error writing files in queue to memory: %v", err)
				http.Error(w, "Error writing your files to memory.", http.StatusInternalServerError)
				return
			}
		}()

		wg.Wait() // Wait for zip to be finished

		if err := zWriter.Close(); err != nil { // Close zipwriter
			ob.Logf("Error closing zip writer: %v", err)
		}

		if err := syscall.Mlock(zBuffer.Bytes()); err != nil { // Lock memory allotted to zBuffer from being used in SWAP
			ob.Logf("Error mlocking allotted memory for zBuffer: %v", err)
		}

		// Create OnionBuffer object
		oBuffer := onionbuffer.OnionBuffer{Name: strings.ToLower(randomdata.SillyName()), Bytes: make([]byte, len(zBuffer.Bytes()))}

		if r.FormValue("password_enabled") == "on" { // If password option was enabled
			var err error
			pass := r.FormValue("password")
			oBuffer.Bytes, err = onionbuffer.Encrypt(zBuffer.Bytes(), pass)
			if err != nil {
				ob.Logf("Error encrypting buffer: %v", err)
				http.Error(w, "Error encrypting buffer.", http.StatusInternalServerError)
				return
			}

			zBuffer.Reset() // Empty temporary zip buffer

			if err := syscall.Mlock(oBuffer.Bytes); err != nil { // Lock memory allotted to oBuffer from being used in SWAP
				ob.Logf("Error mlocking allotted memory for oBuffer: %v", err)
			}

			oBuffer.Encrypted = true

			oBuffer.Checksum, err = oBuffer.GetChecksum()
			if err != nil {
				ob.Logf("Error getting buffer's checksum: %v", err)
				http.Error(w, "Error getting checksum.", http.StatusInternalServerError)
				return
			}
		} else { // If password option was NOT enabled
			oBuffer.Bytes = zBuffer.Bytes()

			zBuffer.Reset() // Empty temporary zip buffer

			if err := syscall.Mlock(oBuffer.Bytes); err != nil { // Lock memory allotted to oBuffer from being used in SWAP
				ob.Logf("Error mlocking allotted memory for oBuffer: %v", err)
			}

			oBuffer.Checksum, err = oBuffer.GetChecksum() // Get checksum
			if err != nil {
				ob.Logf("Error getting checksum: %v", err)
				http.Error(w, "Error getting checksum.", http.StatusInternalServerError)
				return
			}
		}

		if r.FormValue("limit_downloads") == "on" { // If limit downloads was enabled
			form := r.FormValue("download_limit")
			limit, err := strconv.Atoi(form)
			if err != nil {
				ob.Logf("Error converting duration string into time.Duration: %v", err)
				http.Error(w, "Error getting expiration time.", http.StatusInternalServerError)
				return
			}
			oBuffer.DownloadLimit = int64(limit)
		}

		if r.FormValue("expire") == "on" { // if expiration was enabled
			expiration := fmt.Sprintf("%sm", r.FormValue("expiration_time"))
			if err := oBuffer.SetExpiration(expiration); err != nil {
				ob.Logf("Error parsing expiration time: %v", err)
				http.Error(w, "Error parsing expiration time.", http.StatusInternalServerError)
				return
			}
		}

		if err := ob.Store.Add(&oBuffer); err != nil { // Add OnionBuffer to Store
			ob.Logf("Error adding file to store: %v", err)
			http.Error(w, "Error adding file to store.", http.StatusInternalServerError)
			return
		}

		// Write the zip's URL to client for sharing
		_, err = w.Write([]byte(fmt.Sprintf("Files uploaded. Please share this link with your recipients: http://%s.onion/%s",
			ob.OnionURL, oBuffer.Name)))
		if err != nil {
			ob.Logf("Error writing to client: %v", err)
			http.Error(w, "Error writing to client.", http.StatusInternalServerError)
			return
		}

		// Destroy temp OnionBuffer
		oBuffer.Lock()
		defer oBuffer.Unlock()
		if err := oBuffer.Destroy(); err != nil {
			if err.Error() != "invalid argument" {
				ob.Logf("Error destroying temporary onionbuffer: %v", err)
			}
		}
	default:
		http.Error(w, "Invalid HTTP Method.", http.StatusMethodNotAllowed)
		return
	}
}
