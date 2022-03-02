# Operator's Guide

## Setup

#### Public Chainpoint Network
 1. The Makefile process described in the [quickstart](https://github.com/chainpoint/chainpoint-core#quick-start) will install and configure golang on the target box, as well as build the chainpoint-core binary.  
 2. Manually running the binary will start the setup wizard. Be sure to select the "mainnet" and "Public chainpoint network" options, then put in your server's public IP. 
 3. The in-built Lightning Node will then initialize and print a bitcoin address, as well as the amount of bitcoin necessary to stake with the public Chainpoint Network. Fund this address with the requested amount. 
 4. At this point you can kill the process and daemonize the node for long-term running, if you wish. The commands for this are `make install-daemon` and `make start-daemon`. Running `make log-daemon` will let you see the logs. 
 5. After a few confirmations, your Core will be able to send a JWK message declaring its public key, at which point the stake will be checked and verified by the Validators. 
 6. If you wish, and after a certain amount of successful time on the Network, our nodes can collectively vote to elevate your Core to Validator status. 

#### Private Chainpoint Network
 1. The Makefile process described in the [quickstart](https://github.com/chainpoint/chainpoint-core#quick-start) will install and configure golang on the target box, as well as build the chainpoint-core binary.  
 2. Manually running the binary will start the setup wizard. Be sure to select the "mainnet" and "Standalone Mode" options, then put in your server's public IP.
 3. The in-built Lightning Node will then initialize and print a bitcoin address. You'll need to fund this address with funds sufficient to fund bitcoin anchoring (segwit OP_RETURN transactions). 
 4. If you're joining another private node to create a private Chainpoint network, specify `"SEEDS=<seed_node_id@<seed_node_IP>:26656"` in the config file (by default at `~/.chainpoint/core/core.conf`). The seed node id can be found from retrieving the `id` json field at `http://<seed_node_IP>/status`.
 5. At this point you can kill the process and daemonize the node for long-term running, if you wish. The commands for this are `make install-daemon` and `make start-daemon`. Running `make log-daemon` will let you see the logs. 

## Usage

In high-usage or public-facing situations, it is recommended to use [Chainpoint Gateway](https://github.com/chainpoint/chainpoint-gateway) as a public-facing service in front of your Core. 
It will need to open a channel to your Core in order to submit hashes, or use `AGGREGATOR_WHITELIST=<gateway_ips>` (Core) and `NO_LSAT_CORE_WHITELIST=<core_ips>` (Gateway) to skip lightning usage.

However, when it is possible to use Core directly as a proof generator. Configure your Core to accept requests from your client IPs by adding 
```
AGGREGATOR_WHITELIST=<client_ips>
REMOVE_RATE_LIMITS=true
```
to your config file (by default at `~/.chainpoint/core/core.conf`).


#### Sending Hashes

Sending hashes directly to Core from a whitelisted IP is easy. A `proof_id` is returned by core for later retrieval of the full bitcoin proof:

```
$ curl -s -X POST http://18.220.31.138/hash -H 'Accept: application/json' -H 'Content-Type: application/json' -d '{"hash": "1957db7fe23e4be1740ddeb941ddda7ae0a6b782e536a9e00b5aa82db1e84547"}' | jq
{
  "hash": "1957db7fe23e4be1740ddeb941ddda7ae0a6b782e536a9e00b5aa82db1e84547",
  "proof_id": "59c2c108-998a-11ec-a979-017ffb31ef5e",
  "hash_received": "2022-03-01T18:06:43Z",
  "processing_hints": {
    "cal": "2022-03-01T18:09:03Z",
    "btc": "2022-03-01T19:36:43Z"
  }
}
```

#### Retrieving Proofs

After a maximum of two minutes, a calendar proof can be retrieved from core using the previously-returned `proof_id`. After around 90 minutes, a full btc proof can be retrieved. 
Be advised that since bitcoin mining is probabilistic, the time to retrieve a full bitcoin proof can sometimes be as long as 4 hours. This necessitates an asynchronous retrieval process. 

```
$ curl -s -X GET http://18.220.31.138/proofs -H 'proofids: 59c2c108-998a-11ec-a979-017ffb31ef5e' 
[{"proof":{"@context":"https://w3id.org/chainpoint/v4","branches":
[{"branches":[{"label":"btc_anchor_branch","ops":[
{"r":"2a98bcca858e7b9f3528959ee15a8421e60b257281d6dbbf074233f826cc0e95"},{"op":"sha-256"},
{"r":"b0ebda7c9a16a80b10ff39a0efa9ca26c8d2be7fba388fa9d712dab2caba142d"},{"op":"sha-256"},
{"l":"76a32726a49c0e889e6123ff6824b6e37c65082480811fd541514f6997cc075d"},{"op":"sha-256"},
{"r":"98f9d4a78be411eaf4027dd80c4f16a475aa8c1fb8b80f552dfcd3775abf7bd8"},{"op":"sha-256"},
{"r":"cc5988aeaac11831fa1314b6b97b038282510f3b0b250afa686d2abd7b5eaa61"},{"op":"sha-256"},
{"l":"0100000001e341ad663d29d7f6f7f530db48eb07be790a1d7a6f8c7220c19dcadbc3fd5c4f0100000000ffffffff02ec408b0000000000160014f9fdea9125d41aa4b5356cf7c11a776ca4d727e90000000000000000226a20"},
{"r":"00000000"},{"op":"sha-256-x2"},{"r":"61a72323f0ac816c206f66e0a8f279e10a1c2a6430169961162661e811e9ecff"},
{"op":"sha-256-x2"},{"l":"393948d30e34be670919722707ae0b4f722f0a21f0c87617c9226161c7756789"},{"op":"sha-256-x2"},
{"r":"9d6c7122a74494c8a6940300aeb1d3999fc62ed7e2128452d2cc393b584b037c"},{"op":"sha-256-x2"},
{"r":"9c83b4526a48f45e385d95a1a765e080f2183f25d3078d2cdad657a68fe4af46"},{"op":"sha-256-x2"},
{"r":"b6198ff8058249f628ad33c2dacc1d0b7a40dd3dcb161e47e2ceefa316ce10f1"},{"op":"sha-256-x2"},
{"r":"0509d3af25de7a7c268d04d8e363ea28cec6c50df36983c76d885013b31776d8"},{"op":"sha-256-x2"},
{"r":"c6c86f49e243f69cd5bdcb71cc2e227f8e38515da1859364e814591b21b1fb8f"},{"op":"sha-256-x2"},
{"l":"db9e5216c43002015fc814455d46ed2ab213878c1c68764e53686af2b2fce094"},{"op":"sha-256-x2"},
{"l":"eb30fef8bc4b10c845d32180db53c9e387be90f47beeb18dab27454a8279143b"},{"op":"sha-256-x2"},
{"r":"03700fe3ae2837c1b5aa3a91a15010ac7d578c00406cc1524f4e0eb4f11f0603"},{"op":"sha-256-x2"},
{"l":"e92a41c187b96c47bc2e58152605aa842ab2f9b08b010a117c8feadb019f55aa"},{"op":"sha-256-x2"},
{"r":"214d25db30eba22868317fb8d08f0268baec6e73697a1dcf8e4c82f353db9416"},{"op":"sha-256-x2"},
{"anchors":[{"anchor_id":"725484","type":"btc","uris":["http://18.220.31.138/calendar/cd6377d3e27d9798457725ea8a2d30da857f997ea08b777f8400748837aaf5b7/data"]}]}]}],
"label":"cal_anchor_branch","ops":[{"l":"drand:1690899:b47ac925c10114b30623cbc3c5444a4bc888f7a7ad0c4c4d1cf14f8984da7077"},
{"op":"sha-256"},{"anchors":[{"anchor_id":"cde5302e29b9c9596d775feccd36be72af76fce240468b3fdb047f0eb262c5b8","type":"cal",
"uris":["http://18.220.31.138/calendar/cde5302e29b9c9596d775feccd36be72af76fce240468b3fdb047f0eb262c5b8/data"]}]}]}],
"hash":"1957db7fe23e4be1740ddeb941ddda7ae0a6b782e536a9e00b5aa82db1e84547","hash_received":"2022-03-01T18:06:43Z",
"proof_id":"59c2c108-998a-11ec-a979-017ffb31ef5e","type":"Chainpoint"},"proof_id":"59c2c108-998a-11ec-a979-017ffb31ef5e"}]

```

#### Validating Proofs

Chainpoint offers javascript libraries to validate the proof schema and anchors inside a retrieved proof. 

```javascript
const chainpointProofSchema = require('chainpoint-proof-json-schema')
const chp = require('chainpoint-js')

async function run() {
    let objectToValidate = {"@context":"https://w3id.org/chainpoint/v4","branches":[{"branches":[{"branches":[{"label":"btc_anchor_branch","ops":[{"r":"32fe2cdf8e55ce5998ce1962a306eb78a6bb038118f79db0b16339f07919720e"},{"op":"sha-256"},{"l":"4946a39a5886a86b3de22a1d69c2ad5fd88653e8e4cd74d96abaa7d94f096a10"},{"op":"sha-256"},{"l":"2e00187b6ab2680c58634c6adafebb7de87b74a7a809a7b4760ad69b17f1e825"},{"op":"sha-256"},{"l":"a7773c0e5ab87f795e74405a4f1682cd2de3952eb5e9a62535afef8297700276"},{"op":"sha-256"},{"r":"10d55f097179478b36954751d1732bfc98696f94654757e82d185f86b9a1bf1d"},{"op":"sha-256"},{"r":"4480d51200905adfd17422724e1527e301c8d5f7500e6d5453c2fa02c3321ff6"},{"op":"sha-256"},{"r":"a2ac89fa1fb0cf47dc3c1982872bf371905c125e20c165c77b3a94a9aecb8f29"},{"op":"sha-256"},{"l":"0100000001899649e6c377a9610be77649a436813291bffbbb54a4fe741c7809ac26fcdfaa0000000000ffffffff0216d5be0100000000160014f956e78e15c0bf40572caa74432097ccd75e856a0000000000000000226a20"},{"r":"00000000"},{"op":"sha-256-x2"},{"l":"2c4eea3b138eedfeafea8f676264323affe29566ff8f193aba48f25883342915"},{"op":"sha-256-x2"},{"l":"86fe9fa7badfa77cdce14e6d2825efba128683f33f30de650a65e357319d233b"},{"op":"sha-256-x2"},{"l":"86aa499d24376725e65e3e9d38e69585eb193b19c603a0342c469f586f1ca9dc"},{"op":"sha-256-x2"},{"l":"44925866b9d547ae569d45429e890c78dbd4df21177c95cb63b9e37c8f69eb4a"},{"op":"sha-256-x2"},{"r":"3c8977ede14a2449b55f5a903538b380b58a113119fd73d0f26b3e5449f784ac"},{"op":"sha-256-x2"},{"r":"a432d88d239a940065d62a26690ce1dfe1a9e21bf95084cf94752130176690d3"},{"op":"sha-256-x2"},{"l":"7b42eef3988fda04aca785dd281eb87cc618884547524885d902eaf03fffe361"},{"op":"sha-256-x2"},{"r":"42559a2806bfc7c33dd93e8eee06ef29518ca5f0ed2ef7e664a0076d576e9541"},{"op":"sha-256-x2"},{"l":"831e38115957ca54290a5af2723487fb8983259caeea48290513eaa6d3597005"},{"op":"sha-256-x2"},{"r":"ae8c68e46f87f2e31d8ed7c63c74ead5cdc122f2f04530bb93a04183c0fa6b33"},{"op":"sha-256-x2"},{"r":"cff5d1868a9a83c80099e5b59cf885f7a44890a92629172a1ec1f377daec212b"},{"op":"sha-256-x2"},{"anchors":[{"anchor_id":"705514","type":"btc","uris":["http://52.14.86.247/calendar/9f55e60635aa9478a1cb1cb011a04e7269a94fb2b36b1431a0d68dc36b913cd8/data"]}]}]}],"label":"cal_anchor_branch","ops":[{"l":"drand:1303435:efdbec6da24a03fd011f08721a7357611a766c8db47ab353484a12a841042869"},{"op":"sha-256"},{"l":"fa50bc6d9d327f709fef63cddfc964956197e76f82ccb432916c0929d4b1f858"},{"op":"sha-256"},{"anchors":[{"anchor_id":"590f4dd649d414b27290b3ac3d029fc3f74805e88d8eaca21387f39d886c05e4","type":"cal","uris":["http://tendermint.chainpoint.org/calendar/590f4dd649d414b27290b3ac3d029fc3f74805e88d8eaca21387f39d886c05e4/data"]}]}]}],"label":"aggregator","ops":[{"l":"c0f19aa696735543f7075c5c1b32f3c87101fd306fee3fc39b32a05372384a61"},{"op":"sha-256"},{"l":"d4d0c165172969b7b31b1f5770ed4cf3082bb50624a1187edb04fe7c02e4d02f"},{"op":"sha-256"}]}],"hash":"f94ecc316d8f43112a58d102c962d72c250e923f22bee5ec2b5965978072e190","hash_received":"2021-10-18T05:14:05Z","proof_id":"3728c862-2fd2-11ec-a614-016576e002b3","type":"Chainpoint"}
    
    // validate proof schema
    let res = chainpointProofSchema.validate(objectToValidate)
    if (res.valid) {
        console.log('valid')
    } else {
        console.log(res.errors)
    }

    /// validate anchors
    let verifiedProofs = await chp.verifyProofs([objectToValidate])
    console.log('Verified Proof Objects: Expand objects below to inspect.')
    console.log(verifiedProofs)
}
run()
```

#### Retrieving the Merkle Root of a Calendar Anchor

This is used during proof verification to confirm the expected Merkle Root of an anchor. 

```
$ curl http://18.220.31.138/calendar/cd6377d3e27d9798457725ea8a2d30da857f997ea08b777f8400748837aaf5b7/data
032d612fda2c2df9420dad0c6504a638102efdf6897702acfc859ae519966070
```

#### Retrieving Core Status

```
$ curl http://18.220.31.138/status
{"version":"0.0.2","time":"2022-03-02T17:49:10.696Z","base_uri":"http://0.0.0.0",
"jwk":{"kty":"EC","kid":"24ba3a2556ebae073b42d94815836b29594a2456","crv":"P-256","x":"JIVErwpm7UK-LphlGEuCq3kAr5NBIwsJu9EOPifsSG0","y":"E6OMwt1lslzujOpUFdfiwsZxxZBBT0m1QVQiSmofajM"},
"network":"mainnet","identity_pubkey":"02108182a754e0d0e42e7dcbc9d79f145e51afcc3b49ee6a2463d8999274f8aa4f",
"lightning_address":"bc1qa2nddalfe5glzknztujpp4asmy3aw0k8vaek64","lightning_balance":{"total_balance":"15766444","confirmed_balance":"15766444","unconfirmed_balance":"0"},
"public_key":"","uris":["02108182a754e0d0e42e7dcbc9d79f145e51afcc3b49ee6a2463d8999274f8aa4f@18.220.31.138:9735"],
"alias":"02108182a754e0d0e42e","hash_price_satoshis":2,"total_stake_price":6000000,"validator_stake_price":1200000,
"num_channels_count":4,"node_info":{"protocol_version":{"p2p":7,"block":10,"app":1},"id":"24ba3a2556ebae073b42d94815836b29594a2456",
"listen_addr":"18.220.31.138:26656","network":"mainnet-chain-32","version":"0.33.5","channels":"4020212223303800",
"moniker":"46e15ad75513","other":{"tx_index":"on","rpc_address":"tcp://0.0.0.0:26657"}},
"sync_info":{"latest_block_hash":"31711B1D07AF30995BAEFB36860E77D572B224069B185E7056A2F163A22DDF3B",
"latest_app_hash":"8AA2600000000000","latest_block_height":788615,"latest_block_time":"2022-03-02T17:47:51.087775422Z",
"earliest_block_hash":"79615814D4F75FD6FDD9DA09A4CCB95F6B91E1B2F7F7A59242F925581225DDC4","earliest_app_hash":"",
"earliest_block_height":1,"earliest_block_time":"2020-03-10T21:50:32.59624701Z","catching_up":false}}
```

#### Retrieving Core Peers

```
$ curl http://18.220.31.138/peers
["3.142.136.148","3.95.20.189","18.118.26.31","3.133.161.241","18.220.31.138","3.145.43.113"]
```

#### Retrieving Public Gateways

```
$ curl http://18.220.31.138/gateways/public
["18.224.185.143","3.133.135.157","18.191.50.129"]
```

## Updates

Core uses semantic versioning:
- Non-consensus-related updates and critical fixes will result in a patch (point release, ie 1.3.0 => 1.3.1)
- New features will result in a minor version release (ie 1.3.1 => 1.4.0)
- Any changes to the consensus protocol will result in a major version release (ie 1.4.0 => 2.0.0) 

Any updates to the consensus protocol will require the cooperation of all Core operators. 
Notice of an available update will be given through email for regular Core operators, whereas Validator operators will coordinate in real time over Discord. 

## Validators

Chainpoint Validators have the final responsibility for forming blocks and validating transactions.
They also must vote to add additional validators and increase/decrease the lightning stake required to join the network

### Adding Validators

1. New person or entity applies to be a validator node. 
2. Provision a server, clone the Chainpoint-Core repo, and run `make install-deps` and `make build`
3. Upon running Core for the first time, a setup wizard will do the following:
    1. Generate the node's public key and ID
    2. Ingest the new node’s IP
4. Upon 2nd start (normal operation), the node will print something like
    ```
   Core ID set                                  ID=AF12ACE1A4058F4E60723930E96200EB605D0B36
   Core Tendermint Publickey set                Key=+MU67U5bacm7H/2ZWaAltvchl7RyXwHJ8pl6lIq7zYw=
   ```
   These must be given to the validator agreeing to run the next steps.
5. An existing validator operator aggrees to broadcast a VAL transaction at a particular block height agreed upon by the rest of the Validators. 
    1. This consists of starting all validators with the argument `proposed_val=val:<ID>!<b64_public_key>!<voting_power>!<block_height>` to the network, where
        - ID is the Tendermint ID of the node, shown above 
        - PubKey is the base 64-encoded tendermint public key of the node
        - Voting power is the amount of weight the node has while voting on new blocks (should be 1 in nearly all cases)
        - Block height is the height the new validator will take effect at. 
        
6. An existing validator whitelists the new node operator’s IP, opening up ports 26656 and 26657 to the new validator. 
7. The new validator begins operating their node with the `make up` command. It will fastsync and begin voting on new blocks. 

### Change Required Lightning Stake

1. Upon setting the `update_stake` config value, a core will automatically submit a `CHNGSTK` tx to tendermint consensus.
2. Each other core will only confirm `CHNGSTK` if the value is equal to their existing `stake_per_core` configuration value
    1. In practice this means that 2/3 of all validators must share this configuration value. 
    2. If the value changes then these Cores must restart in order to get the new config value, so they can approve the `CHNGSTK` tx
3. Upon initializing, Cores will automatically read the latest `CHNGSTK` value from the tendermint index and use this value for the staking requirement 

## Support

Email:    `team@tierion.com`, `ops@tierion.com`

Report Bugs on our Github: https://github.com/chainpoint/chainpoint-core/issues

We also have a discord. Please email us for an invite. 