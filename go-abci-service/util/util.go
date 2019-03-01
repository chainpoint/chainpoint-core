package util

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
)

// LogError : Log error if it exists
func LogError(err error) error {
	if err != nil {
		fmt.Println(err)
	}
	return err
}

// Int64ToByte : Convert an int64 to a byte for use in the Tendermint tagging system
func Int64ToByte(num int64) []byte {
	return []byte(strconv.FormatInt(num, binary.MaxVarintLen64))
}

// ByteToInt64 : Convert a byte array from tendermint back into an int64
func ByteToInt64(arr string) int64 {
	n, _ := strconv.ParseInt(arr, 10, 64)
	return n
}

// GetEnv : Get an env var but with a default. Untyped, defaults to string.
func GetEnv(key string, def string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return def
	}
	return value
}

// GetSeededRandInt : Given a seed and a maximum size, generates a random int between 0 and upperBound
func GetSeededRandInt(seedBytes []byte, upperBound int) int {
	eightByteHash := seedBytes[0:7]
	seed, _ := binary.Varint(eightByteHash)
	rand.Seed(seed)
	return rand.Intn(upperBound)
}

// DecodeTx accepts a Chainpoint Calendar transaction in base64 and decodes it into abci.Tx struct
func DecodeTx(incoming []byte) (types.Tx, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(incoming))
	var calendar types.Tx
	if err != nil {
		fmt.Println(err)
		return calendar, err
	}
	json.Unmarshal([]byte(decoded), &calendar)
	return calendar, nil
}

// EncodeTx : Encodes a Tendermint transaction to base64
func EncodeTx(outgoing types.Tx) string {
	txJSON, _ := json.Marshal(outgoing)
	return base64.StdEncoding.EncodeToString(txJSON)
}

// WaitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
