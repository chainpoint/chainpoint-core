/* Copyright (C) 2019 Tierion
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
const btcBridge = require('btc-bridge')
const amqp = require('amqplib')
const connections = require('./lib/connections.js')
const logger = require('./lib/logger.js')

// Key for the Redis set of all Bitcoin transaction id objects needing to be monitored for first confirmation
const NEW_BTC_TX_IDS_KEY = 'BTC_Mon:NewBTCTxIds'
// Key for the Redis set of all Bitcoin transaction id objects needing to be monitored for final confirmation
const CONFIRMED_BTC_TX_IDS_KEY = 'BTC_Mon:ConfirmedBTCTxIds'

// The merkle tools object for building trees and generating proof paths
const merkleTools = new MerkleTools()

// The channel used for all amqp communication
// This value is set once the connection has been established
var amqpChannel = null

// This value is set once the connection has been established
let redis = null

let CHECKS_IN_PROGRESS = false

async function consumeBtcMonMessageAsync(msg) {
  if (msg !== null) {
    // determine the source of the message and handle appropriately
    switch (msg.properties.type) {
      case 'newtx':
        try {
          // add the transaction id to the redis set
          // Redis is the sole storage mechanism for this data
          let newBtcTxIdObjJSON = msg.content.toString()
          await redis.sadd(NEW_BTC_TX_IDS_KEY, newBtcTxIdObjJSON)
          amqpChannel.ack(msg)
        } catch (error) {
          amqpChannel.nack(msg)
          logger.error(`${env.RMQ_WORK_IN_BTCMON_QUEUE} : consume message nacked`)
        }
        break
      case 'confirmedtx':
        try {
          // add the transaction id to the redis set
          // Redis is the sole storage mechanism for this data
          let confirmedBtcTxIdObjJSON = msg.content.toString()
          await redis.sadd(CONFIRMED_BTC_TX_IDS_KEY, confirmedBtcTxIdObjJSON)
          amqpChannel.ack(msg)
        } catch (error) {
          amqpChannel.nack(msg)
          logger.error(`${env.RMQ_WORK_IN_BTCMON_QUEUE} : consume message nacked`)
        }
        break
      default:
        console.error('consumeBtcMonMessageAsync : unknown message type', msg.properties.type)
        // cannot handle unknown type messages, ack message and do nothing
        amqpChannel.ack(msg)
    }
  }
}

let monitorTransactionsAsync = async () => {
  // if the amqp channel is null (closed), processing should not continue, defer to next monitorTransactions call
  if (amqpChannel === null || redis === null) return

  CHECKS_IN_PROGRESS = true
  let newBtcTxObjJSONArray = await redis.smembers(NEW_BTC_TX_IDS_KEY)
  let confirmedBtcTxObjJSONArray = await redis.smembers(CONFIRMED_BTC_TX_IDS_KEY)

  let lnd
  try {
    const btcNetwork = env.NETWORK === 'mainnet' ? btcBridge.networks.MAINNET : btcBridge.networks.TESTNET
    lnd = new btcBridge.providers.LndProvider(
      btcNetwork,
      env.LND_SOCKET,
      `/root/.lnd/data/chain/bitcoin/${env.NETWORK}/admin.macaroon`,
      `/root/.lnd/tls.cert`
    )
    await lnd.initWallet()
  } catch (error) {
    logger.error(`Unable to initialize LND provider : ${error.message}`)
    CHECKS_IN_PROGRESS = false
    return
  }
  // Iterate through all new BTCTXIDS objects, checking the confirmation count for each transaction
  // If a given transaction has been confirmed at least once, send the transaction information
  // along with the block height of the confirming block to calendar service, ack message.
  logger.info(`New Btc Tx monitoring check starting for ${newBtcTxObjJSONArray.length} transaction(s)`)
  for (let newbtcTxObjJSON of newBtcTxObjJSONArray) {
    let newBtcTxIdObj = JSON.parse(newbtcTxObjJSON)
    try {
      // Get BTC Transaction Stats
      let txStats
      try {
        txStats = await lnd.getTransactionDataAsync(newBtcTxIdObj.btctx_id)
        logger.info(`blockHash obtained for btctx ${txStats.txId} : ${txStats.blockHash}`)
      } catch (error) {
        throw new Error(`Could not get stats for transaction ${newBtcTxIdObj.btctx_id}`)
      }
      if (txStats.confirmations < 1) {
        logger.info(`${txStats.txId} not yet confirmed`)
        continue
      }

      // if confirmed, Get BTC Block info
      let blockStats
      try {
        blockStats = await lnd.getBlockDataAsync(txStats.blockHash)
      } catch (error) {
        throw new Error(`Could not get stats for block ${txStats.blockHash}`)
      }

      newBtcTxIdObj.btctx_height = blockStats.height
      try {
        await amqpChannel.sendToQueue(env.RMQ_WORK_OUT_CAL_QUEUE, Buffer.from(JSON.stringify(newBtcTxIdObj)), {
          persistent: true,
          type: 'btcmon_new'
        })
        logger.info(`${env.RMQ_WORK_OUT_CAL_QUEUE} : [btcmon] publish message acked : ${newBtcTxIdObj.btctx_id}`)
      } catch (error) {
        logger.error(`${env.RMQ_WORK_OUT_CAL_QUEUE} : [btcmon] publish message nacked : ${newBtcTxIdObj.btctx_id}`)
        throw new Error(error.message)
      }

      await redis.srem(NEW_BTC_TX_IDS_KEY, newbtcTxObjJSON)

      logger.info(`${newBtcTxIdObj.btctx_id} ready with ${txStats.confirmations} confirmations`)
    } catch (error) {
      logger.error(`An unexpected error occurred while monitoring : ${error.message}`)
    }
  }
  logger.info(`New Btc Tx monitoring checks complete`)

  // Iterate through all confirmed BTCTXIDS objects, checking the confirmation count for each transaction
  // If MIN_BTC_CONFIRMS is reached for a given transaction, retrieve the state data needed
  // to build the proof path from the transaction to the block header merkle root value and
  // return that information to calendar service, ack message.
  logger.info(`Confirmed Btc Tx monitoring check starting for ${confirmedBtcTxObjJSONArray.length} transaction(s)`)
  for (let confirmedBtcTxObjJSON of confirmedBtcTxObjJSONArray) {
    let confirmedBtcTxIdObj = JSON.parse(confirmedBtcTxObjJSON)
    try {
      let txId = confirmedBtcTxIdObj.tx_id
      let txBlockHeight = confirmedBtcTxIdObj.block_height
      // Get current BTC block height
      let confirmCount = 0
      try {
        let info = await lnd.getChainInfoAsync()
        confirmCount = info.topBlockHeight - txBlockHeight + 1
        if (confirmCount < env.MIN_BTC_CONFIRMS) {
          logger.info(`${txId} not ready : ${confirmCount} of ${env.MIN_BTC_CONFIRMS} confirmations`)
          continue
        }
      } catch (error) {
        throw new Error(`Could not retrieve node info`)
      }

      // if ready, Get BTC Block Stats with Transaction Ids
      let blockStats
      try {
        blockStats = await lnd.getBlockDataAsync(txBlockHeight)
      } catch (error) {
        throw new Error(`Could not get stats for block ${txBlockHeight}`)
      }
      let txIndex = blockStats.tx.indexOf(txId)
      if (txIndex === -1) throw new Error(`transaction ${txId} not found in block ${txBlockHeight}`)
      // adjusting for endieness, reverse txids for further processing
      blockStats.tx = blockStats.tx.map(txId =>
        txId
          .match(/.{2}/g)
          .reverse()
          .join('')
      )

      if (blockStats.tx.length === 0) throw new Error(`No transactions found in block ${txBlockHeight}`)

      // build BTC merkle tree with txIds
      merkleTools.resetTree()
      merkleTools.addLeaves(blockStats.tx)
      merkleTools.makeBTCTree(true)
      let rootValueBuffer = merkleTools.getMerkleRoot()
      // re-adjust for endieness, reverse and convert back to hex
      let rootValueHex = rootValueBuffer
        .toString('hex')
        .match(/.{2}/g)
        .reverse()
        .join('')
      if (rootValueHex !== blockStats.merkleRoot)
        throw new Error(
          `calculated merkle root (${rootValueHex}) does not match block merkle root (${
            blockStats.merkleRoot
          }) for tx ${txId}`
        )
      // get proof path from tx to block root
      let proofPath = merkleTools.getProof(txIndex)
      // send data back to calendar
      let messageObj = {}
      messageObj.btctx_id = txId
      messageObj.btchead_height = blockStats.height
      messageObj.btchead_root = rootValueHex
      messageObj.path = proofPath
      try {
        await amqpChannel.sendToQueue(env.RMQ_WORK_OUT_CAL_QUEUE, Buffer.from(JSON.stringify(messageObj)), {
          persistent: true,
          type: 'btcmon_confirmed'
        })
        logger.info(`${env.RMQ_WORK_OUT_CAL_QUEUE} : [btcmon] publish message acked : ${messageObj.btctx_id}`)
      } catch (error) {
        logger.error(`${env.RMQ_WORK_OUT_CAL_QUEUE} : [btcmon] publish message nacked : ${messageObj.btctx_id}`)
        throw new Error(error.message)
      }

      await redis.srem(CONFIRMED_BTC_TX_IDS_KEY, confirmedBtcTxObjJSON)

      logger.info(`${confirmedBtcTxIdObj.tx_id} ready with ${confirmCount} confirmations`)
    } catch (error) {
      logger.error(`An unexpected error occurred while monitoring : ${error.message}`)
    }
  }
  logger.info(`Confirmed Btc Tx monitoring checks complete`)

  CHECKS_IN_PROGRESS = false
}

/**
 * Opens a Redis connection
 *
 * @param {string} redisURI - The connection string for the Redis instance, an Redis URI
 */
function openRedisConnection(redisURIs) {
  connections.openRedisConnection(
    redisURIs,
    newRedis => {
      redis = newRedis
    },
    () => {
      redis = null
      setTimeout(() => {
        openRedisConnection(redisURIs)
      }, 5000)
    }
  )
}

/**
 * Opens an AMPQ connection and channel
 * Retry logic is included to handle losses of connection
 *
 * @param {string} connectURI - The connection URI for the RabbitMQ instance
 */
async function openRMQConnectionAsync(connectURI) {
  await connections.openStandardRMQConnectionAsync(
    amqp,
    connectURI,
    [env.RMQ_WORK_IN_BTCMON_QUEUE, env.RMQ_WORK_OUT_CAL_QUEUE],
    env.RMQ_PREFETCH_COUNT_BTCMON,
    {
      queue: env.RMQ_WORK_IN_BTCMON_QUEUE,
      method: msg => {
        consumeBtcMonMessageAsync(msg)
      }
    },
    chan => {
      amqpChannel = chan
    },
    () => {
      amqpChannel = null
      setTimeout(() => {
        openRMQConnectionAsync(connectURI)
      }, 5000)
    }
  )
}

function startIntervals() {
  let intervals = [
    {
      function: () => {
        if (!CHECKS_IN_PROGRESS) monitorTransactionsAsync()
      },
      ms: env.MONITOR_INTERVAL_SECONDS * 1000
    }
  ]
  connections.startIntervals(intervals)
}

// process all steps need to start the application
async function start() {
  if (env.NODE_ENV === 'test') return
  try {
    // init Redis
    openRedisConnection(env.REDIS_CONNECT_URIS)
    // init RabbitMQ
    await openRMQConnectionAsync(env.RABBITMQ_CONNECT_URI)
    // init interval functions
    startIntervals()
    logger.info(`Startup completed successfully`)
  } catch (error) {
    logger.error(`An error has occurred on startup : ${error.message}`)
    process.exit(1)
  }
}

// get the whole show started
start()

// export these functions for unit tests
module.exports = {
  getAMQPChannel: function() {
    return amqpChannel
  },
  setAMQPChannel: chan => {
    amqpChannel = chan
  },
  openRMQConnectionAsync: openRMQConnectionAsync,
  consumeBtcMonMessageAsync: consumeBtcMonMessageAsync,
  monitorTransactionsAsync: monitorTransactionsAsync
}
