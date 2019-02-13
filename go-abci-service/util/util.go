package util

import (
	"encoding/binary"
	"math/rand"
	"os"
	"strconv"
)

func Int64ToByte(num int64) []byte {
	return []byte(strconv.FormatInt(num, binary.MaxVarintLen64))
}

func ByteToInt64(arr string) int64 {
	n, _ := strconv.ParseInt(arr, 10, 64)
	return n
}

func GetEnv(key string, def string) string{
	value := os.Getenv(key)
	if len(value) == 0 {
		return def
	}
	return value
}

func GetSeededRandInt(seedBytes []byte, upperBound int) int {
	eightByteHash := seedBytes[0:7]
	seed, _ := binary.Varint(eightByteHash)
	rand.Seed(seed)
	return rand.Intn(upperBound)
}
