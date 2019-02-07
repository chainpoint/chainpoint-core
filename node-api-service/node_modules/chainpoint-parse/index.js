/* Copyright 2017 Tierion
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*     http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/

const crypto = require('crypto')
const sha3512 = require('js-sha3').sha3_512
const sha3384 = require('js-sha3').sha3_384
const sha3256 = require('js-sha3').sha3_256
const sha3224 = require('js-sha3').sha3_224
const chpSchema = require('chainpoint-proof-json-schema')
const chpBinary = require('chainpoint-binary')

function parse (chainpointObject, callback) {
  // if the supplied is a Buffer, Hex, or Base64 string, convert to JS object
  if (typeof chainpointObject === 'string' || Buffer.isBuffer(chainpointObject)) chainpointObject = chpBinary.binaryToObjectSync(chainpointObject)

  let schemaCheck = chpSchema.validate(chainpointObject)
  if (!schemaCheck.valid) throw new Error(schemaCheck.errors)

  // initialize the result object
  let result = {}
  // identify this result set with the basic information on the hash
  result.hash = chainpointObject.hash
  result.hash_id_node = chainpointObject.hash_id_node
  result.hash_submitted_node_at = chainpointObject.hash_submitted_node_at
  result.hash_id_core = chainpointObject.hash_id_core
  result.hash_submitted_core_at = chainpointObject.hash_submitted_core_at
  // acquire all anchor points and calcaulte expected values for all branches, recursively
  result.branches = parseBranches(chainpointObject.hash, chainpointObject.branches)
  return result
}

function parseBranches (startHash, branchArray) {
  var branches = []
  var currentHashValue = Buffer.from(startHash, 'hex')

  // iterate through all branches in the current branch array
  for (var b = 0; b < branchArray.length; b++) {
    // initialize anchors array for this branch
    let anchors = []
    // iterate through all operations in the operations array for this branch
    let currentbranchOps = branchArray[b].ops
    for (var o = 0; o < currentbranchOps.length; o++) {
      if (currentbranchOps[o].r) {
        // hex data gets treated as hex, otherwise it is converted to bytes assuming a ut8 encoded string
        let concatValue = isHex(currentbranchOps[o].r) ? Buffer.from(currentbranchOps[o].r, 'hex') : Buffer.from(currentbranchOps[o].r, 'utf8')
        currentHashValue = Buffer.concat([currentHashValue, concatValue])
      } else if (currentbranchOps[o].l) {
        // hex data gets treated as hex, otherwise it is converted to bytes assuming a ut8 encoded string
        let concatValue = isHex(currentbranchOps[o].l) ? Buffer.from(currentbranchOps[o].l, 'hex') : Buffer.from(currentbranchOps[o].l, 'utf8')
        currentHashValue = Buffer.concat([concatValue, currentHashValue])
      } else if (currentbranchOps[o].op) {
        switch (currentbranchOps[o].op) {
          case 'sha-224':
            currentHashValue = crypto.createHash('sha224').update(currentHashValue).digest()
            break
          case 'sha-256':
            currentHashValue = crypto.createHash('sha256').update(currentHashValue).digest()
            break
          case 'sha-384':
            currentHashValue = crypto.createHash('sha384').update(currentHashValue).digest()
            break
          case 'sha-512':
            currentHashValue = crypto.createHash('sha512').update(currentHashValue).digest()
            break
          case 'sha3-224':
            currentHashValue = Buffer.from(sha3224.array(currentHashValue))
            break
          case 'sha3-256':
            currentHashValue = Buffer.from(sha3256.array(currentHashValue))
            break
          case 'sha3-384':
            currentHashValue = Buffer.from(sha3384.array(currentHashValue))
            break
          case 'sha3-512':
            currentHashValue = Buffer.from(sha3512.array(currentHashValue))
            break
          case 'sha-256-x2':
            currentHashValue = crypto.createHash('sha256').update(currentHashValue).digest()
            currentHashValue = crypto.createHash('sha256').update(currentHashValue).digest()
            break
        }
      } else if (currentbranchOps[o].anchors) {
        anchors = anchors.concat(parseAnchors(currentHashValue, currentbranchOps[o].anchors))
      }
    }

    let branchObj = {
      label: branchArray[b].label || undefined,
      anchors: anchors
    }
    if (branchArray[b].branches) branchObj.branches = parseBranches(currentHashValue.toString('hex'), branchArray[b].branches)

    // if this branch is a standard Chaipoint BTC anchor branch,
    // output the OP_RETURN value and the BTC transaction id
    if (branchObj.label === 'btc_anchor_branch') {
      let btcAnchorInfo = getBtcAnchorInfo(startHash, currentbranchOps)
      branchObj.opReturnValue = btcAnchorInfo.opReturnValue
      branchObj.btcTxId = btcAnchorInfo.btcTxId
    }

    branches.push(branchObj)
  }

  return branches
}

function parseAnchors (currentHashValue, anchorsArray) {
  var anchors = []
  for (var x = 0; x < anchorsArray.length; x++) {
    let expectedValue = currentHashValue.toString('hex')
    // BTC merkle root values is in little endian byte order
    // All hashes and calculations in a Chainpoint proof are in big endian byte order
    // If we are determining the expected value for a BTC anchor, the expected value
    // result byte order must be reversed to match the BTC merkle root byte order
    // before making any comparisons
    if (anchorsArray[x].type === 'btc') expectedValue = expectedValue.match(/.{2}/g).reverse().join('')
    anchors.push(
      {
        type: anchorsArray[x].type,
        anchor_id: anchorsArray[x].anchor_id,
        uris: anchorsArray[x].uris || undefined,
        expected_value: expectedValue
      }
    )
  }
  return anchors
}

function getBtcAnchorInfo (startHash, ops) {
  // This calculation depends on the branch using the standard format
  // for btc_anchor_branch type branches created by Chainpoint services
  let currentHashValue = Buffer.from(startHash, 'hex')
  let has256x2 = false
  let isFirst256x2 = false

  let opResultTable = ops.map((op) => {
    if (op.r) {
      // hex data gets treated as hex, otherwise it is converted to bytes assuming a ut8 encoded string
      let concatValue = isHex(op.r) ? Buffer.from(op.r, 'hex') : Buffer.from(op.r, 'utf8')
      currentHashValue = Buffer.concat([currentHashValue, concatValue])
      return { opResult: currentHashValue, op: op, isFirst256x2: isFirst256x2 }
    } else if (op.l) {
      // hex data gets treated as hex, otherwise it is converted to bytes assuming a ut8 encoded string
      let concatValue = isHex(op.l) ? Buffer.from(op.l, 'hex') : Buffer.from(op.l, 'utf8')
      currentHashValue = Buffer.concat([concatValue, currentHashValue])
      return { opResult: currentHashValue, op: op, isFirst256x2: isFirst256x2 }
    } else if (op.op) {
      switch (op.op) {
        case 'sha-224':
          currentHashValue = crypto.createHash('sha224').update(currentHashValue).digest()
          break
        case 'sha-256':
          currentHashValue = crypto.createHash('sha256').update(currentHashValue).digest()
          break
        case 'sha-384':
          currentHashValue = crypto.createHash('sha384').update(currentHashValue).digest()
          break
        case 'sha-512':
          currentHashValue = crypto.createHash('sha512').update(currentHashValue).digest()
          break
        case 'sha3-224':
          currentHashValue = Buffer.from(sha3224.array(currentHashValue))
          break
        case 'sha3-256':
          currentHashValue = Buffer.from(sha3256.array(currentHashValue))
          break
        case 'sha3-384':
          currentHashValue = Buffer.from(sha3384.array(currentHashValue))
          break
        case 'sha3-512':
          currentHashValue = Buffer.from(sha3512.array(currentHashValue))
          break
        case 'sha-256-x2':
          currentHashValue = crypto.createHash('sha256').update(currentHashValue).digest()
          currentHashValue = crypto.createHash('sha256').update(currentHashValue).digest()
          if (!has256x2) {
            isFirst256x2 = true
            has256x2 = true
          } else {
            isFirst256x2 = false
          }
          break
      }
      return { opResult: currentHashValue, op: op, isFirst256x2: isFirst256x2 }
    }
  })

  let btcTxIdOpIndex = opResultTable.findIndex((result) => { return result.isFirst256x2 })
  let opReturnOpIndex = btcTxIdOpIndex - 3

  return {
    opReturnValue: opResultTable[opReturnOpIndex].opResult.toString('hex'),
    btcTxId: opResultTable[btcTxIdOpIndex].opResult.toString('hex').match(/.{2}/g).reverse().join('')
  }
}

function isHex (value) {
  var hexRegex = /^[0-9A-Fa-f]{2,}$/
  var result = hexRegex.test(value)
  if (result) result = !(value.length % 2)
  return result
}

module.exports = {
  parse: parse
}
