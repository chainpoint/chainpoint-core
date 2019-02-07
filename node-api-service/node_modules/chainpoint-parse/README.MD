# Chainpoint Parse

[![npm](https://img.shields.io/npm/l/chainpoint-parse.svg)](https://www.npmjs.com/package/chainpoint-parse)
[![npm](https://img.shields.io/npm/v/chainpoint-parse.svg)](https://www.npmjs.com/package/chainpoint-parse)

A Javascript library for parsing a [Chainpoint v3 proof](https://chainpoint.org/) and discovering all anchor points in the proof as well as the expected values to be stored at those anchor points.

Note: This is a low level tool meant for calculating the proof operations and determining the values claimed to be stored at the proof's anchor points. This tool will not perform the actual proof verification against the Chainpoint
Calendar or other public blockchains. It simply calculates locally the values
that you should expect to see in the URI's to validation data provided.

## Installation

```
$ npm install --save chainpoint-parse
```

or

```
yarn add chainpoint-parse
```

## Usage

### Node.js

#### `parse`

This function parses a [Chainpoint v3 proof](https://chainpoint.org/). chainpointProofObject may be a proof in Javascript Object form, Buffer, Hex string, or Base64 string.

```js
const chainpointParse = require('chainpoint-parse')

// Valid proof in JS Object, Buffer, Hex String, or Base64 String form
let chainpointProofObject = {...} 

try {
  let result = chainpointParse.parse(chainpointProofObject)
} catch (error) {
  console.error(`An error has occurred: ${error.message}`)
}
```

### Parse Result Object

The Result object contains basic information about the hash referenced in the proof, and the anchor data for that hash. The anchor data contains both the locations and the value the proof claims shoulds exist at those locations. The original branch structure of the proof is preserved, and the anchor data is included in the approriate branch. For btc_anchor_branch branches, the OP_RETURN value anchored and BTC transaction id it was included in are also returned.

The following is an example of a parse result object: 
```json
{  
  "hash":"ffff27222fe366d0b8988b7312c6ba60ee422418d92b62cdcb71fe2991ee7391",
  "hash_id_node":"66a34bd0-f4e7-11e7-a52b-016a36a9d789",
  "hash_submitted_node_at":"2018-01-09T02:47:15Z",
  "hash_id_core":"66bd6380-f4e7-11e7-895d-0176dc2220aa",
  "hash_submitted_core_at":"2018-01-09T02:47:15Z",
  "branches":[  
    {  
      "label":"cal_anchor_branch",
      "anchors":[  
        {  
          "type":"cal",
          "anchor_id":"985635",
          "uris":[  
            "https://a.chainpoint.org/calendar/985635/hash"
          ],
          "expected_value":"4690932f928fb7f7ce6e6c49ee95851742231709360be28b7ce2af7b92cfa95b"
        }
      ],
      "branches":[  
        {  
          "label":"btc_anchor_branch",
          "anchors":[  
            {  
              "type":"btc",
              "anchor_id":"503275",
              "uris":[  
                "https://a.chainpoint.org/calendar/985814/data"
              ],
              "expected_value":"c617f5faca34474bea7020d75c39cb8427a32145f9646586ecb9184002131ad9"
            }
          ],
          "opReturnValue":"267335262e21e7adb4220068b4b90b7ff066324935d7f61ceab2a64080b06b1b",
          "btcTxId":"ba3c8c3e547ed73471c28a69659373f3f0a3b726aab31cdecd14513d9c581f1e"
        }
      ]
    }
  ]
}
```

The parse result object is similar to the original proof in most ways. The key difference is that all the hash operations have been performed and extracted from the proof. The `expected_value` is inserted for each anchor point as a result of these calculations. With these `expected_value` determined, you can proceed to validate these anchor claims against an external blockchain source.

NOTE: The `expected_value` byte order is reversed for all BTC anchor results. This is because all hashes and calculations in a Chainpoint proof are in big endian byte order, but BTC Merkle root values are in little endian byte order in Bitcoin, and as a result, in the Tierion Calendar as well. The value must be reversed before any comparisons are made.


### Browser

You can copy `bundle.js` into your app to include in a script tag.

Or install the npm package in a place available to your web pages and set the script `src` tag to something like the following. A window global function `chainpointParse.parse()` will be available and operate the same as the Node.js example above.


```
<script src="./node_modules/chainpoint-parse/bundle.js">

```
