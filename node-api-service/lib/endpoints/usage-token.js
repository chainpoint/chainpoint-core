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
const ethers = require('ethers')
let env = require('../parse-env.js')('api')
let utils = require('../utils.js')
let activeToken = require('../models/ActiveToken.js')
const jwt = require('jsonwebtoken')
const crypto = require('crypto')
const uuidv1 = require('uuid/v1')
const jose = require('node-jose')
const tokenUtils = require('../token-utils.js')
const logger = require('../logger.js')
const url = require('url').URL

const network = env.NETWORK === 'mainnet' ? 'homestead' : 'ropsten'
const infuraProvider = new ethers.providers.InfuraProvider(network, env.ETH_INFURA_API_KEY)
const etherscanProvider = new ethers.providers.EtherscanProvider(network, env.ETH_ETHERSCAN_API_KEY)
let fallbackProvider = new ethers.providers.FallbackProvider([infuraProvider, etherscanProvider])

let tknDefinition = require('../../artifacts/ethcontracts/TierionNetworkToken.json')
const tokenABI = tknDefinition.abi
const tokenContractInterface = new ethers.utils.Interface(tokenABI)
let tokenContractAddress = tknDefinition.networks[network === 'homestead' ? '1' : '3'].address

async function postTokenRefreshAsync(req, res, next) {
  // ensure that aud was supplied
  if (!req.params.aud) {
    return next(new errors.InvalidArgumentError('invalid request, aud must be supplied'))
  }

  let aud = req.params.aud.toString()
  let ips = aud.split(',')
  // ensure that aud is a csv with three values
  if (ips.length !== 3) {
    return next(new errors.InvalidArgumentError('invalid request, aud must contain 3 values'))
  }

  // ensure each IP is unique
  let ipSet = new Set(ips)
  if (ipSet.size !== 3) {
    return next(new errors.InvalidArgumentError('invalid request, aud must contain 3 unique values'))
  }

  // ensure that each ip value is a real ip
  for (let ip of ips) {
    if (!utils.isIP(ip)) {
      return next(new errors.InvalidArgumentError(`invalid request, bad IP value in aud - ${ip}`))
    }
  }

  // ensure that the ip values contain this Core ip
  let coreURL = new url(env.CHAINPOINT_CORE_BASE_URI)
  let coreIP = coreURL.hostname
  if (!ips.includes(coreIP)) {
    return next(new errors.InvalidArgumentError(`invalid request, aud must include this Core IP`))
  }

  const tokenString = req.params.token
  // ensure that token is supplied
  if (!tokenString) {
    return next(new errors.InvalidArgumentError('invalid request, token must be supplied'))
  }

  let decodedToken = null
  // attempt to parse token, if not valid, return error
  try {
    decodedToken = jwt.decode(tokenString, { complete: true })
    if (!decodedToken) throw new Error()
  } catch (error) {
    return next(new errors.InvalidArgumentError('invalid request, token cannot be decoded'))
  }

  // verify signature of token
  let verifyError = await tokenUtils.verifySigAsync(tokenString, decodedToken)
  // verifyError will be a restify error on error, or null on successful verification
  if (verifyError !== null) return next(verifyError)

  // ensure that we can retrieve the Node IP from the request
  let submittingNodeIP = utils.getClientIP(req)
  if (submittingNodeIP === null) {
    logger.error('Unable to determine Node IP of request')
    logger.error('connection = ', req.connection || 'undefined')
    logger.error('socket = ', req.socket || 'undefined')
    return next(new errors.BadRequestError('bad request, unable to determine Node IP'))
  }
  logger.info(`Received request from Node at ${submittingNodeIP}`)

  // get the token's subject
  let sub = decodedToken.payload.sub
  if (!sub) return next(new errors.InvalidArgumentError('invalid request, token missing `sub` value'))

  // ensure the Node IP is the subject of the JWT
  if (sub !== submittingNodeIP)
    return next(new errors.InvalidArgumentError('invalid request, token subject does not match Node IP'))

  // cannot refresh a token with a balance of 0
  if (decodedToken.payload.bal < 1)
    return next(new errors.InvalidArgumentError('invalid request, token with 0 balance cannot be refreshed'))

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

  logger.info(`NodeIP ${submittingNodeIP} : Active Token ${activeTokenHash} : Provided Token ${tokenHash}`)

  if (activeTokenHash !== tokenHash)
    return next(new errors.InvalidArgumentError('invalid request, supplied token is not an active token'))

  // At this point, we've established that the submitted token is a valid JWT with valid signature,
  // it has a balance > 0, and it is the active token. Proceed with refresh.

  // construct the token payload
  // set the expiration time to an hour in the future
  // if active token has time remaining, add one hour to existing expiration
  let nowSeconds = Math.ceil(Date.now() / 1000)
  let base = decodedToken.payload.exp > nowSeconds ? decodedToken.payload.exp : nowSeconds
  let exp = base + 60 * 60 // 1 hour in the future from base time
  // set new balance value
  let bal = decodedToken.payload.bal - 1
  // set the audit update limit remaining count
  let aulr = 3
  let payload = constructTokenPayload(submittingNodeIP, exp, bal, aud, aulr)

  // Create token
  let refreshedTokenString = null
  try {
    refreshedTokenString = await createAndSignJWTAsync(payload)
  } catch (error) {
    return next(new errors.InternalServerError('server error, could not sign refreshed token'))
  }

  // calculate hash of new token
  let refreshTokenHash = crypto
    .createHash('sha256')
    .update(refreshedTokenString)
    .digest('hex')

  logger.info(
    `NodeIP ${submittingNodeIP} : Active Token ${activeTokenHash} : Provided Token ${tokenHash} : Refresh Token ${refreshTokenHash}`
  )

  let cacheSuccess = await tokenUtils.cacheTokenHashesAsync(activeTokenHash, refreshTokenHash)
  if (!cacheSuccess) {
    return next(new errors.InternalServerError('server error, could cache token hashes for confirmation'))
  }

  res.contentType = 'application/json'
  res.send({ token: refreshedTokenString })
  return next()
}

async function createAndSignJWTAsync(payload) {
  let privateKeyPEM = env.ECDSA_PKPEM
  let jwk = await jose.JWK.asKey(privateKeyPEM, 'pem')
  return jwt.sign(payload, privateKeyPEM, { algorithm: 'ES256', keyid: jwk.toJSON().kid })
}

function constructTokenPayload(submittingNodeIP, exp, bal, aud, aulr) {
  // create a new JWT id
  let jti = uuidv1()
  // set the issuer (this Core's identifier)
  let iss = env.CHAINPOINT_CORE_BASE_URI
  // set the subject (the Node IP)
  let sub = submittingNodeIP
  // construct and return a JWT payload
  return { jti, iss, sub, exp, bal, aud, aulr }
}

async function postTokenCreditAsync(req, res, next) {
  // ensure that aud was supplied
  if (!req.params.aud) {
    return next(new errors.InvalidArgumentError('invalid request, aud must be supplied'))
  }

  let aud = req.params.aud.toString()
  let ips = aud.split(',')
  // ensure that aud is a csv with three values
  if (ips.length !== 3) {
    return next(new errors.InvalidArgumentError('invalid request, aud must contain 3 values'))
  }

  // ensure each IP is unique
  let ipSet = new Set(ips)
  if (ipSet.size !== 3) {
    return next(new errors.InvalidArgumentError('invalid request, aud must contain 3 unique values'))
  }

  // ensure that each ip value is a real ip
  for (let ip of ips) {
    if (!utils.isIP(ip)) {
      return next(new errors.InvalidArgumentError(`invalid request, bad IP value in aud - ${ip}`))
    }
  }

  // ensure that the ip values contain this Core ip
  let coreURL = new url(env.CHAINPOINT_CORE_BASE_URI)
  let coreIP = coreURL.hostname
  if (!ips.includes(coreIP)) {
    return next(new errors.InvalidArgumentError(`invalid request, aud must include this Core IP`))
  }

  // ensure that tx was supplied
  if (!req.params.tx) {
    return next(new errors.InvalidArgumentError('invalid request, tx must be supplied'))
  }

  const rawTx = req.params.tx.toString()
  // ensure that rawTx represents a valid hex value starting wiht 0x
  if (!rawTx.startsWith('0x')) {
    return next(new errors.InvalidArgumentError('invalid request, tx must begin with 0x'))
  }
  // ensure that rawTx represents a valid hex value
  let txContent = rawTx.slice(2)
  if (!utils.isHex(txContent)) {
    return next(new errors.InvalidArgumentError('invalid request, non hex tx value supplied'))
  }

  // ensure that rawTx represents a valid ethereum transaction
  let decodedTx = null
  try {
    decodedTx = ethers.utils.parseTransaction(rawTx)
  } catch (error) {
    return next(new errors.InvalidArgumentError('invalid request, invalid ethereum tx body supplied'))
  }

  // ensure that the raw Eth Tx provided is interacting with the Chainpoint Token Contract
  if (decodedTx.to !== tokenContractAddress) {
    return next(
      new errors.InvalidArgumentError('invalid request, transaction must interact with Chainpoint token contract')
    )
  }

  // ensure that this is a 'purchaseUsage' method call
  let parsedTx = tokenContractInterface.parseTransaction(decodedTx)
  if (parsedTx.name !== 'purchaseUsage') {
    return next(new errors.InvalidArgumentError(`invalid request, transaction may only make a call to 'purchaseUsage'`))
  }

  // ensure the purchaseUsage amount is valid
  let txDataArgs = parsedTx.args
  let spendAmount = txDataArgs[0].toNumber()
  if (spendAmount < 10 ** 8) {
    return next(new errors.InvalidArgumentError(`invalid request, must purchase with at least ${10 ** 8} $TKN`))
  }

  // ensure that we can retrieve the Node IP from the request
  let submittingNodeIP = utils.getClientIP(req)
  if (submittingNodeIP === null) {
    logger.error('Unable to determine Node IP of request')
    logger.error('connection = ', req.connection || 'undefined')
    logger.error('socket = ', req.socket || 'undefined')
    return next(new errors.BadRequestError('bad request, unable to determine Node IP'))
  }
  logger.info(`Received request from Node at ${submittingNodeIP}`)

  // broadcast the ETH transaction and await inclusion in a block
  try {
    let sendResponse = await fallbackProvider.sendTransaction(rawTx)
    await fallbackProvider.waitForTransaction(sendResponse.hash)
  } catch (error) {
    logger.error(`Error when attempting to broadcast ETH Tx : ${error.message}`)
    return next(new errors.InternalServerError(error.message))
  }

  let creditPrice = 0.1 // TODO: Build and request from exchange rate service

  // determine the number of credits to issue in new token
  let bal = Math.floor(spendAmount / 10 ** 8) / creditPrice - 1

  // construct the token payload
  // set the expiration time to an hour in the future
  let oneHourFromNow = Math.ceil(Date.now() / 1000) + 60 * 60 // 1 hour in the future from now
  let exp = oneHourFromNow
  // set the audit update limit remaining count
  let aulr = 3
  let payload = constructTokenPayload(submittingNodeIP, exp, bal, aud, aulr)

  // Create token
  let newTokenString = null
  try {
    newTokenString = await createAndSignJWTAsync(payload)
  } catch (error) {
    return next(new errors.InternalServerError('server error, could not sign new token'))
  }

  // calculate hash of new token
  let newTokenHash = crypto
    .createHash('sha256')
    .update(newTokenString)
    .digest('hex')

  // save new active token information in local database
  // this is to allow multiple consecutive JWT method calls
  // from the same Core without waiting for the broadcast delay
  // if this fails, we can proceed because the subsequent broadcast
  // call will update the local database eventually as well,
  // and Nodes will eventually recover
  try {
    await activeToken.writeActiveTokenAsync({
      nodeIp: submittingNodeIP,
      tokenHash: newTokenHash
    })
  } catch (error) {
    logger.warn(`Could not update active token data in local database on credit`)
  }

  // broadcast Node IP and new token hash for Cores to update their local active token table
  try {
    let coreId = await tokenUtils.getCachedCoreIDAsync()
    await tokenUtils.broadcastCoreTxAsync(coreId, submittingNodeIP, newTokenHash)
  } catch (error) {
    return next(new errors.InternalServerError(`server error, ${error.message}`))
  }

  res.contentType = 'application/json'
  res.send({ token: newTokenString })
  return next()
}

async function postTokenAudienceUpdateAsync(req, res, next) {
  // ensure that aud was supplied
  if (!req.params.aud) {
    return next(new errors.InvalidArgumentError('invalid request, aud must be supplied'))
  }

  let aud = req.params.aud.toString()
  let ips = aud.split(',')
  // ensure that aud is a csv with three values
  if (ips.length !== 3) {
    return next(new errors.InvalidArgumentError('invalid request, aud must contain 3 values'))
  }

  // ensure each IP is unique
  let ipSet = new Set(ips)
  if (ipSet.size !== 3) {
    return next(new errors.InvalidArgumentError('invalid request, aud must contain 3 unique values'))
  }

  // ensure that each ip value is a real ip
  for (let ip of ips) {
    if (!utils.isIP(ip)) {
      return next(new errors.InvalidArgumentError(`invalid request, bad IP value in aud - ${ip}`))
    }
  }

  // ensure that the ip values contain this Core ip
  let coreURL = new url(env.CHAINPOINT_CORE_BASE_URI)
  let coreIP = coreURL.hostname
  if (!ips.includes(coreIP)) {
    return next(new errors.InvalidArgumentError(`invalid request, aud must include this Core IP`))
  }

  const tokenString = req.params.token
  // ensure that token is supplied
  if (!tokenString) {
    return next(new errors.InvalidArgumentError('invalid request, token must be supplied'))
  }

  let decodedToken = null
  // attempt to parse token, if not valid, return error
  try {
    decodedToken = jwt.decode(tokenString, { complete: true })
    if (!decodedToken) throw new Error()
  } catch (error) {
    return next(new errors.InvalidArgumentError('invalid request, token cannot be decoded'))
  }

  // verify signature of token
  let verifyError = await tokenUtils.verifySigAsync(tokenString, decodedToken)
  // verifyError will be a restify error on error, or null on successful verification
  if (verifyError !== null) return next(verifyError)

  // ensure that we can retrieve the Node IP from the request
  let submittingNodeIP = utils.getClientIP(req)
  if (submittingNodeIP === null) {
    logger.error('Unable to determine Node IP of request')
    logger.error('connection = ', req.connection || 'undefined')
    logger.error('socket = ', req.socket || 'undefined')
    return next(new errors.BadRequestError('bad request, unable to determine Node IP'))
  }
  logger.info(`Received request from Node at ${submittingNodeIP}`)

  // get the token's subject
  let sub = decodedToken.payload.sub
  if (!sub) return next(new errors.InvalidArgumentError('invalid request, token missing `sub` value'))

  // ensure the Node IP is the subject of the JWT
  if (sub !== submittingNodeIP)
    return next(new errors.InvalidArgumentError('invalid request, token subject does not match Node IP'))

  // cannot accept expired token
  let tokenExp = decodedToken.payload.exp
  if (!tokenExp) return next(new errors.InvalidArgumentError('invalid request, token missing `exp` value'))
  if (isNaN(tokenExp)) return next(new errors.InvalidArgumentError('invalid request, `exp` value must be a number'))
  let tokenExpMS = tokenExp * 1000
  if (tokenExpMS < Date.now()) return next(new errors.UnauthorizedError('not authorized, token has expired'))

  // get the token's audience update limit remaining
  let tokenAulr = decodedToken.payload.aulr
  if (!tokenAulr && tokenAulr !== 0)
    return next(new errors.InvalidArgumentError('invalid request, token missing `aulr` value'))
  if (isNaN(tokenAulr)) return next(new errors.InvalidArgumentError('invalid request, `aulr` value must be a number'))

  // ensure audience update limit remaining is greater than 0
  if (tokenAulr < 1)
    return next(new errors.TooManyRequestsError('request rejected, aud update rate limit exceeded for this token'))

  // ensure that the submitted token is the active token for the Node
  let activeTokenHash = null
  try {
    let resultRow = await activeToken.getActiveTokenByNodeIPAsync(submittingNodeIP)
    if (resultRow === null)
      return next(new errors.InvalidArgumentError('invalid request, no active token available to be updated'))
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
  // it has a update limit remaining > 0, and it is the active token. Proceed with aud update.

  // construct the token payload
  // get the current balance
  let bal = decodedToken.payload.bal
  // reduce the audit update limit remaining count by 1
  let aulr = tokenAulr - 1
  let payload = constructTokenPayload(submittingNodeIP, tokenExp, bal, aud, aulr)

  // Create token
  let updatedTokenString = null
  try {
    updatedTokenString = await createAndSignJWTAsync(payload)
  } catch (error) {
    return next(new errors.InternalServerError('server error, could not sign updated token'))
  }

  // calculate hash of new token
  let updatedTokenHash = crypto
    .createHash('sha256')
    .update(updatedTokenString)
    .digest('hex')

  // save new active token information in local database
  // this is to allow multiple consecutive JWT method calls
  // from the same Core without waiting for the broadcast delay
  // if this fails, we can proceed because the subsequent broadcast
  // call will update the local database eventually as well,
  // and Nodes will eventually recover
  try {
    await activeToken.writeActiveTokenAsync({
      nodeIp: submittingNodeIP,
      tokenHash: updatedTokenHash
    })
  } catch (error) {
    logger.warn(`Could not update active token data in local database on audience update`)
  }

  // broadcast Node IP and new token hash for Cores to update their local active token table
  try {
    let coreId = await tokenUtils.getCachedCoreIDAsync()
    await tokenUtils.broadcastCoreTxAsync(coreId, submittingNodeIP, updatedTokenHash)
  } catch (error) {
    return next(new errors.InternalServerError(`server error, ${error.message}`))
  }

  res.contentType = 'application/json'
  res.send({ token: updatedTokenString })
  return next()
}

module.exports = {
  postTokenRefreshAsync: postTokenRefreshAsync,
  postTokenCreditAsync: postTokenCreditAsync,
  postTokenAudienceUpdateAsync: postTokenAudienceUpdateAsync,
  // additional functions for testing purposes
  setFP: fp => {
    fallbackProvider = fp
  },
  setAT: at => {
    activeToken = at
  },
  setENV: e => {
    env = e
  },
  setTA: ta => {
    tokenContractAddress = ta
  },
  setRedis: r => {
    tokenUtils.setRedis(r)
  },
  setStatus: s => {
    tokenUtils.setStatus(s)
  },
  setRP: rp => {
    tokenUtils.setRP(rp)
  },
  setGetIP: func => {
    utils.getClientIP = func
  },
  setSC: sc => {
    tokenUtils.setSC(sc)
  }
}
