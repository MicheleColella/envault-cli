package vault

// StoreChanges describes what changed between two snapshots of a Store.
type StoreChanges struct {
	Added   []string // entries present in after but not in before
	Removed []string // entries present in before but not in after
	Rotated []string // entries present in both but with a newer UpdatedAt
}

// IsEmpty reports whether there are no changes.
func (c StoreChanges) IsEmpty() bool {
	return len(c.Added) == 0 && len(c.Removed) == 0 && len(c.Rotated) == 0
}

// Total returns the total number of changed entries.
func (c StoreChanges) Total() int {
	return len(c.Added) + len(c.Removed) + len(c.Rotated)
}

// DiffStores computes what changed between before and after.
// Comparison is by entry name only (kind is ignored for reporting).
func DiffStores(before, after *Store) StoreChanges {
	beforeMap := make(map[string]Entry, len(before.Entries))
	for _, e := range before.Entries {
		beforeMap[e.Name] = e
	}

	afterMap := make(map[string]Entry, len(after.Entries))
	for _, e := range after.Entries {
		afterMap[e.Name] = e
	}

	var changes StoreChanges

	for name, afterEntry := range afterMap {
		beforeEntry, existed := beforeMap[name]
		if !existed {
			changes.Added = append(changes.Added, name)
		} else if afterEntry.UpdatedAt.After(beforeEntry.UpdatedAt) {
			changes.Rotated = append(changes.Rotated, name)
		}
	}

	for name := range beforeMap {
		if _, still := afterMap[name]; !still {
			changes.Removed = append(changes.Removed, name)
		}
	}

	return changes
}
