package set

import (
	"fmt"

	"github.com/hnimtadd/termio/terminal/utils"
)

type Hashable interface {
	Hash() uint64
	Equals(t Hashable) bool
	Delete()
}

type ID uint64

// Metadata for an item in the set.
type metadata struct {
	bucketID uint64 // The ID of the bucket this item belongs to.

	// The length of the probe sequence for this item.
	psl uint64

	// Ref is the reference count of the item.
	ref int64
}

type elem struct {
	data Hashable
	meta metadata
}

type RefCountedSet struct {
	// The backing store of items.
	items []*elem
	// A hash table of item indexes.
	table map[uint64]ID

	// Maximum probe sequence length.
	maxPSL uint64

	// pslStats keeps track of number of each items for each probe sequence
	// length. We keep this to shrink maxPSL when we delete items.
	pslStats []int64

	// The next index to store an item at.
	// Id 0 is reserved for unused items.
	nextID ID

	// The number of living items currently in the set.
	living int
}

type Options struct {
	// Cap is the maximum number of items in the set.
	// If not set, it defaults to 1000.
	Cap *uint64
}

func NewRefCountedSet(opts Options) *RefCountedSet {
	var cap uint64
	if opts.Cap == nil {
		cap = 1000 // Default capacity.
	} else {
		cap = *opts.Cap
	}
	return &RefCountedSet{
		items:    make([]*elem, cap),
		table:    make(map[uint64]ID, cap),
		pslStats: make([]int64, 32),
		maxPSL:   0,
		nextID:   1, // Start from 1, since 0 is reserved for unused items.
	}
}

// Add an item to the set if not present and increment its ref count.
//
// Returns the item's ID.
//
// If the set has no more room, then an OutOfMemory error is returned.
func (s *RefCountedSet) Add(value Hashable) ID {
	items := s.items

	// Trim dead items from the end of list.
checkLoop:
	for s.nextID > 1 {
		prev := items[s.nextID-1]
		switch {
		case prev != nil && prev.meta.ref == 0:
			s.nextID--
			s.DeleteItem(s.nextID)
		default:
			break checkLoop
		}

	}

	// If the item already exists, return it.
	if id, found := s.Lookup(value); found {
		items[id].meta.ref++
		return id
	}

	id := s.Insert(uint64(s.nextID), value)
	items[id].meta.ref += 1
	utils.Assert(
		items[id].meta.ref == 1,
		fmt.Sprintf("item ref count should be 1 instead of %d",
			items[id].meta.ref),
	)
	s.living++

	// id is different nextID if we already resurrect an item on the way.
	if id == ID(s.nextID) {
		s.nextID++
	}
	return id
}

// Insert the given value into the hash table with the given items ID.
// asssert that this item is already not present in the table.
func (s *RefCountedSet) Insert(newID uint64, value Hashable) ID {
	_, found := s.Lookup(value)
	utils.Assert(!found, "item already exists in the set")

	table := s.table
	items := s.items

	// The new item that we're inserting.
	newItem := &elem{
		data: value,
		meta: metadata{
			psl: 0,
			ref: 0,
		},
	}

	// ID that we are currently hold, we use this while swap elements or
	// resurrect them.
	heldID := newID
	heldItem := newItem

	// The final ID that the new item will be inserted to
	chosenID := newID

	hash := value.Hash()

	for i := 0; i <= cap(items); i++ {
		p := (hash + uint64(i)) % uint64(len(items))
		id := table[p]

		// If we met empty bucket, we can insert the item here.
		if id == 0 {
			table[p] = ID(heldID)
			heldItem.meta.bucketID = p
			heldItem.meta.psl = uint64(i)
			s.pslStats[heldItem.meta.psl]++
			s.maxPSL = max(s.maxPSL, heldItem.meta.psl)
			break
		}

		item := items[id]

		// If there's a dead item then we resurrect it
		// for our value so that we can re-use its ID,
		// unless its ID is greater than the one we're
		// given (i.e. prefer smaller IDs).
		if item.meta.ref == 0 {
			// Reap the dead item.
			s.pslStats[item.meta.psl] -= 1
			*item = elem{}

			// Only resurrect this item if it has a
			// smaller id than the one we were given.
			if id < ID(newID) {
				chosenID = uint64(id)
			}
			// Put the currently held item in to the
			// bucket of the item that we just reaped.
			table[p] = ID(heldID)
			heldItem.meta.bucketID = p
			s.pslStats[heldItem.meta.psl] += 1
			s.maxPSL = max(s.maxPSL, heldItem.meta.psl)

			break
		}

		// If this item has a lower PSL, or has equal PSL and lower ref count,
		// then we swap it our with the held item. By doing this, items with
		// higher reference counts are prioritized for earlier placement.
		// This assumption is that an item which has a higher ref count will
		// be accessed more frequently, and thus should be placed earlier
		// in the table.
		if item.meta.psl < heldItem.meta.psl ||
			(item.meta.psl == heldItem.meta.psl &&
				item.meta.ref < heldItem.meta.ref) {
			// Put our held item in the bucket.
			table[p] = ID(heldID)
			s.pslStats[heldItem.meta.psl]++
			s.maxPSL = max(s.maxPSL, heldItem.meta.psl)

			// Picket the item that has a lower PSL.
			heldID = uint64(id)
			heldItem = item
			s.pslStats[item.meta.psl]--
		}

		// Advance to the next probe position for our held item.
		heldItem.meta.psl++
	}

	// Our chosen ID may have changed if we decided
	// to re-use a dead item's ID, so we make sure
	// the chosen bucket contains the correct ID.
	table[newItem.meta.bucketID] = ID(chosenID)

	fmt.Println("add", newItem.data.Hash(), "to", chosenID)
	// Finally place our new item in to our array.
	items[chosenID] = newItem

	return ID(chosenID)
}

// Delete an item, removing any references from the table, and freeing its ID
// to be re-used.
func (s *RefCountedSet) DeleteItem(id ID) {
	table := s.table
	items := s.items
	item := items[id]

	utils.Assert(table[item.meta.bucketID] == id, "item not found in table")

	s.pslStats[item.meta.psl]-- // Decrement the PSL stats for this item.
	table[item.meta.bucketID] = 0
	items[id] = nil // Remove the item from the items slice.

	prev := item.meta.bucketID
	next := (prev + 1) % uint64(len(items))

	// clean up subsequence items in this same bucket.
	for table[next] != 0 && items[table[next]].meta.psl > 0 {
		// assign the bucketID to the previous item.
		items[table[next]].meta.bucketID = prev
		items[table[next]].meta.psl--

		// Move the item to the previous index.
		table[prev] = table[next]

		prev = next
		next = (next + 1) % uint64(len(items))
	}

	// Shrink the maxPSL
	for s.maxPSL > 0 && s.pslStats[s.maxPSL] == 0 {
		s.maxPSL--
	}

	// mark the previous item as unused.
	table[prev] = 0

	// Hack so if the ref is not 0, means we delete this item out of the set
	// without releasing it. That means, we have to manually decrement the
	// living count.
	if item.meta.ref > 0 {
		s.living--
	}
}

// Releases a reference to an item by its ID.
//
// Asserts that the item's reference count is greater than 0.
func (s *RefCountedSet) Release(id ID) {
	utils.Assert(id > 0, "cannot release item with ID 0")
	items := s.items
	item := items[id]

	utils.Assert(item.meta.ref > 0)
	item.meta.ref -= 1
	if item.meta.ref == 0 {
		s.living -= 1
	}
}

// Lookup find an item in the table and return its ID.
// If the item doesn't exist in the table, return nil and false.
func (s *RefCountedSet) Lookup(val Hashable) (ID, bool) {
	table := s.table
	items := s.items

	hash := val.Hash()

	for i := uint64(0); i <= s.maxPSL; i++ {
		p := (hash + i) % uint64(len(items))
		id := table[p]

		// Empty bucket, our item cannot have probed to any point after this
		// meaning it's not present.
		if id == 0 {
			return 0, false
		}

		item := items[id]

		// An item with a shorter probe sequence length cannot have probed
		// to this point, since it would be swapped out after previous item
		// delete.
		if item.meta.psl < i {
			return 0, false
		}

		// Check dead items also.
		if item.meta.ref == 0 {
			continue
		}

		// If the item is a part of the same probe sequence, check if it
		// matches the value we're looking for.
		if item.meta.psl == i && item.data.Equals(val) {
			return id, true
		}

	}
	return 0, false
}

func (s *RefCountedSet) Use(id ID) {
	utils.Assert(id > 0, "cannot use item with ID 0")
	items := s.items
	item := items[id]

	// If `use` is being called on an item with 0 references, then
	// either someone forgot to call it before, released too early
	// or lied about releasing. In any case something is wrong and
	// shouldn't be allowed.
	utils.Assert(item.meta.ref > 0)

	item.meta.ref++ // Increment the reference count.
}

func (s *RefCountedSet) Count() int {
	return s.living
}
