/* Copyright 2019 Tierion
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*     http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package merkletools

import (
	"bytes"
	"crypto/sha256"
	"sync"
)

// Node : A node on the tree
type Node struct {
	Parent     *Node
	Sibling    *Node
	IsLeftNode bool
	IsRoot     bool
	Hash       []byte
}

// MerkleTree : The complete tree structure
type MerkleTree struct {
	Leaves []*Node
	Nodes  []*Node
	Root   []byte
}

// ProofStep : A sibling position and value used to descibe an inclusion proof
type ProofStep struct {
	Left  bool
	Value []byte
}

// Reset : Clears all values in tree
func (mt *MerkleTree) Reset() {
	mt.Leaves = []*Node{}
	mt.Nodes = []*Node{}
	mt.Root = []byte{}
}

// AddLeaf : Adds one leaf to the tree
func (mt *MerkleTree) AddLeaf(hash []byte) {
	node := Node{nil, nil, false, false, hash}
	mt.Leaves = append(mt.Leaves, &node)
}

// AddLeaves : Adds multiple leaves to the tree
func (mt *MerkleTree) AddLeaves(hashes [][]byte) {
	for _, hash := range hashes {
		node := Node{nil, nil, false, false, hash}
		mt.Leaves = append(mt.Leaves, &node)
	}
}

// GetLeaf : Gets the Leaf node at a given index
func (mt *MerkleTree) GetLeaf(index int) *Node {
	if index < 0 || index > len(mt.Leaves)-1 {
		return nil
	}
	return mt.Leaves[index]
}

// GetLeafCount : Returns the total number of leaves
func (mt *MerkleTree) GetLeafCount() int {
	return len(mt.Leaves)
}

// GetMerkleRoot : Returns the Root for this Tree
func (mt *MerkleTree) GetMerkleRoot() []byte {
	return mt.Root
}

// GetProof : Returns the proof for a leaf at the given index
func (mt *MerkleTree) GetProof(index int) []ProofStep {
	if index < 0 || index > len(mt.Leaves)-1 {
		return nil // the index it out of the bounds of the leaf array
	}
	results := []ProofStep{}

	currentNode := mt.Leaves[index]
	for currentNode.IsRoot == false {
		if currentNode.Sibling != nil {
			proofStep := ProofStep{
				Left:  !currentNode.IsLeftNode,
				Value: currentNode.Sibling.Hash,
			}
			results = append(results, proofStep)
		}
		currentNode = currentNode.Parent
	}

	return results
}

// VerifyProof : Checks the validity of the proof and returns true or false
func VerifyProof(proofSteps []ProofStep, targetHash []byte, merkleRoot []byte) bool {
	performHashesTwice := false
	return verifyProof(proofSteps, targetHash, merkleRoot, performHashesTwice)
}

// VerifyBTCProof : Checks the validity of the proof generated with MakeBTCTree and returns true or false
func VerifyBTCProof(proofSteps []ProofStep, targetHash []byte, merkleRoot []byte) bool {
	performHashesTwice := true
	return verifyProof(proofSteps, targetHash, merkleRoot, performHashesTwice)
}

// MakeTree : Builds the tree using the given Leaves
func (mt *MerkleTree) MakeTree() {
	useOddNodeDuplication := false
	performHashesTwice := false
	makeTree(mt, useOddNodeDuplication, performHashesTwice)
}

// MakeBTCTree : Builds the tree using the given Leaves
// These tree will duplicate odd nodes to enforce even number on each level
// This is for compatability with how Bitcoin builds Merkle trees
// Hash operations are performed twice
func (mt *MerkleTree) MakeBTCTree() {
	useOddNodeDuplication := true
	performHashesTwice := true
	makeTree(mt, useOddNodeDuplication, performHashesTwice)
}

func makeTree(mt *MerkleTree, useOddNodeDuplication bool, performHashesTwice bool) {
	if len(mt.Leaves) > 0 {
		// Initialize mt.Nodes and newNodeSet to start off containing all the Leaves
		mt.Nodes = make([]*Node, len(mt.Leaves))
		newNodeSet := make([]*Node, len(mt.Leaves))
		copy(mt.Nodes, mt.Leaves)
		copy(newNodeSet, mt.Leaves)

		// Process newNodeSet while there are still node pairs to process
		for len(newNodeSet) > 1 {
			// Set the current working set to newNodeSet and clear newNodeSet
			currentNodeSet := make([]*Node, len(newNodeSet))
			copy(currentNodeSet, newNodeSet)
			newNodeSet = nil

			// Initialize the WaitGroup
			// This will allow us to wait until all tasks are complete
			var wg sync.WaitGroup
			// Initialize the channnel for our go routines
			c := make(chan *Node)
			// For every hash pair, calculate and return its parent nde
			for i := 0; i < len(currentNodeSet); i += 2 {
				wg.Add(1)
				// if we are making a btc style tree, identify any lone odd nodes
				// and add a duplicate to be paired with
				if useOddNodeDuplication {
					if len(currentNodeSet) % 2 == 1 {
						// create a copy of the entire Node structure
						duplicateNode := *(currentNodeSet[len(currentNodeSet) - 1])
						// add the duplicate Node address to currentNodeSet
						currentNodeSet = append(currentNodeSet, &duplicateNode)
						// add the duplicate Node to mt.Nodes master list
						mt.Nodes = append(mt.Nodes, &duplicateNode)
					}
				}
				go hashNodePair(&currentNodeSet, i, useOddNodeDuplication, performHashesTwice, &wg, &c)
				newNodeSet = append(newNodeSet, <- c)
			}
			// Wait for all tasks to complete
			wg.Wait()

			// add the new parent nodes to mt.Nodes
			mt.Nodes = append(mt.Nodes, newNodeSet...)
		}

		// This is the lone reminaing parent Node,
		// it is the Root of the entire tree.
		newNodeSet[0].IsRoot = true
		mt.Root = newNodeSet[0].Hash

	}
}

func hashNodePair(nodes *[]*Node, index int, useOddNodeDuplication bool, performHashesTwice bool, wg *sync.WaitGroup, c *chan *Node) {
	// Always call Done() before exiting
	defer wg.Done()
	// Initialize the hash pair nodes we will be working with
	var hashPair []*Node
	hashPair = append(hashPair, (*nodes)[index])
	if index + 1 < len(*nodes) {
		hashPair = append(hashPair, (*nodes)[index + 1])
	}
	var newParentNode Node
	if len(hashPair) > 1 {
		// If we have a pair, generate the parent Node for this pair
		// make sibling connection
		hashPair[0].IsLeftNode = true
		hashPair[0].Sibling = hashPair[1]
		hashPair[1].Sibling = hashPair[0]
		// concat hashes values
		leftHash := hashPair[0].Hash[:]
		rightHash := hashPair[1].Hash[:]
		// calculate the hash, respecting the optional performHashesTwice setting
		hash := sha256.Sum256(append(leftHash, rightHash...))
		if performHashesTwice {
			hash = sha256.Sum256(hash[:])
		}
		// create new parent node
		newParentNode = Node{nil, nil, false, false, hash[:]}
		// make parent links
		hashPair[0].Parent = &newParentNode
		hashPair[1].Parent = &newParentNode
	} else {
		// We have a single odd node, make a copy as it's parent
		// create new parent node
		newParentNode = Node{nil, nil, false, false, hashPair[0].Hash}
		// make parent links
		hashPair[0].Parent = &newParentNode
	}
	// Send a pointer to the new parent Node back through channel c
	*c <- &newParentNode
}

func verifyProof(proofSteps []ProofStep, targetHash []byte, merkleRoot []byte, performHashesTwice bool) bool {
	if len(proofSteps) == 0 {
		return bytes.Equal(targetHash, merkleRoot)
	}
	currentValue := targetHash
	for _, proofStep := range proofSteps {
		var leftHash, rightHash []byte
		if proofStep.Left {
			leftHash = proofStep.Value
			rightHash = currentValue
		} else {
			leftHash = currentValue
			rightHash = proofStep.Value
		}
		concatHashes := append(leftHash, rightHash...)
		// calculate the hash, respecting the optional performHashesTwice setting
		newHash := sha256.Sum256(concatHashes)
		if performHashesTwice {
			newHash = sha256.Sum256(newHash[:])
		}
		currentValue = newHash[:]
	}
	return bytes.Equal(currentValue, merkleRoot)
}
