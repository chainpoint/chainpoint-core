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

	"github.com/tendermint/tendermint/libs/log"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/google/uuid"

	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"
	"github.com/streadway/amqp"
)

const msgType = "aggregator"
const aggQueueIn = "work.agg"
const proofStateQueueOut = "work.proofstate"

// Aggregator : object includes rabbitURI and Logger
type Aggregator struct {
	RabbitmqURI string
	Logger      log.Logger
}

// Aggregate retrieves hash messages from rabbitmq, stores them, and creates a proof path for each
func (aggregator *Aggregator) Aggregate(nist string) (agg []types.Aggregation) {
	var session rabbitmq.Session
	aggThreads, _ := strconv.Atoi(util.GetEnv("AGGREGATION_THREADS", "4"))
	hashBatchSize, _ := strconv.Atoi(util.GetEnv("HASHES_PER_MERKLE_TREE", "25000"))
	sleep := int(60 / aggThreads)

	//Consume queue in goroutines with output slice guarded by mutex
	aggStructSlice := make([]types.Aggregation, 0)
	shutdown := make(chan struct{})
	var wg sync.WaitGroup
	var mux sync.Mutex
	endConsume := false

	session, err := rabbitmq.ConnectAndConsume(aggregator.RabbitmqURI, aggQueueIn)
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
					msgStructSlice = append(msgStructSlice, hash)
					aggregator.Logger.Info(fmt.Sprintf("Hash: %s\n", string(hash.Body)))
					//create new agg roots under heavy load
					if len(msgStructSlice) > hashBatchSize {
						if agg := aggregator.ProcessAggregation(msgStructSlice, nist); agg.AggRoot != "" {
							mux.Lock()
							aggStructSlice = append(aggStructSlice, agg)
							mux.Unlock()
							msgStructSlice = make([]amqp.Delivery, 0)
						}
					}
				}
			}
			if len(msgStructSlice) > 0 {
				if agg := aggregator.ProcessAggregation(msgStructSlice, nist); agg.AggRoot != "" {
					mux.Lock()
					aggStructSlice = append(aggStructSlice, agg)
					mux.Unlock()
				}
			}
		}()
	}

	time.Sleep(time.Duration(sleep) * time.Second)
	close(shutdown)
	wg.Wait()
	aggregator.Logger.Debug(fmt.Sprintf("Aggregated %d items", len(aggStructSlice)))
	return aggStructSlice
}

// ProcessAggregation creates merkle trees of received hashes a la https://github.com/chainpoint/chainpoint-services/blob/develop/node-aggregator-service/server.js#L66
func (aggregator *Aggregator) ProcessAggregation(msgStructSlice []amqp.Delivery, nist string) types.Aggregation {
	var agg types.Aggregation
	hashSlice := make([][]byte, 0)               // byte array
	hashStructSlice := make([]types.HashItem, 0) // keep record for building proof path

	for _, msgHash := range msgStructSlice {
		unPackedHashItem := types.HashItem{}
		if err := json.Unmarshal(msgHash.Body, &unPackedHashItem); err != nil || len(msgHash.Body) == 0 {
			util.LogError(err)
			continue
		}
		hashStructSlice = append(hashStructSlice, unPackedHashItem)
		var buffer bytes.Buffer

		//concatenate ID and hash
		_, err := buffer.WriteString(fmt.Sprintf("core_id:%s", unPackedHashItem.HashID))
		hashBytes, _ := hex.DecodeString(unPackedHashItem.Hash)
		_, err = buffer.Write(hashBytes)

		rabbitmq.LogError(err, "failed to write hashes to byte buffer")

		//Create checksum
		newHash := sha256.Sum256(buffer.Bytes())

		if nist != "" {
			var nistBuffer bytes.Buffer
			nistBuffer.WriteString(fmt.Sprintf("nistv2:%s", nist))
			nistBuffer.Write(newHash[:])
			newHash = sha256.Sum256(nistBuffer.Bytes())
		}
		hashSlice = append(hashSlice, newHash[:])
	}

	if len(msgStructSlice) == 0 || len(hashSlice) == 0 {
		return types.Aggregation{}
	}

	//Merkle tree creation
	var tree merkletools.MerkleTree
	tree.AddLeaves(hashSlice)
	tree.MakeTree()
	uuid, err := uuid.NewUUID()
	rabbitmq.LogError(err, "can't generate uuid")
	agg.AggID = uuid.String()
	agg.AggRoot = hex.EncodeToString(tree.GetMerkleRoot())

	//Create proof paths
	proofSlice := make([]types.ProofData, 0)
	for i, unPackedHash := range hashStructSlice {
		var proofData types.ProofData
		proofData.HashID = unPackedHash.HashID
		proofData.Hash = unPackedHash.Hash
		proofs := tree.GetProof(i)
		if nist != "" {
			proofs = append([]merkletools.ProofStep{merkletools.ProofStep{Left: true, Value: []byte(fmt.Sprintf("nistv2:%s", nist))}}, proofs...)
		}
		proofs = append([]merkletools.ProofStep{merkletools.ProofStep{Left: true, Value: []byte(fmt.Sprintf("core_id:%s", unPackedHash.HashID))}}, proofs...)
		proofData.Proof = make([]types.ProofLineItem, 0)
		for _, p := range proofs {
			if p.Left {
				if strings.Contains(string(p.Value), "nistv2") || strings.Contains(string(p.Value), "core_id") {
					proofData.Proof = append(proofData.Proof, types.ProofLineItem{Left: string(p.Value)})
				} else {
					proofData.Proof = append(proofData.Proof, types.ProofLineItem{Left: hex.EncodeToString(p.Value)})
				}
			} else {
				proofData.Proof = append(proofData.Proof, types.ProofLineItem{Right: hex.EncodeToString(p.Value)})
			}
			proofData.Proof = append(proofData.Proof, types.ProofLineItem{Op: "sha-256"})
		}
		proofSlice = append(proofSlice, proofData)
	}
	agg.ProofData = proofSlice

	//Publish to proof-state service
	aggJSON, err := json.Marshal(agg)
	if aggregator.RabbitmqURI != "" {
		err = rabbitmq.Publish(aggregator.RabbitmqURI, proofStateQueueOut, msgType, aggJSON)

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
	}
	return agg
}
