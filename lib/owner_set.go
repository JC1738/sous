package sous

import (
	"fmt"
	"sort"
)

// OwnerSet collects the names of the owners of a deployment.
type OwnerSet map[string]struct{}

// NewOwnerSet creates a new owner set from the provided list of owners.
func NewOwnerSet(owners ...string) OwnerSet {
	os := OwnerSet{}
	for _, o := range owners {
		os.Add(o)
	}
	return os
}

// Add adds an owner to an ownerset.
func (os OwnerSet) Add(owner string) {
	os[owner] = struct{}{}
}

// Remove removes an owner from an ownerset.
func (os OwnerSet) Remove(owner string) {
	delete(os, owner)
}

// Clone returns a deep copy of this OwnerSet.
func (os OwnerSet) Clone() OwnerSet {
	o := make(OwnerSet, len(os))
	for k, v := range os {
		o[k] = v
	}
	return o
}

// Merge adds the owners from other to this OwnerSet
func (os OwnerSet) Merge(other OwnerSet) {
	for k := range other {
		os.Add(k)
	}
}

// Equal returns true if two ownersets contain the same owner names.
func (os OwnerSet) Equal(o OwnerSet) bool {
	diff, _ := os.Diff(o)
	return !diff
}

// Diff returns (false, nil) if os == o. Otherwise it returns false and a list
// of short descriptions of each difference.
func (os OwnerSet) Diff(o OwnerSet) (bool, []string) {
	var diffs []string
	diff := func(format string, a ...interface{}) { diffs = append(diffs, fmt.Sprintf(format, a...)) }

	if len(os) != len(o) {
		diff("different lengths: %d vs %d", len(os), len(o))
	}
	for ownr := range os {
		if _, has := o[ownr]; !has {
			diff("Owner %q missing from other", ownr)
		}
	}

	return len(diffs) != 0, diffs
}

// Slice returns this set as a slice of unique strings, in alphabetical order.
func (os OwnerSet) Slice() []string {
	slice := make([]string, 0, len(os))
	for owner := range os {
		slice = append(slice, owner)
	}
	sort.Strings(slice)
	return slice
}
