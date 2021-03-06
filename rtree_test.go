package rtree

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func TestRandom(t *testing.T) {
	for population := 0; population < 75; population++ {
		t.Run(fmt.Sprintf("bulk_%d", population), func(t *testing.T) {
			rnd := rand.New(rand.NewSource(0))
			boxes := make([]BBox, population)
			for i := range boxes {
				boxes[i] = randomBox(rnd, 0.9, 0.1)
			}

			inserts := make([]InsertItem, len(boxes))
			for i := range inserts {
				inserts[i].BBox = boxes[i]
				inserts[i].DataIndex = i
			}
			rt := BulkLoad(inserts)

			checkInvariants(t, rt)
			checkSearch(t, rt, boxes, rnd)
		})
		for maxCapacity := 2; maxCapacity <= 10; maxCapacity++ {
			for minCapacity := 1; minCapacity <= maxCapacity/2; minCapacity++ {
				name := fmt.Sprintf("min_%d_max_%d_pop_%d", minCapacity, maxCapacity, population)
				t.Run(name, func(t *testing.T) {
					rnd := rand.New(rand.NewSource(0))
					boxes := make([]BBox, population)
					for i := range boxes {
						boxes[i] = randomBox(rnd, 0.9, 0.1)
					}

					ins, err := NewInsertionPolicy(minCapacity, maxCapacity)
					if err != nil {
						t.Fatal(err)
					}
					var rt RTree
					for i, bb := range boxes {
						rt.Insert(bb, i, ins)
						checkInvariants(t, rt)
					}

					checkSearch(t, rt, boxes, rnd)
				})
			}
		}
	}
}

func checkSearch(t *testing.T, rt RTree, boxes []BBox, rnd *rand.Rand) {
	for i := 0; i < 10; i++ {
		searchBB := randomBox(rnd, 0.5, 0.5)
		var got []int
		rt.Search(searchBB, func(idx int) {
			got = append(got, idx)
		})

		var want []int
		for i, bb := range boxes {
			if overlap(bb, searchBB) {
				want = append(want, i)
			}
		}

		sort.Ints(want)
		sort.Ints(got)

		if !reflect.DeepEqual(want, got) {
			t.Logf("search bbox: %v", searchBB)
			t.Errorf("search failed, got: %v want: %v", got, want)
		}
	}
}

func randomBox(rnd *rand.Rand, maxStart, maxWidth float64) BBox {
	bb := BBox{
		MinX: rnd.Float64() * maxStart,
		MinY: rnd.Float64() * maxStart,
	}
	bb.MaxX = bb.MinX + rnd.Float64()*maxWidth
	bb.MaxY = bb.MinY + rnd.Float64()*maxWidth

	bb.MinX = float64(int(bb.MinX*100)) / 100
	bb.MinY = float64(int(bb.MinY*100)) / 100
	bb.MaxX = float64(int(bb.MaxX*100)) / 100
	bb.MaxY = float64(int(bb.MaxY*100)) / 100
	return bb
}

func checkInvariants(t *testing.T, rt RTree) {
	t.Logf("")
	t.Logf("RTree description:")
	t.Logf("node_count=%v, root=%d", len(rt.Nodes), rt.RootIndex)
	for i, n := range rt.Nodes {
		t.Logf("%d: leaf=%t numEntries=%d parent=%d", i, n.IsLeaf, len(n.Entries), n.Parent)
		for j, e := range n.Entries {
			t.Logf("\t%d: index=%d bbox=%v", j, e.Index, e.BBox)
		}
	}

	// Each node has the correct parent set.
	for i, node := range rt.Nodes {
		if i == rt.RootIndex {
			if node.Parent != -1 {
				t.Fatalf("expected root to have parent -1, but has %d", node.Parent)
			}
			continue
		}
		if node.Parent == -1 {
			t.Fatalf("expected parent for non-root not to be -1, but was -1")
		}

		var matchingChildren int
		for _, entry := range rt.Nodes[node.Parent].Entries {
			if entry.Index == i {
				matchingChildren++
			}
		}
		if matchingChildren != 1 {
			t.Fatalf("expected parent to have 1 matching child, but has %d", matchingChildren)
		}
	}

	// For each non-leaf node, its entries should have the smallest bounding boxes that cover its children.
	for i, parentNode := range rt.Nodes {
		if parentNode.IsLeaf {
			continue
		}
		for j, parentEntry := range parentNode.Entries {
			childNode := rt.Nodes[parentEntry.Index]
			union := childNode.Entries[0].BBox
			for _, childEntry := range childNode.Entries[1:] {
				union = combine(childEntry.BBox, union)
			}
			if union != parentEntry.BBox {
				t.Fatalf("expected parent to have smallest bbox that covers its children (node=%d, entry=%d)", i, j)
			}
		}
	}

	// Each leaf should be reached exactly once from the root. This implies
	// that the tree has no loops, and there are no orphan leafs. Also checks
	// that each non-leaf is visited at least once (i.e. no orphan non-leaves).
	leafCount := make(map[int]int)
	visited := make(map[int]bool)
	var recurse func(int)
	recurse = func(n int) {
		visited[n] = true
		node := &rt.Nodes[n]
		if node.IsLeaf {
			leafCount[n]++
			return
		}
		for _, entry := range node.Entries {
			recurse(entry.Index)
		}
	}
	recurse(rt.RootIndex)
	for leaf, count := range leafCount {
		if count != 1 {
			t.Fatalf("leaf %d visited %d times", leaf, count)
		}
	}
	for i := range rt.Nodes {
		if !visited[i] {
			t.Fatalf("node %d was not visited", i)
		}
	}
}
