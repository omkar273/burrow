// Package errors defines the engine's operational failure taxonomy.
// The codes are operational rather than HTTP-shaped: this is a daemon,
// not an API.
package errors

type Code string

const (
	CodeCheckpointExpired Code = "checkpoint_expired"
	CodeChecksumMismatch  Code = "checksum_mismatch"
	CodeBlobMissing       Code = "blob_missing"
	CodeReplicaCorrupt    Code = "replica_corrupt"
	CodeCredentialExpired Code = "credential_expired"
	CodeSourceUnavailable Code = "source_unavailable"
	CodeProfileLocked     Code = "profile_locked"
	CodeNotFound          Code = "not_found"
	CodeValidation        Code = "validation"
	CodeInternal          Code = "internal"
)

type sentinel struct {
	code Code
	msg  string
}

func (s *sentinel) Error() string { return string(s.code) + ": " + s.msg }

func (s *sentinel) Code() Code { return s.code }

func newSentinel(c Code, msg string) error { return &sentinel{code: c, msg: msg} }

var (
	// ErrCheckpointExpired means the source rejected our resume token and a
	// full resync is required. Gmail returns 404 on a stale historyId,
	// typically after about a week — this is expected, not exceptional.
	ErrCheckpointExpired = newSentinel(CodeCheckpointExpired, "source checkpoint expired; full resync required")
	ErrChecksumMismatch  = newSentinel(CodeChecksumMismatch, "content hash does not match stored hash")
	ErrBlobMissing       = newSentinel(CodeBlobMissing, "blob not present in storage")
	ErrReplicaCorrupt    = newSentinel(CodeReplicaCorrupt, "replica failed verification")
	ErrCredentialExpired = newSentinel(CodeCredentialExpired, "credential expired and could not be refreshed")
	ErrSourceUnavailable = newSentinel(CodeSourceUnavailable, "source is unavailable")
	ErrProfileLocked     = newSentinel(CodeProfileLocked, "another burrowd holds this profile")
	ErrNotFound          = newSentinel(CodeNotFound, "not found")
	ErrValidation        = newSentinel(CodeValidation, "validation failed")
	ErrInternal          = newSentinel(CodeInternal, "internal error")
)
