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

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/tendermint/libs/log"

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
	RabbitmqURI  string
	Logger       log.Logger
	LatestNist   string
	Aggregations []types.Aggregation
	AggMutex     sync.Mutex
	RestartMutex sync.Mutex
	TempStop     chan struct{}
	WaitGroup    sync.WaitGroup
}

func (aggregator *Aggregator) AggregateAndReset() []types.Aggregation {
	aggregator.Logger.Info(fmt.Sprintf("Retrieving aggregation tree and resetting...."))
	close(aggregator.TempStop)
	aggregator.RestartMutex.Lock()
	aggregations := make([]types.Aggregation, len(aggregator.Aggregations))
	if len(aggregator.Aggregations) > 0 {
		copy(aggregations, aggregator.Aggregations)
		aggregator.Aggregations = make([]types.Aggregation, 0)
	}
	aggregator.RestartMutex.Unlock()
	return aggregations
}

// ReceiveCalRMQ : Continually consume the calendar work queue and
// process any resulting messages from the tx and monitor services
func (aggregator *Aggregator) StartAggregation() error {
	var session rabbitmq.Session
	var err error
	aggThreads, _ := strconv.Atoi(util.GetEnv("AGGREGATION_THREADS", "4"))
	hashBatchSize, _ := strconv.Atoi(util.GetEnv("HASHES_PER_MERKLE_TREE", "25000"))

	//Consume queue in goroutines with output slice guarded by mutex
	aggregator.Aggregations = make([]types.Aggregation, 0)
	for {
		aggregator.RestartMutex.Lock()
		aggregator.TempStop = make(chan struct{})
		session, err = rabbitmq.ConnectAndConsume(aggregator.RabbitmqURI, aggQueueIn)
		if rabbitmq.LogError(err, "RabbitMQ connection failed") != nil {
			rabbitmq.LogError(err, "failed to dial for work.in queue")
			time.Sleep(5 * time.Second)
			aggregator.RestartMutex.Unlock()
			continue
		}
		// Spin up {aggThreads} number of threads to process incoming hashes from the API
		for i := 0; i < aggThreads; i++ {
			go func() {
				msgStructSlice := make([]amqp.Delivery, 0)
				aggregator.WaitGroup.Add(1)
				defer aggregator.WaitGroup.Done()
				for {
					select {
					case <-aggregator.TempStop:
						return
					case err = <-session.Notify:
						return
					case hash := <-session.Msgs:
						if len(hash.Body) > 0 {
							msgStructSlice = append(msgStructSlice, hash)
							aggregator.Logger.Info(fmt.Sprintf("Hash: %s\n", string(hash.Body)))
							//create new agg roots under heavy load
							if len(msgStructSlice) > hashBatchSize {
								if agg := aggregator.ProcessAggregation(msgStructSlice, aggregator.LatestNist); agg.AggRoot != "" {
									aggregator.AggMutex.Lock()
									aggregator.Aggregations = append(aggregator.Aggregations, agg)
									aggregator.AggMutex.Unlock()
									msgStructSlice = make([]amqp.Delivery, 0)
								}
							}
						}
					}
					if len(msgStructSlice) > 0 {
						if agg := aggregator.ProcessAggregation(msgStructSlice, aggregator.LatestNist); agg.AggRoot != "" {
							aggregator.AggMutex.Lock()
							aggregator.Aggregations = append(aggregator.Aggregations, agg)
							aggregator.AggMutex.Unlock()
						}
					}
				}
			}()
		}
		aggregator.WaitGroup.Wait()
		aggregator.Logger.Debug(fmt.Sprintf("Aggregator resetting RabbitMQ connection...", len(aggregator.Aggregations)))
		session.End()
		aggregator.RestartMutex.Unlock()
	}
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
