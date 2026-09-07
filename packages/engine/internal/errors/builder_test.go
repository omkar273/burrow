package errors_test

import (
	stderrors "errors"
	"testing"

	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
)

func TestMarkIsDiscoverableByErrorsIs(t *testing.T) {
	err := ierr.New("history id rejected by provider").
		Mark(ierr.ErrCheckpointExpired)

	if !stderrors.Is(err, ierr.ErrCheckpointExpired) {
		t.Fatal("marked error is not discoverable via errors.Is")
	}
	if stderrors.Is(err, ierr.ErrNotFound) {
		t.Fatal("error matched a sentinel it was not marked with")
	}
}

func TestWrapPreservesTheUnderlyingError(t *testing.T) {
	inner := stderrors.New("connection reset")
	err := ierr.Wrap(inner, "fetching message").Mark(ierr.ErrSourceUnavailable)

	if !stderrors.Is(err, inner) {
		t.Fatal("wrapped error lost the underlying cause")
	}
	if !stderrors.Is(err, ierr.ErrSourceUnavailable) {
		t.Fatal("wrapped error lost its mark")
	}
}

func TestHintIsRetained(t *testing.T) {
	err := ierr.New("checksum mismatch").
		WithHint("run `burrowd verify --repair` to refetch this object").
		Mark(ierr.ErrChecksumMismatch)

	if got := err.Hint(); got == "" {
		t.Fatal("hint was dropped")
	}
}

func TestUnmarkedErrorMatchesNoSentinel(t *testing.T) {
	err := ierr.New("something happened")
	if stderrors.Is(err, ierr.ErrInternal) {
		t.Fatal("unmarked error matched a sentinel")
	}
}
