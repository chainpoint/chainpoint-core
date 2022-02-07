package util

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	random "crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lestrrat-go/jwx/jwk"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	core_types "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/google/uuid"

	"github.com/chainpoint/chainpoint-core/types"
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
		logger.Error(fmt.Sprintf("Error in %s: %s", GetCurrentFuncName(2), err.Error()))
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

// GetEnv : GetArray an env var but with a default. Untyped, defaults to string.
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

func GetIPOnly(ip string) string {
	listenAddr := ip
	if strings.Contains(listenAddr, "//") {
		listenAddr = listenAddr[strings.LastIndex(listenAddr, "/")+1:]
	}
	if strings.Contains(listenAddr, ":") {
		listenAddr = listenAddr[:strings.LastIndex(listenAddr, ":")]
	}
	return listenAddr
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

func DecodeJWK(jwkType types.Jwk) (string, *ecdsa.PublicKey, error) {
	jsonJwk, err := json.Marshal(jwkType)
	if LogError(err) != nil {
		return "", &ecdsa.PublicKey{}, err
	}
	set, err := jwk.ParseBytes(jsonJwk)
	if LogError(err) != nil {
		return "", &ecdsa.PublicKey{}, err
	}
	for _, k := range set.Keys {
		pubKeyInterface, err := k.Materialize()
		if LogError(err) != nil {
			continue
		}
		pubKey := pubKeyInterface.(*ecdsa.PublicKey)
		pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
		pubKeyHex := fmt.Sprintf("Loading self pubkey as %x", pubKeyBytes)
		return pubKeyHex, pubKey, err
	}
	return "", &ecdsa.PublicKey{}, errors.New("unable to create public key from JWK")
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
	if calendar.TxType == "JWK" {
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
		err := LogError(errors.New(fmt.Sprintf("Can't validate signature of Tx from Core %s", calendar.CoreID)))
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

func NormalizeUri(uri string, removePort bool, http bool, https bool) string {
	proofUri := strings.ReplaceAll(uri, "http://", "")
	proofUri = strings.ReplaceAll(proofUri, "https://", "")
	if removePort {
		uriParts := strings.Split(proofUri, ":")
		proofUri = uriParts[0]
	}
	if http {
		proofUri = "http://" + proofUri
	} else if https {
		proofUri = "https://" + proofUri
	}
	return proofUri
}

func GetClientIP(r *http.Request) string {
	return r.RemoteAddr
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

func ValidateIPAddress(ip string) error {
	if net.ParseIP(ip) == nil {
		return errors.New("IP address invalid")
	}
	return nil
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

// GetCurrentFuncName : get name of function being called
func GetCurrentFuncName(numCallStack int) string {
	pc, _, _, _ := runtime.Caller(numCallStack)
	return fmt.Sprintf("%s", runtime.FuncForPC(pc).Name())
}

func ArrayContains(arr []string, item string) bool {
	for _, v := range arr {
		if v == item {
			return true
		}
	}
	return false
}

func ArrayContainsIndex(arr []string, item string) (bool, int) {
	for i, v := range arr {
		if v == item {
			return true, i
		}
	}
	return false, -1
}

func MaxInt64(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}
