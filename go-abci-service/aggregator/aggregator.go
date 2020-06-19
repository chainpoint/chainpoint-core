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
	"github.com/tendermint/tendermint/libs/log"

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
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered", r)
		}
	}()
	close(aggregator.TempStop)
	aggregator.RestartMutex.Lock()
	aggregations := make([]types.Aggregation, len(aggregator.Aggregations))
	if len(aggregator.Aggregations) > 0 {
		copy(aggregations, aggregator.Aggregations)
		aggregator.Aggregations = make([]types.Aggregation, 0)
	}
	aggregator.Logger.Info(fmt.Sprintf("Retrieved aggregation tree of %d items and resetting", len(aggregations)))
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
	aggregator.Logger.Info(fmt.Sprintf("Starting aggregation with %d threads and %d batch size", aggThreads, hashBatchSize))
	connected := false
	//Consume queue in goroutines with output slice guarded by mutex
	aggregator.Aggregations = make([]types.Aggregation, 0)
	for {
		aggregator.RestartMutex.Lock()
		aggregator.TempStop = make(chan struct{})
		if !connected {
			session, err = rabbitmq.ConnectAndConsume(aggregator.RabbitmqURI, aggQueueIn)
			if rabbitmq.LogError(err, "RabbitMQ connection failed") != nil {
				rabbitmq.LogError(err, "failed to dial for work.in queue")
				time.Sleep(5 * time.Second)
				aggregator.RestartMutex.Unlock()
				continue
			}
			connected = true
		}
		consume := true
		// Spin up {aggThreads} number of threads to process incoming hashes from the API
		for i := 0; i < aggThreads; i++ {
			aggregator.WaitGroup.Add(1)
			go func() {
				msgStructSlice := make([]amqp.Delivery, 0)
				defer aggregator.WaitGroup.Done()
				for connected && consume {
					select {
					case <-aggregator.TempStop:
						consume = false
						break
					case err = <-session.Notify:
						connected = false
						break
					case hash := <-session.Msgs:
						if len(hash.Body) > 0 {
							msgStructSlice = append(msgStructSlice, hash)
							aggregator.Logger.Info(fmt.Sprintf("Hash: %s", string(hash.Body)))
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
				}
				if len(msgStructSlice) > 0 {
					if agg := aggregator.ProcessAggregation(msgStructSlice, aggregator.LatestNist); agg.AggRoot != "" {
						aggregator.AggMutex.Lock()
						aggregator.Aggregations = append(aggregator.Aggregations, agg)
						aggregator.AggMutex.Unlock()
					}
				}
			}()
		}
		aggregator.WaitGroup.Wait()
		aggregator.Logger.Info("aggregation threads stopped")
		if !connected {
			session.End()
		}
		aggregator.RestartMutex.Unlock()
		time.Sleep(5 * time.Second)
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

		//decode hash to bytes and concatenate onto nist bytes
		hashBytes, _ := hex.DecodeString(unPackedHashItem.Hash)

		//Create checksum
		var newHash [32]byte

		if nist != "" {
			var nistBuffer bytes.Buffer
			nistBuffer.WriteString(fmt.Sprintf("nistv2:%s", nist))
			newHash = sha256.Sum256(append(nistBuffer.Bytes(), hashBytes...))
		}else {
			copy(newHash[:], hashBytes)
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
		proofData.ProofID = unPackedHash.ProofID
		proofData.Hash = unPackedHash.Hash
		proofs := tree.GetProof(i)
		if nist != "" {
			proofs = append([]merkletools.ProofStep{merkletools.ProofStep{Left: true, Value: []byte(fmt.Sprintf("nistv2:%s", nist))}}, proofs...)
		}
		proofData.Proof = make([]types.ProofLineItem, 0)
		for _, p := range proofs {
			if p.Left {
				if strings.Contains(string(p.Value), "nistv2") {
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
	aggregator.Logger.Debug(fmt.Sprintf("Aggregated: %#v", agg))

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
