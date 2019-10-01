package rtree

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func TestRandom(t *testing.T) {
	for maxCapacity := 2; maxCapacity <= 3; maxCapacity++ {
		for minCapacity := 1; minCapacity <= maxCapacity/2; minCapacity++ {
			for population := 0; population < 14; population++ {
				name := fmt.Sprintf("min_%d_max_%d_pop_%d", minCapacity, maxCapacity, population)
				t.Run(name, func(t *testing.T) {
					fmt.Println("running test ", name)
					rnd := rand.New(rand.NewSource(0))
					boxes := make([]BBox, population)
					for i := range boxes {
						boxes[i] = randomBox(rnd, 0.9, 0.1)
						fmt.Println("\tbox", i, boxes[i])
					}

					rt, err := New(minCapacity, maxCapacity)
					if err != nil {
						t.Fatal(err)
					}
					for i, bb := range boxes {
						fmt.Println("\tinserting", i)
						rt.Insert(bb, i)
						checkInvariants(t, rt)
					}

					for i := 0; i < 10; i++ {
						searchBB := randomBox(rnd, 0.5, 0.5)
						var got []int
						fmt.Println("\tseacrhing", i)
						rt.Search(searchBB, func(idx int) error {
							got = append(got, idx)
							return nil
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
				})
			}
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
	t.Logf("node count: %v", len(rt.Nodes))
	for i, n := range rt.Nodes {
		t.Logf("%d: leaf=%t numEntries=%d", i, n.IsLeaf, len(n.Entries))
		for j, e := range n.Entries {
			t.Logf("\t%d: index=%d bbox=%v", j, e.Index, e.BBox)
		}
	}

	// Only one node (the root) should have -1 as the parent.
	/*
		var roots []int
		for i, n := range rt.Nodes {
			if n.ParentIndex == -1 {
				roots = append(roots, i)
			}
		}
		if len(roots) != 1 {
			t.Fatalf("expected 1 node with parent -1, but got: %v", roots)
		}
		if roots[0] != rt.RootIndex {
			t.Fatalf("expected the root to have parent -1")
		}
	*/

	// From every node except the root, the parent node (via the parent index)
	// should contain an entry for that node.
	/*
		for i, node := range rt.Nodes {
			if node.ParentIndex == -1 {
				continue
			}
			var count int
			parent := rt.Nodes[node.ParentIndex]
			for _, e := range parent.Entries {
				if e.Index == i {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("expected 1 parent/child return trip, but got: %d (node index %d)", count, i)
			}
		}
	*/

	// TODO: each leaf can reach the root node by traversing the parents.

	// TODO: there are no loops

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
}

// TODO: test for error propagation
