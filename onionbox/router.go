package onionbox

import (
	"net/http"
	"regexp"
)

var downloadURLreg = regexp.MustCompile(`((?:[a-z]+))`)

func (ob *Onionbox) Router(w http.ResponseWriter, r *http.Request) {
	// If base URL, send to upload handler
	if r.URL.Path == "/" {
		ob.upload(w, r)
	} else if matches := downloadURLreg.FindStringSubmatch(r.URL.Path[1:]); matches != nil {
		if ob.Store != nil {
			if _, err := ob.Store.Get(r.URL.Path[1:]); err != nil {
				ob.download(w, r)
			} else {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
		} else {
			// Do not state the store is empty to the user
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
	} else {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
}
