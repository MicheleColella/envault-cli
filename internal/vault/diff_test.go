package vault

import (
	"testing"
	"time"
)

func makeStore(entries []Entry) *Store {
	return &Store{Version: storeVersion, Entries: entries}
}

func entry(name string, updatedAt time.Time) Entry {
	return Entry{Name: name, Kind: KindEnv, UpdatedAt: updatedAt}
}

var (
	t0 = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 = time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
)

func TestDiffStores_NoChanges(t *testing.T) {
	before := makeStore([]Entry{entry("FOO", t0), entry("BAR", t0)})
	after := makeStore([]Entry{entry("FOO", t0), entry("BAR", t0)})

	changes := DiffStores(before, after)
	if !changes.IsEmpty() {
		t.Errorf("expected no changes, got %+v", changes)
	}
}

func TestDiffStores_Added(t *testing.T) {
	before := makeStore([]Entry{entry("FOO", t0)})
	after := makeStore([]Entry{entry("FOO", t0), entry("BAR", t0)})

	changes := DiffStores(before, after)
	if len(changes.Added) != 1 || changes.Added[0] != "BAR" {
		t.Errorf("expected Added=[BAR], got %+v", changes)
	}
	if len(changes.Removed) != 0 || len(changes.Rotated) != 0 {
		t.Errorf("unexpected changes: %+v", changes)
	}
}

func TestDiffStores_Removed(t *testing.T) {
	before := makeStore([]Entry{entry("FOO", t0), entry("BAR", t0)})
	after := makeStore([]Entry{entry("FOO", t0)})

	changes := DiffStores(before, after)
	if len(changes.Removed) != 1 || changes.Removed[0] != "BAR" {
		t.Errorf("expected Removed=[BAR], got %+v", changes)
	}
	if len(changes.Added) != 0 || len(changes.Rotated) != 0 {
		t.Errorf("unexpected changes: %+v", changes)
	}
}

func TestDiffStores_Rotated(t *testing.T) {
	before := makeStore([]Entry{entry("FOO", t0)})
	after := makeStore([]Entry{entry("FOO", t1)}) // newer UpdatedAt

	changes := DiffStores(before, after)
	if len(changes.Rotated) != 1 || changes.Rotated[0] != "FOO" {
		t.Errorf("expected Rotated=[FOO], got %+v", changes)
	}
	if len(changes.Added) != 0 || len(changes.Removed) != 0 {
		t.Errorf("unexpected changes: %+v", changes)
	}
}

func TestDiffStores_Mixed(t *testing.T) {
	before := makeStore([]Entry{
		entry("FOO", t0),
		entry("OLD", t0),
	})
	after := makeStore([]Entry{
		entry("FOO", t1), // rotated
		entry("NEW", t0), // added
		// OLD removed
	})

	changes := DiffStores(before, after)
	if len(changes.Added) != 1 {
		t.Errorf("expected 1 added, got %v", changes.Added)
	}
	if len(changes.Removed) != 1 {
		t.Errorf("expected 1 removed, got %v", changes.Removed)
	}
	if len(changes.Rotated) != 1 {
		t.Errorf("expected 1 rotated, got %v", changes.Rotated)
	}
	if changes.Total() != 3 {
		t.Errorf("expected Total()=3, got %d", changes.Total())
	}
}

func TestDiffStores_EmptyToPopulated(t *testing.T) {
	before := makeStore(nil)
	after := makeStore([]Entry{entry("FOO", t0), entry("BAR", t0)})

	changes := DiffStores(before, after)
	if len(changes.Added) != 2 {
		t.Errorf("expected 2 added, got %v", changes.Added)
	}
}

func TestDiffStores_PopulatedToEmpty(t *testing.T) {
	before := makeStore([]Entry{entry("FOO", t0), entry("BAR", t0)})
	after := makeStore(nil)

	changes := DiffStores(before, after)
	if len(changes.Removed) != 2 {
		t.Errorf("expected 2 removed, got %v", changes.Removed)
	}
}

func TestStoreChanges_IsEmpty(t *testing.T) {
	if !(StoreChanges{}).IsEmpty() {
		t.Error("zero StoreChanges should be empty")
	}
	if (StoreChanges{Added: []string{"X"}}).IsEmpty() {
		t.Error("non-empty StoreChanges reported as empty")
	}
}
