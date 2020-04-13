package rtree

import (
	"errors"
	"math"
	"math/bits"
)

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

// Insert adds a new data item to the RTree.
func (t *RTree) Insert(bb BBox, dataIndex int, policy InsertionPolicy) {
	if len(t.Nodes) == 0 {
		t.Nodes = append(t.Nodes, Node{IsLeaf: true, Entries: nil, Parent: -1})
		t.RootIndex = 0
	}

	leaf := t.chooseLeafNode(bb)
	t.Nodes[leaf].Entries = append(t.Nodes[leaf].Entries, Entry{BBox: bb, Index: dataIndex})

	current := leaf
	for current != t.RootIndex {
		parent := t.Nodes[current].Parent
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
		Parent: -1,
	})
	t.RootIndex = len(t.Nodes) - 1
	t.Nodes[r1].Parent = len(t.Nodes) - 1
	t.Nodes[r2].Parent = len(t.Nodes) - 1
}

func (t *RTree) adjustTree(n, nn int, policy InsertionPolicy) (int, int) {
	for {
		if n == t.RootIndex {
			return n, nn
		}
		parent := t.Nodes[n].Parent
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
			t.Nodes[nn].Parent = parent
			if len(t.Nodes[parent].Entries) > policy.maxChildren {
				pp = t.splitNode(parent, policy)
			}
		}

		n, nn = parent, pp
	}
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
		IsLeaf:  t.Nodes[n].IsLeaf,
		Entries: entriesB,
		Parent:  -1,
	})
	if !t.Nodes[n].IsLeaf {
		for _, entry := range entriesB {
			t.Nodes[entry.Index].Parent = len(t.Nodes) - 1
		}
	}
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
