// Package types holds the shared vocabulary of the engine: identifiers,
// enums, and filters. It depends on nothing else in the tree.
package types

import (
	"strings"

	"github.com/oklog/ulid/v2"
)

// Prefixes exist so the four distinct identities in the archive (object,
// version, content, replica) cannot be silently substituted.
type IDPrefix string

const (
	PrefixObject  IDPrefix = "obj"
	PrefixVersion IDPrefix = "ver"
	PrefixBlob    IDPrefix = "blob"
	PrefixReplica IDPrefix = "rep"
	PrefixSource  IDPrefix = "src"
	PrefixJob     IDPrefix = "job"
	PrefixBackend IDPrefix = "bkd"
)

func NewID(p IDPrefix) string {
	return string(p) + "_" + ulid.Make().String()
}

func HasPrefix(id string, p IDPrefix) bool {
	return strings.HasPrefix(id, string(p)+"_")
}
