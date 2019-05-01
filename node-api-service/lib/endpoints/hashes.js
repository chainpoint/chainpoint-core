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
const env = require('../parse-env.js')('api')
const utils = require('../utils.js')
const BLAKE2s = require('blake2s-js')
const _ = require('lodash')
const jwt = require('jsonwebtoken')
const tokenUtils = require('../middleware/token-utils.js')

// Generate a v1 UUID (time-based)
// see: https://github.com/broofa/node-uuid
const uuidv1 = require('uuid/v1')

// The channel used for all amqp communication
// This value is set once the connection has been established
let amqpChannel = null

/**
 * Converts an array of hash strings to a object suitable to
 * return to HTTP clients.
 *
 * @param {string} hash - A hash string to process
 * @returns {Object} An Object with 'hash_id', 'hash', 'submitted_at' and 'processing_hints' properties
 *
 */
function generatePostHashResponse(hash) {
  hash = hash.toLowerCase()

  // Compute a five byte BLAKE2s hash of the
  // timestamp that will be embedded in the UUID.
  // This allows the UUID to verifiably reflect the
  // combined NTP time, and the hash submitted. Thus these values
  // are represented both in the BLAKE2s hash and in
  // the full timestamp embedded in the v1 UUID.
  //
  // RFC 4122 allows the MAC address in a version 1
  // (or 2) UUID to be replaced by a random 48-bit Node ID,
  // either because the node does not have a MAC address, or
  // because it is not desirable to expose it. In that case, the
  // RFC requires that the least significant bit of the first
  // octet of the Node ID should be set to `1`. This code
  // uses a five byte BLAKE2s hash as a verifier in place
  // of the MAC address. This also prevents leakage of server
  // info.
  //
  // This value can be checked on receipt of the hash_id UUID
  // by extracting the bytes of the last segment of the UUID.
  // e.g. If the UUID is 'b609358d-7979-11e7-ae31-01ba7816bf8f'
  // the Node ID hash is the six bytes shown in '01ba7816bf8f'.
  // Any client that can access the timestamp in the UUID,
  // and the original hash can recompute
  // the verification hash and compare it.
  //
  // The UUID can also be verified for correct time by a
  // client that itself has an accurate NTP clock at the
  // moment when returned to the client. This allows
  // a client to verify, likely within a practical limit
  // of approximately 500ms depending on network latency,
  // the accuracy of the returned UUIDv1 timestamp.
  //
  // See JS API for injecting time and Node ID in the UUID API:
  // https://github.com/kelektiv/node-uuid/blob/master/README.md
  //
  let timestampDate = new Date()
  let timestampMS = timestampDate.getTime()
  // 5 byte length BLAKE2s hash w/ personalization
  let h = new BLAKE2s(5, { personalization: Buffer.from('CHAINPNT') })
  let hashStr = [timestampMS.toString(), timestampMS.toString().length, hash, hash.length].join(':')

  h.update(Buffer.from(hashStr))

  let hashId = uuidv1({
    msecs: timestampMS,
    node: Buffer.concat([Buffer.from([0x01]), h.digest()])
  })

  let result = {}
  result.hash_id = hashId
  result.hash = hash
  result.submitted_at = utils.formatDateISO8601NoMs(timestampDate)
  result.processing_hints = generateProcessingHints(timestampDate)

  return result
}

/**
 * Generate the expected proof ready times for each proof stage
 *
 * @param {Date} timestampDate - The hash submission timestamp
 * @returns {Object} An Object with 'cal', 'eth', and 'btc' properties
 *
 */
function generateProcessingHints(timestampDate) {
  let twoHoursFromTimestamp = utils.addMinutes(timestampDate, 120)
  let oneHourFromTopOfTheHour = new Date(twoHoursFromTimestamp.setHours(twoHoursFromTimestamp.getHours(), 0, 0, 0))
  let calHint = utils.formatDateISO8601NoMs(utils.addSeconds(timestampDate, 10))
  let ethHint = utils.formatDateISO8601NoMs(utils.addMinutes(timestampDate, 41))
  let btcHint = utils.formatDateISO8601NoMs(oneHourFromTopOfTheHour)

  return {
    cal: calHint,
    eth: ethHint,
    btc: btcHint
  }
}

/**
 * POST /hash handler
 *
 * Expects a JSON body with the form:
 *   {"hash": "11cd8a380e8d5fd3ac47c1f880390341d40b11485e8ae946d8fa3d466f23fe89"}
 *
 * The `hash` key must reference valid hex string representing the hash to anchor.
 *
 * Each hash must be:
 * - in Hexadecimal form [a-fA-F0-9]
 * - minimum 40 chars long (e.g. 20 byte SHA1)
 * - maximum 128 chars long (e.g. 64 byte SHA512)
 * - an even length string
 */
async function postHashV1Async(req, res, next) {
  // validate content-type sent was 'application/json'
  if (req.contentType() !== 'application/json') {
    return next(new errors.InvalidArgumentError('invalid content type'))
  }

  // validate params has parse a 'hash' key
  if (!req.params.hasOwnProperty('hash')) {
    return next(new errors.InvalidArgumentError('invalid JSON body: missing hash'))
  }

  // validate 'hash' is a string
  if (!_.isString(req.params.hash)) {
    return next(new errors.InvalidArgumentError('invalid JSON body: bad hash submitted'))
  }

  // validate hash param is a valid hex string
  let isValidHash = /^([a-fA-F0-9]{2}){20,64}$/.test(req.params.hash)
  if (!isValidHash) {
    return next(new errors.InvalidArgumentError('invalid JSON body: bad hash submitted'))
  }

  // validate amqp channel has been established
  if (!amqpChannel) {
    return next(new errors.InternalServerError('Message could not be delivered'))
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

  // cannot accept expired token
  let exp = decodedToken.payload.exp
  if (!exp) return next(new errors.InvalidArgumentError('invalid request, token missing `exp` value'))
  if (isNaN(exp)) return next(new errors.InvalidArgumentError('invalid request, `exp` value must be a number'))
  let expMS = exp * 1000
  if (expMS < Date.now()) return next(new errors.UnauthorizedError('not authorized, token has expired'))

  // ensure that we can retrieve the Node IP from the request
  let submittingNodeIP = req.clientIp
  if (submittingNodeIP === null) return next(new errors.BadRequestError('bad request, unable to determine Node IP'))

  let responseObj = generatePostHashResponse(req.params.hash)

  let hashObj = {
    hash_id: responseObj.hash_id,
    hash: responseObj.hash
  }

  try {
    await amqpChannel.sendToQueue(env.RMQ_WORK_OUT_AGG_QUEUE, Buffer.from(JSON.stringify(hashObj)), {
      persistent: true
    })
  } catch (error) {
    console.error(env.RMQ_WORK_OUT_AGG_QUEUE, 'publish message nacked')
    return next(new errors.InternalServerError('Message could not be delivered'))
  }

  res.send(responseObj)
  return next()
}

module.exports = {
  postHashV1Async: postHashV1Async,
  generatePostHashResponse: generatePostHashResponse,
  // additional functions for testing purposes
  setAMQPChannel: chan => {
    amqpChannel = chan
  },
  setRedis: r => {
    tokenUtils.setRedis(r)
  },
  setRP: rp => {
    tokenUtils.setRP(rp)
  }
}
