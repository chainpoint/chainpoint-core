# Application BlockChain Interface (ABCI)

## Introduction

The purpose of this package is to interface with the specific Tendermint blockchain ("the Calendar") 
which serves as the event bus, storage layer, and consensus mechanism for the Chainpoint Network. 

The entrypoint, called by the main `abci-service.go` file in the parent directory, is `abci.go`. From here
the application coordinates the corresponding functions from the following files:

* **tasks.go**: Periodic tasks 
* **anchor.go**: Calendar anchoring every tendermint block and BTC anchoring every 60 blocks
* **api.go**: Http API for submitting hashes and retrieving proofs
* **identity.go**: Methods for submitting, saving, and verifying core identities
* **leader.go**: Elects a lead core based on a shared random seed

## Chainpoint Protocol

All Chainpoint transactions follow this json format:

```
{
"type":<string>,
"data":<string>,
"version:<int64>,
"time":<unix;int64>,
"CoreID":<string>,
"Meta":<string>,
"Sig":<ECDSA signature of all fields above>
}
```

Each new Chainpoint Core shall open a lightning channel to all other Cores, then issue a JWK tendermint transaction
that includes their ECDSA public key and lightning public key.

Once a minute, all Cores shall submit the merkle root of all hashes collected over the past minute as a CAL tendermint transaction.

Once every hour, a Core is elected to aggregated all CAL merkle roots over the past hour into a new merkle tree. 
The new merkle root is placed in a OP_RETURN transaction on the bitcoin blockchain. Upon 1 confirmation, the anchoring Core
issues a BTC-A transaction with the associated bitcoin tx body, block height, and ID. 

After 6 confirmations of the previous bitcoin transaction, a validator Core will be elected to confirm the transaction. If the transaction is
confirmed, this Core issues a BTC-C transaction to the rest of the chain. 

Upon receiving a BTC-C transaction, all Cores generate their final btc timestamp proofs for the hour covered by that bitcoin transaction.

## Proof Generation

The Chainpoint Network generates calender proofs every minute and bitcoin proofs roughly every 90 minutes. 
This is essentially a multi-level Merkle Tree that builds until the top root reaches bitcoin.

The initial Merkle Tree construction creates Calendar roots from submitted hashes:

1. A hash is submitted to the Merkle Tree aggregator via the API. 
2. An *agg state* is created, representing the initial hashes being merkelized. 
3. When aggregation states are in the process of being accumulated into a calendar anchor, a *cal state* is created. 
4. The cal states are queried and added to each individual aggregation state to produce a compelete cryptographic chain demonstrating 
inclusion of each hash in the Chainpoint Calendar. The proofs are then cached in the `proof` table in json format for download.

A similar process occurs for turning Calendar roots into BTC proofs:

1. A group of Calendar roots is aggregated into a Merkle tree. The root is submitted to Bitcoin.
2. The submitted Calendar roots are turned into `Anchor Btc Agg State` objects and stored. 
3. Information about the BTC transaction is stored in the `btctx state`.
4. After 6 confirmations, information about the BTC block is used to reconstruct the Merkle Tree in the
Bitcoin block header. This information is stored in `btchead state` table and is used to demonstrate
that our btc transaction is included in the Bitcoin blockchain.
5. The Merkle Trees stored in (2) are added to the Bitcoin block Merkle Tree to show how each calendar root relates
to the Bitcoin block header. The `proof` object for every hash that has a calendar root included in this btc batch 
is then updated with this new information, available for download. 