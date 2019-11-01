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
const env = require('./lib/parse-env.js')('lnd-mon')

const connections = require('./lib/connections.js')
const logger = require('./lib/logger.js')
const utils = require('./lib/utils.js')
const lndClient = require('lnrpc-node-client')
const fs = require('fs')

const LND_SOCKET = env.LND_SOCKET
const LND_CERTPATH = `/root/.lnd/tls.cert`
const LND_MACAROONPATH = `/root/.lnd/data/chain/bitcoin/${env.NETWORK}/admin.macaroon`

const LAST_KNOWN_INVOICE_INDEX_KEY = 'LastKnownInvoiceIndex'

// This value is set once the connection has been established
let redis = null

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

async function startInvoiceMonitoring() {
  let subscriptionEstablished = false
  if (!fs.existsSync(LND_CERTPATH)) {
    throw new Error(`LND TLS Cert not yet generated, restarting...`)
  }
  while (!subscriptionEstablished) {
    try {
      // establish a connection to lnd
      try {
        await ensureLndNodeClientWalletUnlockedAsync()
        lndClient.setCredentials(LND_SOCKET, LND_MACAROONPATH, LND_CERTPATH)
      } catch (error) {
        throw new Error(`Cannot establish active LND connection : ${error.message}`)
      }
      // retrieve the add_index of the most recently handled invoice
      let lastKnownInvoiceIndex = await getLastKnownInvoiceIndexAsync()
      // starting listening for and handle new invoice activity
      await establishInvoiceSubscriptionAsync(lastKnownInvoiceIndex)
      subscriptionEstablished = true
    } catch (error) {
      // catch errors when attempting to connect and establish invoice subscription
      logger.error(`Invoice monitoring : ${error.message} : Retrying in 5 seconds...`)
      await utils.sleepAsync(5000)
    }
  }
}

async function ensureLndNodeClientWalletUnlockedAsync() {
  lndClient.setTls(LND_SOCKET, LND_CERTPATH)
  let unlocker = lndClient.unlocker()
  try {
    await unlocker.unlockWalletAsync({ wallet_password: env.HOT_WALLET_PASS })
  } catch (error) {
    if (error.code === 12) return // already unlocked
  }
  throw new Error(`Unable to unlock wallet`)
}

async function establishInvoiceSubscriptionAsync(addIndex) {
  try {
    let invoiceSubscription = lndClient.lightning().subscribeInvoices({ add_index: addIndex })
    invoiceSubscription.on('data', async invoice => {
      let invoiceAddIndex = await processInvoiceBatchAsync([invoice])
      await updateLastKnownInvoiceIndexAsync(invoiceAddIndex)
    })
    invoiceSubscription.on('status', function(status) {
      logger.warn(`LND invoice subscription status has changed (${status.code}) ${status.details}`)
    })
    invoiceSubscription.on('end', function() {
      logger.error(`The LND invoice subscription has unexpectedly ended`)
      setTimeout(startInvoiceMonitoring, 1000)
    })
    logger.info('LND invoice subscription has been established')
  } catch (error) {
    throw new Error(`Unable to establish LND invoice subscription : ${error.message}`)
  }
}

async function getLastKnownInvoiceIndexAsync() {
  // return the add_index value of the newest invoice processed
  let lastKnownInvoiceIndex
  try {
    lastKnownInvoiceIndex = parseInt(await redis.get(LAST_KNOWN_INVOICE_INDEX_KEY))
    if (isNaN(lastKnownInvoiceIndex)) lastKnownInvoiceIndex = 0 // no value set yet, do not retrieve past invoices
  } catch (error) {
    logger.warn(`Unable to retrieve last known invoice index, skipping : ${error.message}`)
    lastKnownInvoiceIndex = 0
  }
  return lastKnownInvoiceIndex
}

async function updateLastKnownInvoiceIndexAsync(newInvoiceAddIndex) {
  try {
    // retrieve the current last known invoice index value
    // update this value only if the new value is greater than the current value
    let lastKnownIndex = await getLastKnownInvoiceIndexAsync()
    if (newInvoiceAddIndex > lastKnownIndex) await redis.set(LAST_KNOWN_INVOICE_INDEX_KEY, newInvoiceAddIndex)
  } catch (error) {
    logger.error(`Unable to update LAST_KNOWN_INVOICE_INDEX_KEY : value = ${newInvoiceAddIndex} : ${error.message}`)
  }
}

async function processInvoiceBatchAsync(invoices) {
  let invoiceRedisOps = invoices
    .map(invoice => {
      if (!invoice.settled) {
        // This invoice was created but not yet paid
        logger.info(`Invoice generated : ${invoice.memo} : index ${invoice.add_index}, ${invoice.settle_index}`)
        return null
      } else {
        // This invoice has been paid
        logger.info(`Invoice paid : ${invoice.memo} : index ${invoice.add_index}, ${invoice.settle_index}`)
        // Add a key to redis indicating the payment has been made
        // With this key added, the invoice id can be used to submit a hash one time
        let invoiceId = invoice.memo.split(':')[1]
        let paidInvoiceKey = `PaidSubmitHashInvoiceId:${invoiceId}`
        return ['set', paidInvoiceKey, '1', 'EX', '120']
      }
    })
    .filter(invoice => invoice !== null)
  // process all the redis oprations and return the most recently processed invoice add_index
  let lastInvoiceAddIndex = invoices[invoices.length - 1].add_index
  try {
    await redis.multi(invoiceRedisOps).exec()
    return lastInvoiceAddIndex
  } catch (error) {
    logger.error(
      `Redis MULTI SET error : error setting item batch ending with invoice index = ${lastInvoiceAddIndex} : ${
        error.message
      }`
    )
    return null
  }
}

// process all steps need to start the application
async function start() {
  if (env.NODE_ENV === 'test') return
  try {
    // init Redis
    openRedisConnection(env.REDIS_CONNECT_URIS)
    // init listening for lnd invoice update events
    await startInvoiceMonitoring()
    logger.info(`Startup completed successfully`)
  } catch (error) {
    logger.error(`An error has occurred on startup : ${error.message}`)
    process.exit(1)
  }
}

// get the whole show started
start()
