/**
 * Copyright 2019 Tierion
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

let env = require('../parse-env.js').env
const crypto = require('crypto')

const prefixBuffers = {
  PROOF_TIME_INDEX: Buffer.from('b1a1', 'hex'),
  PROOF_CONTENT: Buffer.from('b1a2', 'hex')
}

const PRUNE_BATCH_SIZE = 1000
const PRUNE_INTERVAL_SECONDS = 10
let PRUNE_IN_PROGRESS = false

let db = null

// #endregion SCHEMAS

/****************************************************************************************************
 * PROOF FUNCTIONS
 ****************************************************************************************************/
// #region PROOF FUNCTIONS

function createBinaryProofContentKey(hashId) {
  let uuidBuffer = Buffer.from(hashId.replace(/-/g, ''), 'hex')
  return Buffer.concat([prefixBuffers.PROOF_CONTENT, uuidBuffer])
}

function createBinaryProofTimeIndexKey() {
  // generate a new key for the current time
  let timestampBuffer = Buffer.alloc(8)
  timestampBuffer.writeDoubleBE(Date.now())
  let rndBuffer = crypto.randomBytes(16)
  return Buffer.concat([prefixBuffers.PROOF_TIME_INDEX, timestampBuffer, rndBuffer])
}

function createBinaryProofTimeIndexMin() {
  // generate the minimum key value for range query
  let minBoundsBuffer = Buffer.alloc(24, 0)
  return Buffer.concat([prefixBuffers.PROOF_TIME_INDEX, minBoundsBuffer])
}

function createBinaryProofTimeIndexMax(timestamp) {
  // generate the maximum key value for range query up to given timestamp
  let timestampBuffer = Buffer.alloc(8, 0)
  timestampBuffer.writeDoubleBE(timestamp)
  let rndBuffer = Buffer.alloc(16, 'ff', 'hex')
  return Buffer.concat([prefixBuffers.PROOF_TIME_INDEX, timestampBuffer, rndBuffer])
}

async function getProofBatchByHashIdsAsync(hashIds) {
  let results = []
  for (let hashId of hashIds) {
    try {
      let proofContentKey = createBinaryProofContentKey(hashId)
      let proofContent = await db.get(proofContentKey)
      let coreProof = JSON.parse(proofContent)
      results.push({
        hashId: hashId,
        proof: coreProof
      })
    } catch (error) {
      if (error.notFound) {
        results.push({
          hashId: hashId,
          proof: null
        })
      } else {
        let err = `Unable to read proof for hash with hashId = ${hashId} : ${error.message}`
        throw err
      }
    }
  }
  return results
}

async function saveProofBatchAsync(proofs) {
  let ops = []

  for (let proof of proofs) {
    let proofContentKey = createBinaryProofContentKey(proof.hash_id_core)
    let proofTimeIndexKey = createBinaryProofTimeIndexKey()
    let proofContent = JSON.stringify(proof)
    ops.push({ type: 'put', key: proofContentKey, value: proofContent })
    ops.push({ type: 'put', key: proofTimeIndexKey, value: proofContentKey })
  }

  try {
    await db.batch(ops)
  } catch (error) {
    let err = `Unable to write proofs : ${error.message}`
    throw err
  }
}

async function pruneProofsSince(timestampMS) {
  return new Promise((resolve, reject) => {
    let delOps = []
    let minKey = createBinaryProofTimeIndexMin()
    let maxKey = createBinaryProofTimeIndexMax(timestampMS)
    db.createReadStream({ gt: minKey, lte: maxKey })
      .on('data', async data => {
        delOps.push({ type: 'del', key: data.key })
        delOps.push({ type: 'del', key: data.value })
        // Execute in batches of PRUNE_BATCH_SIZE
        if (delOps.length >= PRUNE_BATCH_SIZE) {
          try {
            let delOpsBatch = delOps.splice(0)
            await db.batch(delOpsBatch)
          } catch (error) {
            let err = `Error during proof batch delete : ${error.message}`
            return reject(err)
          }
        }
      })
      .on('error', error => {
        let err = `Error reading proof keys for pruning : ${error.message}`
        return reject(err)
      })
      .on('end', async () => {
        try {
          await db.batch(delOps)
        } catch (error) {
          return reject(error.message)
        }
        return resolve()
      })
  })
}

async function pruneOldProofsAsync() {
  let pruneTime = Date.now() - env.PROOF_EXPIRE_MINUTES * 60 * 1000
  try {
    await pruneProofsSince(pruneTime)
  } catch (error) {
    console.error(`ERROR : An error occurred during proof pruning : ${error.message}`)
  }
}

// #endregion PROOF FUNCTIONS

/****************************************************************************************************
 * SET AUTOMATIC PRUNING INTERVALS
 ****************************************************************************************************/
// #region SET AUTOMATIC PRUNING INTERVALS

function startPruningInterval() {
  return setInterval(async () => {
    if (!PRUNE_IN_PROGRESS) {
      PRUNE_IN_PROGRESS = true
      await pruneOldProofsAsync()
      PRUNE_IN_PROGRESS = false
    }
  }, PRUNE_INTERVAL_SECONDS * 1000)
}

// #endregion SET AUTOMATIC PRUNING INTERVALS

function setConnection(rocksConn) {
  db = rocksConn
}

module.exports = {
  getProofBatchByHashIdsAsync: getProofBatchByHashIdsAsync,
  saveProofBatchAsync: saveProofBatchAsync,
  startPruningInterval: startPruningInterval,
  pruneOldProofsAsync: pruneOldProofsAsync,
  setConnection: setConnection
}
