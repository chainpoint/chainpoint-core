//Copyright (c) 2016, The Gocql authors
//All rights reserved.
//
//Redistribution and use in source and binary forms, with or without
//modification, are permitted provided that the following conditions are met:
//
//* Redistributions of source code must retain the above copyright notice, this
//list of conditions and the following disclaimer.
//
//* Redistributions in binary form must reproduce the above copyright notice,
//this list of conditions and the following disclaimer in the documentation
//and/or other materials provided with the distribution.
//
//* Neither the name of the copyright holder nor the names of its
//contributors may be used to endorse or promote products derived from
//this software without specific prior written permission.
//
//THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
//AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
//IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
//DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
//FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
//DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
//SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
//CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
//OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
//OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package uuid

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

type UUID [16]byte

var hardwareAddr []byte
var clockSeq uint32

const (
	VariantNCSCompat = 0
	VariantIETF      = 2
	VariantMicrosoft = 6
	VariantFuture    = 7
)

func init() {
	if interfaces, err := net.Interfaces(); err == nil {
		for _, i := range interfaces {
			if i.Flags&net.FlagLoopback == 0 && len(i.HardwareAddr) > 0 {
				hardwareAddr = i.HardwareAddr
				break
			}
		}
	}
	if hardwareAddr == nil {
		// If we failed to obtain the MAC address of the current computer,
		// we will use a randomly generated 6 byte sequence instead and set
		// the multicast bit as recommended in RFC 4122.
		hardwareAddr = make([]byte, 6)
		_, err := io.ReadFull(rand.Reader, hardwareAddr)
		if err != nil {
			panic(err)
		}
		hardwareAddr[0] = hardwareAddr[0] | 0x01
	}

	// initialize the clock sequence with a random number
	var clockSeqRand [2]byte
	io.ReadFull(rand.Reader, clockSeqRand[:])
	clockSeq = uint32(clockSeqRand[1])<<8 | uint32(clockSeqRand[0])
}

// ParseUUID parses a 32 digit hexadecimal number (that might contain hypens)
// representing an UUID.
func ParseUUID(input string) (UUID, error) {
	var u UUID
	j := 0
	for _, r := range input {
		switch {
		case r == '-' && j&1 == 0:
			continue
		case r >= '0' && r <= '9' && j < 32:
			u[j/2] |= byte(r-'0') << uint(4-j&1*4)
		case r >= 'a' && r <= 'f' && j < 32:
			u[j/2] |= byte(r-'a'+10) << uint(4-j&1*4)
		case r >= 'A' && r <= 'F' && j < 32:
			u[j/2] |= byte(r-'A'+10) << uint(4-j&1*4)
		default:
			return UUID{}, fmt.Errorf("invalid UUID %q", input)
		}
		j += 1
	}
	if j != 32 {
		return UUID{}, fmt.Errorf("invalid UUID %q", input)
	}
	return u, nil
}

// UUIDFromBytes converts a raw byte slice to an UUID.
func UUIDFromBytes(input []byte) (UUID, error) {
	var u UUID
	if len(input) != 16 {
		return u, errors.New("UUIDs must be exactly 16 bytes long")
	}

	copy(u[:], input)
	return u, nil
}

// RandomUUID generates a totally random UUID (version 4) as described in
// RFC 4122.
func RandomUUID() (UUID, error) {
	var u UUID
	_, err := io.ReadFull(rand.Reader, u[:])
	if err != nil {
		return u, err
	}
	u[6] &= 0x0F // clear version
	u[6] |= 0x40 // set version to 4 (random uuid)
	u[8] &= 0x3F // clear variant
	u[8] |= 0x80 // set to IETF variant
	return u, nil
}

var timeBase = time.Date(1582, time.October, 15, 0, 0, 0, 0, time.UTC).Unix()

// getTimestamp converts time to UUID (version 1) timestamp.
// It must be an interval of 100-nanoseconds since timeBase.
func getTimestamp(t time.Time) int64 {
	utcTime := t.In(time.UTC)
	ts := int64(utcTime.Unix()-timeBase)*10000000 + int64(utcTime.Nanosecond()/100)

	return ts
}

// TimeUUID generates a new time based UUID (version 1) using the current
// time as the timestamp.
func TimeUUID() UUID {
	return UUIDFromTime(time.Now())
}

// The min and max clock values for a UUID.
//
// Cassandra's TimeUUIDType compares the lsb parts as signed byte arrays.
// Thus, the min value for each byte is -128 and the max is +127.
const (
	minClock = 0x8080
	maxClock = 0x7f7f
)

// The min and max node values for a UUID.
//
// See explanation about Cassandra's TimeUUIDType comparison logic above.
var (
	minNode = []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	maxNode = []byte{0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f}
)

// MinTimeUUID generates a "fake" time based UUID (version 1) which will be
// the smallest possible UUID generated for the provided timestamp.
//
// UUIDs generated by this function are not unique and are mostly suitable only
// in queries to select a time range of a Cassandra's TimeUUID column.
func MinTimeUUID(t time.Time) UUID {
	return TimeUUIDWith(getTimestamp(t), minClock, minNode)
}

// MaxTimeUUID generates a "fake" time based UUID (version 1) which will be
// the biggest possible UUID generated for the provided timestamp.
//
// UUIDs generated by this function are not unique and are mostly suitable only
// in queries to select a time range of a Cassandra's TimeUUID column.
func MaxTimeUUID(t time.Time) UUID {
	return TimeUUIDWith(getTimestamp(t), maxClock, maxNode)
}

// UUIDFromTime generates a new time based UUID (version 1) as described in
// RFC 4122. This UUID contains the MAC address of the node that generated
// the UUID, the given timestamp and a sequence number.
func UUIDFromTime(t time.Time) UUID {
	ts := getTimestamp(t)
	clock := atomic.AddUint32(&clockSeq, 1)

	return TimeUUIDWith(ts, clock, hardwareAddr)
}

// UUIDFromTimeNode generates a new time based UUID (version 1) based on a
// user-submitted node and time struct
func UUIDFromTimeNode(t time.Time, node []byte) UUID {
	ts := getTimestamp(t)
	clock := atomic.AddUint32(&clockSeq, 1)
	return TimeUUIDWith(ts, clock, node)
}

// TimeUUIDWith generates a new time based UUID (version 1) as described in
// RFC4122 with given parameters. t is the number of 100's of nanoseconds
// since 15 Oct 1582 (60bits). clock is the number of clock sequence (14bits).
// node is a slice to gurarantee the uniqueness of the UUID (up to 6bytes).
// Note: calling this function does not increment the static clock sequence.
func TimeUUIDWith(t int64, clock uint32, node []byte) UUID {
	var u UUID

	u[0], u[1], u[2], u[3] = byte(t>>24), byte(t>>16), byte(t>>8), byte(t)
	u[4], u[5] = byte(t>>40), byte(t>>32)
	u[6], u[7] = byte(t>>56)&0x0F, byte(t>>48)

	u[8] = byte(clock >> 8)
	u[9] = byte(clock)

	copy(u[10:], node)

	u[6] |= 0x10 // set version to 1 (time based uuid)
	u[8] &= 0x3F // clear variant
	u[8] |= 0x80 // set to IETF variant

	return u
}

// String returns the UUID in it's canonical form, a 32 digit hexadecimal
// number in the form of xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (u UUID) String() string {
	var offsets = [...]int{0, 2, 4, 6, 9, 11, 14, 16, 19, 21, 24, 26, 28, 30, 32, 34}
	const hexString = "0123456789abcdef"
	r := make([]byte, 36)
	for i, b := range u {
		r[offsets[i]] = hexString[b>>4]
		r[offsets[i]+1] = hexString[b&0xF]
	}
	r[8] = '-'
	r[13] = '-'
	r[18] = '-'
	r[23] = '-'
	return string(r)

}

// Bytes returns the raw byte slice for this UUID. A UUID is always 128 bits
// (16 bytes) long.
func (u UUID) Bytes() []byte {
	return u[:]
}

// Variant returns the variant of this UUID. This package will only generate
// UUIDs in the IETF variant.
func (u UUID) Variant() int {
	x := u[8]
	if x&0x80 == 0 {
		return VariantNCSCompat
	}
	if x&0x40 == 0 {
		return VariantIETF
	}
	if x&0x20 == 0 {
		return VariantMicrosoft
	}
	return VariantFuture
}

// Version extracts the version of this UUID variant. The RFC 4122 describes
// five kinds of UUIDs.
func (u UUID) Version() int {
	return int(u[6] & 0xF0 >> 4)
}

// Node extracts the MAC address of the node who generated this UUID. It will
// return nil if the UUID is not a time based UUID (version 1).
func (u UUID) Node() []byte {
	if u.Version() != 1 {
		return nil
	}
	return u[10:]
}

// Clock extracts the clock sequence of this UUID. It will return zero if the
// UUID is not a time based UUID (version 1).
func (u UUID) Clock() uint32 {
	if u.Version() != 1 {
		return 0
	}

	// Clock sequence is the lower 14bits of u[8:10]
	return uint32(u[8]&0x3F)<<8 | uint32(u[9])
}

// Timestamp extracts the timestamp information from a time based UUID
// (version 1).
func (u UUID) Timestamp() int64 {
	if u.Version() != 1 {
		return 0
	}
	return int64(uint64(u[0])<<24|uint64(u[1])<<16|
		uint64(u[2])<<8|uint64(u[3])) +
		int64(uint64(u[4])<<40|uint64(u[5])<<32) +
		int64(uint64(u[6]&0x0F)<<56|uint64(u[7])<<48)
}

// Time is like Timestamp, except that it returns a time.Time.
func (u UUID) Time() time.Time {
	if u.Version() != 1 {
		return time.Time{}
	}
	t := u.Timestamp()
	sec := t / 1e7
	nsec := (t % 1e7) * 100
	return time.Unix(sec+timeBase, nsec).UTC()
}

// Marshaling for JSON
func (u UUID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + u.String() + `"`), nil
}

// Unmarshaling for JSON
func (u *UUID) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), `"`)
	if len(str) > 36 {
		return fmt.Errorf("invalid JSON UUID %s", str)
	}

	parsed, err := ParseUUID(str)
	if err == nil {
		copy(u[:], parsed[:])
	}

	return err
}

func (u UUID) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *UUID) UnmarshalText(text []byte) (err error) {
	*u, err = ParseUUID(string(text))
	return
}
