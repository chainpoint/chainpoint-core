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

const { URL } = require('url')
const utils = require('./utils.js')

/**
 * Opens a Tendermint RPC connection
 */
async function openTendermintConnectionAsync(tendermintURI, debug) {
  let { RpcClient } = require('tendermint')
  let tmConnected = false
  let rpcClient
  while (!tmConnected) {
    try {
      rpcClient = RpcClient(tendermintURI)
      logMessage('Tendermint connection established', debug, 'general')
      tmConnected = true
    } catch (error) {
      // catch errors when attempting to establish connection
      console.error('Cannot establish Tendermint connection. Attempting in 5 seconds...')
      await utils.sleepAsync(5000)
    }
  }
  return rpcClient
}

/**
 * Opens a Redis connection
 *
 * @param {string} redisURI - The connection string for the Redis instance, an Redis URI
 * @param {function} onReady - Function to call with commands to execute when `ready` event fires
 * @param {function} onError - Function to call with commands to execute  when `error` event fires
 */
function openRedisConnection(redisURIs, onReady, onError, debug) {
  const Redis = require('ioredis')

  let redisURIList = redisURIs.split(',')

  // If redisURIs contains just a single URI, treat it as a connection to a single Redis host
  // If it contains a CSV of URIs, treat it as multiple Sentinel URIs
  let redisConfigObj = null
  if (redisURIList.length === 1) {
    // this is a single Redis host URI
    let redisURL = new URL(redisURIList[0])
    redisConfigObj = {
      port: redisURL.port, // Redis port
      host: redisURL.hostname, // Redis host
      password: redisURL.password
    }
  } else {
    // this is a list if Redis Sentinel URIs
    let password = null
    redisConfigObj = {
      sentinels: redisURIList.map(uri => {
        let redisURL = new URL(uri)
        // use the first password found as the password for all sentinels
        // store this value in 'password' for use in redisConfigObj
        if (!password) password = redisURL.password
        return {
          port: redisURL.port, // Redis port
          host: redisURL.hostname // Redis host
        }
      }),
      name: 'mymaster',
      password: password
    }
  }

  var newRedis = new Redis(redisConfigObj)

  newRedis.on('error', err => {
    console.error(`A redis error has occurred: ${err}`)
    newRedis.quit()
    onError()
    console.error('Redis connection lost. Attempting reconnect...')
  })

  newRedis.on('ready', () => {
    onReady(newRedis)
    logMessage('Redis connection established', debug, 'general')
  })
}

/**
 * Opens the Postgres connection
 **/
async function openPostgresConnectionAsync(modelSqlzArray, debug) {
  const Sequelize = require('sequelize')
  const envalid = require('envalid')

  const env = envalid.cleanEnv(process.env, {
    POSTGRES_CONNECT_PROTOCOL: envalid.str({ default: 'postgres:', desc: 'Postgres server connection protocol' }),
    POSTGRES_CONNECT_USER: envalid.str({ default: 'chainpoint', desc: 'Postgres server connection user name' }),
    POSTGRES_CONNECT_PW: envalid.str({ default: 'chainpoint', desc: 'Postgres server connection password' }),
    POSTGRES_CONNECT_HOST: envalid.str({ default: 'postgres', desc: 'Postgres server connection host' }),
    POSTGRES_CONNECT_PORT: envalid.num({ default: 5432, desc: 'Postgres server connection port' }),
    POSTGRES_CONNECT_DB: envalid.str({ default: 'chainpoint', desc: 'Postgres server connection database name' })
  })

  // Connection URI for Postgres
  const POSTGRES_CONNECT_URI = `${env.POSTGRES_CONNECT_PROTOCOL}//${env.POSTGRES_CONNECT_USER}:${
    env.POSTGRES_CONNECT_PW
  }@${env.POSTGRES_CONNECT_HOST}:${env.POSTGRES_CONNECT_PORT}/${env.POSTGRES_CONNECT_DB}`

  const sequelize = new Sequelize(POSTGRES_CONNECT_URI, { logging: null, operatorsAliases: false })

  let dbConnected = false
  let synchedModels = []
  while (!dbConnected) {
    try {
      for (let model of modelSqlzArray) {
        synchedModels.push(model.defineFor(sequelize))
      }
      await sequelize.sync({ logging: false })
      logMessage('Postgres connection established', debug, 'general')
      dbConnected = true
    } catch (error) {
      // catch errors when attempting to establish connection
      console.error('Cannot establish Postgres connection. Attempting in 5 seconds...')
      await utils.sleepAsync(5000)
    }
  }

  return {
    sequelize: sequelize,
    models: synchedModels
  }
}

/**
 * Opens an AMPQ connection and channel
 * Retry logic is included to handle losses of connection
 *
 * @param {string} connectionString - The connection URI for the RabbitMQ instance
 */
async function openStandardRMQConnectionAsync(
  amqpClient,
  connectURI,
  queues,
  prefetchCount,
  consumeObj,
  onInit,
  onClose,
  debug
) {
  let rmqConnected = false
  while (!rmqConnected) {
    try {
      // connect to rabbitmq server
      let conn = await amqpClient.connect(connectURI)
      // create communication channel
      let chan = await conn.createConfirmChannel()
      // assert all queues supplied
      queues.forEach(queue => {
        chan.assertQueue(queue, { durable: true })
      })
      // optionally set prefetch count
      if (prefetchCount !== null) chan.prefetch(prefetchCount)
      // optionally confifgure message consumption
      if (consumeObj !== null) chan.consume(consumeObj.queue, consumeObj.method)
      // initialize variables using new communication channel
      onInit(chan)
      // if the channel closes for any reason, attempt to reconnect
      conn.on('close', async () => {
        onClose()
        console.error('Connection to RabbitMQ closed.  Reconnecting in 5 seconds...')
      })
      // if the channel closes for any reason, attempt to reconnect
      conn.on('error', async error => {
        console.error(`Connection to RabbitMQ caught an error : ${error}`)
        conn.close()
      })
      logMessage('RabbitMQ connection established', debug, 'general')
      rmqConnected = true
    } catch (error) {
      // catch errors when attempting to establish connection
      console.error('Cannot establish RabbitMQ connection. Attempting in 5 seconds...')
      await utils.sleepAsync(5000)
    }
  }
}

// Instruct REST server to begin listening for request
async function listenRestifyAsync(server, port, debug) {
  return new Promise((resolve, reject) => {
    server.listen(port, err => {
      if (err) return reject(err)
      logMessage(`${server.name} listening at ${server.url}`, debug, 'general')
      return resolve()
    })
  })
}

function startIntervals(intervals, debug) {
  logMessage('starting intervals', debug, 'general')

  intervals.forEach(interval => {
    if (interval.immediate) interval.function()
    setInterval(interval.function, interval.ms)
  })
}

// SUPPORT FUNCTIONS ****************

function logMessage(message, debug, msgType) {
  if (debug && debug[msgType]) {
    debug[msgType](message)
  } else {
    console.log(message)
  }
}

module.exports = {
  openTendermintConnectionAsync: openTendermintConnectionAsync,
  openRedisConnection: openRedisConnection,
  openPostgresConnectionAsync: openPostgresConnectionAsync,
  openStandardRMQConnectionAsync: openStandardRMQConnectionAsync,
  listenRestifyAsync: listenRestifyAsync,
  startIntervals: startIntervals
}
