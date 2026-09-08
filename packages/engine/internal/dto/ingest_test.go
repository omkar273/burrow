package dto_test

import (
	stderrors "errors"
	"testing"

	"github.com/omkar273/burrow/packages/engine/internal/dto"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
)

func TestIngestRequestValidateRejectsMissingExternalID(t *testing.T) {
	req := &dto.IngestRequest{}
	if err := req.Validate(); !stderrors.Is(err, ierr.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestIngestRequestValidatePassesWithExternalID(t *testing.T) {
	req := &dto.IngestRequest{ExternalID: "m1"}
	if err := req.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}
