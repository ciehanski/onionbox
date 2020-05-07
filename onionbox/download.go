package onionbox

import (
	"crypto/subtle"
	"fmt"
	"html/template"
	"net/http"
	"syscall"

	"github.com/ciehanski/onionbox/onionbuffer"
	"github.com/ciehanski/onionbox/templates"
)

func (ob *Onionbox) download(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:

		oBuffer, err := ob.Store.Get(r.URL.Path[1:])
		if err != nil {
			ob.Logf("File %s not found in store", oBuffer.Name)
			http.Error(w, "Error finding requested file.", http.StatusInternalServerError)
			return
		}

		if oBuffer.Encrypted {
			csrf, err := createCSRF()
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

			t, err := template.New("download_encrypted").Parse(templates.DownloadHTML) // Parse template
			if err != nil {
				ob.Logf("Error loading template: %v", err)
				http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
				return
			}

			if err := t.Execute(w, csrf); err != nil { // Execute template
				ob.Logf("Error executing template: %v", err)
				http.Error(w, "Error displaying web page, please try refreshing.", http.StatusInternalServerError)
				return
			}
		} else {
			if oBuffer.DownloadLimit != 0 {
				if oBuffer.Downloads >= oBuffer.DownloadLimit {
					ob.Logf("Download limit reached for %s", oBuffer.Name)
					if err := ob.Store.Destroy(oBuffer); err != nil {
						ob.Logf("Error deleting onion file from Store: %v", err)
					}
					http.Error(w, "Download limit reached.", http.StatusUnauthorized)
					return
				} else {
					// Increment files download count
					oBuffer.Downloads++
				}
			}
			chksmValid, err := oBuffer.ValidateChecksum() // Validate checksum
			if err != nil {
				ob.Logf("Error validating checksum: %v", err)
				http.Error(w, "Error validating checksum.", http.StatusInternalServerError)
				return
			}
			if !chksmValid {
				ob.Logf("Invalid checksum for file %s", oBuffer.Name)
				http.Error(w, "Invalid checksum.", http.StatusInternalServerError)
				return
			}
			// Set headers for browser to initiate download
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", oBuffer.Name))
			w.Header().Set("Content-Type", "application/zip; charset=utf-8")
			// Write the zip bytes to the response for download
			_, err = w.Write(oBuffer.Bytes)
			if err != nil {
				ob.Logf("Error writing to client: %v", err)
				http.Error(w, "Error writing to client.", http.StatusInternalServerError)
				return
			}
		}
	case http.MethodPost: // If buffer was password protected
		if err := r.ParseForm(); err != nil {
			ob.Logf("Error parsing upload form: %v", err)
			http.Error(w, "Error parsing password form.", http.StatusInternalServerError)
			return
		}

		// Check CSRF
		csrfForm := r.PostFormValue(formCSRF)
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

		oBuffer, err := ob.Store.Get(r.URL.Path[1:])
		if err != nil {
			ob.Logf("File %s not found in store", oBuffer.Name)
			http.Error(w, "Error finding requested file.", http.StatusInternalServerError)
			return
		}

		if oBuffer.DownloadLimit != 0 {
			if oBuffer.Downloads >= oBuffer.DownloadLimit {
				if err := ob.Store.Destroy(oBuffer); err != nil {
					ob.Logf("Error deleting onionbuffer from store: %v", err)
				}
				ob.Logf("Download limit reached for %s", oBuffer.Name)
				http.Error(w, "Download limit reached.", http.StatusUnauthorized)
				return
			} else {
				// Increment files download count
				oBuffer.Downloads++
			}
		}
		// Validate checksum
		chksmValid, err := oBuffer.ValidateChecksum()
		if err != nil {
			ob.Logf("Error validating checksum: %v", err)
			http.Error(w, "Error validating checksum.", http.StatusInternalServerError)
			return
		}
		if !chksmValid {
			ob.Logf("Invalid checksum for file %s", oBuffer.Name)
			http.Error(w, "Invalid checksum.", http.StatusInternalServerError)
			return
		}
		// Get password and decrypt zip for download
		pass := r.FormValue("password")
		decryptedBytes, err := onionbuffer.Decrypt(oBuffer.Bytes, pass)
		if err != nil {
			ob.Logf("Error decrypting buffer: %v", err)
			http.Error(w, "Error decrypting buffer.", http.StatusInternalServerError)
			return
		}
		// Lock memory allotted to decryptedBytes from being used in SWAP
		if err := syscall.Mlock(decryptedBytes); err != nil {
			ob.Logf("Error mlocking allotted memory for decryptedBytes: %v", err)
		}
		// Set headers for browser to initiate download
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", oBuffer.Name))
		w.Header().Set("Content-Type", "application/zip; charset=utf-8")
		// Write the zip bytes to the response for download
		_, err = w.Write(decryptedBytes)
		if err != nil {
			ob.Logf("Error writing to client: %v", err)
			http.Error(w, "Error writing to client.", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Invalid HTTP Method.", http.StatusMethodNotAllowed)
		return
	}
}
