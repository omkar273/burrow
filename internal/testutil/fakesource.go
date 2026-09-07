package testutil

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/omkar273/burrow/internal/domain/source"
	ierr "github.com/omkar273/burrow/internal/errors"
)

// FakeSource is an in-memory source.Connector and source.Restorer.
type FakeSource struct {
	mu       sync.Mutex
	id       string
	messages map[string][]byte
	order    []string
	nextID   int

	// ChangesErr, when set, is returned by Changes. Set it to
	// ierr.ErrCheckpointExpired to exercise the full-resync path.
	ChangesErr error
	// FetchErr, when set, is returned by Fetch.
	FetchErr error
}

// NewFakeSource returns an empty in-memory provider.
func NewFakeSource(sourceID string) *FakeSource {
	return &FakeSource{id: sourceID, messages: map[string][]byte{}}
}

// AddMessage seeds a message at the provider.
func (f *FakeSource) AddMessage(externalID string, body []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, seen := f.messages[externalID]; !seen {
		f.order = append(f.order, externalID)
	}
	f.messages[externalID] = body
}

func (f *FakeSource) SourceID() string { return f.id }

func (f *FakeSource) FullList(_ context.Context, _ source.Cursor) ([]source.ObjectRef, source.Cursor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	refs := make([]source.ObjectRef, 0, len(f.order))
	for _, id := range f.order {
		refs = append(refs, source.ObjectRef{ExternalID: id})
	}
	return refs, "", nil
}

func (f *FakeSource) Changes(_ context.Context, _ source.Checkpoint) ([]source.Change, source.Checkpoint, error) {
	if f.ChangesErr != nil {
		return nil, "", f.ChangesErr
	}
	return nil, "cp-1", nil
}

func (f *FakeSource) Fetch(_ context.Context, ref source.ObjectRef) (io.ReadCloser, source.ObjectMeta, error) {
	if f.FetchErr != nil {
		return nil, source.ObjectMeta{}, f.FetchErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	body, ok := f.messages[ref.ExternalID]
	if !ok {
		return nil, source.ObjectMeta{}, ierr.
			New("no message with external id " + ref.ExternalID).
			Mark(ierr.ErrNotFound)
	}
	meta := source.ObjectMeta{
		ExternalID:   ref.ExternalID,
		InternalDate: time.Unix(0, 0).UTC(),
		SizeEstimate: int64(len(body)),
	}
	return io.NopCloser(bytes.NewReader(body)), meta, nil
}

func (f *FakeSource) Health(context.Context) error { return nil }

// Restore stores the bytes under a NEW provider ID, exactly as Gmail's
// messages.insert does. This is what makes the duplicate-on-restore
// hazard reproducible in tests without touching a real mailbox.
func (f *FakeSource) Restore(_ context.Context, r io.Reader, _ source.RestoreOpts) (string, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	newID := "restored-" + strconv.Itoa(f.nextID)
	f.messages[newID] = body
	f.order = append(f.order, newID)
	return newID, nil
}
