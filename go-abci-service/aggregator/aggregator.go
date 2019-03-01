package aggregator

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/google/uuid"

	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"
	beacon "github.com/chainpoint/go-nist-beacon"
	"github.com/streadway/amqp"
)

type Aggregation struct {
	AggId     string      `json:"agg_id"`
	AggRoot   string      `json:"agg_root"`
	ProofData []ProofData `json:"proofData"`
}

type Hash struct {
	HashID string `json:"hash_id"`
	Hash   string `json:"hash"`
	Nist   string `json:"nist"`
}

type ProofData struct {
	HashID string  `json:"hash_id"`
	Hash   string  `json:"hash"`
	Proof  []Proof `json:"proof"`
}

type Proof struct {
	Left  string `json:"l,omitempty"`
	Right string `json:"r,omitempty"`
	Op    string `json:"op,omitempty"`
}

const msgType = "aggregator"
const aggQueueIn = "work.agg"
const aggQueueOut = "work.agg"
const proofStateQueueOut = "work.proofstate"

// Aggregate retrieves hash messages from rabbitmq, stores them, and creates a proof path for each
func Aggregate(rabbitmqConnectUri string, nist beacon.Record) (agg []Aggregation) {
	var session rabbitmq.Session
	aggThreads, _ := strconv.Atoi(util.GetEnv("AGGREGATION_THREADS", "4"))
	hashBatchSize, _ := strconv.Atoi(util.GetEnv("HASHES_PER_MERKLE_TREE", "200"))
	sleep := int(60 / aggThreads)

	//Consume queue in goroutines with output slice guarded by mutex
	aggStructSlice := make([]Aggregation, 0)
	shutdown := make(chan struct{})
	var wg sync.WaitGroup
	var mux sync.Mutex
	endConsume := false

	session, err := rabbitmq.ConnectAndConsume(rabbitmqConnectUri, aggQueueIn)
	rabbitmq.LogError(err, "RabbitMQ connection failed")
	defer session.End()

	// Spin up {aggThreads} number of threads to process incoming hashes from the API
	for i := 0; i < aggThreads; i++ {

		go func() {
			msgStructSlice := make([]amqp.Delivery, 0)
			wg.Add(1)
			defer wg.Done()

			//loop consumes queue and appends to mutex protected data slice
			for !endConsume {
				select {
				//if we close the shutdown channel, we exit. Otherwise we process incoming messages
				case <-shutdown:
					endConsume = true
					break //exit
				case hash := <-session.Msgs:
					fmt.Println(string(hash.Body))
					msgStructSlice = append(msgStructSlice, hash)
					//create new agg roots under heavy load
					if len(msgStructSlice) > hashBatchSize {
						if agg := ProcessAggregation(rabbitmqConnectUri, msgStructSlice, nist.ChainpointFormat()); agg.AggRoot != "" {
							mux.Lock()
							aggStructSlice = append(aggStructSlice, agg)
							mux.Unlock()
							msgStructSlice = make([]amqp.Delivery, 0)
						}
					}
				}
			}
			if len(msgStructSlice) > 0 {
				if agg := ProcessAggregation(rabbitmqConnectUri, msgStructSlice, nist.ChainpointFormat()); agg.AggRoot != "" {
					mux.Lock()
					aggStructSlice = append(aggStructSlice, agg)
					mux.Unlock()
				}
			}
			fmt.Println("Ending aggregation")
		}()
	}

	time.Sleep(time.Duration(sleep) * time.Second)
	close(shutdown)
	wg.Wait()
	fmt.Printf("Aggregation consists of %d items\n", len(aggStructSlice))

	return aggStructSlice

}

// ProcessAggregation creates merkle trees of received hashes a la https://github.com/chainpoint/chainpoint-services/blob/develop/node-aggregator-service/server.js#L66
func ProcessAggregation(rabbitmqConnectUri string, msgStructSlice []amqp.Delivery, nist string) Aggregation {
	var agg Aggregation
	hashSlice := make([][]byte, 0)     // byte array
	hashStructSlice := make([]Hash, 0) // keep record for building proof path

	for _, msgHash := range msgStructSlice {
		unPackedHash := Hash{}
		if err := json.Unmarshal(msgHash.Body, &unPackedHash); err != nil || len(msgHash.Body) == 0 {
			util.LogError(err)
			continue
		}
		if nist != "" {
			unPackedHash.Nist = nist
		}
		hashStructSlice = append(hashStructSlice, unPackedHash)
		var buffer bytes.Buffer
		_, err := buffer.WriteString(fmt.Sprintf("core_id:%s%s", unPackedHash.HashID, unPackedHash.Hash))
		rabbitmq.LogError(err, "failed to write hashes to byte buffer")
		newHash := sha256.Sum256(buffer.Bytes())
		if unPackedHash.Nist != "" {
			var nistBuffer bytes.Buffer
			nistBuffer.WriteString(fmt.Sprintf("nistv2:%s", unPackedHash.Nist))
			nistBuffer.Write(newHash[:])
			newHash = sha256.Sum256(nistBuffer.Bytes())
		}
		hashSlice = append(hashSlice, newHash[:])
	}

	if len(msgStructSlice) == 0 || len(hashSlice) == 0 {
		return Aggregation{}
	}

	//Merkle tree creation
	var tree merkletools.MerkleTree
	tree.AddLeaves(hashSlice)
	tree.MakeTree()
	uuid, err := uuid.NewUUID()
	rabbitmq.LogError(err, "can't generate uuid")
	agg.AggId = uuid.String()
	agg.AggRoot = hex.EncodeToString(tree.GetMerkleRoot())

	//Create proof paths
	proofSlice := make([]ProofData, 0)
	for i, unPackedHash := range hashStructSlice {
		var proofData ProofData
		proofData.HashID = unPackedHash.HashID
		proofData.Hash = unPackedHash.Hash
		proofs := tree.GetProof(i)
		if unPackedHash.Nist != "" {
			proofs = append([]merkletools.ProofStep{merkletools.ProofStep{Left: true, Value: []byte(fmt.Sprintf("nistv2:%s", unPackedHash.Nist))}}, proofs...)
		}
		proofs = append([]merkletools.ProofStep{merkletools.ProofStep{Left: true, Value: []byte(fmt.Sprintf("core_id:%s", unPackedHash.HashID))}}, proofs...)
		proofData.Proof = make([]Proof, 0)
		for _, p := range proofs {
			if p.Left {
				if strings.Contains(string(p.Value), "nistv2") || strings.Contains(string(p.Value), "core_id") {
					proofData.Proof = append(proofData.Proof, Proof{Left: string(p.Value)})
				} else {
					proofData.Proof = append(proofData.Proof, Proof{Left: hex.EncodeToString(p.Value)})
				}
			} else {
				proofData.Proof = append(proofData.Proof, Proof{Right: hex.EncodeToString(p.Value)})
			}
			proofData.Proof = append(proofData.Proof, Proof{Op: "sha-256"})
		}
		proofSlice = append(proofSlice, proofData)
	}
	agg.ProofData = proofSlice

	//Publish to proof-state service
	aggJson, err := json.Marshal(agg)
	err = rabbitmq.Publish(rabbitmqConnectUri, proofStateQueueOut, msgType, aggJson)

	if err != nil {
		rabbitmq.LogError(err, "problem publishing aggJSON message to queue")
		for _, msg := range msgStructSlice {
			msg.Nack(false, true)
		}
	} else {
		for _, msg := range msgStructSlice {
			errAck := msg.Ack(false)
			rabbitmq.LogError(errAck, "error acking queue item")
		}
	}
	return agg
}
