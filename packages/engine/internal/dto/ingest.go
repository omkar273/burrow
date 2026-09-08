// Package dto holds the request and response shapes for the engine's use
// cases. A request validates itself before a service acts on it.
package dto

import "github.com/omkar273/burrow/packages/engine/internal/validator"

// IngestRequest is what IngestOne needs to copy one object into the archive.
type IngestRequest struct {
	ExternalID string `validate:"required"`
	ThreadID   string
}

func (r *IngestRequest) Validate() error {
	return validator.ValidateRequest(r)
}

type IngestResponse struct {
	ObjectID    string
	VersionID   string
	ContentHash string
	// Deduplicated: this provider id was already held, directly or via an
	// alias recorded by a restore.
	Deduplicated bool
	BlobReused   bool
}
