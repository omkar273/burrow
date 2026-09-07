// Package blob models stored content. Where the bytes physically live is
// Replica's concern, not this package's.
package blob

import "time"

type Blob struct {
	ID          string
	ContentHash string
	SizeBytes   int64
	CreatedAt   time.Time
}
