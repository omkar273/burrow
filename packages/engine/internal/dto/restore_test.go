package dto_test

import (
	stderrors "errors"
	"testing"

	"github.com/omkar273/burrow/packages/engine/internal/dto"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
)

func TestRestoreRequestValidateRejectsMissingSourceID(t *testing.T) {
	req := &dto.RestoreRequest{ObjectID: "obj_1"}
	if err := req.Validate(); !stderrors.Is(err, ierr.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestRestoreRequestValidateRejectsMissingObjectID(t *testing.T) {
	req := &dto.RestoreRequest{SourceID: "src_1"}
	if err := req.Validate(); !stderrors.Is(err, ierr.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestRestoreRequestValidatePasses(t *testing.T) {
	req := &dto.RestoreRequest{SourceID: "src_1", ObjectID: "obj_1"}
	if err := req.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}
