package onionbox

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ciehanski/onionbox/onionstore"
)

func TestRouter(t *testing.T) {
	tests := []struct {
		name         string
		req          *http.Request
		expectedCode int
	}{
		{
			name:         "1: Test Upload Valid",
			req:          newRequest(t, "GET", "/", nil),
			expectedCode: http.StatusOK,
		},
		{
			name:         "2: Test Download Invalid",
			req:          newRequest(t, "GET", "/tastyred-tastyblue", nil),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "3: Test Download Invalid",
			req:          newRequest(t, "GET", "/tastyred7", nil),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "4: Test Download Invalid",
			req:          newRequest(t, "GET", "/Uglyduck", nil),
			expectedCode: http.StatusNotFound,
		},
	}

	ob := Onionbox{Store: onionstore.NewStore()}
	handler := http.HandlerFunc(ob.Router)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new httptest writer
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tt.req)
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
