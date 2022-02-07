package seededelection

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"reflect"
	"sync"
)

var randSourceLock sync.Mutex

func ElectLeaders(slice interface{}, numLeaders int, seed string) interface{} {
	peers := reflect.ValueOf(slice)
	validatorLength := peers.Len()
	index := GetSeededRandInt([]byte(seed), validatorLength)    //seed the first time
	if err := RotateLeft(slice, index); err != nil { //get a wrapped-around slice of numLeader leaders
		return nil
	}
	if numLeaders <= validatorLength {
		return peers.Slice(0, numLeaders).Interface()
	} else {
		return peers.Slice(0,1).Interface()
	}
}

// GetSeededRandInt : Given a seed and a maximum size, generates a random int between 0 and upperBound
func GetSeededRandInt(seedBytes []byte, upperBound int) int {
	eightByteHash := seedBytes[0:7]
	seed, _ := binary.Varint(eightByteHash)
	randSourceLock.Lock()
	rand.Seed(seed)
	index := rand.Intn(upperBound)
	randSourceLock.Unlock()
	return index
}

// GetSeededRandFloat : Given a seed return a random float
func GetSeededRandFloat(seedBytes []byte) float32 {
	eightByteHash := seedBytes[0:7]
	seed, _ := binary.Varint(eightByteHash)
	randSourceLock.Lock()
	rand.Seed(seed)
	index := rand.Float32()
	randSourceLock.Unlock()
	return index
}

// Rotate rotates the values of a slice by rotate positions, preserving the
// rotated values by wrapping them around the slice
func rotate(slice interface{}, rotate int) error {
	// check slice is not nil
	if slice == nil {
		return errors.New("non-nil slice interface required")
	}
	// get the slice value
	sliceV := reflect.ValueOf(slice)
	if sliceV.Kind() != reflect.Slice {
		return errors.New("slice kind required")
	}
	// check slice value is not nil
	if sliceV.IsNil() {
		return errors.New("non-nil slice value required")
	}

	// slice length
	sLen := sliceV.Len()
	// shortcut when empty slice
	if sLen == 0 {
		return nil
	}

	// limit rotates via modulo
	rotate %= sLen
	// Go's % operator returns the remainder and thus can return negative
	// values. More detail can be found under the `Integer operators` section
	// here: https://golang.org/ref/spec#Arithmetic_operators
	if rotate < 0 {
		rotate += sLen
	}
	// shortcut when shift == 0
	if rotate == 0 {
		return nil
	}

	// get gcd to determine number of juggles
	gcd := gcd(rotate, sLen)

	// do the shifting
	for i := 0; i < gcd; i++ {
		// remember the first value
		temp := reflect.ValueOf(sliceV.Index(i).Interface())
		j := i

		for {
			k := j + rotate
			// wrap around slice
			if k >= sLen {
				k -= sLen
			}
			// end when we're back to where we started
			if k == i {
				break
			}
			// slice[j] = slice[k]
			sliceV.Index(j).Set(sliceV.Index(k))
			j = k
		}
		// slice[j] = slice
		sliceV.Index(j).Set(temp)
		// elemJ.Append(temp)
	}
	// success
	return nil
}

func gcd(x, y int) int {
	for y != 0 {
		x, y = y, x%y
	}
	return x
}

// RotateRight is the analogue to RotateLeft
func RotateRight(slice interface{}, by int) error {
	return rotate(slice, -by)
}

// RotateLeft is an alias for Rotate
func RotateLeft(slice interface{}, by int) error {
	return rotate(slice, by)
}
