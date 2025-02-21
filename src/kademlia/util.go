package kademlia

import (
	"errors"
	"math/bits"
	"math/rand"
)

// Returns a randomly generated id.
func RandomID() [5]uint32 {
	var res [5]uint32
	for res == [5]uint32{0, 0, 0, 0, 0} {
		for i := 0; i < 5; i++ {
			res[i] = rand.Uint32()
		}
	}
	return res
}

func RandomIP() [4]byte {
	var res [4]byte
	for i := 0; i < 4; i++ {
		b, _ := RandU32(0, 256)
		res[i] = byte(b)
	}
	return res
}

// returns a pseudo-random uint32 in the range (min, max]
func RandU32(min uint32, max uint32) (uint32, error) {
	if min >= max {
		return 0, errors.New("invalid range")
	}
	x := rand.Uint32()
	x %= (max - min)
	x += min
	return x, nil
}

// returns the xor distance metric for between the nodes
func RelativeDistance(nodeA [5]uint32, nodeB [5]uint32) [5]uint32 {
	relDist := [5]uint32{0, 0, 0, 0, 0}
	for i := 0; i < 5; i++ {
		relDist[i] = nodeA[i] ^ nodeB[i]
	}
	return relDist
}

// Returns true if node A is closer to the target than node B, returns false if node B is closer to target than node A.
func CloserNode(nodeA [5]uint32, nodeB [5]uint32, target [5]uint32) bool {
	distA := RelativeDistance(nodeA, target)
	distB := RelativeDistance(nodeB, target)
	for i := 0; i < 5; i++ {
		if distA[i] < distB[i] {
			return true
		} else if distB[i] < distA[i] {
			return false
		}
	}
	return false
}

// Returns true if node A and B are the same distance from the target, otherwise returns false.
func EquiDistantNode(nodeA [5]uint32, nodeB [5]uint32, target [5]uint32) bool {
	distA := RelativeDistance(nodeA, target)
	distB := RelativeDistance(nodeB, target)
	if distA == distB {
		return true
	}
	return false
}

// Returns the shared prefix length between the supplied ID's
func DistPrefixLength(idA [5]uint32, idB [5]uint32) int {
	length := 0
	for i := 0; i < len(idA); i++ {
		segDist := bits.LeadingZeros32(idA[i] ^ idB[i])
		length += segDist
		if segDist != 32 {
			break
		}
	}
	return length
}

// returns true if node A is larger than node node
// returns false if node B is larger than or equal to node A
func LargerNode(nodeA [5]uint32, nodeB [5]uint32) bool {
	for i := 0; i < 5; i++ {
		if nodeA[i] > nodeB[i] {
			return true
		} else if nodeA[i] < nodeB[i] {
			return false
		}
	}
	return false
}

// sorts contact slice based on distance to the target
func SortContactsByDistance(input *[]Contact, target [5]uint32) {
	for i := 1; i < len(*input); i++ {
		for j := 0; j < len(*input)-1; j++ {
			nodeA := (*input)[j]
			nodeB := (*input)[j+1]
			sortCriterion := CloserNode(nodeB.ID(), nodeA.ID(), target)

			// if node B is close to target than node A
			if sortCriterion {
				(*input)[j] = nodeB
				(*input)[j+1] = nodeA

				//if node A and B are the same distance from target
			} else if EquiDistantNode(nodeA.ID(), nodeB.ID(), target) {

				// if node A is larger than node B
				if LargerNode(nodeA.ID(), nodeB.ID()) {
					(*input)[j] = nodeB
					(*input)[j+1] = nodeA
				}
			}
		}
	}
}

// Merges two slices of Contacts and removes all duplicates.
func MergeContactsByDistance(setA *[]Contact, setB *[]Contact, target [5]uint32) []Contact {
	res := make([]Contact, 0)
	res = append(res, (*setA)...)
	res = append(res, (*setB)...)
	SortContactsByDistance(&res, target)
	RemoveDuplicateContacts(&res)
	return res
}

func RemoveDuplicateContacts(set *[]Contact) {
	for i := len(*set) - 1; i > 0; i-- {
		if (*set)[i].ID() == (*set)[i-1].ID() {
			*set = append((*set)[:i-1], (*set)[i:]...)
		}
	}
}

// Returns true if the slice contains a node with provided ID.
func SliceContains(id [5]uint32, slice *[]Contact) bool {
	for _, node := range *slice {
		if node.ID() == id {
			return true
		}
	}
	return false
}

// Returns true if slice B contains all nodes from slice A.
func SliceContainsAll(sliceA *[]Contact, sliceB *[]Contact) bool {
	for _, node := range *sliceA {
		contained := SliceContains(node.ID(), sliceB)
		if !contained {
			return false
		}
	}
	return true
}
