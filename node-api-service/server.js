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
const proof = require('./lib/models/Proof.js')
const tmRpc = require('./lib/tendermint-rpc.js')
const logger = require('./lib/logger.js')
const utils = require('./lib/utils.js')
const lndClient = require('lnrpc-node-client')
const fs = require('fs')

const { LND_SOCKET, LND_TLS_CERT, LND_MACAROON } = env

const applyProductionMiddleware = (middlewares = []) => {
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
  // server.pre(function(request, response, next) {
  //   logger.info(`req: ${request.method} ${request.url}`, 'API-REQUEST')
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

  // retrieve information associated with an LSAT attached in the header
  server.get(
    { path: '/boltwall/invoice', version: '1.0.0' },
    ...applyProductionMiddleware([throttle(5, 2)]),
    hashes.boltwall
  )

  // retrieve basic information about the node that will be receiving
  // payments for hash submissions
  server.get(
    { path: '/boltwall/node', version: '1.0.0' },
    ...applyProductionMiddleware([throttle(5, 2)]),
    hashes.boltwall
  )

  // boltwall protected hash submission
  // if no hodl invoice has been requested or a requested one has not been
  // paid then this will fail
  server.post(
    { path: '/hash', version: '1.0.0' },
    ...applyProductionMiddleware([throttle(5, 2)]),
    hashes.validatePostHashRequest,
    hashes.parsePostHashRequest,
    hashes.boltwall,
    hashes.postHashV1Async
  )

  // get the block objects for the calendar in the specified block range
  server.get(
    { path: '/calendar/:txid', version: '1.0.0' },
    ...applyProductionMiddleware([throttle(50, 10)]),
    calendar.getCalTxAsync
  )

  // get the data value of a txId
  server.get(
    { path: '/calendar/:txid/data', version: '1.0.0' },
    ...applyProductionMiddleware([throttle(50, 10)]),
    calendar.getCalTxDataAsync
  )
  // get proofs from storage
  server.get(
    { path: '/proofs', version: '1.0.0' },
    ...applyProductionMiddleware([throttle(50, 10)]),
    proofs.getProofsByIDsAsync
  )
  // get random core peers
  server.get({ path: '/peers', version: '1.0.0' }, ...applyProductionMiddleware([throttle(15, 3)]), peers.getPeersAsync)
  // serve aggregator whitelist for clients to find free service
  server.get(
    { path: '/gateways/public', version: '1.0.0' },
    ...applyProductionMiddleware([throttle(15, 3)]),
    peers.getGatewayWhitelistAsync
  )
  // get status
  server.get(
    { path: '/status', version: '1.0.0' },
    ...applyProductionMiddleware([throttle(15, 3)]),
    status.getCoreStatusAsync
  )
  // teapot
  server.get({ path: '/', version: '1.0.0' }, root.getV1)

  // error handler
  // eslint-disable-next-line no-unused-vars
  server.on('restifyError', (req, res, err, cb) => {
    logger.error(err)
    return cb()
  })
}

// HTTP Server
async function startAPIServerAsync() {
  let restifyServer = restify.createServer({ name: 'Chainpoint Core' })
  setupRestifyConfigAndRoutes(restifyServer)

  // Begin listening for requests
  await connections.listenRestifyAsync(restifyServer, 8080)
  return restifyServer
}

/**
 * Opens a Postgres connection
 **/
async function openPostgresConnectionAsync() {
  let sqlzModelArray = [proof]
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

async function openLightningConnection() {
  let connectionEstablished = false
  if (!LND_TLS_CERT && !fs.existsSync(LND_TLS_CERT)) {
    logger.warn(`LND TLS Cert not found or not yet generated. Trying without...`)
  }
  while (!connectionEstablished) {
    try {
      // establish a connection to lnd
      try {
        if (!LND_TLS_CERT) lndClient.setCredentials(LND_SOCKET, LND_MACAROON)
        else lndClient.setCredentials(LND_SOCKET, LND_MACAROON, LND_TLS_CERT)
        let lightning = lndClient.lightning()
        let invoicesClient = lndClient.invoice()
        // attempt a get info call, this will fail if wallet is still locked
        await lightning.getInfoAsync({})
        hashes.setLND(lightning)
        hashes.setInvoiceClient(invoicesClient)
        status.setLND(lightning)
      } catch (error) {
        let message = error.message
        if (error.code === 12) message = 'Wallet locked'
        throw new Error(`Cannot establish active LND connection : ${message}`)
      }
      connectionEstablished = true
    } catch (error) {
      // catch errors when attempting to connect and establish invoice subscription
      logger.error(`LND connection : ${error.message} : Retrying in 5 seconds...`)
      await utils.sleepAsync(5000)
    }
  }
}

// process all steps need to start the application
async function start() {
  if (env.NODE_ENV === 'test') return
  try {
    // init DB
    await openPostgresConnectionAsync()
    // init Tendermint
    await openTendermintConnectionAsync()
    // init RabbitMQ
    await openRMQConnectionAsync(env.RABBITMQ_CONNECT_URI)
    // Init Restify
    await startAPIServerAsync()
    // Init listening for lnd transaction update events
    await openLightningConnection()
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
  startAPIServerAsync: startAPIServerAsync,
  setThrottle: t => {
    throttle = t
  }
}
