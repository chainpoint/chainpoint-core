package util

import (
	"fmt"
	"os"
	"testing"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"

	"github.com/stretchr/testify/assert"
)

func TestInt64ToByte(t *testing.T) {
	bytes := Int64ToByte(9223372036854775806)
	assert.Equal(t,
		bytes,
		[]byte{0x39, 0x32, 0x32, 0x33, 0x33, 0x37, 0x32, 0x30, 0x33, 0x36, 0x38, 0x35, 0x34, 0x37, 0x37, 0x35, 0x38, 0x30, 0x36},
		"Byte conversion of Int64 255 should be "+
			"[]byte{0x39, 0x32, 0x32, 0x33, 0x33, 0x37, 0x32, 0x30, 0x33, 0x36, 0x38, 0x35, 0x34, 0x37, 0x37, 0x35, 0x38, 0x30, 0x36}")
}

func TestByteToInt64(t *testing.T) {
	bytes := Int64ToByte(9223372036854775806)
	num := ByteToInt64(string(bytes))
	assert.Equal(t, int64(9223372036854775806), num, "Byte to Int64 should render 255")

}

func TestSeededRandInt(t *testing.T) {
	seed := "3719ADA3EEE198F3A7A33616EA60ED6D72D94D31A2B2422FA12E2BCDDCABD4D4"
	output := GetSeededRandInt([]byte(seed), 4)
	assert.Equal(t, output, 3, "Seeded output should be equal to 3")
}

func TestGetSeededRandFloat(t *testing.T) {
	seed := "3719ADA3EEE198F3A7A33616EA60ED6D72D94D31A2B2422FA12E2BCDDCABD4D4"
	output := GetSeededRandFloat([]byte(seed))
	floatStr := fmt.Sprintf("%f", output)
	assert.Equal(t, floatStr, "0.546605", "Seeded random float should be 0.546605")
}

func TestGetEnv(t *testing.T) {
	assert := assert.New(t)
	envvar := GetEnv("om", "nom2")
	assert.Equal(envvar, "nom2", "GetEnv('om') output should fall through to default value, which is nom2")
	os.Setenv("om", "nom")
	envvar = GetEnv("om", "nom")
	assert.Equal(envvar, "nom", "GetEnv('om') output should fall through to default value, which is nom")
}

func TestUUIDFromHash(t *testing.T) {
	assert := assert.New(t)
	_, testerr := UUIDFromHash([]byte{})
	assert.NotEqual(testerr, nil, "Seeding UUID with nil byte array should return an error")
	uuid, _ := UUIDFromHash([]byte("abcdefghijklmnop"))
	assert.Equal(uuid.String(), "61626364-6566-6768-696a-6b6c6d6e6f70", "UUID should be 61626364-6566-6768-696a-6b6c6d6e6f70")
}

func TestEncodeTx(t *testing.T) {
	txStr := EncodeTx(types.Tx{TxType: "CAL", Data: "msg", Version: 2, Time: 0000000001})
	assert.Equal(t, txStr, "eyJ0eXBlIjoiQ0FMIiwiZGF0YSI6Im1zZyIsInZlcnNpb24iOjIsInRpbWUiOjF9", "Tx should be in base64 ")
}

func TestDecodeTx(t *testing.T) {
	assert := assert.New(t)
	tx, _ := DecodeTx([]byte("eyJ0eXBlIjoiQ0FMIiwiZGF0YSI6Im1zZyIsInZlcnNpb24iOjIsInRpbWUiOjF9"))
	assert.Equal(tx.Data, "msg", "Tx data section should be 'msg'")
	_, err := DecodeTx([]byte{})
	assert.NotEqual(err, nil, "Error from DecodeTx([]byte{}) should be non-nil")
}

func TestDecodeIP(t *testing.T) {
	ipStr := DecodeIP("AAAAAAAAAAAAAP//I7zuug==")
	assert.Equal(t, ipStr, "35.188.238.186", "DecodeIP mismatch, please check initial IP encoding")
}
