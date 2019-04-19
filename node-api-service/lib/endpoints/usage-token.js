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
// const ethers = require('ethers')
const env = require('../parse-env.js')('api')
const utils = require('../utils.js')
const activeToken = require('../models/ActiveToken.js')
const jwt = require('jsonwebtoken')
const crypto = require('crypto')
const uuidv1 = require('uuid/v1')
const jose = require('node-jose')
const rp = require('request-promise-native')
const retry = require('async-retry')

const CORE_JWK_KEY_PREFIX = 'CoreJWK'

/*
const network = env.NODE_ENV === 'production' ? 'homestead' : 'ropsten'
const infuraProvider = new ethers.providers.InfuraProvider(network, env.ETH_INFURA_API_KEY)
const etherscanProvider = new ethers.providers.EtherscanProvider(network, env.ETH_ETHERSCAN_API_KEY)
const fallbackProvider = new ethers.providers.FallbackProvider([infuraProvider, etherscanProvider])
*/

// The redis connection used for all redis communication
// This value is set once the connection has been established
let redis = null

async function postTokenRefreshAsync(req, res, next) {
  const tokenString = req.params.token
  // ensure that token is supplied
  if (!tokenString) {
    return next(new errors.InvalidArgumentError('invalid request, token must be supplied'))
  }

  let decodedToken = null
  // attempt to parse token, if not valid, return error
  try {
    decodedToken = jwt.decode(tokenString, { complete: true })
  } catch (error) {
    return next(new errors.InvalidArgumentError('invalid request, token cannot be decoded'))
  }

  // verify signature of token
  try {
    // get the token's key id
    let kid = decodedToken.header.kid
    if (!kid) return next(new errors.InvalidArgumentError('invalid request, token missing `kid` value'))
    // get the token's issuer, the Core URI
    let iss = decodedToken.payload.iss
    if (!iss) return next(new errors.InvalidArgumentError('invalid request, token missing `iss` value'))
    // get the JWK for the given token
    let jwk = await getCachedJWKAsync(kid, iss)
    if (!jwk) return next(new errors.InvalidArgumentError('invalid request, unable to find public key for given kid'))
    jwt.verify(tokenString, jwk.toPEM(), { complete: true })
  } catch (error) {
    return next(new errors.InvalidArgumentError('invalid request, token signature cannot be verified'))
  }

  // cannot refresh a token with a balance of 0
  if (decodedToken.payload.bal < 1)
    return next(new errors.InvalidArgumentError('invalid request, token with 0 balance cannot be refreshed'))

  // ensure that we can retrieve the Node IP from the request
  let submittingNodeIP = utils.getRequestSourceIP(req)
  if (submittingNodeIP === null) return next(new errors.BadRequestError('bad request, unable to determine Node IP'))

  // ensure that the submitted token is the active token for the Node
  let activeTokenHash = null
  try {
    let resultRow = await activeToken.getActiveTokenByNodeIPAsync(submittingNodeIP)
    if (resultRow === null)
      return next(new errors.InvalidArgumentError('invalid request, no active token available to be refreshed'))
    activeTokenHash = resultRow.tokenHash
  } catch (error) {
    return next(new errors.InternalServerError('server error, unable to read active token data'))
  }
  let tokenHash = crypto
    .createHash('sha256')
    .update(tokenString)
    .digest('hex')
  if (activeTokenHash !== tokenHash)
    return next(new errors.InvalidArgumentError('invalid request, supplied token is not an active token'))

  // At this point, we've established that the submitted token is a valid JWT with valid signature,
  // it has a balance > 0, and it is the active token. Proceed with refresh.

  // create a new JWT id
  let jti = uuidv1()
  // set the issuer (this Core's identifier)
  let iss = env.CHAINPOINT_CORE_BASE_URI
  // set the subject (the Node IP)
  let sub = submittingNodeIP
  // set the expiration time to an hour in the future
  // if active token has time remaining, add one hour to existing expiration
  let nowSeconds = Math.ceil(Date.now() / 1000)
  let base = decodedToken.payload.exp > nowSeconds ? decodedToken.payload.exp : nowSeconds
  let exp = base + 60 * 60 // 1 hour in the future from base time
  // set new balance value
  let bal = decodedToken.payload.bal - 1

  // construct JWT payload
  const payload = { jti, iss, sub, exp, bal }

  // Create token
  let refreshedTokenString = null
  try {
    let privateKeyPEM = env.ECDSA_KEYPAIR
    let jwk = await jose.JWK.asKey(privateKeyPEM, 'pem')
    refreshedTokenString = jwt.sign(payload, privateKeyPEM, { algorithm: 'ES256', keyid: jwk.toJSON().kid })
  } catch (error) {
    return next(new errors.InternalServerError('server error, could not sign refreshed token'))
  }

  // TODO: broadcast Node IP and new token hash for Cores to update their local active token table

  // calculate hash of new token
  // let refreshTokenHash = crypto.createHash('sha256').update(refreshedTokenString).digest('hex')

  res.contentType = 'application/json'
  res.send({ token: refreshedTokenString })
  return next()
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

async function coreStatusRequestAsync(coreURI, retryCount = 3) {
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

async function postTokenCreditAsync(req, res, next) {
  const rawTx = req.params.tx

  let result = rawTx

  // TODO: Implmement me

  res.contentType = 'application/json'
  res.send(result)
  return next()
}

module.exports = {
  postTokenRefreshAsync: postTokenRefreshAsync,
  postTokenCreditAsync: postTokenCreditAsync,
  setRedis: r => {
    redis = r
  }
}