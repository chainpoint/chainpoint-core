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
const env = require('./lib/parse-env.js')('api')

const amqp = require('amqplib')
const restify = require('restify')
const corsMiddleware = require('restify-cors-middleware')
const hashes = require('./lib/endpoints/hashes.js')
const nodes = require('./lib/endpoints/nodes.js')
const calendar = require('./lib/endpoints/calendar.js')
const config = require('./lib/endpoints/config.js')
const root = require('./lib/endpoints/root.js')
const cnsl = require('consul')
const registeredNode = require('./lib/models/RegisteredNode.js')
const calendarBlock = require('./lib/models/CalendarBlock.js')
const auditChallenge = require('./lib/models/AuditChallenge.js')
const zeromq = require('zeromq')
const connections = require('./lib/connections.js')

// The redis connection used for all redis communication
// This value is set once the connection has been established
let redis = null

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

let consul = null

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
server.get({ path: '/calendar/blockrange/:index', version: '1.0.0' }, calendar.getCalBlockRangeV2Async)
// get the block hash for the calendar at the specified height
server.get({ path: '/calendar/:height/hash', version: '1.0.0' }, calendar.getCalBlockHashByHeightV1Async)
// get the dataVal item for the calendar at the specified height
server.get({ path: '/calendar/:height/data', version: '1.0.0' }, calendar.getCalBlockDataByHeightV1Async)
// get the block object for the calendar at the specified height
server.get({ path: '/calendar/:height', version: '1.0.0' }, calendar.getCalBlockByHeightV1Async)
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
    calendarBlock,
    auditChallenge
  ]
  let cxObjects = await connections.openStorageConnectionAsync(sqlzModelArray)
  nodes.setDatabase(cxObjects.sequelize, cxObjects.models[0])
  hashes.setDatabase(cxObjects.sequelize, cxObjects.models[0])
  config.setDatabase(cxObjects.sequelize, cxObjects.models[1], cxObjects.models[2])
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

/**
 * Initializes ZeroMQ request and subscribe sockets
 * Requests the latest NIST value immediately
 * Receives NIST value updates when broadcasted
 *
 */
function initNISTSockets () {
  const requestSocket = zeromq.socket(`req`)
  const subscribeSocket = zeromq.socket(`sub`)

  requestSocket.connect(env.NIST_REQ_ZEROMQ_SOCKET_URI)
  subscribeSocket.connect(env.NIST_SUB_ZEROMQ_SOCKET_URI)

  requestSocket.on(`message`, function (msg) {
    console.log(`Received initial NIST value : ${msg}`)
    hashes.setNistLatest(String(msg || ''))
  })

  console.log(`Requesting initial NIST value`)
  requestSocket.send('get nist')

  subscribeSocket.subscribe(`nist`)

  subscribeSocket.on(`message`, function (topic, msg) {
    let newValue = String(msg || '')
    if (msg && hashes.getNistLatest() !== newValue) {
      console.log(`Received new NIST value : ${msg}`)
      hashes.setNistLatest(newValue)
    }
  })
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
      hashes.setRedis(redis)
      config.setRedis(redis)
      nodes.setRedis(redis)
    }, () => {
      redis = null
      hashes.setRedis(null)
      config.setRedis(null)
      nodes.setRedis(null)
      setTimeout(() => { openRedisConnection(redisURIs) }, 5000)
    })
}

// This initializes all the consul watches
function startConsulWatches () {
  let watches = [{
    key: env.MIN_NODE_VERSION_EXISTING_KEY,
    onChange: (data, res) => {
      // process only if a value has been returned
      if (data && data.Value) {
        config.setMinNodeVersionExisting(data.Value)
        nodes.setMinNodeVersionExisting(data.Value)
      }
    },
    onError: null
  }, {
    key: env.MIN_NODE_VERSION_NEW_KEY,
    onChange: (data, res) => {
      // process only if a value has been returned
      if (data && data.Value) {
        nodes.setMinNodeVersionNew(data.Value)
      }
    },
    onError: null
  }, {
    key: env.AUDIT_CHALLENGE_RECENT_KEY,
    onChange: (data, res) => {
      // process only if a value has been returned
      if (data && data.Value) {
        config.setMostRecentChallengeKey(data.Value)
      }
    },
    onError: null
  }, {
    key: env.ENFORCE_PRIVATE_STAKE_KEY,
    onChange: (data, res) => {
      // process only if a value has been returned
      if (data && data.Value) {
        hashes.setEnforcePrivateStakeState(data.Value)
      }
    },
    onError: null
  },
  {
    key: env.NODE_AGGREGATION_INTERVAL_SECONDS_KEY,
    onChange: (data, res) => {
      // process only if a value has been returned
      if (data && data.Value) {
        let newVal = parseInt(data.Value, 10)
        config.setNodeAggregationInterval(newVal)
      }
    },
    onError: null
  }]

  let defaults = [
    { key: env.MIN_NODE_VERSION_EXISTING_KEY, value: '0.0.1' },
    { key: env.MIN_NODE_VERSION_NEW_KEY, value: '0.0.1' },
    { key: env.ENFORCE_PRIVATE_STAKE_KEY, value: 'true' },
    { key: env.NODE_AGGREGATION_INTERVAL_SECONDS_KEY, value: `${env.NODE_AGGREGATION_INTERVAL_SECONDS_DEFAULT}` }
  ]
  connections.startConsulWatches(consul, watches, defaults)
}

// process all steps need to start the application
async function start () {
  if (env.NODE_ENV === 'test') return
  try {
    // init ZeroMQ sockets
    initNISTSockets()
    // init consul
    consul = connections.initConsul(cnsl, env.CONSUL_HOST, env.CONSUL_PORT)
    await config.setConsul(consul)
    // init Redis
    openRedisConnection(env.REDIS_CONNECT_URIS)
    // init DB
    await openStorageConnectionAsync()
    // init RabbitMQ
    await openRMQConnectionAsync(env.RABBITMQ_CONNECT_URI)
    // init consul watches
    startConsulWatches()
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
  setRedis: (redisClient) => {
    redis = redisClient
    hashes.setRedis(redis)
    nodes.setRedis(redis)
  },
  setAMQPChannel: (chan) => {
    hashes.setAMQPChannel(chan)
  },
  setNistLatest: (val) => { hashes.setNistLatest(val) },
  server: server,
  config: config,
  setMinNodeVersionNew: (val) => { nodes.setMinNodeVersionNew(val) },
  setMinNodeVersionExisting: (val) => { nodes.setMinNodeVersionExisting(val) },
  overrideGetTNTGrainsBalanceForAddressAsync: (func) => { nodes.overrideGetTNTGrainsBalanceForAddressAsync(func) }
}
