/* Copyright 2018 Tierion
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
	"encoding/hex"
	"testing"
)

var hLeft, _ = hex.DecodeString("a292780cc748697cb499fdcc8cb89d835609f11e502281dfe3f6690b1cc23dcb")
var hRight, _ = hex.DecodeString("cb4990b9a8936bbc137ddeb6dcab4620897b099a450ecdc5f3e86ef4b3a7135c")
var mRoot = sha256.Sum256(append(hLeft, hRight...))

func TestEmptyTree(t *testing.T) {
	mt := MerkleTree{}
	mt.MakeTree()
	root := mt.GetMerkleRoot()
	if root != nil {
		t.Errorf("merkle root value should be null, got: %d, want: nil.", root)
	}
}

func TestAddLeaves(t *testing.T) {
	mt := MerkleTree{}
	leaves := [][]byte{hLeft, hRight}
	mt.AddLeaves(leaves)
	mt.MakeTree()
	root := mt.GetMerkleRoot()
	if !bytes.Equal(mt.GetMerkleRoot(), mRoot[:]) {
		t.Errorf("merkle root value should be correct, got: %x, want: %x.", root, mRoot)
	}
}

func TestGetLeaf(t *testing.T) {
	mt := MerkleTree{}
	leaves := [][]byte{hLeft, hRight}
	mt.AddLeaves(leaves)
	mt.MakeTree()
	leaf := mt.GetLeaf(0)
	if !bytes.Equal(leaf.Hash, hLeft) {
		t.Errorf("returned leaf should be in right spot, got: %x, want: %x.", leaf.Hash, hLeft)
	}
}
func TestGetLeafBadIndex(t *testing.T) {
	mt := MerkleTree{}
	leaves := [][]byte{hLeft, hRight}
	mt.AddLeaves(leaves)
	mt.MakeTree()
	leaf := mt.GetLeaf(-1)
	if leaf != nil {
		t.Errorf("out of bounds leaf position returns null, got: %x, want: nil.", leaf.Hash)
	}
	leaf = mt.GetLeaf(5)
	if leaf != nil {
		t.Errorf("out of bounds leaf position returns null, got: %x, want: nil.", leaf.Hash)
	}
}

func TestGetLeafnoMake(t *testing.T) {
	mt := MerkleTree{}
	leaves := [][]byte{hLeft, hRight}
	mt.AddLeaves(leaves)

	leaf := mt.GetLeaf(0)
	if !bytes.Equal(leaf.Hash, hLeft) {
		t.Errorf("returned leaf should be in right spot, got: %x, want: %x.", leaf.Hash, hLeft)
	}
}

func TestGetLeafBadIndexNoMake(t *testing.T) {
	mt := MerkleTree{}
	leaves := [][]byte{hLeft, hRight}
	mt.AddLeaves(leaves)

	leaf := mt.GetLeaf(-1)
	if leaf != nil {
		t.Errorf("out of bounds leaf position returns null, got: %x, want: nil.", leaf.Hash)
	}
	leaf = mt.GetLeaf(5)
	if leaf != nil {
		t.Errorf("out of bounds leaf position returns null, got: %x, want: nil.", leaf.Hash)
	}
}

func TestResetTree(t *testing.T) {
	mt := MerkleTree{}
	leaves := [][]byte{hLeft, hRight}
	mt.AddLeaves(leaves)
	mt.MakeTree()
	mt.Reset()

	if len(mt.Leaves) > 0 {
		t.Errorf("Leaves returns null, got: %d, want: nil.", len(mt.Leaves))
	}
	if len(mt.Nodes) > 0 {
		t.Errorf("Nodes returns null, got: %d, want: nil.", len(mt.Nodes))
	}
	if len(mt.Root) > 0 {
		t.Errorf("Root returns null, got: %x, want: nil.", mt.Root)
	}
}

func TestAddLeaf(t *testing.T) {
	mt := MerkleTree{}
	mt.AddLeaf(hLeft)
	mt.AddLeaf(hRight)
	mt.MakeTree()
	root := mt.GetMerkleRoot()
	if !bytes.Equal(mt.GetMerkleRoot(), mRoot[:]) {
		t.Errorf("merkle root value should be correct, got: %x, want: %x.", root, mRoot)
	}
}

func TestAddLeafProoflength(t *testing.T) {
	mt := MerkleTree{}
	mt.AddLeaf(hLeft)
	mt.AddLeaf(hRight)
	mt.MakeTree()
	p1 := mt.GetProof(0)
	p2 := mt.GetProof(1)

	if len(p1) != 1 {
		t.Errorf("bad proof length, got: %d, want: %d.", len(p1), 1)
	}
	if len(p2) != 1 {
		t.Errorf("bad proof length, got: %d, want: %d.", len(p1), 1)
	}
}

func TestOneLeaf(t *testing.T) {
	mt := MerkleTree{}
	mt.AddLeaf(hLeft)
	mt.MakeTree()
	root := mt.GetMerkleRoot()
	if !bytes.Equal(root, hLeft) {
		t.Errorf("merkle root value should be correct, got: %x, want: %x.", root, hLeft)
	}
}

func TestFiveLeaves(t *testing.T) {
	mt := MerkleTree{}
	var h1, _ = hex.DecodeString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb")
	var h2, _ = hex.DecodeString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d")
	var h3, _ = hex.DecodeString("2e7d2c03a9507ae265ecf5b5356885a53393a2029d241394997265a1a25aefc6")
	var h4, _ = hex.DecodeString("18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4")
	var h5, _ = hex.DecodeString("3f79bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea")
	var expRoot, _ = hex.DecodeString("d71f8983ad4ee170f8129f1ebcdd7440be7798d8e1c80420bf11f1eced610dba")
	mt.AddLeaf(h1)
	mt.AddLeaf(h2)
	mt.AddLeaf(h3)
	mt.AddLeaf(h4)
	mt.AddLeaf(h5)
	mt.MakeTree()
	root := mt.GetMerkleRoot()
	if !bytes.Equal(root, expRoot) {
		t.Errorf("merkle root value should be correct, got: %x, want: %x.", root, expRoot)
	}
}

func TestProofLeft(t *testing.T) {
	mt := MerkleTree{}
	mt.AddLeaf(hLeft)
	mt.AddLeaf(hRight)
	mt.MakeTree()
	proof := mt.GetProof(0)
	if !bytes.Equal(proof[0].Value, hRight) {
		t.Errorf("proof array should be correct, got: %x, want: %x.", proof[0].Value, hRight)
	}
}

func TestProofRight(t *testing.T) {
	mt := MerkleTree{}
	mt.AddLeaf(hLeft)
	mt.AddLeaf(hRight)
	mt.MakeTree()
	proof := mt.GetProof(1)
	if !bytes.Equal(proof[0].Value, hLeft) {
		t.Errorf("proof array should be correct, got: %x, want: %x.", proof[0].Value, hLeft)
	}
}

func TestProofOneNode(t *testing.T) {
	mt := MerkleTree{}
	mt.AddLeaf(hLeft)
	mt.MakeTree()
	proof := mt.GetProof(0)
	if len(proof) != 0 {
		t.Errorf("proof should be [], got: %x, want: %x.", len(proof), 0)
	}
}

func TestGoodProof2Leaves(t *testing.T) {
	mt := MerkleTree{}
	mt.AddLeaf(hLeft)
	mt.AddLeaf(hRight)
	mt.MakeTree()
	proof := mt.GetProof(1)
	isValid := VerifyProof(proof, hRight, mt.GetMerkleRoot())
	if !isValid {
		t.Errorf("proof should be valid, got: %t, want: %t.", isValid, true)
	}
}

func TestBadProof2Leaves(t *testing.T) {
	mt := MerkleTree{}
	mt.AddLeaf(hLeft)
	mt.AddLeaf(hRight)
	mt.MakeTree()
	proof := mt.GetProof(1)
	var badRoot, _ = hex.DecodeString("3f79bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea")
	isValid := VerifyProof(proof, hRight, badRoot)
	if isValid {
		t.Errorf("proof should be invalid, got: %t, want: %t.", isValid, true)
	}
}

func TestGoodProof5Leaves(t *testing.T) {
	mt := MerkleTree{}
	var h1, _ = hex.DecodeString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb")
	var h2, _ = hex.DecodeString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d")
	var h3, _ = hex.DecodeString("2e7d2c03a9507ae265ecf5b5356885a53393a2029d241394997265a1a25aefc6")
	var h4, _ = hex.DecodeString("18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4")
	var h5, _ = hex.DecodeString("3f79bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea")
	var expRoot, _ = hex.DecodeString("d71f8983ad4ee170f8129f1ebcdd7440be7798d8e1c80420bf11f1eced610dba")
	mt.AddLeaf(h1)
	mt.AddLeaf(h2)
	mt.AddLeaf(h3)
	mt.AddLeaf(h4)
	mt.AddLeaf(h5)
	mt.MakeTree()
	proof := mt.GetProof(1)
	isValid := VerifyProof(proof, h2, expRoot)
	if !isValid {
		t.Errorf("proof should be valid, got: %t, want: %t.", isValid, true)
	}
}

func TestBadProof5Leaves(t *testing.T) {
	mt := MerkleTree{}
	var h1, _ = hex.DecodeString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb")
	var h2, _ = hex.DecodeString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d")
	var h3, _ = hex.DecodeString("2e7d2c03a9507ae265ecf5b5356885a53393a2029d241394997265a1a25aefc6")
	var h4, _ = hex.DecodeString("18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4")
	var h5, _ = hex.DecodeString("3f79bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea")
	var expRoot, _ = hex.DecodeString("a71f8983ad4ee170f8129f1ebcdd7440be7798d8e1c80420bf11f1eced610dba")
	mt.AddLeaf(h1)
	mt.AddLeaf(h2)
	mt.AddLeaf(h3)
	mt.AddLeaf(h4)
	mt.AddLeaf(h5)
	mt.MakeTree()
	proof := mt.GetProof(3)
	isValid := VerifyProof(proof, h4, expRoot)
	if isValid {
		t.Errorf("proof should be invalid, got: %t, want: %t.", isValid, true)
	}
}

func TestOddBTCThreeLeaves(t *testing.T) {
	mt := MerkleTree{}
	var h1, _ = hex.DecodeString("1a02db5db5a24c5edc5b653051d8aaaddec3f9abc30354f7df358c49fe40f735")
	var h2, _ = hex.DecodeString("d3f3eb471e368a27f5320ff7a961bed748519139435cf8348e84ebd6225d7150")
	var h3, _ = hex.DecodeString("7cbcf6b5378e3e43b39734baa578efa501d02abf90289547f0e6621ee959f0e3")
	var expRoot, _ = hex.DecodeString("7099d100635a0e5f62ef12a8420c99426a408951078f191a0f63ddedc4dcd198")
	mt.AddLeaf(reverseBytes(h1))
	mt.AddLeaf(reverseBytes(h2))
	mt.AddLeaf(reverseBytes(h3))
	mt.MakeBTCTree()
	root := mt.GetMerkleRoot()
	if !bytes.Equal(reverseBytes(root), expRoot) {
		t.Errorf("merkle root value should be correct, got: %x, want: %x.", root, expRoot)
	}
}

func TestOddBTCOneLeaf(t *testing.T) {
	mt := MerkleTree{}
	var h1, _ = hex.DecodeString("9c397f783042029888ec02f0a461cfa2cc8e3c7897f476e338720a2a86731c60")
	var expRoot, _ = hex.DecodeString("9c397f783042029888ec02f0a461cfa2cc8e3c7897f476e338720a2a86731c60")
	mt.AddLeaf(reverseBytes(h1))
	mt.MakeBTCTree()
	root := mt.GetMerkleRoot()
	if !bytes.Equal(reverseBytes(root), expRoot) {
		t.Errorf("merkle root value should be correct, got: %x, want: %x.", root, expRoot)
	}
}

func TestEvenBTCFourLeaves(t *testing.T) {
	mt := MerkleTree{}
	var h1, _ = hex.DecodeString("6584fd6a4d0a96e27f1f0f8549a206bc9367134064d45decd2116ca7d73e6cc4")
	var h2, _ = hex.DecodeString("7e2087abb091d059749a6bfd36840743d818de95a39975c18fc5969459eb00b2")
	var h3, _ = hex.DecodeString("d45f9b209556d52db69a900703dacd934701bb523cd2a03bf48ec658133e511a")
	var h4, _ = hex.DecodeString("5ec499041da320458cf1719d06af02fecc97d3178739f4d331c4fb84c764933d")
	var expRoot, _ = hex.DecodeString("b02c190b3a4d8a32b2f053ffd6353495fb857ad03ff600002c581a3a2232f696")
	mt.AddLeaf(reverseBytes(h1))
	mt.AddLeaf(reverseBytes(h2))
	mt.AddLeaf(reverseBytes(h3))
	mt.AddLeaf(reverseBytes(h4))
	mt.MakeBTCTree()
	root := mt.GetMerkleRoot()
	if !bytes.Equal(reverseBytes(root), expRoot) {
		t.Errorf("merkle root value should be correct, got: %x, want: %x.", root, expRoot)
	}
}

func TestBTCProofThreeLeaves(t *testing.T) {
	mt := MerkleTree{}
	var h1, _ = hex.DecodeString("1a02db5db5a24c5edc5b653051d8aaaddec3f9abc30354f7df358c49fe40f735")
	var h2, _ = hex.DecodeString("d3f3eb471e368a27f5320ff7a961bed748519139435cf8348e84ebd6225d7150")
	var h3, _ = hex.DecodeString("7cbcf6b5378e3e43b39734baa578efa501d02abf90289547f0e6621ee959f0e3")
	mt.AddLeaf(h1)
	mt.AddLeaf(h2)
	mt.AddLeaf(h3)
	mt.MakeBTCTree()
	proof := mt.GetProof(2)
	isValid := VerifyBTCProof(proof, h3, mt.GetMerkleRoot())
	if !isValid {
		t.Errorf("proof should be valid, got: %t, want: %t.", isValid, true)
	}
}

func TestBTCProofFiveLeaves(t *testing.T) {
	mt := MerkleTree{}
	var h1, _ = hex.DecodeString("ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb")
	var h2, _ = hex.DecodeString("3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d")
	var h3, _ = hex.DecodeString("2e7d2c03a9507ae265ecf5b5356885a53393a2029d241394997265a1a25aefc6")
	var h4, _ = hex.DecodeString("18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4")
	var h5, _ = hex.DecodeString("3f79bb7b435b05321651daefd374cdc681dc06faa65e374e38337b88ca046dea")
	mt.AddLeaf(h1)
	mt.AddLeaf(h2)
	mt.AddLeaf(h3)
	mt.AddLeaf(h4)
	mt.AddLeaf(h5)
	mt.MakeBTCTree()
	proof := mt.GetProof(4)
	isValid := VerifyBTCProof(proof, h5, mt.GetMerkleRoot())
	if !isValid {
		t.Errorf("proof should be valid, got: %t, want: %t.", isValid, true)
	}
}

func reverseBytes(bytes []byte) []byte {
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
	return bytes
}
