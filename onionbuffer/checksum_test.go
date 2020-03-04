package onionbuffer

import (
	"testing"
)

func TestGetChecksum(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
	}{
		{
			name:  "1",
			bytes: []byte("LUHF*Hufi4f8ufhilaueh8d9w84h3jfnkjsdfiuw4ufhsjkdbnfskdfu4"),
		},
		{
			name:  "2",
			bytes: []byte("This is a test"),
		},
		{
			name: "3",
			bytes: []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nulla scelerisque, massa vitae " +
				"hendrerit sollicitudin, libero ligula pharetra dolor, at ultrices dolor turpis id elit. Donec diam sem, " +
				"dictum vel sagittis vitae, commodo sed lectus. Nam egestas nisi orci, vitae feugiat mi hendrerit id." +
				" Quisque ac nisl sit amet urna ornare laoreet a nec dolor."),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := OnionBuffer{Bytes: tt.bytes}
			if _, err := b.GetChecksum(); err != nil {
				t.Error(err)
			}
		})
	}
}

func BenchmarkGetChecksum(b *testing.B) {
	ob := OnionBuffer{Bytes: []byte("Testing checksum")}
	for n := 0; n < b.N; n++ {
		ob.GetChecksum()
	}
}

func TestValidateChecksum(t *testing.T) {
	b := OnionBuffer{Bytes: []byte("Testing checksum")}
	b.Checksum, _ = b.GetChecksum()
	validChksm, err := b.ValidateChecksum()
	if err != nil {
		t.Error(err)
	}
	if !validChksm {
		t.Error("Expected checksum to be valid")
	}
}

func BenchmarkValidateChecksum(b *testing.B) {
	ob := OnionBuffer{Bytes: []byte("Testing checksum")}
	ob.Checksum, _ = ob.GetChecksum()
	for n := 0; n < b.N; n++ {
		ob.ValidateChecksum()
	}
}
