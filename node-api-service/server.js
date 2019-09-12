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
const env = require('./lib/parse-env.js')('api')

const amqp = require('amqplib')
const restify = require('restify')
const corsMiddleware = require('restify-cors-middleware')
const hashes = require('./lib/endpoints/hashes.js')
const calendar = require('./lib/endpoints/calendar.js')
const peers = require('./lib/endpoints/peers.js')
const proofs = require('./lib/endpoints/proofs.js')
const status = require('./lib/endpoints/status.js')
const root = require('./lib/endpoints/root.js')
const connections = require('./lib/connections.js')
const stakedCore = require('./lib/models/StakedCore.js')
const proof = require('./lib/models/Proof.js')
const tmRpc = require('./lib/tendermint-rpc.js')
const logger = require('./lib/logger.js')
const bunyan = require('bunyan')
const apicache = require('apicache')
const LndGrpc = require('lnd-grpc')
const utils = require('./lib/utils.js')

let redisCache = null

var apiLogs = bunyan.createLogger({
  name: 'audit',
  stream: process.stdout
})

// RESTIFY SETUP
// 'version' : all routes will default to this version
const httpOptions = {
  name: 'chainpoint',
  version: '1.0.0',
  log: apiLogs
}

const applyMiddleware = (middlewares = []) => {
  if (process.env.NODE_ENV === 'development' || process.env.NETWORK === 'testnet') {
    return []
  } else {
    return middlewares
  }
}

let throttle = (burst, rate, opts = { ip: true }) => {
  return restify.plugins.throttle(Object.assign({}, { burst, rate }, opts))
}

function setupRestifyConfigAndRoutes(server) {
  // LOG EVERY REQUEST
  // server.pre(function (request, response, next) {
  //   request.log.info({ req: [request.url, request.method, request.rawHeaders] }, 'API-REQUEST')
  //   next()
  // })

  // Clean up sloppy paths like //todo//////1//
  server.pre(restify.plugins.pre.sanitizePath())

  // Checks whether the user agent is curl. If it is, it sets the
  // Connection header to "close" and removes the "Content-Length" header
  // See : http://restify.com/#server-api
  server.pre(restify.plugins.pre.userAgentConnection())

  // CORS
  // See : https://github.com/TabDigital/restify-cors-middleware
  // See : https://github.com/restify/node-restify/issues/1151#issuecomment-271402858
  //
  // Test w/
  //
  // curl \
  // --verbose \
  // --request OPTIONS \
  // http://127.0.0.1/hashes \
  // --header 'Origin: http://localhost:9292' \
  // --header 'Access-Control-Request-Headers: Origin, Accept, Content-Type' \
  // --header 'Access-Control-Request-Method: POST'
  //
  var cors = corsMiddleware({
    preflightMaxAge: 600,
    origins: ['*']
  })
  server.pre(cors.preflight)
  server.use(cors.actual)

  server.use(restify.plugins.gzipResponse())
  server.use(restify.plugins.queryParser())
  server.use(restify.plugins.bodyParser({ maxBodySize: env.MAX_BODY_SIZE, mapParams: true }))

  // API RESOURCES

  // get hash invoice
  server.get(
    { path: '/hash/invoice', version: '1.0.0' },
    ...applyMiddleware([throttle(5, 1)]),
    hashes.getHashInvoiceV1Async
  )
  // submit hash
  server.post({ path: '/hash', version: '1.0.0' }, ...applyMiddleware([throttle(5, 1)]), hashes.postHashV1Async)
  // get the block objects for the calendar in the specified block range
  server.get(
    { path: '/calendar/:txid', version: '1.0.0' },
    ...applyMiddleware([throttle(50, 10)]),
    calendar.getCalTxAsync
  )
  // get the data value of a txId
  server.get(
    { path: '/calendar/:txid/data', version: '1.0.0' },
    ...applyMiddleware([throttle(50, 10)]),
    calendar.getCalTxDataAsync
  )
  // get proofs from storage
  if (redisCache) {
    server.get(
      { path: '/proofs', version: '1.0.0' },
      ...applyMiddleware(throttle(50, 10)),
      redisCache('1 minute'),
      proofs.getProofsByIDsAsync
    )
    logger.info('Redis caching middleware added')
  } else {
    server.get(
      { path: '/proofs', version: '1.0.0' },
      ...applyMiddleware([throttle(50, 10)]),
      proofs.getProofsByIDsAsync
    )
  }
  // get random core peers
  server.get({ path: '/peers', version: '1.0.0' }, ...applyMiddleware([throttle(15, 3)]), peers.getPeersAsync)
  // get status
  server.get({ path: '/status', version: '1.0.0' }, ...applyMiddleware([throttle(15, 3)]), status.getCoreStatusAsync)
  // teapot
  server.get({ path: '/', version: '1.0.0' }, root.getV1)
}

// HTTP Server
async function startInsecureRestifyServerAsync() {
  let restifyServer = restify.createServer(httpOptions)
  setupRestifyConfigAndRoutes(restifyServer)

  // Begin listening for requests
  await connections.listenRestifyAsync(restifyServer, 8080)
  return restifyServer
}

/**
 * Opens a Redis connection
 *
 * @param {string} redisURI - The connection string for the Redis instance, an Redis URI
 */
function openRedisConnection(redisURIs) {
  return new Promise(resolve => {
    connections.openRedisConnection(
      redisURIs,
      newRedis => {
        hashes.setRedis(newRedis)
        resolve(newRedis)
        redisCache = apicache.options({
          redisClient: newRedis,
          debug: true,
          appendKey: req => req.headers.hashids
        }).middleware
      },
      () => {
        hashes.setRedis(null)
        setTimeout(() => {
          openRedisConnection(redisURIs).then(() => resolve())
        }, 5000)
      }
    )
  })
}

/**
 * Opens a Postgres connection
 **/
async function openPostgresConnectionAsync() {
  let sqlzModelArray = [proof, stakedCore]
  let cxObjects = await connections.openPostgresConnectionAsync(sqlzModelArray)
  proof.setDatabase(cxObjects.sequelize, cxObjects.op, cxObjects.models[0])
}

/**
 * Opens a Tendermint connection
 **/
async function openTendermintConnectionAsync() {
  let rpcClient = await connections.openTendermintConnectionAsync(env.TENDERMINT_URI)
  tmRpc.setRpcClient(rpcClient)
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
    [env.RMQ_WORK_OUT_AGG_QUEUE],
    null,
    null,
    chan => {
      hashes.setAMQPChannel(chan)
    },
    () => {
      hashes.setAMQPChannel(null)
      setTimeout(() => {
        openRMQConnectionAsync(connectURI)
      }, 5000)
    }
  )
}

async function connectToLndAsync() {
    try {
      // initialize lightning grpc object
      console.log(`connecting to ${env.LND_SOCKET}`)
      let lnd = new LndGrpc({
          host: env.LND_SOCKET,
          cert: `/root/.lnd/tls.cert`,
          macaroon: `/root/.lnd/data/chain/bitcoin/${env.NETWORK}/admin.macaroon`
      })
      try {
        await lnd.disconnect()
      } catch (error) {
        console.error(`LND disconnect failed: ${error.message}`)
      }
      try {
        lnd.once('active', async () => {
          console.info('GRPC state active')
        })
        await lnd.connect()
        if (lnd.state === 'locked') {
          try {
            await lnd.services.WalletUnlocker.unlockWallet({
                wallet_password: Buffer.from(env.HOT_WALLET_PASS),
            })
            await lnd.activateLightning()
          } catch (error) {
            console.error(`Can't unlock LND: ${error.message}`)
          }
        }
      } catch (error) {
        throw new Error(`Unable to connect to LND : ${error.message}`)
      }
      return lnd
    } catch (error) {
      throw new Error(`Unable to connect to LND : ${error.message}`)
    }
}

async function establishTransactionSubscriptionAsync(lnd) {
  try {
    let transactionSubscription = lnd.services.Lightning.subscribeTransactions()
    transactionSubscription.on('data', () => {})
    transactionSubscription.on('error', err => {
      logger.warn(`An transaction subscription error occurred : ${JSON.stringify(err)}`)
    })
    transactionSubscription.on('end', async () => {
      logger.error(`The transaction subscription has unexpectedly ended`)
      hashes.setLND(null)
      status.setLND(null)
      try {
        await lnd.disconnect()
      } catch (error) {
        logger.error(`Unable to disconnect : ${error.message}`)
      }
      openLndConnectionAsync()
    })
    logger.info('Transaction subscription established')
  } catch (error) {
    throw new Error(`Unable to establish LND transaction subscription : ${error.message}`)
  }
}

/**
 * Opens the Lightning node connection
 * Retry logic is included inside transaction subscription event handlers
 *
 * @param {string} connectURI - The connection URI for the RabbitMQ instance
 */
async function openLndConnectionAsync() {
  let connectionEstablished = false
  while (!connectionEstablished) {
    try {
      // establish a connection to lnd
      let lnd = await connectToLndAsync()
      // starting listening for new transaction activity and handle subscription/connection loss
      await establishTransactionSubscriptionAsync(lnd)
      hashes.setLND(lnd)
      status.setLND(lnd)
      connectionEstablished = true
    } catch (error) {
      // catch errors when attempting to connect and establish subscription
      logger.error(`Lightning connection : ${error.message} : Attempting in 5 seconds...`)
      await utils.sleepAsync(5000)
    }
  }
}

// process all steps need to start the application
async function start() {
  if (env.NODE_ENV === 'test') return
  try {
    // init Redis
    await openRedisConnection(env.REDIS_CONNECT_URIS)
    // init DB
    await openPostgresConnectionAsync()
    // init Tendermint
    await openTendermintConnectionAsync()
    // init RabbitMQ
    await openRMQConnectionAsync(env.RABBITMQ_CONNECT_URI)
    // Init Restify
    await startInsecureRestifyServerAsync()
    // Init LND
    await openLndConnectionAsync()
    logger.info(`Startup completed successfully`)
  } catch (error) {
    logger.error(`An error has occurred on startup : ${error.message}`)
    process.exit(1)
  }
}

// get the whole show started
start()

// export these functions for testing purposes
module.exports = {
  setAMQPChannel: chan => {
    hashes.setAMQPChannel(chan)
  },
  // additional functions for testing purposes
  startInsecureRestifyServerAsync: startInsecureRestifyServerAsync,
  setThrottle: t => {
    throttle = t
  }
}
