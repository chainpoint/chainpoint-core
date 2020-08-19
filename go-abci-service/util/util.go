package util

import (
	"bufio"
	"crypto/ecdsa"
	random "crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/lestrrat-go/jwx/jwk"

	"github.com/ethereum/go-ethereum/common"

	core_types "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/google/uuid"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
)

var randSourceLock sync.Mutex

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

func ReverseTxHex(str string) string {
	re := regexp.MustCompile(`(\S{2})`)
	x := re.FindAllString(str, -1)
	reverseAny(x)
	return strings.Join(x, "")
}

func reverseAny(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
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

// GetAPIStatus : get metadata from a core, given its IP
func GetAPIStatus(ip string) types.CoreAPIStatus {
	selfStatusURL := fmt.Sprintf("http://%s/status", ip)
	response, err := http.Get(selfStatusURL)
	if LogError(err) != nil {
		return types.CoreAPIStatus{}
	}
	contents, err := ioutil.ReadAll(response.Body)
	if LogError(err) != nil {
		return types.CoreAPIStatus{}
	}
	var apiStatus types.CoreAPIStatus
	err = json.Unmarshal(contents, &apiStatus)
	if LogError(err) != nil {
		return types.CoreAPIStatus{}
	}
	return apiStatus
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

// UUIDFromHash : generate a uuid from a byte hash, must be 16 bytes
func UUIDFromHash(seedBytes []byte) (uuid.UUID, error) {
	return uuid.FromBytes(seedBytes)
}

func DecodePubKey(tx types.Tx) (*ecdsa.PublicKey, error) {
	var jwkType types.Jwk
	json.Unmarshal([]byte(tx.Data), &jwkType)
	jsonJwk, err := json.Marshal(jwkType)
	if LogError(err) != nil {
		return &ecdsa.PublicKey{}, err
	}
	set, err := jwk.ParseBytes(jsonJwk)
	if LogError(err) != nil {
		return &ecdsa.PublicKey{}, err
	}
	for _, k := range set.Keys {
		pubKeyInterface, err := k.Materialize()
		if LogError(err) != nil {
			continue
		}
		pubKey := pubKeyInterface.(*ecdsa.PublicKey)
		return pubKey, err
	}
	return &ecdsa.PublicKey{}, errors.New("unable to create public key from JWK")
}

// DecodeTxAndVerifySig accepts a Chainpoint Calendar transaction in base64 and decodes it into abci.Tx struct
func DecodeTx(incoming []byte) (types.Tx, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(incoming))
	var calendar types.Tx
	if err != nil {
		fmt.Println(err)
		return types.Tx{}, err
	}
	err = json.Unmarshal([]byte(decoded), &calendar)
	return calendar, err
}

// VerifySig : verifies an ecdsa signature
func VerifySig(data string, originalSig string, key ecdsa.PublicKey) bool {
	der, err := base64.StdEncoding.DecodeString(originalSig)
	if LogError(err) != nil {
		return false
	}
	sig := &types.EcdsaSignature{}
	_, err = asn1.Unmarshal(der, sig)
	if LogError(err) != nil {
		return false
	}
	hash := sha256.Sum256([]byte(data))
	var pubKey *ecdsa.PublicKey
	pubKey = &key
	if !ecdsa.Verify(pubKey, hash[:], sig.R, sig.S) {
		LogError(errors.New("Can't validate signature of Tx"))
		return false
	}
	return true
}

// CreateSig : create signature from data in base64
func CreateSig(data string, key ecdsa.PrivateKey) string {
	hash := sha256.Sum256([]byte(data))
	sig, err := key.Sign(random.Reader, hash[:], nil)
	if LogError(err) != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(sig)
}

// DecodeTxAndVerifySig accepts a Chainpoint Calendar transaction in base64 and decodes it into abci.Tx struct
func DecodeTxAndVerifySig(incoming []byte, CoreKeys map[string]ecdsa.PublicKey) (types.Tx, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(incoming))
	var calendar types.Tx
	if err != nil {
		fmt.Println(err)
		return types.Tx{}, err
	}
	err = json.Unmarshal([]byte(decoded), &calendar)
	/* Skip sig verification if this is a TOKEN tx */
	if calendar.TxType == "TOKEN" || calendar.TxType == "JWK" {
		return calendar, nil
	}
	/* Verify Signature */
	var pubKey *ecdsa.PublicKey
	if pubKeyInterface, keyExists := CoreKeys[calendar.CoreID]; keyExists {
		pubKey = &pubKeyInterface
	} else {
		return types.Tx{}, errors.New(fmt.Sprintf("Can't find corresponding key for message from Core: %s", calendar.CoreID))
	}
	oldSig := calendar.Sig
	der, err := base64.StdEncoding.DecodeString(calendar.Sig)
	if LogError(err) != nil {
		return types.Tx{}, err
	}
	sig := &types.EcdsaSignature{}
	_, err = asn1.Unmarshal(der, sig)
	if LogError(err) != nil {
		return types.Tx{}, err
	}
	calendar.Sig = ""
	txNoSig, err := json.Marshal(calendar)
	if LogError(err) != nil {
		return types.Tx{}, err
	}
	hash := sha256.Sum256(txNoSig)
	if !ecdsa.Verify(pubKey, hash[:], sig.R, sig.S) {
		err := LogError(errors.New("Can't validate signature of Tx"))
		return types.Tx{}, err
	}
	calendar.Sig = oldSig
	return calendar, nil
}

// EncodeTxWithKey : Encodes a Tendermint transaction to base64
func EncodeTxWithKey(outgoing types.Tx, privateKey *ecdsa.PrivateKey) string {
	txNoSig, err := json.Marshal(outgoing)
	if LogError(err) != nil {
		return ""
	}
	hash := sha256.Sum256(txNoSig)
	sig, err := privateKey.Sign(random.Reader, hash[:], nil)
	if LogError(err) != nil {
		return ""
	}
	outgoing.Sig = base64.StdEncoding.EncodeToString(sig)
	txJSON, _ := json.Marshal(outgoing)
	return base64.StdEncoding.EncodeToString(txJSON)
}

func GenerateKey(privateKey *ecdsa.PrivateKey, kid string) types.Jwk {
	jwk, err := jwk.New(privateKey.Public())
	if err != nil {
		panic(err)
	}
	jwkJson, err := json.MarshalIndent(jwk, "", "  ")
	if err != nil {
		panic(err)
	}
	var jwkType types.Jwk
	err = json.Unmarshal([]byte(jwkJson), &jwkType)
	if err != nil {
		panic(err)
	}
	jwkType.Kid = kid
	return jwkType
}

//EncodeTx : encode a tx to base64
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

// DetermineIP : use remoteIP if routable, use listenAddr if not
func DetermineIP(peer core_types.Peer) string {
	remoteIP := peer.RemoteIP
	firstOctet := remoteIP[:strings.Index(remoteIP, ".")]
	if firstOctet == "10" || firstOctet == "172" || firstOctet == "192" {
		listenAddr := peer.NodeInfo.ListenAddr
		if len(listenAddr) > 0 {
			if strings.Contains(listenAddr, "//") && strings.Contains(listenAddr, ":") {
				return listenAddr[strings.LastIndex(listenAddr, "/"):strings.LastIndex(listenAddr, ":")]
			}
			return listenAddr[:strings.LastIndex(listenAddr, ":")]
		}
		return ""
	}
	return remoteIP
}

// BytesToIP : takes an IP byte array and converts it to corresponding dot string format
func BytesToIP(encoded_ip []byte) string {
	ip := strconv.Itoa(int(encoded_ip[0])) + "." +
		strconv.Itoa(int(encoded_ip[1])) + "." +
		strconv.Itoa(int(encoded_ip[2])) + "." +
		strconv.Itoa(int(encoded_ip[3]))
	return ip
}

//Ip2Int : converts IP to uint32
func Ip2Int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

//Int2Ip : converts uint32 to IP
func Int2Ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

//Contains : generic method for testing set (slice) inclusion
func Contains(s interface{}, elem interface{}) bool {
	arrV := reflect.ValueOf(s)
	if arrV.Kind() == reflect.Slice {
		for i := 0; i < arrV.Len(); i++ {
			if arrV.Index(i).Interface() == elem {
				return true
			}
		}
	}
	return false
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
		// elemJ.Set(temp)
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

//ReadContractJSON : reads in TierionNetworkToken and ChainpointRegistry json files
//and extracts addresses
func ReadContractJSON(file string, testnet bool) string {
	jsonFile, err := os.Open(file)
	if LogError(err) != nil {
		return ""
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var jsonMap map[string]interface{}
	if LogError(json.Unmarshal(byteValue, &jsonMap)) != nil {
		return ""
	}
	if testnet {
		return jsonMap["networks"].(map[string]interface{})["3"].(map[string]interface{})["address"].(string)
	}
	return jsonMap["networks"].(map[string]interface{})["1"].(map[string]interface{})["address"].(string)
}

//UniquifyAddresses: make unique array of addresses
func UniquifyAddresses(s []common.Address) []common.Address {
	seen := make(map[common.Address]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}

//UniquifyStrings : make unique array of strings
func UniquifyStrings(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}

// Copy a file safely
func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

//GetNISTTimestamp : gets Unix timestamp from record
func GetNISTTimestamp(record string) int64 {
	timeSplit := strings.Split(record, ":")
	var timeRecord int64
	if len(timeSplit) == 2 {
		timeRecord, _ = strconv.ParseInt(timeSplit[0], 10, 64)
	}
	return timeRecord
}

// ReadLines reads a whole file into memory
// and returns a slice of its lines.
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}
	return lines, scanner.Err()
}
