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
const env = require('./lib/parse-env.js')('btc-tx')

const amqp = require('amqplib')
const btcBridge = require('btc-bridge')
const connections = require('./lib/connections.js')
const logger = require('./lib/logger.js')
const BigNumber = require('bignumber.js')

// The channel used for all amqp communication
// This value is set once the connection has been established
let amqpChannel = null

// track the `feeSatPerByte` values being used so that, in the event of a estimated fee retrieval
// failure, we fall back to the last retrieved value
let lastKnownFeeSatPerByte = null
// use a default constant value if the service has just started, lastKnownFeeSatPerByte is null,
// and estimated fee retrieval fails
const defaultFeeSatPerByte = 250

const btcNetwork = env.NETWORK === 'mainnet' ? btcBridge.networks.MAINNET : btcBridge.networks.TESTNET
let providers = []
let rpcUris = []
for (let rpcUri of rpcUris) {
  providers.push(new btcBridge.providers.JsonRpcProvider(btcNetwork, rpcUri))
}
const fallbackProvider = new btcBridge.providers.FallbackProvider(providers, false)

/**
 * Send a POST request to /wallet/:id/send with a POST body
 * containing an OP_RETURN TX
 *
 * @param {string} hash - The hash to embed in an OP_RETURN
 */
const sendTxToBTCAsync = async hash => {
  let privateKeyWIF = env.BITCOIN_WIF

  let feeSatPerByte
  let feeTotalSatoshi
  try {
    let result = await fallbackProvider.getEstimatedFeeAsync(2)
    let feeBtcPerKb = result.feerate
    feeSatPerByte = BigNumber(feeBtcPerKb)
      .div(1024)
      .times(10 ** 8)
      .toNumber()
    lastKnownFeeSatPerByte = feeSatPerByte
  } catch (error) {
    logger.warn(`Error retrieving estimated fee: ${error.message}`)
    feeSatPerByte = lastKnownFeeSatPerByte || defaultFeeSatPerByte
    logger.warn(`Falling back to a Satoshi per byte fee value of '${feeSatPerByte}'`)
  }

  // if the fee exceeds the maximum, revert to BTC_MAX_FEE_SAT_PER_BYTE for the fee
  if (feeSatPerByte > env.BTC_MAX_FEE_SAT_PER_BYTE) {
    logger.warn(
      `Fee of '${feeSatPerByte}' Satoshi per byte exceeded BTC_MAX_FEE_SAT_PER_BYTE of '${
        env.BTC_MAX_FEE_SAT_PER_BYTE
      }'`
    )
    logger.warn(`Falling back to a Satoshi per byte fee value of '${env.BTC_MAX_FEE_SAT_PER_BYTE}'`)
    feeSatPerByte = env.BTC_MAX_FEE_SAT_PER_BYTE
  }
  let feeExtra = 1.1 // the factor to use to increase the final transaction fee in order to better position this transaction for fast confirmation
  feeSatPerByte = Math.ceil(feeSatPerByte * feeExtra) // Math.ceil to keep the value an integer, as expected in the log db
  let averageTxInBytes = 235 // 235 represents the average btc anchor transaction size in bytes
  feeTotalSatoshi = feeSatPerByte * averageTxInBytes

  let txResult
  try {
    let wallet = new btcBridge.Wallet(privateKeyWIF, fallbackProvider)
    let btcFee = BigNumber(feeTotalSatoshi)
      .div(10 ** 8)
      .toNumber()
    txResult = await wallet.generateOpReturnTxWithFeeAsync(hash, btcFee, true)
    txResult.publishDate = Date.now()
    txResult.feeSatoshiPerByte = feeSatPerByte
    txResult.feePaidSatoshi = feeTotalSatoshi
    return txResult
  } catch (error) {
    throw new Error(`Error sending anchor transaction: ${error.message}`)
  }
}

/**
 * Parses a message and performs the required work for that message
 *
 * @param {amqp message object} msg - The AMQP message received from the queue
 */
async function processIncomingAnchorBTCJobAsync(msg) {
  if (msg !== null) {
    let messageObj = JSON.parse(msg.content.toString())
    // the value to be anchored, likely a merkle root hex string
    let anchorData = messageObj.anchor_btc_agg_root

    // if amqpChannel is null for any reason, don't bother sending transaction until that is resolved, return error
    if (!amqpChannel) throw new Error('no amqpConnection available')

    try {
      // create and publish the transaction
      let txResult
      try {
        txResult = await sendTxToBTCAsync(anchorData)
      } catch (error) {
        throw new Error(`Unable to publish BTC transaction: ${error.message}`)
      }

      // queue return message for calendar containing the new transaction information
      // adding btc transaction id and full transaction body to original message and returning
      messageObj.btctx_id = txResult.txId
      messageObj.btctx_body = txResult.txHex
      try {
        await amqpChannel.sendToQueue(env.RMQ_WORK_OUT_CAL_QUEUE, Buffer.from(JSON.stringify(messageObj)), {
          persistent: true,
          type: 'btctx'
        })
        logger.info(`${env.RMQ_WORK_OUT_CAL_QUEUE} : [btctx] publish message acked : ${messageObj.btctx_id}`)
      } catch (error) {
        logger.error(`${env.RMQ_WORK_OUT_CAL_QUEUE} : [btctx] publish message nacked : ${messageObj.btctx_id}`)
        throw new Error(`Unable to publish to RMQ_WORK_OUT_CAL_QUEUE: ${error.message}`)
      }
      amqpChannel.ack(msg)
    } catch (error) {
      // An error has occurred publishing the transaction, nack consumption of message
      // set a 30 second delay for nacking this message to prevent a flood of retries hitting the bitcoin rpc provider
      let retryMS = 30000
      logger.error(`Unable to process BTC anchor message : ${error.message} : Retrying in ${retryMS / 1000} seconds`)
      setTimeout(() => {
        amqpChannel.nack(msg)
        logger.error(`${env.RMQ_WORK_IN_BTCTX_QUEUE} : consume message nacked`)
      }, retryMS)
    }
  }
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
    [env.RMQ_WORK_IN_BTCTX_QUEUE, env.RMQ_WORK_OUT_CAL_QUEUE],
    env.RMQ_PREFETCH_COUNT_BTCTX,
    {
      queue: env.RMQ_WORK_IN_BTCTX_QUEUE,
      method: msg => {
        processIncomingAnchorBTCJobAsync(msg)
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

// process all steps need to start the application
async function start() {
  if (env.NODE_ENV === 'test') return
  try {
    // init RabbitMQ
    await openRMQConnectionAsync(env.RABBITMQ_CONNECT_URI)
    logger.info(`Startup completed successfully`)
  } catch (error) {
    logger.error(`An error has occurred on startup : ${error.message}`)
    process.exit(1)
  }
}

// get the whole show started
start()
