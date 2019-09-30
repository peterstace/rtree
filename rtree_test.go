package rtree_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"github.com/peterstace/rtree"
)

func TestRandom(t *testing.T) {
	// TODO: parameterise over the number of inserted boxes.
	// TODO: parameterise over the min and max capacity of the tree.
	for maxCapacity := 2; maxCapacity <= 10; maxCapacity++ {
		for minCapacity := 1; minCapacity <= maxCapacity/2; minCapacity++ {
			for population := 0; population < 100; population++ {
				name := fmt.Sprintf("min_%d_max_%d_pop_%d", minCapacity, maxCapacity, population)
				t.Run(name, func(t *testing.T) {
					fmt.Println("running test ", name)
					rnd := rand.New(rand.NewSource(0))
					boxes := make([]rtree.BBox, population)
					for i := range boxes {
						boxes[i] = randomBox(rnd, 0.9, 0.1)
						fmt.Println("\tbox", i, boxes[i])
					}

					rt, err := rtree.New(minCapacity, maxCapacity)
					if err != nil {
						t.Fatal(err)
					}
					for i, bb := range boxes {
						fmt.Println("\tinserting", i)
						rt.Insert(bb, i)
						// TODO: check tree invariants
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
							t.Errorf("got: %v want: %v", got, want)
						}
					}
				})
			}
		}
	}
}

func randomBox(rnd *rand.Rand, maxStart, maxWidth float64) rtree.BBox {
	bb := rtree.BBox{
		MinX: rnd.Float64() * maxStart,
		MinY: rnd.Float64() * maxStart,
	}
	bb.MaxX = bb.MinX + rnd.Float64()*maxWidth
	bb.MaxY = bb.MinY + rnd.Float64()*maxWidth
	return bb
}

func overlap(bbox1, bbox2 rtree.BBox) bool {
	return true &&
		(bbox1.MinX <= bbox2.MaxX) && (bbox1.MaxX >= bbox2.MinX) &&
		(bbox1.MinY <= bbox2.MaxY) && (bbox1.MaxY >= bbox2.MinY)
}

// TODO: test for error propagation
