package testutil_test

import (
	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
	"github.com/omkar273/burrow/packages/engine/internal/testutil"
)

// Compile-time proof that the fakes satisfy the real ports.
var (
	_ source.Connector = (*testutil.FakeSource)(nil)
	_ source.Restorer  = (*testutil.FakeSource)(nil)
	_ storage.Store    = (*testutil.FakeStore)(nil)
)
