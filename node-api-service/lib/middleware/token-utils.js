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
let status = require('../endpoints/status.js')
const env = require('../parse-env.js')('api')

// NOTE: This is not restify middleware in the traditional sense
// These functions contain async functions and thus cannot be directly included in Restify routes
// These are token signature verification methods that are used in more than one of the API endpoints
// The purpose of this file is to prevent duplication of these token signature verification methods

const CORE_JWK_KEY_PREFIX = 'CorePublicKey'
const CORE_ID_KEY = 'CoreID'

// The redis connection used for all redis communication
// This value is set once the connection has been established
let redis = null

async function verifySigAsync(tokenString, decodedToken) {
  // verify signature of token
  try {
    // get the token's key id
    let kid = decodedToken.header.kid
    if (!kid) return new errors.InvalidArgumentError('invalid request, token missing `kid` value')
    // get the token's issuer, the Core URI
    let iss = decodedToken.payload.iss
    if (!iss) return new errors.InvalidArgumentError('invalid request, token missing `iss` value')
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

async function getCachedJWKAsync(kid, iss) {
  // first, attempt to read value from Redis
  let redisKey = `${CORE_JWK_KEY_PREFIX}:${kid}`
  if (redis) {
    try {
      let cacheResult = await redis.get(redisKey)
      if (cacheResult) return JSON.parse(cacheResult)
    } catch (error) {
      console.error(`Redis read error : getTokenJWKAsync : ${error.message}`)
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
    console.error(`Core request error : getTokenJWKAsync : ${error.message}`)
  }
  // if a non cached value was found, add to the cache
  if (result && redis) {
    try {
      await redis.set(redisKey, JSON.stringify(result))
    } catch (error) {
      console.error(`Redis write error : getTokenJWKAsync : ${error.message}`)
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
      console.error(`Redis read error : getCachedCoreIDAsync : ${error.message}`)
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
    console.error(`Core request error : getCachedCoreIDAsync : ${error.message}`)
  }
  // if a non cached value was found, add to the cache
  if (result && redis) {
    try {
      await redis.set(CORE_ID_KEY, result)
    } catch (error) {
      console.error(`Redis write error : getCachedCoreIDAsync : ${error.message}`)
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
        console.log(`Error on request to Core : ${error.statusCode || 'no response'} : ${error.message} : retrying`)
      }
    }
  )

  return response.body
}

module.exports = {
  verifySigAsync: verifySigAsync,
  getCachedCoreIDAsync: getCachedCoreIDAsync,
  setRedis: r => {
    redis = r
  },
  // additional functions for testing purposes
  setRP: r => {
    rp = r
  },
  setStatus: s => {
    status = s
  }
}
