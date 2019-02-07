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
const env = require('./lib/parse-env.js')('btc-mon')

const MerkleTools = require('merkle-tools')
const BlockchainAnchor = require('blockchain-anchor')
const amqp = require('amqplib')
const connections = require('./lib/connections.js')

// Key for the Redis set of all Bitcoin transaction id objects needing to be monitored.
const BTC_TX_IDS_KEY = 'BTC_Mon:BTCTxIds'

// The merkle tools object for building trees and generating proof paths
const merkleTools = new MerkleTools()

// The channel used for all amqp communication
// This value is set once the connection has been established
var amqpChannel = null

// This value is set once the connection has been established
let redis = null

let CHECKS_IN_PROGRESS = false

// Initialize BlockchainAnchor object
let anchor = new BlockchainAnchor({
  btcUseTestnet: env.USE_BTCETH_TESTNET,
  service: 'insightapi',
  insightApiBase: env.INSIGHT_API_BASE_URI,
  insightFallback: true
})

async function consumeBtcTxIdMessageAsync (msg) {
  if (msg !== null) {
    let btcTxIdObjJSON = msg.content.toString()

    try {
      // add the transaction id to the redis set
      // Redis is the sole storage mechanism for this data
      await redis.sadd(BTC_TX_IDS_KEY, btcTxIdObjJSON)
      amqpChannel.ack(msg)
    } catch (error) {
      amqpChannel.nack(msg)
      console.error(`${env.RMQ_WORK_IN_AGG_QUEUE} consume message nacked`)
    }
  }
}

// Iterate through all BTCTXIDS objects, checking the confirmation count for each transaction
// If MIN_BTC_CONFIRMS is reached for a given transaction, retrieve the state data needed
// to build the proof path from the transaction to the block header merkle root value and
// return that information to calendar service, ack message.
let monitorTransactionsAsync = async () => {
  // if the amqp channel is null (closed), processing should not continue, defer to next monitorTransactions call
  if (amqpChannel === null || redis === null) return

  CHECKS_IN_PROGRESS = true
  let btcTxObjJSONArray = await redis.smembers(BTC_TX_IDS_KEY)
  console.log(`Btc Tx monitoring check starting for ${btcTxObjJSONArray.length} transaction(s)`)

  for (let btcTxObjJSON of btcTxObjJSONArray) {
    let btcTxIdObj = JSON.parse(btcTxObjJSON)
    try {
      // Get BTC Transaction Stats
      let txStats
      try {
        txStats = await anchor.btcGetTxStatsAsync(btcTxIdObj.tx_id)
      } catch (error) {
        throw new Error(`Could not get stats for transaction ${btcTxIdObj.tx_id}`)
      }
      if (txStats.confirmations < env.MIN_BTC_CONFIRMS) {
        console.log(`${txStats.id} not ready : ${txStats.confirmations} of ${env.MIN_BTC_CONFIRMS} confirmations`)
        continue
      }

      // if ready, Get BTC Block Stats with Transaction Ids
      let blockStats
      try {
        blockStats = await anchor.btcGetBlockStatsAsync(txStats.blockHash)
      } catch (error) {
        throw new Error(`Could not get stats for block ${txStats.blockHeight} (${txStats.blockHash})`)
      }
      let txIndex = blockStats.txIds.indexOf(txStats.id)
      if (txIndex === -1) throw new Error(`transaction ${txStats.id} not found in block ${txStats.blockHeight}`)
      // adjusting for endieness, reverse txids for further processing
      blockStats.txIds = blockStats.txIds.map((txId) => txId.match(/.{2}/g).reverse().join(''))

      if (blockStats.txIds.length === 0) throw new Error(`No transactions found in block ${txStats.blockHeight}`)

      // build BTC merkle tree with txIds
      merkleTools.resetTree()
      merkleTools.addLeaves(blockStats.txIds)
      merkleTools.makeBTCTree(true)
      let rootValueBuffer = merkleTools.getMerkleRoot()
      // re-adjust for endieness, reverse and convert back to hex
      let rootValueHex = rootValueBuffer.toString('hex').match(/.{2}/g).reverse().join('')
      if (rootValueHex !== blockStats.merkleRoot) throw new Error(`calculated merkle root (${rootValueHex}) does not match block merkle root (${blockStats.merkleRoot}) for tx ${txStats.id}`)
      // get proof path from tx to block root
      let proofPath = merkleTools.getProof(txIndex)
      // send data back to calendar
      let messageObj = {}
      messageObj.btctx_id = txStats.id
      messageObj.btchead_height = txStats.blockHeight
      messageObj.btchead_root = rootValueHex
      messageObj.path = proofPath
      try {
        await amqpChannel.sendToQueue(env.RMQ_WORK_OUT_CAL_QUEUE, Buffer.from(JSON.stringify(messageObj)), { persistent: true, type: 'btcmon' })
        console.log(env.RMQ_WORK_OUT_CAL_QUEUE, '[btcmon] publish message acked', messageObj.btctx_id)
      } catch (error) {
        console.error(env.RMQ_WORK_OUT_CAL_QUEUE, '[btcmon] publish message nacked', messageObj.btctx_id)
        throw new Error(error.message)
      }

      await redis.srem(BTC_TX_IDS_KEY, btcTxObjJSON)

      console.log(`${btcTxIdObj.tx_id} ready with ${txStats.confirmations} confirmations`)
    } catch (error) {
      console.error(error.message)
    }
  }

  console.log(`Btc Tx monitoring checks complete`)
  CHECKS_IN_PROGRESS = false
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
    }, () => {
      redis = null
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
    [env.RMQ_WORK_IN_BTCMON_QUEUE, env.RMQ_WORK_OUT_CAL_QUEUE],
    env.RMQ_PREFETCH_COUNT_BTCMON,
    { queue: env.RMQ_WORK_IN_BTCMON_QUEUE, method: (msg) => { consumeBtcTxIdMessageAsync(msg) } },
    (chan) => { amqpChannel = chan },
    () => {
      amqpChannel = null
      setTimeout(() => { openRMQConnectionAsync(connectURI) }, 5000)
    }
  )
}

function startIntervals () {
  let intervals = [{
    function: () => {
      if (!CHECKS_IN_PROGRESS) monitorTransactionsAsync()
    },
    ms: env.MONITOR_INTERVAL_SECONDS * 1000
  }]
  connections.startIntervals(intervals)
}

// process all steps need to start the application
async function start () {
  if (env.NODE_ENV === 'test') return
  try {
    // init Redis
    openRedisConnection(env.REDIS_CONNECT_URIS)
    // init RabbitMQ
    await openRMQConnectionAsync(env.RABBITMQ_CONNECT_URI)
    // init interval functions
    startIntervals()
    console.log('startup completed successfully')
  } catch (error) {
    console.error(`An error has occurred on startup: ${error.message}`)
    process.exit(1)
  }
}

// get the whole show started
start()

// export these functions for unit tests
module.exports = {
  getAMQPChannel: function () { return amqpChannel },
  setAMQPChannel: (chan) => { amqpChannel = chan },
  openRMQConnectionAsync: openRMQConnectionAsync,
  consumeBtcTxIdMessageAsync: consumeBtcTxIdMessageAsync,
  monitorTransactionsAsync: monitorTransactionsAsync
}
