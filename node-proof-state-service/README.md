# chainpoint-node-proof-state-service

## Configuration
Configuration parameters will be stored in environment variables. Environment variables can be overridden through the use of a .env file. 

The following are the descriptions of the configuration parameters:

| Name           | Description  |
| :------------- |:-------------|
| RMQ\_PREFETCH\_COUNT | the maximum number of messages sent over the channel that can be awaiting acknowledgement |
| RMQ\_WORK\_IN\_QUEUE     | the queue name for message consumption originating from the api service |
| RMQ\_WORK\_OUT\_GEN\_QUEUE       | the queue name for outgoing message to the proof gen service | 
| RMQ\_WORK\_OUT\_STATE\_QUEUE       | the queue name for outgoing message to the proof state service |
| RABBITMQ\_CONNECT\_URI       | the RabbitMQ connection URI |
| PRUNE\_FREQUENCY\_MINUTES       | The frequency of proof state and hash tracker log data pruning |

The following are the types, defaults, and acceptable ranges of the configuration parameters: 

| Name           | Type         | Default | 
| :------------- |:-------------|:-------------|
| RMQ\_PREFETCH\_COUNT      | integer      | 10 | 0 | - | 
| RMQ\_WORK\_IN\_QUEUE      | string      | 'work.proofstate' |  |  | 
| RMQ\_WORK\_OUT\_GEN\_QUEUE       | string      | 'work.gen' |  |  | 
| RMQ\_WORK\_OUT\_STATE\_QUEUE       | string      | 'work.proofstate' |  |  |   
| RABBITMQ\_CONNECT\_URI       | string      | 'amqp://chainpoint:chainpoint@rabbitmq' | 
| PRUNE\_FREQUENCY\_MINUTES       | integer      | 1 | 


## Data In
The proof state service serves as the a proof state storage mechanism for all hashes as they are being processed. As proofs are constructed for each hash, state data is received and stored in CRDB from other services. As anchors objects are completed and added to the proof, a proof ready message is also queued for the proof generator service indicating that a Chainpoint proof is ready to be created for the current state data. These proof ready messages are both published and consumed by this service. Milestone events occurring during the proof building process are logged to a hash tracker table.


#### Aggregator Message
When an aggregation event occurs, the aggregation service will queue a message bound for the proof state service containing data for each hash in that aggregation event.
The following is an example of state data published from the aggregator service: 
```json
{
  "agg_id": "0cdecc3e-2452-11e7-93ae-92361f002671", // a UUIDv1 for this aggregation event
  "agg_root": "419001851bcf08329f0c34bb89570028ff500fc85707caa53a3e5b8b2ecacf05",
  "proofData": [
    {
      "hash_id": "34712680-14bb-11e7-9598-0800200c9a66",
      "hash": "a0ec06301bf1814970a70f89d1d373afdff9a36d1ba6675fc02f8a975f4efaeb",
      "proof": [ /* Chainpoint v3 ops list for leaf 0 ... */ ]
    },
    {
      "hash_id": "6d627180-1883-11e7-a8f9-edb8c212ef23",
      "hash": "2222d5f509d86e2627b1f498d7b44db1f8e70aae1528634580a9b68f05d57a9f",
      "proof": [ /* Chainpoint v3 ops list for leaf 1 ... */ ]
    },
    { /* more ... */ },
  ]
}
```
| Name             | Description                                                            |
| :--------------- |:-----------------------------------------------------------------------|
| agg_id           | The UUIDv1 unique identifier for the aggregation event with embedded timestamp |
| agg_root        | A hex string representing the merkle root for this aggregation tree |
| proofData        | An array of hash state data |
| proofData.hash_id          | The UUIDv1 unique identifier for a hash object with embedded timestamp |
| proofData.hash          | A hex string representing the hash to be processed  |
| proofData.agg_state  | The state data being stored, in this case, aggregation operations |


#### Calendar Message
When a new calendar entry is created, the calendar service will queue messages bound for the proof state service for each aggregation event in that calendar entry.
The following is an example of state data published in a calendar message: 
```json
{
  "agg_id": "0cdecc3e-2452-11e7-93ae-92361f002671",
  "cal_id": "1027",
  "cal_state": {
    "ops": [
      { "l": "315be5d46580b617928b53f3bac5bac3d5e0d10a1c6143cc1fdab224cd1450ea" },
      { "op": "sha-256" },
      { "r": "585a960c51c665432f52d2ceb5a31a11bdc375bac136ffa0af84afa1b1e7840f" },
      { "op": "sha-256" }
    ],
    "anchor": {
      "anchor_id" : "1027",
      "uris": [
        "http://a.cal.chainpoint.org/1027/hash"
      ]
    }
  }
}
```
| Name             | Description                                                            |
| :--------------- |:-----------------------------------------------------------------------|
| agg_id          | The UUIDv1 unique identifier for the aggregation event with embedded timestamp |
| cal_id          | The block height for the calendar block |
| cal_state  | The state data being stored, in this case, calendar aggregation operations and cal anchor information |


#### Anchor Agg Message
Periodically, calendar entries are read and aggregated for anchoring. The calendar service will queue messages bound for the proof state service for these anchor aggregation events.
The following is an example of state data published in an anchor agg message: 
```json
{
  "cal_id": "1027",
  "anchor_btc_agg_id": "af884cde-422b-11e7-a919-92ebcb67fe33",
  "anchor_btc_agg_state": {
    "ops": [
      { "l": "c380779f6175766fdbe90940851fff3995d343c63bbb82f816843c1d5100865e" },
      { "op": "sha-256" },
      { "r": "2f05c19fbcea874a66c33431da22df8c062decf363c0ab4cbde48edf0be8ab09" },
      { "op": "sha-256" }
    ]
  }
}
```
| Name             | Description                                                            |
| :--------------- |:-----------------------------------------------------------------------|
| cal\_id          | The block height for the calendar block |
| anchor\_agg\_id          | The UUIDv1 unique identifier for an anchor aggregation event with embedded timestamp |
| anchor\_agg\_state  | The state data being stored, in this case, anchor aggregation operations |


#### Btctx Message
When data has been anchored to the Bitcoin blockchain, additional state data is saved to connect the anchor aggreagtion root value to the new transaction body. The calendar service will queue messages bound for the proof state service for this bitcoin transaction data.
The following is an example of state data published in an anchor agg message: 
```json
{
  "anchor_btc_agg_id": "af884cde-422b-11e7-a919-92ebcb67fe33",
  "btctx_id": "2265a48bcf9b72c1bc4f0a70ae946e9f438a783947947a309a3b2e458f81c63b",
  "btctx_state": {
    "ops": [
      { "l": "010000000183c19d576272403ad4dede9ed488b0c73913e32611fd9fb0c7d060bcc1606f7e010000006a473044022065ce187f65536ce6be1a7bd45170e04dd7dab2f3c1cf2c161e6fb3d1b446120b0220229028e672afd2385e1a693ad287308851e03d65325378486873f6142f968b010121035b690114679d44d75b75aa170e34596c94c778f589bcb9063b0e4e293fcacd1dffffffff020000000000000000226a20" },
      { "r": "48120203000000001976a9147003cc5915f6c23fd512b38daeeecfdde7a587e988ac00000000" }
    ]
  }
}
```
| Name             | Description                                                            |
| :--------------- |:-----------------------------------------------------------------------|
| anchor\_agg\_id          | The UUIDv1 unique identifier for an anchor aggregation event with embedded timestamp |
| btctx\_id          | The bitcoin transaction id |
| btctx\_state  | The state data being stored, in this case, bitcoin transaction information |


#### Proof State Service
After the proof state service consumes a calendar message and stores the calendar entry state data, the proof state service will also queue proof ready messages bound for the proof state service for each hash part of the aggregation event for that calendar message.
The following is an example of proof ready data published from the proof state service: 
```json
{
  "type": "cal",
  "hash_id": "34712680-14bb-11e7-9598-0800200c9a66"
}
```
| Name             | Description                                                            |
| :--------------- |:-----------------------------------------------------------------------|
| type          | The type of proof ready to be generated |
| hash_id          | The UUIDv1 unique identifier for the hash with embedded timestamp |



## Proof State Storage
As state data is consumed from the queue, proof state objects are created from that state data and saved to storage.

In addition to storing state data, the proof state service also updates the hash tracker log for milestone events that occurring during the proof generation process. The events being tracked are shown in the following table.

| Name | Description                                                            |
| :--- |:-----------------------------------------------------------------------|
| aggregator_at          | A timestamp value indicating when the hash was processed by the aggregator service |
| calendar_at          | A timestamp value indicating when calendar proof generation has begun for this hash |
| btc_at          | A timestamp value indicating when btc proof generation has begun for this hash |
| eth_at          | A timestamp value indicating when eth proof generation has begun for this hash |


## Data Out 
The service will publish proof ready messages and proof generation message to durable queues within RabbitMQ. The queue names are defined by the RMQ\_WORK\_OUT\_STATE\_QUEUE and RMQ\_WORK\_OUT\_GEN\_QUEUE configuration parameters.

When consuming a calendar message, the proof state service will queue proof ready messages bound for the proof state service for each hash part of the aggregation event for that calendar message.

The following is an example of a proof state object message sent to the proof state service: 
```json
{
  "type": "cal",
  "hash_id": "34712680-14bb-11e7-9598-0800200c9a66"
}
```
| Name             | Description                                                            |
| :--------------- |:-----------------------------------------------------------------------|
| type          | The type of proof ready to be generated |
| hash_id          | The UUIDv1 unique identifier for the hash with embedded timestamp |

In addition to publishing these proof ready messages, the proof state service also consumes them. All state data for the specified hash is read from storage and included in a proof generation message bound for the proof gen service. 

The following is an example of a proof generation message sent to the proof gen service: 
```json
{
  "type": "cal",
  "hash_id": "34712680-14bb-11e7-9598-0800200c9a66",
  "hash": "a0ec06301bf1814970a70f89d1d373afdff9a36d1ba6675fc02f8a975f4efaeb",
  "agg_state": {
    "ops": [
        { "l": "fab4a0b99def4631354ca8b3a7f7fe026623ade9c8c5b080b16b2c744d2b9c7d" },
        { "op": "sha-256" },
        { "r": "7fb6bb6387d1ffa74671ecf5d337f7a8881443e5b5532106f9bebb673dd72bc9" },
        { "op": "sha-256" }
      ]
  },
  "cal_state": {
    "ops": [
      { "l": "315be5d46580b617928b53f3bac5bac3d5e0d10a1c6143cc1fdab224cd1450ea" },
      { "op": "sha-256" },
      { "r": "585a960c51c665432f52d2ceb5a31a11bdc375bac136ffa0af84afa1b1e7840f" },
      { "op": "sha-256" }
    ],
    "anchor": {
      "anchor_id" : "1027",
      "uris": [
        "http://a.cal.chainpoint.org/1027/hash"
      ]
    }
  }
}
```
| Name             | Description                                                            |
| :--------------- |:-----------------------------------------------------------------------|
| type          | The type of proof ready to be generated |
| hash_id          | The UUIDv1 unique identifier for the hash with embedded timestamp |
| hash          | A hex string representing the hash being processed  |
| agg_state  | The aggregation state data for this hash to be used for proof generation |
| cal_state  | The calendar state data for this hash to be used for proof generation |

## Service Failure
In the event of any error occurring, the service will log that error to STDERR and kill itself with a process.exit(). RabbitMQ will be configured so that upon service exit, unacknowledged messages will be requeued to ensure than unfinished work lost due to failure will be processed again in full.


## Notable NPM packages
| Name         | Description                                                            |
| :---         |:-----------------------------------------------------------------------|
| envalid       | for managing and validating environment variables |
| amqplib      | for communication between the service and RabbitMQ |
| async      | for handling flow control for some asynchronous operations |





