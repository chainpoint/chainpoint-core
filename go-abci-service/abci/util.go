package abci

import (
	"encoding/binary"
	"strconv"
)

func int64ToByte(num int64) []byte {
	return []byte(strconv.FormatInt(num, binary.MaxVarintLen64))
}

func byteToInt64(arr string) int64 {
	n, _ := strconv.ParseInt(arr, 10, 64)
	return n
}
