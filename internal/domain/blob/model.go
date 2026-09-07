// Package blob models stored content: bytes identified by their hash.
// It knows nothing about where those bytes physically live — that is
// Replica's concern.
package blob

import "time"

// Blob is content, identified by the hash of its bytes.
type Blob struct {
	ID          string
	ContentHash string
	SizeBytes   int64
	CreatedAt   time.Time
}
