package util

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/google/uuid"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
)

// LogError : Log error if it exists
func LogError(err error) error {
	if err != nil {
		fmt.Println(err)
	}
	return err
}

// LoggerError : Log error if it exists using a logger
func LoggerError(logger log.Logger, err error) error {
	if err != nil {
		logger.Error(fmt.Sprintf("Error: %s", err.Error()))
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

// UUIDFromHash : generate a uuid from a byte hash, must be 16 bytes
func UUIDFromHash(seedBytes []byte) (uuid.UUID, error) {
	return uuid.FromBytes(seedBytes)
}

// DecodeTx accepts a Chainpoint Calendar transaction in base64 and decodes it into abci.Tx struct
func DecodeTx(incoming []byte) (types.Tx, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(incoming))
	var calendar types.Tx
	if err != nil {
		fmt.Println(err)
		return calendar, err
	}
	err = json.Unmarshal([]byte(decoded), &calendar)
	return calendar, err
}

// EncodeTx : Encodes a Tendermint transaction to base64
func EncodeTx(outgoing types.Tx) string {
	txJSON, _ := json.Marshal(outgoing)
	return base64.StdEncoding.EncodeToString(txJSON)
}

// DecodeIP: decode tendermint's arcane remote_ip format
func DecodeIP(remote_ip string) string {
	data, err := base64.StdEncoding.DecodeString(remote_ip)
	if LogError(err) != nil {
		return ""
	}
	encoded_ip := data[len(data)-4:]
	// No nice joins for individual bytes, bytes.Join only likes arrays of arrays
	ip := BytesToIP(encoded_ip)
	return ip
}

// BytesToIP : takes an IP byte array and converts it to corresponding dot string format
func BytesToIP(encoded_ip []byte) string {
	ip := strconv.Itoa(int(encoded_ip[0])) + "." +
		strconv.Itoa(int(encoded_ip[1])) + "." +
		strconv.Itoa(int(encoded_ip[2])) + "." +
		strconv.Itoa(int(encoded_ip[3]))
	return ip
}
