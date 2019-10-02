package rtree

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func TestRandom(t *testing.T) {
	for maxCapacity := 2; maxCapacity <= 10; maxCapacity++ {
		for minCapacity := 1; minCapacity <= maxCapacity/2; minCapacity++ {
			for population := 0; population < 50; population++ {
				name := fmt.Sprintf("min_%d_max_%d_pop_%d", minCapacity, maxCapacity, population)
				t.Run(name, func(t *testing.T) {
					//fmt.Println("running test ", name)
					rnd := rand.New(rand.NewSource(0))
					boxes := make([]BBox, population)
					for i := range boxes {
						boxes[i] = randomBox(rnd, 0.9, 0.1)
						//fmt.Println("\tbox", i, boxes[i])
					}

					rt, err := New(minCapacity, maxCapacity)
					if err != nil {
						t.Fatal(err)
					}
					for i, bb := range boxes {
						//fmt.Println("\tinserting", i)
						rt.Insert(bb, i)
						checkInvariants(t, rt)
					}

					for i := 0; i < 10; i++ {
						searchBB := randomBox(rnd, 0.5, 0.5)
						var got []int
						//fmt.Println("\tseacrhing", i)
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
