package onionbox

import (
	"testing"
)

func TestCreateCSRF(t *testing.T) {
	_, err := createCSRF()
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkCreateCSRF(b *testing.B) {
	for n := 0; n < b.N; n++ {
		createCSRF()
	}
}
