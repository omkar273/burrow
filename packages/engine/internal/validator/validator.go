// Package validator wraps go-playground/validator so struct tags are the
// single place a constraint is written, and every failure arrives as an
// ErrValidation with per-field detail.
package validator

import (
	stderrors "errors"
	"strings"
	"sync"

	playground "github.com/go-playground/validator/v10"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
)

var (
	once     sync.Once
	validate *playground.Validate
)

func instance() *playground.Validate {
	once.Do(func() {
		validate = playground.New(playground.WithRequiredStructEnabled())
		// A profile name becomes a path segment under ~/.burrow/profiles,
		// so traversal here would let a flag value write anywhere on the
		// filesystem.
		_ = validate.RegisterValidation("profilename", func(fl playground.FieldLevel) bool {
			v := fl.Field().String()
			if v == "" {
				return true // optional; the caller substitutes a default
			}
			return v != "." && v != ".." && !strings.ContainsAny(v, `/\`)
		})
	})
	return validate
}

// ValidateRequest checks a struct against its validate tags. Field errors
// are folded into the hint so an operator sees which field failed rather
// than a bare "validation failed".
func ValidateRequest(req any) error {
	if err := instance().Struct(req); err != nil {
		var fieldErrs playground.ValidationErrors
		if !stderrors.As(err, &fieldErrs) {
			return ierr.Wrap(err, "validating request").Mark(ierr.ErrValidation)
		}
		fields := make([]string, 0, len(fieldErrs))
		for _, fe := range fieldErrs {
			fields = append(fields, fe.Field()+" fails "+fe.Tag())
		}
		return ierr.Wrap(err, "request validation failed").
			WithHint(strings.Join(fields, "; ")).
			Mark(ierr.ErrValidation)
	}
	return nil
}
