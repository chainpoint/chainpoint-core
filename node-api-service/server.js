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
const nodes = require('./lib/endpoints/nodes.js')
const calendar = require('./lib/endpoints/calendar.js')
const cores = require('./lib/endpoints/cores.js')
const config = require('./lib/endpoints/config.js')
const root = require('./lib/endpoints/root.js')
const registeredNode = require('./lib/models/RegisteredNode.js')
const calendarBlock = require('./lib/models/CalendarBlock.js')
const connections = require('./lib/connections.js')

const bunyan = require('bunyan')

var logger = bunyan.createLogger({
  name: 'audit',
  stream: process.stdout
})

// RESTIFY SETUP
// 'version' : all routes will default to this version
let server = restify.createServer({
  name: 'chainpoint',
  version: '1.0.0',
  log: logger
})

// LOG EVERY REQUEST
// server.pre(function (request, response, next) {
//   request.log.info({ req: [request.url, request.method, request.rawHeaders] }, 'API-REQUEST')
//   next()
// })

// Clean up sloppy paths like //todo//////1//
server.pre(restify.pre.sanitizePath())

// Checks whether the user agent is curl. If it is, it sets the
// Connection header to "close" and removes the "Content-Length" header
// See : http://restify.com/#server-api
server.pre(restify.pre.userAgentConnection())

// CORS
// See : https://github.com/TabDigital/restify-cors-middleware
// See : https://github.com/restify/node-restify/issues/1151#issuecomment-271402858
//
// Test w/
//
// curl \
// --verbose \
// --request OPTIONS \
// http://127.0.0.1:8080/hashes \
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

server.use(restify.gzipResponse())
server.use(restify.queryParser())
server.use(restify.bodyParser({
  maxBodySize: env.MAX_BODY_SIZE
}))

// API RESOURCES

// submit hash(es)
server.post({ path: '/hashes', version: '1.0.0' }, hashes.postHashV1Async)
// get the block objects for the calendar in the specified block range
server.get({ path: '/calendar/:txid', version: '1.0.0' }, calendar.getCalTxAsync)
// get the data value of a txId
server.get({ path: '/calendar/:txid/data', version: '1.0.0' }, calendar.getCalTxDataAsync)
// get proofs from rocksdb
server.get({ path: '/proofs', version: '1.0.0' }, calendar.getCoreProofsAsync)
// get random core peers
server.get({ path: '/peers', version: '1.0.0' }, cores.getCoresRandomAsync)
// get status
server.get({ path: '/status', version: '1.0.0' }, cores.getCoreStatusAsync)

// get random subset of nodes list
server.get({ path: '/nodes/random', version: '1.0.0' }, nodes.getNodesRandomV1Async)
// get nodes blacklist
server.get({ path: '/nodes/blacklist', version: '1.0.0' }, nodes.getNodesBlacklistV1Async)
// register a new node
server.post({ path: '/nodes', version: '1.0.0' }, nodes.postNodeV1Async)
// update an existing node
server.put({ path: '/nodes/:tnt_addr', version: '1.0.0' }, nodes.putNodeV1Async)
// get configuration information for this stack
server.get({ path: '/config', version: '1.0.0' }, config.getConfigInfoV1Async)
// get heartbeat
server.get({ path: '/heartbeat', version: '1.0.0' }, root.getHeartbeatV1)
// teapot
server.get({ path: '/', version: '1.0.0' }, root.getV1)

/**
 * Opens a storage connection
 **/
async function openStorageConnectionAsync () {
  let sqlzModelArray = [
    registeredNode,
    calendarBlock
  ]
  let cxObjects = await connections.openStorageConnectionAsync(sqlzModelArray)
  nodes.setDatabase(cxObjects.sequelize, cxObjects.models[0])
  hashes.setDatabase(cxObjects.sequelize, cxObjects.models[0])
  calendar.setDatabase(cxObjects.sequelize, cxObjects.models[1])
}

/**
 * Opens an AMPQ connection and channel
 * Retry logic is included to handle losses of connection
 *
 * @param {string} connectURI - The connection URI for the RabbitMQ instance
 */
async function openRMQConnectionAsync (connectURI) {
  await connections.openStandardRMQConnectionAsync(amqp, connectURI,
    [env.RMQ_WORK_OUT_AGG_QUEUE],
    null,
    null,
    (chan) => { hashes.setAMQPChannel(chan) },
    () => {
      hashes.setAMQPChannel(null)
      setTimeout(() => { openRMQConnectionAsync(connectURI) }, 5000)
    }
  )
}

// process all steps need to start the application
async function start () {
  if (env.NODE_ENV === 'test') return
  try {
    // init DB
    await openStorageConnectionAsync()
    // init RabbitMQ
    await openRMQConnectionAsync(env.RABBITMQ_CONNECT_URI)
    // Init Restify
    await connections.listenRestifyAsync(server, 8080)
    console.log('startup completed successfully')
  } catch (error) {
    console.error(`An error has occurred on startup: ${error.message}`)
    process.exit(1)
  }
}

// get the whole show started
start()

// export these functions for testing purposes
module.exports = {
  setAMQPChannel: (chan) => {
    hashes.setAMQPChannel(chan)
  },
  server: server,
  config: config,
  overrideGetTNTGrainsBalanceForAddressAsync: (func) => { nodes.overrideGetTNTGrainsBalanceForAddressAsync(func) }
}
