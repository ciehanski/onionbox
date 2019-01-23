package onion_file

import (
	"time"
)

// OnionFile struct
type OnionFile struct {
	Name          string
	Bytes         []byte
	Encrypted     bool
	Downloads     int
	DownloadLimit int
	CreatedAt     time.Time
	ExpiresAfter  time.Duration
}
