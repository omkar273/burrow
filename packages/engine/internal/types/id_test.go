package types_test

import (
	"strings"
	"testing"
	"time"

	"github.com/omkar273/burrow/packages/engine/internal/types"
)

func TestNewIDCarriesItsPrefix(t *testing.T) {
	id := types.NewID(types.PrefixObject)
	if !strings.HasPrefix(id, "obj_") {
		t.Fatalf("NewID(PrefixObject) = %q, want obj_ prefix", id)
	}
	if got, want := len(id), len("obj_")+26; got != want {
		t.Fatalf("len = %d, want %d (prefix + 26-char ULID)", got, want)
	}
}

func TestNewIDIsKSortable(t *testing.T) {
	first := types.NewID(types.PrefixObject)
	time.Sleep(2 * time.Millisecond)
	second := types.NewID(types.PrefixObject)
	if first >= second {
		t.Fatalf("later ID %q does not sort after earlier %q", second, first)
	}
}

func TestHasPrefixRejectsAnotherFamily(t *testing.T) {
	replica := types.NewID(types.PrefixReplica)
	if types.HasPrefix(replica, types.PrefixObject) {
		t.Fatalf("%q accepted as an object ID", replica)
	}
	if !types.HasPrefix(replica, types.PrefixReplica) {
		t.Fatalf("%q rejected as a replica ID", replica)
	}
}
