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
const LndGrpc = require('lnd-grpc')

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
let lnd = new LndGrpc({
  host: env.LND_HOST,
  cert: Buffer.from(env.LND_CERT, 'base64').toString(),
  macaroon: Buffer.from(env.LND_MACAROON, 'base64').toString('hex')
})

async function startInvoiceSubscription() {
  let subscriptionEstablished = false
  while (!subscriptionEstablished) {
    try {
      try {
        await lnd.connect()
      } catch (error) {
        throw new Error(`Unable to connect to LND : ${error.message}`)
      }
      try {
        let invoiceSubscription = lnd.services.Lightning.subscribeInvoices()

        invoiceSubscription.on('data', async invoice => {
          if (invoice.settled) {
            logger.info('Invoice paid: ' + invoice.memo)
            // This invoice has been paid
            // Add a short lived key to redis indicating the payment has been made
            // With this key added, the invoice id can be used to submit a hash one time
            let invoiceId = invoice.memo.split(':')[1]
            let paidInvoiceKey = `PaidSubmitHashInvoiceId:${invoiceId}`
            try {
              await redis.set(paidInvoiceKey, 1, 'EX', 1) // this key will expire 1 minute after invoice payment
            } catch (error) {
              logger.error(`Redis SET error : error setting item with key = ${paidInvoiceKey}`)
            }
          } else {
            // This invoice was just created and delivered
            logger.info('Invoice generated: ' + invoice.memo)
          }
        })
        invoiceSubscription.on('error', err => {
          logger.warn(`An invoice subscription error occurred : ${JSON.stringify(err)}`)
        })
        invoiceSubscription.on('end', async () => {
          logger.error(`The invoice subscription has unexpectedly ended`)
          try {
            await lnd.disconnect()
          } catch (error) {
            logger.error(`Unable to disconnect : ${error.message}`)
          }
          startInvoiceSubscription()
        })
        logger.info('Invoices subscription established')
        subscriptionEstablished = true
      } catch (error) {
        throw new Error(`Unable to establish LND invoice subscription : ${error.message}`)
      }
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
    startInvoiceSubscription()
    logger.info(`Startup completed successfully`)
  } catch (error) {
    logger.error(`An error has occurred on startup : ${error.message}`)
    process.exit(1)
  }
}

// get the whole show started
start()
