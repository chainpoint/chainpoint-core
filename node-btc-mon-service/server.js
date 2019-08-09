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
const utils = require('./lib/utils.js')
const lnService = require('ln-service')

// Key for the Redis set of all Bitcoin transaction id objects needing to be monitored.
const BTC_TX_IDS_KEY = 'BTC_Mon:BTCTxIds'

// The merkle tools object for building trees and generating proof paths
const merkleTools = new MerkleTools()

// The channel used for all amqp communication
// This value is set once the connection has been established
var amqpChannel = null

// This value is set once the connection has been established
let redis = null

// initialize lightning grpc object
let { lnd } = lnService.authenticatedLndGrpc({ cert: env.LND_CERT, macaroon: env.LND_MACAROON, socket: env.LND_HOST })

let CHECKS_IN_PROGRESS = false

const btcNetwork = env.NETWORK === 'mainnet' ? btcBridge.networks.MAINNET : btcBridge.networks.TESTNET
let providers = []
let rpcUris = env.BTC_RPC_URI_LIST.split(',')
for (let rpcUri of rpcUris) {
  providers.push(new btcBridge.providers.JsonRpcProvider(btcNetwork, rpcUri))
}
if (env.BLOCKCYPHER_API_TOKEN)
  providers.push(new btcBridge.providers.BlockcypherProvider(btcNetwork, env.BLOCKCYPHER_API_TOKEN))
const fallbackProvider = new btcBridge.providers.FallbackProvider(providers, false)

async function consumeBtcTxIdMessageAsync(msg) {
  if (msg !== null) {
    let btcTxIdObjJSON = msg.content.toString()

    try {
      // add the transaction id to the redis set
      // Redis is the sole storage mechanism for this data
      await redis.sadd(BTC_TX_IDS_KEY, btcTxIdObjJSON)
      amqpChannel.ack(msg)
    } catch (error) {
      amqpChannel.nack(msg)
      logger.error(`${env.RMQ_WORK_IN_AGG_QUEUE} : consume message nacked`)
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
  logger.info(`Btc Tx monitoring check starting for ${btcTxObjJSONArray.length} transaction(s)`)

  for (let btcTxObjJSON of btcTxObjJSONArray) {
    let btcTxIdObj = JSON.parse(btcTxObjJSON)
    try {
      // Get BTC Transaction Stats
      let txStats
      try {
        txStats = await fallbackProvider.getTransactionDataAsync(btcTxIdObj.tx_id)
      } catch (error) {
        throw new Error(`Could not get stats for transaction ${btcTxIdObj.tx_id}`)
      }
      if (txStats.confirmations < env.MIN_BTC_CONFIRMS) {
        logger.info(`${txStats.txId} not ready : ${txStats.confirmations} of ${env.MIN_BTC_CONFIRMS} confirmations`)
        continue
      }

      // if ready, Get BTC Block Stats with Transaction Ids
      let blockStats
      try {
        blockStats = await fallbackProvider.getBlockDataAsync(txStats.blockHash)
      } catch (error) {
        throw new Error(`Could not get stats for block ${txStats.blockHash}`)
      }
      let txIndex = blockStats.tx.indexOf(txStats.txId)
      if (txIndex === -1) throw new Error(`transaction ${txStats.txId} not found in block ${blockStats.height}`)
      // adjusting for endieness, reverse txids for further processing
      blockStats.tx = blockStats.tx.map(txId =>
        txId
          .match(/.{2}/g)
          .reverse()
          .join('')
      )

      if (blockStats.tx.length === 0) throw new Error(`No transactions found in block ${blockStats.height}`)

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
          }) for tx ${txStats.txId}`
        )
      // get proof path from tx to block root
      let proofPath = merkleTools.getProof(txIndex)
      // send data back to calendar
      let messageObj = {}
      messageObj.btctx_id = txStats.txId
      messageObj.btchead_height = blockStats.height
      messageObj.btchead_root = rootValueHex
      messageObj.path = proofPath
      try {
        await amqpChannel.sendToQueue(env.RMQ_WORK_OUT_CAL_QUEUE, Buffer.from(JSON.stringify(messageObj)), {
          persistent: true,
          type: 'btcmon'
        })
        logger.info(`${env.RMQ_WORK_OUT_CAL_QUEUE} : [btcmon] publish message acked : ${messageObj.btctx_id}`)
      } catch (error) {
        logger.error(`${env.RMQ_WORK_OUT_CAL_QUEUE} : [btcmon] publish message nacked : ${messageObj.btctx_id}`)
        throw new Error(error.message)
      }

      await redis.srem(BTC_TX_IDS_KEY, btcTxObjJSON)

      logger.info(`${btcTxIdObj.tx_id} ready with ${txStats.confirmations} confirmations`)
    } catch (error) {
      logger.error(`An unexpected error occurred while monitoring : ${error.message}`)
    }
  }

  logger.info(`Btc Tx monitoring checks complete`)
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
        consumeBtcTxIdMessageAsync(msg)
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

function startInvoiceSubscription() {
  try {
    let invoiceSubscription = lnService.subscribeToInvoices({ lnd })
    invoiceSubscription.on('invoice_updated', async invoice => {
      if (invoice.is_confirmed) {
        logger.info('Invoice paid: ' + invoice.description)
        // This invoice has been paid
        // Add a short lived key to redis indicating the payment has been made
        // With this key added, the invoice id can be used to submit a hash one time
        let invoiceId = invoice.description.split(':')[1]
        let paidInvoiceKey = `PaidSubmitHashInvoiceId:${invoiceId}`
        try {
          await redis.set(paidInvoiceKey, 1, 'EX', 1) // this key will expire 1 minute after invoice payment
        } catch (error) {
          logger.error(`Redis SET error : error setting item with key = ${paidInvoiceKey}`)
        }
      } else {
        logger.info('Invoice generated: ' + invoice.description)
      }
    })
    invoiceSubscription.on('error', async err => {
      logger.error(`An invoice subscription error occurred : ${err.details}`)
      if (err.details === 'Stream removed') {
        let needReconnect = true
        while (needReconnect) {
          try {
            invoiceSubscription = null
            startInvoiceSubscription()
            logger.warn('Invoices subscription stream had failed, but has been recovered')
            needReconnect = false
          } catch (error) {
            // catch errors when attempting to establish invoice subscription
            logger.warn('Cannot establish lnd invoice subscription : Attempting in 5 seconds...')
            await utils.sleepAsync(5000)
          }
        }
      }
    })
    logger.info('Invoices subscription established')
  } catch (error) {
    throw new Error(`Unable to subscribe to lnd invoices : ${error.message}`)
  }
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
    // init listening for lnd invoice update events
    startInvoiceSubscription()
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
  consumeBtcTxIdMessageAsync: consumeBtcTxIdMessageAsync,
  monitorTransactionsAsync: monitorTransactionsAsync
}
