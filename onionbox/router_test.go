package onionbox

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter(t *testing.T) {
	// Create new httptest writer
	w := httptest.NewRecorder()

	tests := []struct {
		name         string
		r            *http.Request
		expectedCode int
	}{
		{
			name:         "1: Test Download Invalid",
			r:            newRequest(t, "GET", "/tastyred-tastyblue", nil),
			expectedCode: 404,
		},
		{
			name:         "2: Test Download Invalid",
			r:            newRequest(t, "GET", "/tastyred7", nil),
			expectedCode: 404,
		},
		{
			name:         "3: Test Download Invalid",
			r:            newRequest(t, "GET", "/Uglyduck", nil),
			expectedCode: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			http.DefaultServeMux.ServeHTTP(w, tt.r)
			// check for expected response here.
			if w.Code != tt.expectedCode {
				t.Error(fmt.Sprintf("Expected response code %v, got %v", tt.expectedCode, w.Code))
			}
		})
	}
}

func newRequest(t *testing.T, method, url string, body io.Reader) *http.Request {
	r, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
