const { URL } = require('url')
const utils = require('./utils.js')

/**
 * Opens a Redis connection
 *
 * @param {string} redisURI - The connection string for the Redis instance, an Redis URI
 * @param {function} onReady - Function to call with commands to execute when `ready` event fires
 * @param {function} onError - Function to call with commands to execute  when `error` event fires
 */
function openRedisConnection (redisURIs, onReady, onError, debug) {
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
      sentinels: redisURIList.map((uri) => {
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

  newRedis.on('error', (err) => {
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
 * Initializes the connection to the Resque queue when Redis is ready
 */
async function initResqueQueueAsync (redisClient, namespace, debug) {
  const nodeResque = require('node-resque')
  const exitHook = require('exit-hook')
  var connectionDetails = { redis: redisClient }

  const queue = new nodeResque.Queue({ connection: connectionDetails })
  queue.on('error', function (error) { console.error(error.message) })
  await queue.connect()

  exitHook(async () => {
    await queue.end()
  })

  logMessage('Resque queue connection established', debug, 'general')

  return queue
}

/**
 * Initializes and configures the connection to the Resque worker when Redis is ready
 */
async function initResqueWorkerAsync (redisClient, namespace, queues, minTasks, maxTasks, taskTimeout, jobs, setMWHandlers, debug) {
  const nodeResque = require('node-resque')
  const exitHook = require('exit-hook')
  var connectionDetails = { redis: redisClient }

  var multiWorkerConfig = {
    connection: connectionDetails,
    queues: queues,
    minTaskProcessors: minTasks,
    maxTaskProcessors: maxTasks
  }

  await cleanUpWorkersAndRequequeJobsAsync(nodeResque, connectionDetails, taskTimeout)

  let multiWorker = new nodeResque.MultiWorker(multiWorkerConfig, jobs, debug)

  setMWHandlers(multiWorker)

  multiWorker.start()

  exitHook(async () => {
    await multiWorker.end()
  })

  logMessage(`Resque worker connection established for queues ${JSON.stringify(queues)}`, debug, 'general')
}

async function initResqueSchedulerAsync (redisClient, setSchedulerHandlers, debug) {
  const nodeResque = require('node-resque')
  let connectionDetails = { redis: redisClient }

  // Start Resqueue Scheduler for delayed Jobs
  const scheduler = new nodeResque.Scheduler({ connection: connectionDetails })
  await scheduler.connect()

  setSchedulerHandlers(scheduler)

  scheduler.start()

  logMessage(`Resque Scheduler connection established for queue(s)`, debug, 'general')
}

/**
 * Opens a storage connection
 **/
async function openStorageConnectionAsync (modelSqlzArray, debug) {
  const Sequelize = require('sequelize-cockroachdb')
  const envalid = require('envalid')
  const pg = require('pg')

  const env = envalid.cleanEnv(process.env, {
    COCKROACH_HOST: envalid.str({ devDefault: 'roach1', desc: 'CockroachDB host or IP' }),
    COCKROACH_PORT: envalid.num({ default: 26257, desc: 'CockroachDB port' }),
    COCKROACH_DB_NAME: envalid.str({ default: 'chainpoint', desc: 'CockroachDB name' }),
    COCKROACH_DB_USER: envalid.str({ default: 'chainpoint', desc: 'CockroachDB user' }),
    COCKROACH_DB_PASS: envalid.str({ default: '', desc: 'CockroachDB password' }),
    COCKROACH_TLS_CA_CRT: envalid.str({ devDefault: '', desc: 'CockroachDB TLS CA Cert' }),
    COCKROACH_TLS_CLIENT_KEY: envalid.str({ devDefault: '', desc: 'CockroachDB TLS Client Key' }),
    COCKROACH_TLS_CLIENT_CRT: envalid.str({ devDefault: '', desc: 'CockroachDB TLS Client Cert' })
  })

  let pgConfig = {
    user: env.COCKROACH_DB_USER,
    host: env.COCKROACH_HOST,
    database: env.COCKROACH_DB_NAME,
    port: env.COCKROACH_PORT,
    idleTimeoutMillis: 30000,
    connectionTimeoutMillis: 2000
  }

  // Connect to CockroachDB through Sequelize.
  let sequelizeOptions = {
    dialect: 'postgres',
    host: env.COCKROACH_HOST,
    port: env.COCKROACH_PORT,
    logging: false,
    operatorsAliases: false,
    pool: {
      max: 20,
      min: 0,
      idle: 10000,
      acquire: 10000,
      evict: 10000
    }
  }

  // Present TLS client certificate to production cluster
  if (env.isProduction) {
    sequelizeOptions.dialectOptions = {
      ssl: {
        rejectUnauthorized: false,
        ca: env.COCKROACH_TLS_CA_CRT,
        key: env.COCKROACH_TLS_CLIENT_KEY,
        cert: env.COCKROACH_TLS_CLIENT_CRT
      }
    }
    pgConfig.ssl = {
      rejectUnauthorized: false,
      ca: env.COCKROACH_TLS_CA_CRT,
      key: env.COCKROACH_TLS_CLIENT_KEY,
      cert: env.COCKROACH_TLS_CLIENT_CRT
    }
  }

  let pgClientPool = new pg.Pool(pgConfig)
  let sequelize = new Sequelize(env.COCKROACH_DB_NAME, env.COCKROACH_DB_USER, env.COCKROACH_DB_PASS, sequelizeOptions)

  let dbConnected = false
  let synchedModels = []
  while (!dbConnected) {
    try {
      for (let model of modelSqlzArray) {
        synchedModels.push(model.defineFor(sequelize))
      }
      await sequelize.sync({ logging: false })
      logMessage('Sequelize connection established', debug, 'general')
      dbConnected = true
    } catch (error) {
      // catch errors when attempting to establish connection
      console.error('Cannot establish Sequelize connection. Attempting in 5 seconds...')
      await utils.sleep(5000)
    }
  }

  return {
    sequelize: sequelize,
    pgClientPool: pgClientPool,
    models: synchedModels
  }
}

/**
 * Opens an AMPQ connection and channel
 * Retry logic is included to handle losses of connection
 *
 * @param {string} connectionString - The connection URI for the RabbitMQ instance
 */
async function openStandardRMQConnectionAsync (amqpClient, connectURI, queues, prefetchCount, consumeObj, onInit, onClose, debug) {
  let rmqConnected = false
  while (!rmqConnected) {
    try {
      // connect to rabbitmq server
      let conn = await amqpClient.connect(connectURI)
      // create communication channel
      let chan = await conn.createConfirmChannel()
      // assert all queues supplied
      queues.forEach(queue => { chan.assertQueue(queue, { durable: true }) })
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
      conn.on('error', async (error) => {
        console.error(`Connection to RabbitMQ caught an error : ${error}`)
        conn.close()
      })
      logMessage('RabbitMQ connection established', debug, 'general')
      rmqConnected = true
    } catch (error) {
      // catch errors when attempting to establish connection
      console.error('Cannot establish RabbitMQ connection. Attempting in 5 seconds...')
      await utils.sleep(5000)
    }
  }
}

// Initializes and returns a consul client object
function initConsul (consulClient, host, port, debug) {
  let consul = consulClient({ host: host, port: port })
  logMessage('Consul connection established', debug, 'general')
  return consul
}

// Instruct REST server to begin listening for request
async function listenRestifyAsync (server, port, debug) {
  return new Promise((resolve, reject) => {
    server.listen(port, (err) => {
      if (err) return reject(err)
      logMessage(`${server.name} listening at ${server.url}`, debug, 'general')
      return resolve()
    })
  })
}

// Performs a leader election across all instances using the given leader key
function performLeaderElection (electorClient, leaderKey, host, port, id, onElect, onError, debug) {
  const uuidv1 = require('uuid/v1')
  let clientToken = uuidv1()
  let leaderElectionConfig = {
    key: leaderKey,
    consul: {
      host: host,
      port: port,
      ttl: 15,
      lockDelay: 1,
      token: clientToken
    }
  }

  electorClient(leaderElectionConfig)
    .on('gainedLeadership', () => {
      logMessage(`leaderElection : elected : ${id || 'no id supplied'}`, debug, 'general')
      onElect()
    })
    .on('error', (err) => {
      console.error(`leaderElection : error : lock session invalidated : ${err}`)
      onError()
    })
}

// This initializes all the consul watches
function startConsulWatches (consul, watches, defaults, debug) {
  logMessage('starting watches', debug, 'general')

  // Process any new watches to be initialized
  if (watches !== null) {
    watches.forEach((watchItem) => {
      // Continuous watch on the consul key
      let watch = consul.watch({ method: consul.kv.get, options: { key: watchItem.key } })
      // When the value changes, handle appropriately
      watch.on('change', (data, res) => { watchItem.onChange(data, res) })
      // Handle and log any error events
      watch.on('error', (err) => {
        if (watchItem.onError !== null) watchItem.onError(err)
        console.error(`consul watch error for key ${watchItem.key} : ${err.message}`)
      })
    })
  }

  // Process any new default values to be set
  if (defaults) {
    defaults.forEach((defaultItem) => {
      consul.kv.get(defaultItem.key, function (err, result) {
        if (err) {
          console.error(err)
        } else {
          // Only create key if it doesn't exist or has no value
          if (!result) {
            consul.kv.set(defaultItem.key, defaultItem.value, function (err, result) {
              if (err) throw err
              logMessage(`created ${defaultItem.key} key with default value of ${defaultItem.value} `, debug, 'general')
            })
          }
        }
      })
    })
  }
}

function startIntervals (intervals, debug) {
  logMessage('starting intervals', debug, 'general')

  intervals.forEach((interval) => {
    if (interval.immediate) interval.function()
    setInterval(interval.function, interval.ms)
  })
}

// SUPPORT FUNCTIONS ****************

async function cleanUpWorkersAndRequequeJobsAsync (nodeResque, connectionDetails, taskTimeout, debug) {
  const queue = new nodeResque.Queue({ connection: connectionDetails })
  await queue.connect()
  // Delete stuck workers and move their stuck job to the failed queue
  await queue.cleanOldWorkers(taskTimeout)
  // Get the count of jobs in the failed queue
  let failedCount = await queue.failedCount()
  // Retrieve failed jobs in batches of 100
  // First, determine the batch ranges to retrieve
  let batchSize = 100
  let failedBatches = []
  for (let x = 0; x < failedCount; x += batchSize) {
    failedBatches.push({ start: x, end: x + batchSize - 1 })
  }
  // Retrieve the failed jobs for each batch and collect in 'failedJobs' array
  let failedJobs = []
  for (let failedBatch of failedBatches) {
    let failedJobSet = await queue.failed(failedBatch.start, failedBatch.end)
    failedJobs = failedJobs.concat(failedJobSet)
  }
  // For each job, remove the job from the failed queue and requeue to its original queue
  for (let failedJob of failedJobs) {
    logMessage(`Requeuing job: ${failedJob.payload.queue} : ${failedJob.payload.class} : ${failedJob.error} `, debug, 'worker')
    await queue.retryAndRemoveFailed(failedJob)
  }
}

function logMessage (message, debug, msgType) {
  if (debug && debug[msgType]) {
    debug[msgType](message)
  } else {
    console.log(message)
  }
}

module.exports = {
  openRedisConnection: openRedisConnection,
  initResqueQueueAsync: initResqueQueueAsync,
  initResqueWorkerAsync: initResqueWorkerAsync,
  initResqueSchedulerAsync: initResqueSchedulerAsync,
  openStorageConnectionAsync: openStorageConnectionAsync,
  openStandardRMQConnectionAsync: openStandardRMQConnectionAsync,
  initConsul: initConsul,
  listenRestifyAsync: listenRestifyAsync,
  performLeaderElection: performLeaderElection,
  startConsulWatches: startConsulWatches,
  startIntervals: startIntervals
}
