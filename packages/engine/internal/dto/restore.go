package dto

import "github.com/omkar273/burrow/packages/engine/internal/validator"

type RestoreRequest struct {
	SourceID string `validate:"required"`
	ObjectID string `validate:"required"`
	Labels   []string
}

func (r *RestoreRequest) Validate() error {
	return validator.ValidateRequest(r)
}

type RestoreResponse struct {
	NewExternalID string
	ContentHash   string
	BytesRestored int64
}
