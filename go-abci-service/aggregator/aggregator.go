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
)

const msgType = "aggregator"
const aggQueueIn = "work.agg"
const proofStateQueueOut = "work.proofstate"

// Aggregator : object includes rabbitURI and Logger
type Aggregator struct {
	HashItems	 []types.HashItem
	Logger       log.Logger
	LatestTime   string
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

func (aggregator *Aggregator) AddHashItem(item types.HashItem) {
	aggregator.HashItems = append(aggregator.HashItems, item)
}

func (aggregator *Aggregator) HeadHashItem() (types.HashItem) {
	item := aggregator.HashItems[0]
	aggregator.HashItems = aggregator.HashItems[1:]
	return item
}

// ReceiveCalRMQ : Continually consume the calendar work queue and
// process any resulting messages from the tx and monitor services
func (aggregator *Aggregator) StartAggregation() error {
	aggThreads, _ := strconv.Atoi(util.GetEnv("AGGREGATION_THREADS", "4"))
	hashBatchSize, _ := strconv.Atoi(util.GetEnv("HASHES_PER_MERKLE_TREE", "25000"))
	aggregator.Logger.Info(fmt.Sprintf("Starting aggregation with %d threads and %d batch size", aggThreads, hashBatchSize))
	//Consume queue in goroutines with output slice guarded by mutex
	aggregator.Aggregations = make([]types.Aggregation, 0)
	for {
		aggregator.RestartMutex.Lock()
		aggregator.TempStop = make(chan struct{})
		consume := true
		// Spin up {aggThreads} number of threads to process incoming hashes from the API
		for i := 0; i < aggThreads; i++ {
			aggregator.WaitGroup.Add(1)
			go func() {
				msgStructSlice := make([]types.HashItem, 0)
				defer aggregator.WaitGroup.Done()
				for consume {
					select {
					case <-aggregator.TempStop:
						consume = false
						break
					default:
						hash := aggregator.HeadHashItem()
						if len(hash.Hash) > 0 {
							msgStructSlice = append(msgStructSlice, hash)
							aggregator.Logger.Info(fmt.Sprintf("Hash: %s", string(hash.Hash)))
							//create new agg roots under heavy load
							if len(msgStructSlice) > hashBatchSize {
								if agg := aggregator.ProcessAggregation(msgStructSlice, aggregator.LatestTime); agg.AggRoot != "" {
									aggregator.AggMutex.Lock()
									aggregator.Aggregations = append(aggregator.Aggregations, agg)
									aggregator.AggMutex.Unlock()
									msgStructSlice = make([]types.HashItem, 0)
								}
							}
						}
					}
				}
				if len(msgStructSlice) > 0 {
					if agg := aggregator.ProcessAggregation(msgStructSlice, aggregator.LatestTime); agg.AggRoot != "" {
						aggregator.AggMutex.Lock()
						aggregator.Aggregations = append(aggregator.Aggregations, agg)
						aggregator.AggMutex.Unlock()
					}
				}
			}()
		}
		aggregator.WaitGroup.Wait()
		aggregator.Logger.Info("aggregation threads stopped")
		aggregator.RestartMutex.Unlock()
		time.Sleep(5 * time.Second)
	}
}

// ProcessAggregation creates merkle trees of received hashes a la https://github.com/chainpoint/chainpoint-services/blob/develop/node-aggregator-service/server.js#L66
func (aggregator *Aggregator) ProcessAggregation(msgStructSlice []types.HashItem, drand string) types.Aggregation {
	var agg types.Aggregation
	hashSlice := make([][]byte, 0)               // byte array

	for _, msgHash := range msgStructSlice {
		//decode hash to bytes and concatenate onto nist bytes
		hashBytes, _ := hex.DecodeString(msgHash.Hash)

		//Create checksum
		var newHash [32]byte

		if drand != "" {
			var timeBuffer bytes.Buffer
			timeBuffer.WriteString(fmt.Sprintf("drand:%s", drand))
			newHash = sha256.Sum256(append(timeBuffer.Bytes(), hashBytes...))
		} else {
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
	for i, unPackedHash := range msgStructSlice {
		var proofData types.ProofData
		proofData.ProofID = unPackedHash.ProofID
		proofData.Hash = unPackedHash.Hash
		proofs := tree.GetProof(i)
		if drand != "" {
			proofs = append([]merkletools.ProofStep{{Left: true, Value: []byte(fmt.Sprintf("drand:%s", drand))}}, proofs...)
		}
		proofData.Proof = make([]types.ProofLineItem, 0)
		for _, p := range proofs {
			if p.Left {
				if strings.Contains(string(p.Value), "drand") {
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
