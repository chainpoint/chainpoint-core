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
const lightning = require('lnrpc-node-client')

const LAST_KNOWN_INVOICE_INDEX_KEY = 'LastKnownInvoiceIndex'
const INVOICE_BATCH_SIZE = 1000

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

// initialize lightning grpc object
let unlocker
let client

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

async function connectToLndAsync() {
  try {
    lightning.setTls(env.LND_SOCKET, `/root/.lnd/tls.cert`) //retry instantiating unlocker client if dns resolution fails
    unlocker = lightning.unlocker()
  } catch (error) {
    logger.error(`LND disconnect failed: ${error.message}`)
  }
  try {
    unlocker.unlockWallet({ wallet_password: env.HOT_WALLET_PASS }, (err, res) => {
      console.log(res)
      console.log(err)
    })
  } catch (error) {
    logger.error(`LND unlock failed, already unlocked? : ${error.message}`)
  }
  try {
    await utils.sleepAsync(5000)
    lightning.setCredentials(
      env.LND_SOCKET,
      `/root/.lnd/data/chain/bitcoin/${env.NETWORK}/admin.macaroon`,
      `/root/.lnd/tls.cert`
    )
    client = lightning.lightning()
  } catch (error) {
    throw new Error(`Unable to instantiate authenticated lnd client : ${error.message}`)
  }
}

async function getLastKnownInvoiceIndexAsync() {
  // return the add_index value of the newest invoice processed
  let lastKnownInvoiceIndex
  try {
    lastKnownInvoiceIndex = parseInt(await redis.get(LAST_KNOWN_INVOICE_INDEX_KEY))
    if (isNaN(lastKnownInvoiceIndex)) throw new Error('LAST_KNOWN_INVOICE_INDEX_KEY does not yet have a value')
  } catch (error) {
    logger.warn(`Unable to retrieve last known invoice index, skipping : ${error.message}`)
    lastKnownInvoiceIndex = null
  }
  return lastKnownInvoiceIndex
}

async function establishInvoiceSubscriptionAsync() {
  try {
    let invoiceSubscription = client.subscribeInvoices({})
    invoiceSubscription.on('data', async invoice => {
      let invoiceAddIndex = await processInvoiceBatchAsync([invoice])
      await updateLastKnownInvoiceIndexAsync(invoiceAddIndex)
    })
    invoiceSubscription.on('error', err => {
      logger.warn(`An invoice subscription error occurred : ${JSON.stringify(err)}`)
    })
    invoiceSubscription.on('end', async () => {
      logger.error(`The invoice subscription has unexpectedly ended`)
      startInvoiceMonitoring()
    })
    logger.info('Invoices subscription established')
  } catch (error) {
    throw new Error(`Unable to establish LND invoice subscription : ${error.message}`)
  }
}

async function checkForUnprocessedPayments(lastKnownInvoiceIndex) {
  let indexOffset = lastKnownInvoiceIndex
  let resultLength = 0
  let totalProcessed = 0
  let lastInvoiceAddIndex = null
  logger.info('Checking for unhandled invoice items')
  do {
    let unprocessedInvoices = await client.listInvoices({
      num_max_invoices: INVOICE_BATCH_SIZE,
      index_offset: indexOffset
    })
    if (unprocessedInvoices.invoices.length === 0) break
    lastInvoiceAddIndex = await processInvoiceBatchAsync(unprocessedInvoices.invoices)
    resultLength = unprocessedInvoices.invoices.length
    indexOffset += INVOICE_BATCH_SIZE
    totalProcessed += resultLength
  } while (resultLength >= INVOICE_BATCH_SIZE)
  logger.info(`${totalProcessed} invoice item(s) found and processed`)
  // return the most recent invoice index from all invoices processed in this call
  return lastInvoiceAddIndex
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

async function startInvoiceMonitoring() {
  let subscriptionEstablished = false
  while (!subscriptionEstablished) {
    try {
      // establish a connection to lnd
      await connectToLndAsync()
      // retrieve the add_index of the most recently handled invoice
      let lastKnownInvoiceIndex = await getLastKnownInvoiceIndexAsync()
      // starting listening for and handle new invoice activity
      await establishInvoiceSubscriptionAsync()
      // check if there are any backlogged invoices needing to be processed
      if (lastKnownInvoiceIndex !== null) {
        lastKnownInvoiceIndex = await checkForUnprocessedPayments(lastKnownInvoiceIndex)
        // if any invoices processed, update the last known invoice index to the proper value
        if (lastKnownInvoiceIndex) await updateLastKnownInvoiceIndexAsync(lastKnownInvoiceIndex)
      }
      subscriptionEstablished = true
    } catch (error) {
      // catch errors when attempting to connect and establish invoice subscription
      logger.error(`Invoice monitoring : ${error.message} : Attempting in 5 seconds...`)
      await utils.sleepAsync(5000)
    }
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
