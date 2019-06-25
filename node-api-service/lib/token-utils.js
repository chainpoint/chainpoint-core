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

const errors = require('restify-errors')
const jwt = require('jsonwebtoken')
const jose = require('node-jose')
let rp = require('request-promise-native')
const retry = require('async-retry')
let status = require('./endpoints/status.js')
let stakedCore = require('./models/StakedCore.js')
const env = require('./parse-env.js')('api')
const logger = require('./logger.js')
let tmRpc = require('../tendermint-rpc.js')

// These are token signature verification methods that are used in more than one of the API endpoints
// The purpose of this file is to prevent duplication of these token signature verification methods

const CORE_JWK_KEY_PREFIX = 'CorePublicKey'
const CORE_ID_KEY = 'CoreID'
const CACHED_ISS_VALUES_KEY = 'CachedISSValues'

// The redis connection used for all redis communication
// This value is set once the connection has been established
let redis = null

// cache old token hash so we can confirm node received new token in /hashes
async function cacheTokenHashes(prevTokenHash, currTokenHash) {
  if (redis) {
    try {
      await redis.set(currTokenHash, prevTokenHash, 'EX', 30 * 60 * 60 * 24)
      return true
    } catch (error) {
      logger.warn(`Redis write error : isKnownPeerIPAsync : ${error.message}`)
      return false
    }
  }
}

async function getPrevTokenHash(currTokenHash) {
  if (redis) {
    try {
      let cacheResult = await redis.get(currTokenHash)
      if (cacheResult) {
        return cacheResult
      }
    } catch (error) {
      logger.warn(`Redis read error : getCachedCoreIDAsync : ${error.message}`)
    }
  }
  return null
}

async function delPrevTokenHash(currTokenHash) {
  if (redis) {
    try {
      await redis.del(currTokenHash)
      return true
    } catch (error) {
      logger.warn(`Redis read error : getCachedCoreIDAsync : ${error.message}`)
    }
  }
  return false
}

async function verifySigAsync(tokenString, decodedToken) {
  // verify signature of token
  try {
    // get the token's key id
    let kid = decodedToken.header.kid
    if (!kid) return new errors.InvalidArgumentError('invalid request, token missing `kid` value')
    // get the token's issuer, the Core URI
    let iss = decodedToken.payload.iss
    if (!iss) return new errors.InvalidArgumentError('invalid request, token missing `iss` value')

    // ensure that the issuer ('iss') is a valid, known peer on the network
    let isKnownPeer = await isKnownPeerIPAsync(iss)
    if (!isKnownPeer) return new errors.InvalidArgumentError('invalid request, `iss` not a known network peer')

    // get the JWK for the given token
    let jwkObj = await getCachedJWKAsync(kid, iss)
    if (!jwkObj) return new errors.InvalidArgumentError('invalid request, unable to find public key for given kid')
    let jwk = await jose.JWK.asKey(jwkObj, 'json')
    jwt.verify(tokenString, jwk.toPEM(), { complete: true, ignoreExpiration: true })
  } catch (error) {
    return new errors.InvalidArgumentError('invalid request, token signature cannot be verified')
  }

  return null
}

async function isKnownPeerIPAsync(iss) {
  // Parsed iss
  let parsedISS = iss.replace(/^(http|https):\/\//i, '')
  // first, attempt to read value from Redis
  let redisKey = `${CACHED_ISS_VALUES_KEY}:${iss}`
  if (redis) {
    try {
      let cachedISSValue = await redis.get(redisKey)
      if (cachedISSValue) return cachedISSValue === 'true'
    } catch (error) {
      logger.warn(`Redis read error : isKnownPeerIPAsync : ${error.message}`)
    }
  }
  // a value was not found in Redis, so check the database
  let isKnown = false
  try {
    isKnown = await stakedCore.hasMemberIPAsync(parsedISS)
    logger.info(`Iss: ${iss}, isKnown: ${isKnown}`)
  } catch (error) {
    logger.error(`Database read error : isKnownPeerIPAsync : ${error.message}`)
  }
  // add to the cache
  if (redis) {
    try {
      // cached known IPs for 24 hours, unknown for 1 minute
      await redis.set(redisKey, isKnown, 'EX', isKnown ? 60 * 60 * 24 : 60)
    } catch (error) {
      logger.warn(`Redis write error : isKnownPeerIPAsync : ${error.message}`)
    }
  }

  return isKnown
}

async function getCachedJWKAsync(kid, iss) {
  // first, attempt to read value from Redis
  let redisKey = `${CORE_JWK_KEY_PREFIX}:${kid}`
  if (redis) {
    try {
      let cacheResult = await redis.get(redisKey)
      if (cacheResult) return JSON.parse(cacheResult)
    } catch (error) {
      logger.warn(`Redis read error : getTokenJWKAsync : ${error.message}`)
    }
  }
  // a value was not found in Redis, so ask the specific Core directly
  let result = null
  try {
    let coreStatus = await coreStatusRequestAsync(iss)
    if (!coreStatus) return null
    if (!coreStatus.jwk || coreStatus.jwk.kid !== kid) return null
    result = coreStatus.jwk
  } catch (error) {
    logger.error(`Core request error : getTokenJWKAsync : ${error.message}`)
  }
  // if a non cached value was found, add to the cache
  if (result && redis) {
    try {
      await redis.set(redisKey, JSON.stringify(result))
    } catch (error) {
      logger.warn(`Redis write error : getTokenJWKAsync : ${error.message}`)
    }
  }

  return result
}

async function getCachedCoreIDAsync() {
  // first, attempt to read value from Redis
  if (redis) {
    try {
      let cacheResult = await redis.get(CORE_ID_KEY)
      if (cacheResult) return cacheResult
    } catch (error) {
      logger.warn(`Redis read error : getCachedCoreIDAsync : ${error.message}`)
    }
  }
  // a value was not found in Redis, so ask the specific Core directly
  let result = null
  try {
    let coreStatus = await coreStatusRequestAsync(env.CHAINPOINT_CORE_BASE_URI)
    if (!coreStatus) return null
    if (!coreStatus.node_info || !coreStatus.node_info.id) return null
    result = coreStatus.node_info.id
  } catch (error) {
    logger.error(`Core request error : getCachedCoreIDAsync : ${error.message}`)
  }
  // if a non cached value was found, add to the cache
  if (result && redis) {
    try {
      await redis.set(CORE_ID_KEY, result)
    } catch (error) {
      logger.warn(`Redis write error : getCachedCoreIDAsync : ${error.message}`)
    }
  }

  return result
}

async function coreStatusRequestAsync(coreURI, retryCount = 3) {
  // if we need /status from ourselves, skip the HTTP call and attain directly
  if (coreURI === env.CHAINPOINT_CORE_BASE_URI) {
    let result = await status.buildStatusObjectAsync()
    return result.status
  }
  let options = {
    method: 'GET',
    uri: `${coreURI}/status`,
    json: true,
    gzip: true,
    timeout: 2000,
    resolveWithFullResponse: true
  }

  let response
  await retry(
    async bail => {
      try {
        response = await rp(options)
      } catch (error) {
        // If no response was received or there is a status code >= 500, then we should retry the call, throw an error
        if (!error.statusCode || error.statusCode >= 500) throw error
        // errors like 409 Conflict or 400 Bad Request are not retried because the request is bad and will never succeed
        bail(error)
      }
    },
    {
      retries: retryCount, // The maximum amount of times to retry the operation. Default is 3
      factor: 1, // The exponential factor to use. Default is 2
      minTimeout: 200, // The number of milliseconds before starting the first retry. Default is 200
      maxTimeout: 400,
      randomize: true,
      onRetry: error => {
        logger.warn(`Error on request to Core : ${error.statusCode || 'no response'} : ${error.message} : retrying`)
      }
    }
  )

  return response.body
}

async function broadcastCoreTxAsync(coreId, submittingNodeIP, tokenHash) {
  let tokenTx = {
    type: 'TOKEN',
    data: `${submittingNodeIP}|${tokenHash}`,
    version: 2,
    time: Math.ceil(Date.now() / 1000),
    core_id: coreId
  }
  let tokenTxString = JSON.stringify(tokenTx)
  let tokenTxB64 = Buffer.from(tokenTxString).toString('base64')
  try {
    let txResponse = await tmRpc.broadcastTxAsync(tokenTxB64)
    if (txResponse.error) {
      switch (txResponse.error.responseCode) {
        case 409:
          throw new Error(txResponse.error.message)
        default:
          logger.error(`RPC error communicating with Tendermint : ${txResponse.error.message}`)
          throw new Error('Could not broadcast transaction')
      }
    }
  } catch (error) {
    throw new Error(`server error on transaction broadcast, ${error.message}`)
  }
}

module.exports = {
  cacheTokenHashes: cacheTokenHashes,
  getPrevTokenHash: getPrevTokenHash,
  delPrevTokenHash: delPrevTokenHash,
  verifySigAsync: verifySigAsync,
  getCachedCoreIDAsync: getCachedCoreIDAsync,
  broadcastCoreTxAsync: broadcastCoreTxAsync,
  setRedis: r => {
    redis = r
  },
  // additional functions for testing purposes
  setRP: r => {
    rp = r
  },
  setTMRPC: rpc => {
    tmRpc = rpc
  },
  setStatus: s => {
    status = s
  },
  setSC: sc => {
    stakedCore = sc
  }
}
