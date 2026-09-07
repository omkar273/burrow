package ent_test

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	entrepo "github.com/omkar273/burrow/packages/engine/internal/repository/ent"
	"github.com/omkar273/burrow/packages/engine/internal/types"
)

func TestWithTxCommitsOnSuccess(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)
	srcID := newSourceForTest(t, c)
	repo := entrepo.NewObjectRepository(c)

	id := types.NewID(types.PrefixObject)
	err := c.WithTx(ctx, func(txCtx context.Context) error {
		return repo.Create(txCtx, &object.Object{
			ID: id, SourceID: srcID, Kind: object.KindMessage, ExternalID: "m1",
		})
	})
	if err != nil {
		t.Fatalf("WithTx: %v", err)
	}
	if _, err := repo.Get(ctx, id); err != nil {
		t.Fatalf("committed object not found: %v", err)
	}
}

func TestWithTxRollsBackOnError(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)
	srcID := newSourceForTest(t, c)
	repo := entrepo.NewObjectRepository(c)

	id := types.NewID(types.PrefixObject)
	boom := stderrors.New("service decided to fail")
	err := c.WithTx(ctx, func(txCtx context.Context) error {
		if err := repo.Create(txCtx, &object.Object{
			ID: id, SourceID: srcID, Kind: object.KindMessage, ExternalID: "m1",
		}); err != nil {
			return err
		}
		return boom
	})
	if !stderrors.Is(err, boom) {
		t.Fatalf("err = %v, want the caller's error", err)
	}
	if _, err := repo.Get(ctx, id); err == nil {
		t.Fatal("object survived a rolled-back transaction")
	}
}

// The connection pool is capped at one, so a nested BeginTx would wait forever
// for the connection the outer transaction holds. Nesting must reuse it.
func TestNestedWithTxReusesTheOuterTransaction(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)
	srcID := newSourceForTest(t, c)
	repo := entrepo.NewObjectRepository(c)

	id := types.NewID(types.PrefixObject)
	done := make(chan error, 1)
	go func() {
		done <- c.WithTx(ctx, func(outer context.Context) error {
			return c.WithTx(outer, func(inner context.Context) error {
				return repo.Create(inner, &object.Object{
					ID: id, SourceID: srcID, Kind: object.KindMessage, ExternalID: "m1",
				})
			})
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("nested WithTx: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("nested WithTx deadlocked")
	}

	if _, err := repo.Get(ctx, id); err != nil {
		t.Fatalf("object from the nested transaction not found: %v", err)
	}
}

// An inner failure must roll back the whole outer transaction, not just its
// own part of it.
func TestNestedFailureRollsBackEverything(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)
	srcID := newSourceForTest(t, c)
	repo := entrepo.NewObjectRepository(c)

	outerID := types.NewID(types.PrefixObject)
	boom := stderrors.New("inner failed")

	err := c.WithTx(ctx, func(outer context.Context) error {
		if err := repo.Create(outer, &object.Object{
			ID: outerID, SourceID: srcID, Kind: object.KindMessage, ExternalID: "m1",
		}); err != nil {
			return err
		}
		return c.WithTx(outer, func(context.Context) error { return boom })
	})
	if !stderrors.Is(err, boom) {
		t.Fatalf("err = %v, want the inner error", err)
	}
	if _, err := repo.Get(ctx, outerID); err == nil {
		t.Fatal("the outer write survived an inner failure")
	}
}

// A panic must not leave the transaction open holding the only connection.
func TestPanicRollsBackAndPropagates(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)
	srcID := newSourceForTest(t, c)
	repo := entrepo.NewObjectRepository(c)

	id := types.NewID(types.PrefixObject)

	func() {
		defer func() {
			if recover() == nil {
				t.Error("panic did not propagate")
			}
		}()
		_ = c.WithTx(ctx, func(txCtx context.Context) error {
			_ = repo.Create(txCtx, &object.Object{
				ID: id, SourceID: srcID, Kind: object.KindMessage, ExternalID: "m1",
			})
			panic("service panicked mid-transaction")
		})
	}()

	if _, err := repo.Get(ctx, id); err == nil {
		t.Fatal("write survived a panicking transaction")
	}
	// The connection must be usable again.
	if _, err := repo.Get(ctx, "obj_whatever"); err == nil {
		t.Fatal("expected not-found, got success")
	}
}
