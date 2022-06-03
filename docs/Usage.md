## API

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

#### Upgrading Proofs

If a `cal` proof but not corresponding `btc` proof was retrieved, it is possible to reconstruct or "upgrade" the btc portion of the proof 
using just the `cal` proof anchor_id:

```
$ curl -s -X GET http://3.142.136.148/proofs/upgrade/ca8d25863c51186c0b7d59ffd57df651255c664d584bb2c0636e04c148c84fdd
{"branches":[{"label":"btc_anchor_branch","ops":[{"l":"af483366b679b83859a7b2f3b505f5e794fd0b57b50b9a049b5a3b22c369e68b"},
{"op":"sha-256"},{"r":"5d5578a3832afa52af610c56a62d95f2745fff5647e2a83c47a41cf97ec90e15"},
{"op":"sha-256"},{"l":"d4c0aa28c61e47f12ae4d47bafe59fad63d127dd5cdd2ffdcc421db69c5bfe91"},
{"op":"sha-256"},{"l":"3dbb09bf32464e45bc5466f592bf8066366a9e3453528fae8bf1e918110d0799"},
{"op":"sha-256"},{"l":"72ff745d0c507fc5bd398e73c77c0291e516eb8fb35cbaffa1b7541565e85a22"},
{"op":"sha-256"},{"r":"013cafaf46c15ce9643a092493683eac5b92d17b3100729015a57a06986b7976"},
{"op":"sha-256"},{"l":"01000000014b198fab45c246c4af71826bc4231c1632febb176b249c4d25dad2452ba4aad10100000000ffffffff020000000000000000226a20"},
{"r":"3b6e840000000000160014b71096de8310c8d3b565019b7ddcdc4a26f85c1000000000"},
{"op":"sha-256-x2"},{"r":"92ef83d958ff07487906c02be4cbf79ed6233650f098b005e5166c81e96ad949"},
{"op":"sha-256-x2"},{"l":"3d7d94983b695355af6fb2de85832cfa1f16972c76772a09f5e010997cade9a7"},
{"op":"sha-256-x2"},{"l":"7b5f0f17e6fba60ea05d077bac896e832f1a52daa8591f7bf684f068a824c46a"},
{"op":"sha-256-x2"},{"l":"cc44fad191483298da1d3739e4f90eda2a809a96127f5cd867bd58150561eb2b"},
{"op":"sha-256-x2"},{"r":"637514a0a982573badafe0b3e975a08f7b548a8dc1d1d5c7dd1896a76dc896f8"},
{"op":"sha-256-x2"},{"r":"673a6852f3b1fffa4af9d384a8ede4602af4cc67c8f5aa26b5542fb5ca88c2fc"},
{"op":"sha-256-x2"},{"l":"0ebaedbdde592991464b5b02cc9ed42b32fe7e2411a15dc2cec1efc0c93d7d52"},
{"op":"sha-256-x2"},{"r":"9868dfa316cdbf5b66e1ce2d3ec0729b433d4158923581c1093c9024a866a8f8"},
{"op":"sha-256-x2"},{"l":"173f54b28ce2422f44a025d0f769baed52918b6eeef3468dfbacb4e8a6f67c26"},
{"op":"sha-256-x2"},{"r":"51469f8df770a9d569a0a4479f7436ab1d1d282163c0c1f50e2a63dd50b3f540"},
{"op":"sha-256-x2"},{"r":"3dc01a32178a25507f65c47f13dcee4a23916abe059abd6cc1b7b1d5d7ce3e1b"},
{"op":"sha-256-x2"},{"r":"9e6d375b42dc108d10b242b43281994c95059a87bd8c2e9782fb4b0e20d95de9"},
{"op":"sha-256-x2"},
{"anchors":[{"anchor_id":"738916","type":"btc","uris":["http://3.142.136.148/calendar/3b4f36c00b450f60852e841c30a7e263dc5695be3ce5057e5b240e20a659fe15/data"]}]}]}]}
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