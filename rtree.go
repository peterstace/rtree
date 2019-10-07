package rtree

import (
	"errors"
	"math"
	"math/bits"
)

// BBox is an axis-aligned bounding box.
type BBox struct {
	MinX, MinY, MaxX, MaxY float64
}

// Node is a node in an R-Tree. Nodes can either be leaf nodes holding entries
// for terminal items, or intermediate nodes holding entries for more nodes.
type Node struct {
	IsLeaf  bool
	Entries []Entry
}

// Entry is an entry under a node, leading either to terminal items, or more nodes.
type Entry struct {
	BBox  BBox
	Index int
}

// RTree is an in-memory R-Tree data structure. Its zero value is an empty R-Tree.
type RTree struct {
	RootIndex int
	Nodes     []Node
}

// NewInsertionPolicy creates a new insertion policy with the given node size
// parameters.
func NewInsertionPolicy(minChildren, maxChildren int) (InsertionPolicy, error) {
	if minChildren > maxChildren/2 {
		return InsertionPolicy{}, errors.New("min children must be less than or equal to half of the max children")
	}
	return InsertionPolicy{minChildren, maxChildren}, nil
}

// InsertionPolicy alters the behaviour when inserting new data to an RTree.
type InsertionPolicy struct {
	minChildren int
	maxChildren int
}

// findParent finds the parent of a non-root node n by starting at the root and
// traversing until the parent of the node is found.
func (t *RTree) findParent(n int) int {
	for i, node := range t.Nodes {
		if node.IsLeaf {
			continue
		}
		for _, entry := range node.Entries {
			if entry.Index == n {
				return i
			}
		}
	}
	panic("could not find parent")
}

// Search looks for any items in the tree that overlap with the the given
// bounding box. The callback is called with the item index for each found
// item.
func (t *RTree) Search(bb BBox, callback func(index int)) {
	if len(t.Nodes) == 0 {
		return
	}
	var recurse func(*Node)
	recurse = func(n *Node) {
		for _, entry := range n.Entries {
			if !overlap(entry.BBox, bb) {
				continue
			}
			if n.IsLeaf {
				callback(entry.Index)
			} else {
				recurse(&t.Nodes[entry.Index])
			}
		}
	}
	recurse(&t.Nodes[t.RootIndex])
}

// Insert adds a new data item to the RTree.
func (t *RTree) Insert(bb BBox, dataIndex int, policy InsertionPolicy) {
	if len(t.Nodes) == 0 {
		t.Nodes = append(t.Nodes, Node{IsLeaf: true, Entries: nil})
		t.RootIndex = 0
	}

	leaf := t.chooseLeafNode(bb)
	t.Nodes[leaf].Entries = append(t.Nodes[leaf].Entries, Entry{BBox: bb, Index: dataIndex})

	current := leaf
	for current != t.RootIndex {
		parent := t.findParent(current)
		for i := range t.Nodes[parent].Entries {
			e := &t.Nodes[parent].Entries[i]
			if e.Index == current {
				e.BBox = combine(e.BBox, bb)
				break
			}
		}
		current = parent
	}

	if len(t.Nodes[leaf].Entries) <= policy.maxChildren {
		return
	}

	newNode := t.splitNode(leaf, policy)
	root1, root2 := t.adjustTree(leaf, newNode, policy)

	if root2 != -1 {
		t.joinRoots(root1, root2)
	}
}

func (t *RTree) joinRoots(r1, r2 int) {
	t.Nodes = append(t.Nodes, Node{
		IsLeaf: false,
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
}

func (t *RTree) adjustTree(n, nn int, policy InsertionPolicy) (int, int) {
	for {
		if n == t.RootIndex {
			return n, nn
		}
		parent := t.findParent(n)
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
			if len(t.Nodes[parent].Entries) > policy.maxChildren {
				pp = t.splitNode(parent, policy)
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
func (t *RTree) splitNode(n int, policy InsertionPolicy) int {
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
		if bits.OnesCount64(split) < policy.minChildren {
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
		//ParentIndex: -1,
		IsLeaf:  t.Nodes[n].IsLeaf,
		Entries: entriesB,
	})
	return len(t.Nodes) - 1
}

func (t *RTree) chooseLeafNode(bb BBox) int {
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
