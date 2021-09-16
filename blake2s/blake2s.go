//MIT License
//
//Copyright (c) 2019 QED-it Systems Ltd.
//
//Permission is hereby granted, free of charge, to any person obtaining a copy
//of this software and associated documentation files (the "Software"), to deal
//in the Software without restriction, including without limitation the rights
//to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//copies of the Software, and to permit persons to whom the Software is
//furnished to do so, subject to the following conditions:
//
//The above copyright notice and this permission notice shall be included in all
//copies or substantial portions of the Software.
//
//THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//SOFTWARE.

package blake2s

import (
	"encoding/binary"
	"errors"
	"hash"
)

var (
	useSSE4    = false
	useSSSE3   = false
	useSSE2    = false
	useGeneric = true
)

func hashBlocks(h *[8]uint32, c *[2]uint32, flag uint32, blocks []byte) {
	hashBlocksGeneric(h, c, flag, blocks)
}

const (
	// The blocksize of BLAKE2s in bytes.
	BlockSize = 64
	// The hash size of BLAKE2s-256 in bytes.
	Size = 32
)

var errKeySize = errors.New("blake2s: invalid key size")

var iv = [8]uint32{
	0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
	0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
}

// Sum256 returns the BLAKE2s-256 checksum of the data.
func Sum256(data []byte) [Size]byte {
	var sum [Size]byte
	checkSum(&sum, Size, data)
	return sum
}

// New256 returns a new hash.Hash computing the BLAKE2s-256 checksum. A non-nil
// key turns the hash into a MAC. The key must between zero and 32 bytes long.
func New256(key []byte) (hash.Hash, error) { return newDigest(Size, key, nil) }
func New256WithPersonalization(key, personalization []byte) (hash.Hash, error) {
	return newDigest(Size, key, personalization)
}

func newDigest(hashSize int, key []byte, personalization []byte) (*digest, error) {
	if len(key) > Size {
		return nil, errKeySize
	}
	d := &digest{
		size:            hashSize,
		keyLen:          len(key),
		personalization: personalization,
	}
	copy(d.key[:], key)
	d.Reset()
	return d, nil
}

func checkSum(sum *[Size]byte, hashSize int, data []byte) {
	var (
		h [8]uint32
		c [2]uint32
	)

	h = iv
	h[0] ^= uint32(hashSize) | (1 << 16) | (1 << 24)

	if length := len(data); length > BlockSize {
		n := length &^ (BlockSize - 1)
		if length == n {
			n -= BlockSize
		}
		hashBlocks(&h, &c, 0, data[:n])
		data = data[n:]
	}

	var block [BlockSize]byte
	offset := copy(block[:], data)
	remaining := uint32(BlockSize - offset)

	if c[0] < remaining {
		c[1]--
	}
	c[0] -= remaining

	hashBlocks(&h, &c, 0xFFFFFFFF, block[:])

	for i, v := range h {
		binary.LittleEndian.PutUint32(sum[4*i:], v)
	}
}

type digest struct {
	h      [8]uint32
	c      [2]uint32
	size   int
	block  [BlockSize]byte
	offset int

	key    [BlockSize]byte
	keyLen int

	personalization []byte
}

func (d *digest) BlockSize() int { return BlockSize }

func (d *digest) Size() int { return d.size }

func (d *digest) Reset() {
	d.h = iv
	d.h[0] ^= uint32(d.size) | (uint32(d.keyLen) << 8) | (1 << 16) | (1 << 24)
	if d.personalization != nil {
		for i := uint(0); i < 8; i++ {
			b := uint32(d.personalization[i]) << (8 * uint(i%4))
			d.h[6+i/4] ^= b
		}

	}
	d.offset, d.c[0], d.c[1] = 0, 0, 0
	if d.keyLen > 0 {
		d.block = d.key
		d.offset = BlockSize
	}
}

func (d *digest) Write(p []byte) (n int, err error) {
	n = len(p)

	if d.offset > 0 {
		remaining := BlockSize - d.offset
		if n <= remaining {
			d.offset += copy(d.block[d.offset:], p)
			return
		}
		copy(d.block[d.offset:], p[:remaining])
		hashBlocks(&d.h, &d.c, 0, d.block[:])
		d.offset = 0
		p = p[remaining:]
	}

	if length := len(p); length > BlockSize {
		nn := length &^ (BlockSize - 1)
		if length == nn {
			nn -= BlockSize
		}
		hashBlocks(&d.h, &d.c, 0, p[:nn])
		p = p[nn:]
	}

	d.offset += copy(d.block[:], p)
	return
}

func (d *digest) Sum(b []byte) []byte {
	var block [BlockSize]byte
	h := d.h
	c := d.c

	copy(block[:], d.block[:d.offset])
	remaining := uint32(BlockSize - d.offset)
	if c[0] < remaining {
		c[1]--
	}
	c[0] -= remaining

	hashBlocks(&h, &c, 0xFFFFFFFF, block[:])

	var sum [Size]byte
	for i, v := range h {
		binary.LittleEndian.PutUint32(sum[4*i:], v)
	}

	return append(b, sum[:d.size]...)
}

// the precomputed values for BLAKE2s
// there are 10 16-byte arrays - one for each round
// the entries are calculated from the sigma constants.
var precomputed = [10][16]byte{
	{0, 2, 4, 6, 1, 3, 5, 7, 8, 10, 12, 14, 9, 11, 13, 15},
	{14, 4, 9, 13, 10, 8, 15, 6, 1, 0, 11, 5, 12, 2, 7, 3},
	{11, 12, 5, 15, 8, 0, 2, 13, 10, 3, 7, 9, 14, 6, 1, 4},
	{7, 3, 13, 11, 9, 1, 12, 14, 2, 5, 4, 15, 6, 10, 0, 8},
	{9, 5, 2, 10, 0, 7, 4, 15, 14, 11, 6, 3, 1, 12, 8, 13},
	{2, 6, 0, 8, 12, 10, 11, 3, 4, 7, 15, 1, 13, 5, 14, 9},
	{12, 1, 14, 4, 5, 15, 13, 10, 0, 6, 9, 8, 7, 3, 2, 11},
	{13, 7, 12, 3, 11, 14, 1, 9, 5, 15, 8, 2, 0, 4, 6, 10},
	{6, 14, 11, 0, 15, 9, 3, 8, 12, 13, 1, 10, 2, 7, 4, 5},
	{10, 8, 7, 1, 2, 4, 6, 5, 15, 9, 3, 13, 11, 14, 12, 0},
}

func hashBlocksGeneric(h *[8]uint32, c *[2]uint32, flag uint32, blocks []byte) {
	var m [16]uint32
	c0, c1 := c[0], c[1]

	for i := 0; i < len(blocks); {
		c0 += BlockSize
		if c0 < BlockSize {
			c1++
		}

		v0, v1, v2, v3, v4, v5, v6, v7 := h[0], h[1], h[2], h[3], h[4], h[5], h[6], h[7]
		v8, v9, v10, v11, v12, v13, v14, v15 := iv[0], iv[1], iv[2], iv[3], iv[4], iv[5], iv[6], iv[7]
		v12 ^= c0
		v13 ^= c1
		v14 ^= flag

		for j := range m {
			m[j] = uint32(blocks[i]) | uint32(blocks[i+1])<<8 | uint32(blocks[i+2])<<16 | uint32(blocks[i+3])<<24
			i += 4
		}

		for k := range precomputed {
			s := &(precomputed[k])

			v0 += m[s[0]]
			v0 += v4
			v12 ^= v0
			v12 = v12<<(32-16) | v12>>16
			v8 += v12
			v4 ^= v8
			v4 = v4<<(32-12) | v4>>12
			v1 += m[s[1]]
			v1 += v5
			v13 ^= v1
			v13 = v13<<(32-16) | v13>>16
			v9 += v13
			v5 ^= v9
			v5 = v5<<(32-12) | v5>>12
			v2 += m[s[2]]
			v2 += v6
			v14 ^= v2
			v14 = v14<<(32-16) | v14>>16
			v10 += v14
			v6 ^= v10
			v6 = v6<<(32-12) | v6>>12
			v3 += m[s[3]]
			v3 += v7
			v15 ^= v3
			v15 = v15<<(32-16) | v15>>16
			v11 += v15
			v7 ^= v11
			v7 = v7<<(32-12) | v7>>12

			v0 += m[s[4]]
			v0 += v4
			v12 ^= v0
			v12 = v12<<(32-8) | v12>>8
			v8 += v12
			v4 ^= v8
			v4 = v4<<(32-7) | v4>>7
			v1 += m[s[5]]
			v1 += v5
			v13 ^= v1
			v13 = v13<<(32-8) | v13>>8
			v9 += v13
			v5 ^= v9
			v5 = v5<<(32-7) | v5>>7
			v2 += m[s[6]]
			v2 += v6
			v14 ^= v2
			v14 = v14<<(32-8) | v14>>8
			v10 += v14
			v6 ^= v10
			v6 = v6<<(32-7) | v6>>7
			v3 += m[s[7]]
			v3 += v7
			v15 ^= v3
			v15 = v15<<(32-8) | v15>>8
			v11 += v15
			v7 ^= v11
			v7 = v7<<(32-7) | v7>>7

			v0 += m[s[8]]
			v0 += v5
			v15 ^= v0
			v15 = v15<<(32-16) | v15>>16
			v10 += v15
			v5 ^= v10
			v5 = v5<<(32-12) | v5>>12
			v1 += m[s[9]]
			v1 += v6
			v12 ^= v1
			v12 = v12<<(32-16) | v12>>16
			v11 += v12
			v6 ^= v11
			v6 = v6<<(32-12) | v6>>12
			v2 += m[s[10]]
			v2 += v7
			v13 ^= v2
			v13 = v13<<(32-16) | v13>>16
			v8 += v13
			v7 ^= v8
			v7 = v7<<(32-12) | v7>>12
			v3 += m[s[11]]
			v3 += v4
			v14 ^= v3
			v14 = v14<<(32-16) | v14>>16
			v9 += v14
			v4 ^= v9
			v4 = v4<<(32-12) | v4>>12

			v0 += m[s[12]]
			v0 += v5
			v15 ^= v0
			v15 = v15<<(32-8) | v15>>8
			v10 += v15
			v5 ^= v10
			v5 = v5<<(32-7) | v5>>7
			v1 += m[s[13]]
			v1 += v6
			v12 ^= v1
			v12 = v12<<(32-8) | v12>>8
			v11 += v12
			v6 ^= v11
			v6 = v6<<(32-7) | v6>>7
			v2 += m[s[14]]
			v2 += v7
			v13 ^= v2
			v13 = v13<<(32-8) | v13>>8
			v8 += v13
			v7 ^= v8
			v7 = v7<<(32-7) | v7>>7
			v3 += m[s[15]]
			v3 += v4
			v14 ^= v3
			v14 = v14<<(32-8) | v14>>8
			v9 += v14
			v4 ^= v9
			v4 = v4<<(32-7) | v4>>7
		}

		h[0] ^= v0 ^ v8
		h[1] ^= v1 ^ v9
		h[2] ^= v2 ^ v10
		h[3] ^= v3 ^ v11
		h[4] ^= v4 ^ v12
		h[5] ^= v5 ^ v13
		h[6] ^= v6 ^ v14
		h[7] ^= v7 ^ v15
	}
	c[0], c[1] = c0, c1
}
