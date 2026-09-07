// Package storage defines the contract every storage backend satisfies.
package storage

import (
	"context"
	"io"
	"iter"
	"strings"
	"time"
)

type Stat struct {
	Size       int64
	ModifiedAt time.Time
}

// Caps describes what a backend can actually do. Backends genuinely
// differ; the contract does not pretend otherwise.
type Caps struct {
	Multipart    bool
	RangeReads   bool
	AtomicRename bool
}

type Store interface {
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Stat(ctx context.Context, key string) (Stat, error)
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) iter.Seq2[string, error]
	Capabilities() Caps
}

// KeyForHash maps a content hash to its storage key, fanning out two
// levels so no directory holds an unreasonable number of entries.
//
// The layout is deliberately plain: an archive should be readable with
// standard tools and no Burrow installed.
func KeyForHash(hash string) string {
	hex := hash
	if i := strings.IndexByte(hash, ':'); i >= 0 {
		hex = hash[i+1:]
	}
	if len(hex) < 4 {
		return "objects/" + hex
	}
	return "objects/" + hex[0:2] + "/" + hex[2:4] + "/" + hex
}
