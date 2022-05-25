package aggregator

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/chainpoint/chainpoint-core/ulidthreadsafe"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chainpoint/chainpoint-core/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/chainpoint/chainpoint-core/util"

	merkletools "github.com/chainpoint/merkletools-go"
	"github.com/enriquebris/goconcurrentqueue"
)

// Aggregator : object includes rabbitURI and Logger
type Aggregator struct {
	HashItems    goconcurrentqueue.Queue //In
	Logger       log.Logger
	LatestTime   string
	Aggregations goconcurrentqueue.Queue //Out
	AggMutex     sync.Mutex
	RestartMutex sync.Mutex
	QueueMutex   sync.Mutex
	TempStop     chan struct{}
	WaitGroup    sync.WaitGroup
	UlidGen      *ulidthreadsafe.ThreadSafeUlid
}

func (aggregator *Aggregator) AggregateAndReset() []types.Aggregation {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered", r)
		}
	}()
	close(aggregator.TempStop)
	aggregator.RestartMutex.Lock()
	queueLen := aggregator.Aggregations.GetLen()
	aggregations := make([]types.Aggregation, 0)
	if aggregator.Aggregations.GetLen() > 0 {
		for i := 0; i < queueLen; i++ {
			item, err := aggregator.Aggregations.Dequeue()
			if util.LogError(err) != nil {
				return aggregations
			}
			value, ok := item.(types.Aggregation)
			if !ok {
				return aggregations
			}
			aggregations = append(aggregations, value)
		}
	}
	aggregator.Logger.Info(fmt.Sprintf("Retrieved aggregation tree of %d items and resetting", len(aggregations)))
	aggregator.RestartMutex.Unlock()
	return aggregations
}

func (aggregator *Aggregator) AddHashItem(item types.HashItem) {
	util.LogError(aggregator.HashItems.Enqueue(item))
}

func (aggregator *Aggregator) HeadHashItem() types.HashItem {
	item, err := aggregator.HashItems.Dequeue()
	if err != nil {
		return types.HashItem{}
	}
	value, ok := item.(types.HashItem)
	if !ok {
		return types.HashItem{}
	}
	return value
}

// ReceiveCalRMQ : Continually consume the calendar work queue and
// process any resulting messages from the tx and monitor services
func (aggregator *Aggregator) StartAggregation() error {
	aggThreads, _ := strconv.Atoi(util.GetEnv("AGGREGATION_THREADS", "4"))
	hashBatchSize, _ := strconv.Atoi(util.GetEnv("HASHES_PER_MERKLE_TREE", "25000"))
	aggregator.Logger.Info(fmt.Sprintf("Starting aggregation with %d threads and %d batch size", aggThreads, hashBatchSize))
	aggregator.HashItems = goconcurrentqueue.NewFIFO()
	//Consume queue in goroutines with output slice guarded by mutex
	aggregator.Aggregations = goconcurrentqueue.NewFIFO()
	aggregator.UlidGen = ulidthreadsafe.NewThreadSafeUlid()
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
							//aggregator.Logger.Info(fmt.Sprintf("Hash: %s", hash.Hash))
							//create new agg roots under heavy load
							if len(msgStructSlice) > hashBatchSize {
								if agg := aggregator.ProcessAggregation(msgStructSlice, aggregator.LatestTime); agg.AggRoot != "" {
									aggregator.Aggregations.Enqueue(agg)
									msgStructSlice = make([]types.HashItem, 0)
								}
							}
						}
					}
				}
				if len(msgStructSlice) > 0 {
					if agg := aggregator.ProcessAggregation(msgStructSlice, aggregator.LatestTime); agg.AggRoot != "" {
						aggregator.Aggregations.Enqueue(agg)
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
	aggStates := make([]types.AggState, 0)
	var agg types.Aggregation
	hashSlice := make([][]byte, 0) // byte array

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
	ulid, err := aggregator.UlidGen.NewUlid()
	if util.LogError(err) != nil {
		return types.Aggregation{}
	}
	agg.AggID = ulid.String()
	agg.AggRoot = hex.EncodeToString(tree.GetMerkleRoot())

	//Create proof paths
	for i, unPackedHash := range msgStructSlice {
		var proofData types.ProofData
		proofData.ProofID = unPackedHash.ProofID
		proofData.Hash = unPackedHash.Hash
		proofs := tree.GetProof(i)
		if drand != "" {
			proofs = append([]merkletools.ProofStep{{Left: true, Value: []byte(fmt.Sprintf("drand:%s", drand))}}, proofs...)
		}
		proofOps := make([]types.ProofLineItem, 0)
		for _, p := range proofs {
			if p.Left {
				if strings.Contains(string(p.Value), "drand") {
					proofOps = append(proofOps, types.ProofLineItem{Left: string(p.Value)})
				} else {
					proofOps = append(proofOps, types.ProofLineItem{Left: hex.EncodeToString(p.Value)})
				}
			} else {
				proofOps = append(proofOps, types.ProofLineItem{Right: hex.EncodeToString(p.Value)})
			}
			proofOps = append(proofOps, types.ProofLineItem{Op: "sha-256"})
		}
		aggState := types.AggState{}
		aggState.AggID = agg.AggID
		aggState.AggRoot = agg.AggRoot
		aggState.ProofID = unPackedHash.ProofID
		aggState.Hash = unPackedHash.Hash
		ops := types.OpsState{}
		ops.Ops = proofOps
		opsBytes, err := json.Marshal(ops)
		if err != nil {
			continue
		}
		aggState.AggState = string(opsBytes)
		aggStates = append(aggStates, aggState)
	}
	aggregator.Logger.Debug(fmt.Sprintf("Aggregated: %#v", aggStates))
	agg.AggStates = aggStates
	return agg
}
