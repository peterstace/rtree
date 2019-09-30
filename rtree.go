package rtree

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
)

type BBox struct {
	MinX, MinY, MaxX, MaxY float64
}

type Node struct {
	ParentIndex int
	IsLeaf      bool
	Entries     []Entry
}

type Entry struct {
	BBox  BBox
	Index int
}

type RTree struct {
	MinChildren int
	MaxChildren int
	RootIndex   int
	Nodes       []Node
}

func New(minChildren, maxChildren int) (RTree, error) {
	if minChildren > maxChildren/2 {
		return RTree{}, errors.New("min children must be less than or equal to half of the max children")
	}
	return RTree{
		MinChildren: minChildren,
		MaxChildren: maxChildren,
		RootIndex:   0,
		Nodes:       []Node{Node{ParentIndex: -1, IsLeaf: true, Entries: nil}},
	}, nil
}

func (t *RTree) Search(bb BBox, callback func(index int) error) error {
	var recurse func(*Node) error
	recurse = func(n *Node) error {
		for _, entry := range n.Entries {
			if !overlap(entry.BBox, bb) {
				continue
			}
			var err error
			if n.IsLeaf {
				err = callback(entry.Index)
			} else {
				err = recurse(&t.Nodes[entry.Index])
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
	return recurse(&t.Nodes[t.RootIndex])
}

func (t *RTree) Insert(bb BBox, dataIndex int) {
	leaf := t.chooseLeafNode(bb)
	t.Nodes[leaf].Entries = append(t.Nodes[leaf].Entries, Entry{BBox: bb, Index: dataIndex})
	if len(t.Nodes[leaf].Entries) <= t.MaxChildren {
		return
	}

	newNode := t.splitNode(leaf)
	root1, root2 := t.adjustTree(leaf, newNode)

	if root2 != -1 {
		t.joinRoots(root1, root2)
	}
}

func (t *RTree) joinRoots(r1, r2 int) {
	t.Nodes = append(t.Nodes, Node{
		ParentIndex: -1,
		IsLeaf:      false,
		Entries: []Entry{
			Entry{
				BBox:  t.calculateBound(r1),
				Index: r1,
			},
			Entry{
				BBox:  t.calculateBound(r2),
				Index: r2,
			},
		},
	})
	t.RootIndex = len(t.Nodes) - 1
	t.Nodes[r1].ParentIndex = t.RootIndex
	t.Nodes[r2].ParentIndex = t.RootIndex
}

func (t *RTree) adjustTree(n, nn int) (int, int) {
	fmt.Println("\t\t[adjustTree] n nn ", n, nn)
	for {
		fmt.Println("\t\t[adjustTree] rootIndex", t.RootIndex)
		if n == t.RootIndex {
			return n, nn
		}
		parent := t.Nodes[n].ParentIndex
		fmt.Println("\t\t[adjustTree] parent", parent)
		parentEntry := -1
		for i, entry := range t.Nodes[parent].Entries {
			if entry.Index == n {
				parentEntry = i
				break
			}
		}
		t.Nodes[parent].Entries[parentEntry].BBox = t.calculateBound(n)

		// AT4
		pp := -1
		if nn != -1 {
			newEntry := Entry{
				BBox:  t.calculateBound(nn),
				Index: nn,
			}

			t.Nodes[parent].Entries = append(t.Nodes[parent].Entries, newEntry)

			if len(t.Nodes[parent].Entries) > t.MaxChildren {
				pp = t.splitNode(parent)
			}
		}

		n, nn = parent, pp
	}
}

// calculate bound calculates the smallest bounding box that fits a node.
func (t *RTree) calculateBound(n int) BBox {
	bb := t.Nodes[n].Entries[0].BBox
	for _, entry := range t.Nodes[n].Entries[1:] {
		bb = combine(bb, entry.BBox)
	}
	return bb
}

// splitNode splits node with index n into two nodes. The first node replaces
// n, and the second node is newly created. The return value is the index of
// the new node.
func (t *RTree) splitNode(n int) int {
	var (
		// All zeros would not be valid split, so start at 1.
		minSplit = uint64(1)
		// The MSB should always be 0, to remove duplicates from inverting the
		// bit pattern. So we raise 2 to the power of one less than the number
		// of entries rather than the number of entries.
		//
		// E.g. for 4 entries, we want the following bit patterns:
		// 0001, 0010, 0011, 0100, 0101, 0110, 0111.
		//
		// (1 << (4 - 1)) - 1 == 0111, so the maths checks out.
		maxSplit = uint64((1 << (len(t.Nodes[n].Entries) - 1)) - 1)
	)
	bestArea := math.Inf(+1)
	var bestSplit uint64
	for split := minSplit; split <= maxSplit; split++ {
		if bits.OnesCount64(split) < t.MinChildren {
			continue
		}
		var bboxA, bboxB BBox
		var hasA, hasB bool
		for i, entry := range t.Nodes[n].Entries {
			if split&(1<<i) == 0 {
				if hasA {
					bboxA = combine(bboxA, entry.BBox)
				} else {
					bboxA = entry.BBox
				}
			} else {
				if hasB {
					bboxB = combine(bboxB, entry.BBox)
				} else {
					bboxB = entry.BBox
				}
			}
		}
		combinedArea := area(bboxA) + area(bboxB)
		if combinedArea < bestArea {
			bestArea = combinedArea
			bestSplit = split
		}
	}

	var entriesA, entriesB []Entry
	for i, entry := range t.Nodes[n].Entries {
		if bestSplit&(1<<i) == 0 {
			entriesA = append(entriesA, entry)
		} else {
			entriesB = append(entriesB, entry)
		}
	}

	// Use the existing node for A, and create a new node for B.
	t.Nodes[n].Entries = entriesA
	t.Nodes = append(t.Nodes, Node{
		ParentIndex: -1,
		IsLeaf:      t.Nodes[n].IsLeaf,
		Entries:     entriesB,
	})
	return len(t.Nodes) - 1
}

func (t *RTree) chooseLeafNode(bb BBox) int {
	if len(t.Nodes) == 0 {
		t.Nodes = append(t.Nodes, Node{IsLeaf: true, Entries: nil})
	}
	node := t.RootIndex

	for {
		if t.Nodes[node].IsLeaf {
			return node
		}
		bestDelta := enlargement(bb, t.Nodes[node].Entries[0].BBox)
		bestEntry := 0
		for i, entry := range t.Nodes[node].Entries[1:] {
			delta := enlargement(bb, entry.BBox)
			if delta < bestDelta {
				bestDelta = delta
				bestEntry = i
			} else if delta == bestDelta && area(entry.BBox) < area(t.Nodes[node].Entries[bestEntry].BBox) {
				// Area is used as a tie breaking if the enlargements are the same.
				bestEntry = i
			}
		}
		node = t.Nodes[node].Entries[bestEntry].Index
	}
}

// combine gives the smallest bounding box containing both bbox1 and bbox2.
func combine(bbox1, bbox2 BBox) BBox {
	return BBox{
		MinX: math.Min(bbox1.MinX, bbox2.MinX),
		MinY: math.Min(bbox1.MinY, bbox2.MinY),
		MaxX: math.Max(bbox1.MaxX, bbox2.MaxX),
		MaxY: math.Max(bbox1.MaxY, bbox2.MaxY),
	}
}

// enlargment returns how much additional area the existing BBox would have to
// enlarge by to accomodate the additional BBox.
func enlargement(existing, additional BBox) float64 {
	return area(combine(existing, additional)) - area(existing)
}

func area(bb BBox) float64 {
	return (bb.MaxX - bb.MinX) * (bb.MaxY - bb.MinY)
}

func overlap(bbox1, bbox2 BBox) bool {
	return true &&
		(bbox1.MinX <= bbox2.MaxX) && (bbox1.MaxX >= bbox2.MinX) &&
		(bbox1.MinY <= bbox2.MaxY) && (bbox1.MaxY >= bbox2.MinY)
}
