package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet_AddAndLookup(t *testing.T) {
	set := newTestSet()
	item := &TestSetHashable{val: 42}

	_, found := set.Lookup(item)
	assert.False(t, found, "expected item not to be found before adding")

	id := set.Add(item)
	assert.NotEqual(t, 0, "expected non-zero ID")
	assert.Equal(t, 1, set.Count(), "expected count to be 1 after adding item")

	foundID, found := set.Lookup(item)
	assert.True(t, found, "expected item should be found after adding")
	assert.Equal(t, id, foundID, "expected found ID to be the same as added ID")
}

func TestSet_RefCounting(t *testing.T) {
	set := newTestSet()
	item := &TestSetHashable{val: 1}

	id := set.Add(item)
	assert.NotEqual(t, 0, id, "expected non-zero ID after adding item")
	assert.EqualValues(t, 1, set.items[id].meta.ref, "expected ref count to be 1 after adding item")

	set.Use(id)
	assert.EqualValues(t, 2, set.items[id].meta.ref,
		"expected ref count incremented to 2 after Use",
	)

	set.Use(id)
	assert.EqualValues(t, 3, set.items[id].meta.ref,
		"expected ref count incremented to 3 after Use",
	)

	// Release twice, should still be alive
	set.Release(id)
	assert.EqualValues(t, 2, set.items[id].meta.ref,
		"expected ref count reduce to 2 after Release",
	)

	set.Release(id)
	assert.EqualValues(t, 1, set.items[id].meta.ref,
		"expected ref count reduce to 1 after Release",
	)

	// Release last time, should decrement living
	set.Release(id)
	assert.EqualValues(t, 0, set.items[id].meta.ref,
		"expected ref count reduced to 0 after Release",
	)

	assert.Equal(t, 0, set.Count(), "expected count to be 0 after releasing all references")
}

func TestSet_AddDuplicateIncrementsRef(t *testing.T) {
	set := newTestSet()
	item := &TestSetHashable{val: 99}

	id1 := set.Add(item)
	id2 := set.Add(item)
	assert.Equal(t, id1, id2, "expected same ID for duplicate add")

	// Should have ref count 2 now
	assert.EqualValues(t, 2, set.items[id1].meta.ref,
		"expected ref count to be 2 after duplicate add",
	)
}

func TestSet_DeleteItem(t *testing.T) {
	set := newTestSet()
	item := &TestSetHashable{val: 123}
	id := set.Add(item)

	set.DeleteItem(id)
	_, found := set.Lookup(item)
	assert.False(t, found, "expected item to be deleted")
}

func TestSet_Count(t *testing.T) {
	set := newTestSet()
	item1 := &TestSetHashable{val: 1}
	item2 := &TestSetHashable{val: 2}

	set.Add(item1)
	set.Add(item2)
	assert.Equal(t, 2, set.Count(), "expected count to be 2 after adding two items")
}

func TestSet_AddMultipleUniqueItems(t *testing.T) {
	set := newTestSet()
	item1 := newTestHashable(1)
	item2 := newTestHashable(2)
	item3 := newTestHashable(3)

	id1 := set.Add(item1)
	id2 := set.Add(item2)
	id3 := set.Add(item3)

	assert.NotEqual(t, id1, id2)
	assert.NotEqual(t, id2, id3)
	assert.NotEqual(t, id1, id3)

	assert.Equal(t, 3, set.Count())
}

func TestSet_RefCountingMultipleAdds(t *testing.T) {
	set := newTestSet()
	item := newTestHashable(42)

	id := set.Add(item)
	assert.Equal(t, int64(1), set.items[id].meta.ref)

	id2 := set.Add(item)
	assert.Equal(t, id, id2)
	assert.Equal(t, int64(2), set.items[id].meta.ref)

	set.Release(id)
	assert.Equal(t, int64(1), set.items[id].meta.ref)

	set.Release(id)
	assert.Equal(t, int64(0), set.items[id].meta.ref)
	assert.Equal(t, 0, set.Count())
}

func TestSet_DeleteAndReAdd(t *testing.T) {
	set := newTestSet()
	item := newTestHashable(100)

	id := set.Add(item)
	assert.NotEqual(t, ID(0), id)
	foundID, found := set.Lookup(item)
	assert.Equal(t, id, foundID)
	assert.True(t, found)

	set.DeleteItem(id)

	foundID, found = set.Lookup(item)
	assert.Equal(t, ID(0), foundID)
	assert.False(t, found)

	// Re-add after deletion
	newID := set.Add(item)
	assert.NotEqual(t, id, newID)
	assert.Equal(t, int64(1), set.items[newID].meta.ref)
	assert.Equal(t, 1, set.Count())
}

func TestSet_LookupNonExistent(t *testing.T) {
	set := newTestSet()
	item := newTestHashable(999)

	id, found := set.Lookup(item)
	assert.False(t, found)
	assert.Equal(t, ID(0), id)
}

func TestSet_ReleaseBelowZeroPanics(t *testing.T) {
	set := newTestSet()
	item := newTestHashable(7)
	id := set.Add(item)
	set.Release(id)

	assert.Panics(t, func() {
		set.Release(id)
	})
}

func TestSet_UseIncrementsRef(t *testing.T) {
	set := newTestSet()
	item := newTestHashable(55)
	id := set.Add(item)

	set.Use(id)
	assert.Equal(t, int64(2), set.items[id].meta.ref)
}

func TestSet_AddAfterDeleteDoesNotLeak(t *testing.T) {
	set := newTestSet()
	item := newTestHashable(1234)
	id := set.Add(item)
	set.Release(id)
	set.DeleteItem(id)

	// Add a new item and ensure count is correct
	item2 := newTestHashable(5678)
	id2 := set.Add(item2)
	assert.Equal(t, 1, set.Count())
	assert.NotEqual(t, id, id2)
}

func newTestSet() *RefCountedSet {
	return NewRefCountedSet(Options{
		Cap: nil, // Use default capacity
	})
}

type TestSetHashable struct {
	val uint64
}

func (t *TestSetHashable) Hash() uint64 {
	return t.val
}

func (t *TestSetHashable) Equals(other Hashable) bool {
	o, ok := other.(*TestSetHashable)
	return ok && o.val == t.val
}

func (t *TestSetHashable) Delete() {}

func newTestHashable(val uint64) *TestSetHashable {
	return &TestSetHashable{val: val}
}
