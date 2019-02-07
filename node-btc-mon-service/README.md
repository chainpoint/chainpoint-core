# chainpoint-node-btc-mon-service

## Configuration
Configuration parameters will be stored in environment variables. Environment variables can be overridden through the use of a .env file. 

The following are the descriptions of the configuration parameters:

| Name           | Description  |
| :------------- |:-------------|
| MIN\_BTC\_CONFIRMS | the number of confirmations needed before the transaction is considered ready for proof delivery |
| MONITOR\_INTERVAL\_SECONDS | the interval in which to run the monitoring process |
| RMQ\_PREFETCH\_COUNT | the maximum number of messages sent over the channel that can be awaiting acknowledgement |
| RMQ\_WORK\_IN\_QUEUE     | the queue name for message consumption originating from the calendar service |
| RMQ\_WORK\_OUT\_CAL\_QUEUE       | the queue name for outgoing message to the calendar service | 
| RABBITMQ\_CONNECT\_URI       | the RabbitMQ connection URI | 
| INSIGHT\_API\_BASE\_URI       | the Bitcore Insight-API base URI | 

The following are the types, defaults, and acceptable ranges of the configuration parameters: 

| Name           | Type         | Default | Min | Max |
| :------------- |:-------------|:-------------|:---|:---|
| MIN\_BTC\_CONFIRMS      | integer      | 6 | 1 | 15 |
| MONITOR\_INTERVAL\_SECONDS      | integer      | 30 | 10 | 600 |
| RMQ\_PREFETCH\_COUNT      | integer      | 0 | | |
| RMQ\_WORK\_IN\_QUEUE      | string      | 'work.btcmon' | | |
| RMQ\_WORK\_OUT\_CAL\_QUEUE       | string      | 'work.cal' | | |
| RABBITMQ\_CONNECT\_URI       | string      | 'amqp://chainpoint:chainpoint@rabbitmq' |  | |
| INSIGHT\_API\_BASE\_URI       | string      | null |  | |


## Data In
The service will receive persistent transaction object messages from a durable queue within RabbitMQ. The queue name is defined by the RMQ\_WORK\_IN\_QUEUE  configuration parameter.

The following is an example of a transaction object message body: 
```json
{
  "tx_id": "752d66de3111c308ac16b7e114b855d79b1bbdaa45f0c4a44b64e79bbc69bb78"
}
```
| Name | Description                                                            |
| :--- |:-----------------------------------------------------------------------|
| tx_id   | The transaction id for the transaction to be monitored |


## Monitoring Process
At the interval defined in MONITOR\_INTERVAL\_SECONDS, the transactions to be monitored are inspected and their confirmation counts checked. If a transaction's confirmation count meets or exceeds MIN\_BTC\_CONFIRMS, then information about its block and the proof path connecting the transaction to the its block's merkle root is compiled and delivered back to the calendar service for further processing. Such transaction are then considered final and are removed from the monitoring array.

## Data Out
For each transaction that has achieved minimum confirmations, a block object message is published using the RMQ\_WORK\_OUT\_CAL\_ROUTING\_KEY for consumption by the calendar service.

The following is an example of a block object message sent to both service: 
```json
{
  "btctx_id": "752d66de3111c308ac16b7e114b855d79b1bbdaa45f0c4a44b64e79bbc69bb78",
  "btchead_height": 469222,
  "btchead_root": "3016a73bb0fc915193a3adddf90ef46b643e270665dcdde35fb52eb1f44a48be",
  "path": [
        { "left": "fab4a0b99def4631354ca8b3a7f7fe026623ade9c8c5b080b16b2c744d2b9c7d" },
        { "right": "7fb6bb6387d1ffa74671ecf5d337f7a8881443e5b5532106f9bebb673dd72bc9" }
      ]
}
```
| Name             | Description                                                            |
| :--------------- |:-----------------------------------------------------------------------|
| btctx_id          | The bitcoin transaction id |
| btchead_height | The block height of the block containing the transaction |
| btchead_root | The merkle root of the block containing the transaction |
| path | left and right operations connecting the transaction hash to the block merkle root, double sha256 tree implied |

When a transaction has achieved minimum confirmations, the original transaction message is acked.

## Service Failure
In the event of any error occurring, the service will log that error to STDERR and kill itself with a process.exit(). RabbitMQ will be configured so that upon service exit, unacknowledged messages will be requeued to ensure than unfinished work lost due to failure will be processed again in full.


## Notable NPM packages
| Name         | Description                                                            |
| :---         |:-----------------------------------------------------------------------|
| envalid       | for managing and validating environment variables |
| amqplib      | for communication between the service and RabbitMQ |
| async      | for handling flow control for some asynchronous operations |
| merkle-tools | for constructing merkle tree and calculating merkle paths |
| request | for building and executing HTTP requests |
| lodash | providing various convenience functions |

