/* Copyright (C) 2018 Tierion
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

// load all environment variables into env object
const env = require('./lib/parse-env.js')('gen')

const amqp = require('amqplib')
const chainpointProofSchema = require('chainpoint-proof-json-schema')
const uuidTime = require('uuid-time')
const chpBinary = require('chainpoint-binary')
const utils = require('./lib/utils.js')
const cnsl = require('consul')
const GCPStorage = require('@google-cloud/storage')
const retry = require('async-retry')
const crypto = require('crypto')
const moment = require('moment')
const connections = require('./lib/connections.js')
const parallel = require('async-parallel')

let consul = null

const aggState = require('./lib/models/AggState.js')
const calState = require('./lib/models/CalState.js')
const anchorBtcAggState = require('./lib/models/AnchorBtcAggState.js')
const btcTxState = require('./lib/models/BtcTxState.js')
const btcHeadState = require('./lib/models/BtcHeadState.js')
const cachedProofState = require('./lib/models/cachedProofState.js')

// Variable indicating what proof storage flow to use
// Acceptable values are:
// 'resque' for the Resque queue to proof proxy flow
// 'direct' to write directly to GCP from proof-gen
// 'both' to do both
// Any other value will default to 'resque'
let proofStorageMethod = 'resque'

// The channel used for all amqp communication
// This value is set once the connection has been established
let amqpChannel = null

// The redis connection used for all redis communication
// This value is set once the connection has been established
let redis = null

// This value is set once the connection has been established
let taskQueue = null

// Google Cloud Storage client
const gcpStorage = new GCPStorage({ projectId: env.GCP_STORAGE_PROJECTID })
const gcpBucket = gcpStorage.bucket(env.GCP_STORAGE_BUCKET)

function addChainpointHeader (proof, hash, hashId) {
  proof['@context'] = 'https://w3id.org/chainpoint/v3'
  proof.type = 'Chainpoint'
  proof.hash = hash

  // the following two values are added as placeholders
  // the spec does not allow for missing or empty values here
  // these values will be replaced with proper ones by the Node instance
  proof.hash_id_node = hashId
  proof.hash_submitted_node_at = utils.formatDateISO8601NoMs(new Date(parseInt(uuidTime.v1(hashId))))

  proof.hash_id_core = hashId
  proof.hash_submitted_core_at = proof.hash_submitted_node_at
  return proof
}

function addCalendarBranch (proof, aggState, calState) {
  let calendarBranch = {}
  calendarBranch.label = 'cal_anchor_branch'
  calendarBranch.ops = aggState.ops.concat(calState.ops)

  let calendarAnchor = {}
  calendarAnchor.type = 'cal'
  calendarAnchor.anchor_id = calState.anchor.anchor_id
  calendarAnchor.uris = calState.anchor.uris

  calendarBranch.ops.push({ anchors: [calendarAnchor] })

  proof.branches = [calendarBranch]
  return proof
}

function addBtcBranch (proof, anchorBTCAggState, btcTxState, btcHeadState) {
  let btcBranch = {}
  btcBranch.label = 'btc_anchor_branch'
  btcBranch.ops = anchorBTCAggState.ops.concat(btcTxState.ops, btcHeadState.ops)

  let btcAnchor = {}
  btcAnchor.type = 'btc'
  btcAnchor.anchor_id = btcHeadState.anchor.anchor_id
  btcAnchor.uris = btcHeadState.anchor.uris

  btcBranch.ops.push({ anchors: [btcAnchor] })

  proof.branches[0].branches = [btcBranch]
  return proof
}

/**
* Retrieves all proof state data for a given hash and initiates proof generation
*
* @param {amqp message object} msg - The AMQP message received from the queue
*/
async function consumeProofReadyMessageAsync (msg) {
  let messageObj = JSON.parse(msg.content.toString())

  switch (msg.properties.type) {
    case 'cal_batch':
      try {
        let hashIds = messageObj.hash_ids
        let aggStateRows = await cachedProofState.getAggStateObjectsByHashIdsAsync(hashIds)
        let aggIds = aggStateRows.map((item) => item.agg_id)
        let calStateRows = await cachedProofState.getCalStateObjectsByAggIdsAsync(aggIds)
        // create a lookup table for calStateRows by agg_id
        let calStateLookup = calStateRows.reduce((result, calStateRow) => {
          result[calStateRow.agg_id] = calStateRow.cal_state
          return result
        }, {})

        let proofs = aggStateRows.map((aggStateRow) => {
          let proof = {}
          proof = addChainpointHeader(proof, aggStateRow.hash, aggStateRow.hash_id)
          proof = addCalendarBranch(proof, JSON.parse(aggStateRow.agg_state), JSON.parse(calStateLookup[aggStateRow.agg_id]))

          // ensure the proof is valid according to the defined Chainpoint v3 JSON schema
          let isValidSchema = chainpointProofSchema.validate(proof).valid
          if (!isValidSchema) {
            console.error(`Proof ${aggStateRow.hash_id} has an invalid JSON schema`)
            return null
          }
          return proof
        }).filter((proof) => proof !== null)

        // if taskQueue is null (redis outage), wait one second for recovery,
        // throw error to initiate nack and retry
        if (taskQueue === null) {
          await utils.sleep(1000)
          throw new Error(`Unable to queue up cal storeProofs jobs, taskQueue is null`)
        }
        await storeProofsAsync(proofs, 'cal_batch')

        // Proof ready message has been consumed, ack consumption of original message
        amqpChannel.ack(msg)
        console.log(msg.fields.routingKey, '[' + msg.properties.type + '] consume message acked')
      } catch (error) {
        console.error(`Unable to process proof ready message: ${error.message}`)
        // An error as occurred consuming a message, nack consumption of original message
        amqpChannel.nack(msg)
        console.error(`${msg.fields.routingKey} [${msg.properties.type}] consume message nacked: ${error.message}`)
      }
      break
    case 'btc_batch':
      try {
        let hashIds = messageObj.hash_ids
        let aggStateRows = await cachedProofState.getAggStateObjectsByHashIdsAsync(hashIds)
        let aggIds = aggStateRows.map((item) => item.agg_id)
        let calStateRows = await cachedProofState.getCalStateObjectsByAggIdsAsync(aggIds)
        let calIds = calStateRows.map((item) => item.cal_id)
        let anchorBTCAggStateRows = await cachedProofState.getAnchorBTCAggStateObjectsByCalIdsAsync(calIds)
        let anchorBTCAggIds = anchorBTCAggStateRows.map((item) => item.anchor_btc_agg_id)

        let btcTxStateRow, btcHeadStateRow
        try {
          // if any of these calls fail, there is an unrecoverable problem with the proof state data for these hash_ids
          // in this case, we log an error message and ack the message since it will never be able to process successfully
          //
          // all the anchorBTCAggIds should be the same, all being the event id for this btc transaction
          // use the first one as the identifier to retrieve the remaining 1-to-1 state
          let anchorBTCAggId = anchorBTCAggIds[0]
          btcTxStateRow = await cachedProofState.getBTCTxStateObjectByAnchorBTCAggIdAsync(anchorBTCAggId)
          btcHeadStateRow = await cachedProofState.getBTCHeadStateObjectByBTCTxIdAsync(btcTxStateRow.btctx_id)
        } catch (error) {
          console.error(`Unrecoverable proof state read error for hash_ids ${hashIds} : ${error.message}`)
          amqpChannel.ack(msg)
          return
        }

        // create a lookup table for calStateRows by agg_id
        let calStateLookup = calStateRows.reduce((result, calStateRow) => {
          result[calStateRow.agg_id] = { cal_id: calStateRow.cal_id, state: calStateRow.cal_state }
          return result
        }, {})
        // create a lookup table for anchorBTCAggStateRows by cal_id
        let anchorBTCAggStateLookup = anchorBTCAggStateRows.reduce((result, anchorBTCAggStateRow) => {
          result[anchorBTCAggStateRow.cal_id] = anchorBTCAggStateRow.anchor_btc_agg_state
          return result
        }, {})

        let btcTxState = JSON.parse(btcTxStateRow.btctx_state)
        let btcHeadState = JSON.parse(btcHeadStateRow.btchead_state)

        let proofs = aggStateRows.map((aggStateRow) => {
          let proof = {}
          proof = addChainpointHeader(proof, aggStateRow.hash, aggStateRow.hash_id)
          proof = addCalendarBranch(proof, JSON.parse(aggStateRow.agg_state), JSON.parse(calStateLookup[aggStateRow.agg_id].state))
          proof = addBtcBranch(proof, JSON.parse(anchorBTCAggStateLookup[calStateLookup[aggStateRow.agg_id].cal_id]), btcTxState, btcHeadState)

          // ensure the proof is valid according to the defined Chainpoint v3 JSON schema
          let isValidSchema = chainpointProofSchema.validate(proof).valid
          if (!isValidSchema) {
            console.error(`Proof ${aggStateRow.hash_id} has an invalid JSON schema`)
            return null
          }
          return proof
        }).filter((proof) => proof !== null)

        // if taskQueue is null (redis outage), wait one second for recovery,
        // throw error to initiate nack and retry
        if (taskQueue === null) {
          await utils.sleep(1000)
          throw new Error(`Unable to queue up btc storeProofs jobs, taskQueue is null`)
        }
        await storeProofsAsync(proofs, 'btc_batch')

        // Proof ready message has been consumed, ack consumption of original message
        amqpChannel.ack(msg)
        console.log(msg.fields.routingKey, '[' + msg.properties.type + '] consume message acked')
      } catch (error) {
        console.error(`Unable to process proof ready message: ${error.message}`)
        // An error as occurred consuming a message, nack consumption of original message
        amqpChannel.nack(msg)
        console.error(`${msg.fields.routingKey} [${msg.properties.type}] consume message nacked: ${error.message}`)
      }
      break
    case 'eth':
      console.log('building eth proof')
      amqpChannel.ack(msg)
      break
    default:
      // This is an unknown proof ready type
      console.error('Unknown proof ready type', msg.properties.type)
      // cannot handle unknown type messages, ack message and do nothing
      amqpChannel.ack(msg)
  }
}

async function storeProofsAsync (proofs, batchType) {
  if (proofs.length === 0) return
  let batchStartTimestamp = Date.now()
  let batchId = crypto.randomBytes(4).toString('hex')
  // log information about the first item in the batch
  logGenerationEvent(proofs[0].hash_submitted_node_at, batchType, batchId, 1, proofs.length)
  switch (proofStorageMethod) {
    case 'direct':
      // save proof directly to GCP
      await parallel.each(proofs, async (proof) => {
        try {
          await saveProofToGCPAsync(proof)
        } catch (error) {
          console.error(`Could not save proof to GCP : ${error.message}`)
        }
      }, env.SAVE_CONCURRENCY_COUNT)
      break
    case 'both':
      // save proof directly to GCP
      await parallel.each(proofs, async (proof) => {
        try {
          await saveProofToGCPAsync(proof)
        } catch (error) {
          console.error(`Could not save proof to GCP : ${error.message}`)
        }
      }, env.SAVE_CONCURRENCY_COUNT)
      // save proof to proof proxy
      await parallel.each(proofs, async (proof) => {
        try {
          await taskQueue.enqueue('task-handler-queue', `send_to_proof_proxy`, [proof.hash_id_core, chpBinary.objectToBase64Sync(proof)])
        } catch (error) {
          console.error(`Could not enqueue send_to_proof_proxy task : ${error.message}`)
        }
      }, env.SAVE_CONCURRENCY_COUNT)
      break
    case 'resque':
    default:
      // save proof to proof proxy
      await parallel.each(proofs, async (proof) => {
        try {
          await taskQueue.enqueue('task-handler-queue', `send_to_proof_proxy`, [proof.hash_id_core, chpBinary.objectToBase64Sync(proof)])
        } catch (error) {
          console.error(`Could not enqueue send_to_proof_proxy task : ${error.message}`)
        }
      }, env.SAVE_CONCURRENCY_COUNT)
  }
  if (proofs.length > 1) {
    let batchEndTimestamp = Date.now()
    let batchTotalProcessingMS = batchEndTimestamp - batchStartTimestamp
    // log information about the last item in the batch
    logGenerationEvent(proofs[proofs.length - 1].hash_submitted_node_at, batchType, batchId, proofs.length, proofs.length, batchTotalProcessingMS)
  }
}

async function saveProofToGCPAsync (proof) {
  let proofFilename = `${proof.hash_id_core}.chp`
  let proofGCPFile = gcpBucket.file(proofFilename)

  await retry(async () => {
    await proofGCPFile.save(chpBinary.objectToBinarySync(proof), { resumable: false })
  }, {
    retries: 3,
    minTimeout: 50,
    maxTimeout: 300,
    factor: 1,
    onRetry: (error) => { console.log(`saveProofToGCPAsync : retrying : ${proofFilename} : ${error.message}`) }
  })
}

// use the time difference between now and the time embedded in the hash_id_node UUID
// to log a generation event and total duration
function logGenerationEvent (submitDateString, batchType, batchId, proofIndex, batchSize, batchTotalProcessingMS) {
  let nowTimestamp = Date.now()
  let submitTimestamp = new Date(submitDateString).getTime()
  let generateDuration = moment.duration(nowTimestamp - submitTimestamp)
  let hours = generateDuration.get('h')
  let mins = generateDuration.get('m')
  let secs = generateDuration.get('s')
  let durationString = `${hours} hour${hours !== 1 ? 's' : ''}, ${generateDuration.get('m')} minute${mins !== 1 ? 's' : ''}, and ${generateDuration.get('s')} second${secs !== 1 ? 's' : ''}`
  let totalProcessingString = batchTotalProcessingMS > 1000 ? `[${Math.round(batchTotalProcessingMS / 1000)}s total ]` : (batchTotalProcessingMS > 0 ? `[${batchTotalProcessingMS || 0}ms]` : ``)
  console.log(`Generation ${proofIndex === 1 ? 'starting' : 'complete'} for ${batchType} ${batchId} proof ${proofIndex} of ${batchSize} - ${durationString} after submission ${totalProcessingString}`)
}

// This initializes all the consul watches
function startConsulWatches () {
  let watches = [{
    key: env.PROOF_STORAGE_METHOD_KEY,
    onChange: (data, res) => {
      // process only if a value has been returned
      if (data && data.Value) {
        proofStorageMethod = data.Value
      }
    },
    onError: null
  }]
  let defaults = [
    { key: env.PROOF_STORAGE_METHOD_KEY, value: 'resque' }
  ]
  connections.startConsulWatches(consul, watches, defaults)
}

/**
 * Opens a Redis connection
 *
 * @param {string} redisURI - The connection string for the Redis instance, an Redis URI
 */
function openRedisConnection (redisURIs) {
  connections.openRedisConnection(redisURIs,
    (newRedis) => {
      redis = newRedis
      cachedProofState.setRedis(redis)
      initResqueQueueAsync()
    }, () => {
      redis = null
      cachedProofState.setRedis(null)
      taskQueue = null
      setTimeout(() => { openRedisConnection(redisURIs) }, 5000)
    })
}

/**
 * Opens an AMPQ connection and channel
 * Retry logic is included to handle losses of connection
 *
 * @param {string} connectURI - The connection URI for the RabbitMQ instance
 */
async function openRMQConnectionAsync (connectURI) {
  await connections.openStandardRMQConnectionAsync(amqp, connectURI,
    [env.RMQ_WORK_IN_GEN_QUEUE, env.RMQ_WORK_OUT_TASK_ACC_QUEUE],
    env.RMQ_PREFETCH_COUNT_GEN,
    { queue: env.RMQ_WORK_IN_GEN_QUEUE, method: (msg) => { consumeProofReadyMessageAsync(msg) } },
    (chan) => { amqpChannel = chan },
    () => {
      amqpChannel = null
      setTimeout(() => { openRMQConnectionAsync(connectURI) }, 5000)
    }
  )
}

/**
 * Opens a storage connection
 **/
async function openStorageConnectionAsync () {
  let sqlzModelArray = [
    aggState,
    calState,
    anchorBtcAggState,
    btcTxState,
    btcHeadState
  ]
  let cxObjects = await connections.openStorageConnectionAsync(sqlzModelArray)
  cachedProofState.setDatabase(cxObjects.sequelize, cxObjects.models[0], cxObjects.models[1], cxObjects.models[2], cxObjects.models[3], cxObjects.models[4])
}

/**
 * Initializes the connection to the Resque queue when Redis is ready
 */
async function initResqueQueueAsync () {
  taskQueue = await connections.initResqueQueueAsync(redis, 'resque')
}

// process all steps need to start the application
async function start () {
  if (env.NODE_ENV === 'test') return
  try {
    // init consul
    consul = connections.initConsul(cnsl, env.CONSUL_HOST, env.CONSUL_PORT)
    // init DB
    await openStorageConnectionAsync()
    // init Redis
    openRedisConnection(env.REDIS_CONNECT_URIS)
    // init RabbitMQ
    await openRMQConnectionAsync(env.RABBITMQ_CONNECT_URI)
    // init consul watches
    startConsulWatches()
    console.log('startup completed successfully')
  } catch (error) {
    console.error(`An error has occurred on startup: ${error.message}`)
    process.exit(1)
  }
}

// get the whole show started
start()
