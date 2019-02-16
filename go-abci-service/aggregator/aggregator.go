package aggregator

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"

	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"

	"github.com/google/uuid"
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
	Op    string `json: "op"`
}

const msgType = "aggregator"
const aggQueueIn = "work.agg"
const aggQueueOut = "work.agg"
const proofStateQueueOut = "work.proofstate"

/*Retrieves hash messages from rabbitmq, stores them, and creates a proof path for each*/
func (agg *Aggregation) Aggregate(rabbitmqConnectUri string) {

	var session rabbitmq.Session
	var err error
	//Consume queue in go function with output slice guarded by mutex
	var mux sync.Mutex
	msgStructSlice := make([]amqp.Delivery, 0)
	endConsume := false
	go func() {
		for i := 0; i < 5; i++ {
			session, err = rabbitmq.ConnectAndConsume(rabbitmqConnectUri, aggQueueIn)
			if err != nil {
				continue
			}
			for {
				select {
				case err = <-session.Notify:
					if endConsume {
						return
					}
					time.Sleep(5 * time.Second)
					break //reconnect
				case hash := <-session.Msgs:
					mux.Lock()
					msgStructSlice = append(msgStructSlice, hash)
					mux.Unlock()
				}
			}
		}
	}()
	time.Sleep(60 * time.Second)
	endConsume = true
	session.Ch.Cancel(aggQueueIn, true)
	defer session.Conn.Close()

	//new hashes from concatenated properties
	hashSlice := make([][]byte, len(msgStructSlice))     // byte array
	hashStructSlice := make([]Hash, len(msgStructSlice)) // keep record for building proof path
	for i, msgHash := range msgStructSlice {
		unPackedHash := Hash{}
		json.Unmarshal(msgHash.Body, &unPackedHash)
		hashStructSlice[i] = unPackedHash
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
		hashSlice[i] = newHash[:]
	}

	if len(msgStructSlice) == 0 {
		return
	}

	//Merkle tree creation
	var tree merkletools.MerkleTree
	tree.AddLeaves(hashSlice)
	tree.MakeTree()
	uuid, err := uuid.NewUUID()
	rabbitmq.LogError(err, "can't generate uuid")
	agg.AggId = uuid.String()
	agg.AggRoot = fmt.Sprintf("%x", tree.Root)

	//Create proof paths
	proofSlice := make([]ProofData, len(hashStructSlice))
	for i, unPackedHash := range hashStructSlice {
		var proofData ProofData
		proofData.HashID = unPackedHash.HashID
		proofData.Hash = unPackedHash.Hash
		proofs := tree.GetProof(i)
		if unPackedHash.Nist != "" {
			proofs = append([]merkletools.ProofStep{merkletools.ProofStep{Left: true, Value: []byte(fmt.Sprintf("nistv2:%s", unPackedHash.Nist))}}, proofs...)
		}
		proofs = append([]merkletools.ProofStep{merkletools.ProofStep{Left: true, Value: []byte(fmt.Sprintf("core_id:%s", unPackedHash.HashID))}}, proofs...)
		proofData.Proof = make([]Proof, len(proofs))
		for j, p := range proofs {
			if p.Left {
				proofData.Proof[j] = Proof{Left: string(p.Value), Op: "sha-256"}
			} else {
				proofData.Proof[j] = Proof{Right: string(p.Value), Op: "sha-256"}
			}
		}
		proofSlice = append(proofSlice, proofData)
	}
	agg.ProofData = proofSlice

	aggJson, err := json.Marshal(agg)

	destSession, err := rabbitmq.Dial(rabbitmqConnectUri, proofStateQueueOut)
	defer destSession.Conn.Close()
	defer destSession.Ch.Close()
	err = destSession.Ch.Publish(
		"",
		destSession.Queue.Name,
		false,
		false,
		amqp.Publishing{
			Type:         msgType,
			Body:         aggJson,
			DeliveryMode: 2, //persistent
			ContentType:  "application/json",
		})

	if err != nil {
		for _, msg := range msgStructSlice {
			msg.Nack(false, true)
		}
	} else {
		for _, msg := range msgStructSlice {
			msg.Ack(false)
		}
	}
}
