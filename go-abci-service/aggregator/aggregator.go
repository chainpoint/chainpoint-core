package aggregator

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

type Aggregation struct {
	AggId     string           `json:"agg_id"`
	AggRoot   string           `json:"agg_root"`
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

var RABBITMQ_CONNECT_URI = "amqp://chainpoint:chainpoint@rabbitmq:5672/"

type Session struct {
	conn *amqp.Connection
	ch *amqp.Channel
	queue amqp.Queue
	msgs <-chan amqp.Delivery
	notify chan *amqp.Error
}

//TODO: change to retry logic
func logError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}

func dial(amqpUrl string, queue string)(Session, error){
	var session Session
	var err error
	session.conn, err = amqp.Dial(amqpUrl)
	if err != nil{
		logError(err, "dialing connection error")
		return session, err
	}
	session.notify = session.conn.NotifyClose(make(chan *amqp.Error))
	session.ch, err = session.conn.Channel()
	if err != nil{
		logError(err, "Channel error")
		return session, err
	}
	session.queue, err = session.ch.QueueDeclare(
		queue, // name
		true,      // durable
		false,      // delete when usused
		false,      // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		logError(err, "Problem with queue declare")
		return session, err
	}else{
		return session, nil
	}
}

func connectAndConsume()(session Session, err error){
	//Connect to Queue
	session, err = dial(RABBITMQ_CONNECT_URI, aggQueueIn)
	session.msgs, err = session.ch.Consume(
		session.queue.Name,     // queue
		"", // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	logError(err, "can't consume queue")
	return
}

func (agg *Aggregation) Aggregate() {

	var session Session
	var err error
	//Consume queue in go function with output slice guarded by mutex
	var mux sync.Mutex
	msgStructSlice := make([]amqp.Delivery, 0)
	go func() {
		for i:=0; i < 5; i++ {
			session, err = connectAndConsume()
			if err != nil{
				continue
			}
			for {
				select {
				case err = <-session.notify:
					time.Sleep(5 * time.Second)
					break //reconnect
				case hash := <- session.msgs:
					mux.Lock()
					msgStructSlice = append(msgStructSlice, hash)
					mux.Unlock()
				}
			}
		}
	}()
	time.Sleep(60 * time.Second)
	session.ch.Cancel(aggQueueIn, true)
	defer session.conn.Close()

	//new hashes from concatenated properties
	hashSlice := make([][]byte, len(msgStructSlice))          // byte array
	hashStructSlice := make([]Hash, len(msgStructSlice)) // keep record for building proof path
	for i, msgHash := range msgStructSlice {
		unPackedHash := Hash{}
		json.Unmarshal(msgHash.Body, &unPackedHash)
		hashStructSlice[i] = unPackedHash
		var buffer bytes.Buffer
		_, err := buffer.WriteString(fmt.Sprintf("core_id:%s%s", unPackedHash.HashID, unPackedHash.Hash))
		logError(err, "failed to write hashes to byte buffer")
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
	logError(err, "can't generate uuid")
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

	destSession,err := dial(RABBITMQ_CONNECT_URI, proofStateQueueOut)
	defer destSession.conn.Close()
	defer destSession.ch.Close()
	err = destSession.ch.Publish(
		"",
		destSession.queue.Name,
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