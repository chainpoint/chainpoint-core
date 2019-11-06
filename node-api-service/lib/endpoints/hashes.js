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
let env = require('../parse-env.js')('api')
const utils = require('../utils.js')
const BLAKE2s = require('blake2s-js')
const _ = require('lodash')
const logger = require('../logger.js')
const crypto = require('crypto')

// Generate a v1 UUID (time-based)
// see: https://github.com/broofa/node-uuid
const uuidv1 = require('uuid/v1')

// The channel used for all amqp communication
// This value is set once the connection has been established
let amqpChannel = null

// The lightning connection used for all lightning invoices communication
// This value is set once the connection has been established
let invoiceClient = null
let lightning = null

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
 * @returns {Object} An Object with 'cal' and 'btc' properties
 *
 */
function generateProcessingHints(timestampDate) {
  let twoHoursFromTimestamp = utils.addMinutes(timestampDate, 120)
  let oneHourFromTopOfTheHour = new Date(twoHoursFromTimestamp.setHours(twoHoursFromTimestamp.getHours(), 0, 0, 0))
  let calHint = utils.formatDateISO8601NoMs(utils.addSeconds(timestampDate, 10))
  let btcHint = utils.formatDateISO8601NoMs(oneHourFromTopOfTheHour)

  return {
    cal: calHint,
    btc: btcHint
  }
}

/**
 * GET /hash/invoice handler
 *
 * Returns a lightning invoice with embedded unique id
 * After paying the invoice, use the unique id to access hash submission endpoint
 *
 */
async function getPreimageV1Async(req, res, next) {
  try {
    if (!invoiceClient) throw new Error('LND connection not available')
    let preimage = crypto.randomBytes(32).toString('hex')
    let hash = getHash(preimage)
    let { payment_request } = await invoiceClient.addHoldInvoiceAsync({
      value: env.SUBMIT_HASH_PRICE_SAT,
      memo: `SubmitHashHoldInvoice`,
      hash: Buffer.from(hash, 'hex')
    })
    res.send({ preimage, payment_request })
    return next()
  } catch (error) {
    logger.error(`Unable to generate invoice : ${error.message}`)
    return next(new errors.InternalServerError('Unable to generate invoice'))
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

  // if we aren't part of an aggregator whitelist, use lightning invoicing
  let submittingIP = utils.getClientIP(req)
  // If whitelist does not contain the submitting IP, use invoicing
  if (!env.AGGREGATOR_WHITELIST.includes(submittingIP)) {
    // validate params has parse a 'invoice_id' key
    if (!req.params.hasOwnProperty('preimage')) {
      return next(new errors.InvalidArgumentError('invalid JSON body: missing preimage'))
    }

    // validate 'preimage' is a string
    let preimage = req.params.preimage
    if (!_.isString(preimage)) {
      return next(new errors.InvalidArgumentError('invalid JSON body: bad preimage submitted'))
    }

    // validate preimage param is a valid hex string
    let isValidPreimage = /^([a-fA-F0-9]{2}){32}$/.test(preimage)
    if (!isValidPreimage) {
      return next(new errors.InvalidArgumentError('invalid JSON body: bad preimage submitted'))
    }

    // validate preimage has been paid
    if (!invoiceClient) throw new Error('LND invoices connection not available')
    if (!lightning) throw new Error('LND connection not available')
    let paymentHash
    try {
      paymentHash = getHash(preimage)

      const invoiceInfo = await lightning.lookupInvoiceAsync({ r_hash_str: paymentHash })
      console.log('invoiceINfo:', invoiceInfo)
      if (!invoiceInfo) next(new errors.PaymentRequiredError(`Invoice with that corresponding preimage not found`))
      if (invoiceInfo.settled !== false || invoiceInfo.state === 'OPEN')
        next(new errors.PaymentRequiredError(`invoice ${paymentHash} has not been paid`))
    } catch (error) {
      logger.error(`LND query error for ${paymentHash}: ${error.message}`)
      return next(new errors.InternalServerError('Could not retrieve invoice status'))
    }

    // if invoice status has been confirmed then we can settle it to prevent future submissions
    try {
      await invoiceClient.settleInvoiceAsync({ preimage: Buffer.from(preimage, 'hex') })
    } catch (e) {
      logger.error(`Problem settling invoice ${paymentHash}: ${e.message}`)
      return next(
        new errors.InternalServerError(`Could not settle the hold invoice (${paymentHash}) with preimage (${preimage})`)
      )
    }
  }

  // validate amqp channel has been established
  if (!amqpChannel) {
    return next(new errors.InternalServerError('Message could not be delivered'))
  }

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
    logger.error(`${env.RMQ_WORK_OUT_AGG_QUEUE} : publish message nacked`)
    return next(new errors.InternalServerError('Message could not be delivered'))
  }

  res.send(responseObj)
  return next()
}

function getHash(preimage) {
  return crypto
    .createHash('sha256')
    .update(Buffer.from(preimage, 'hex'))
    .digest()
    .toString('hex')
}

module.exports = {
  postHashV1Async: postHashV1Async,
  getPreimageV1Async: getPreimageV1Async,
  generatePostHashResponse: generatePostHashResponse,
  // additional functions for testing purposes
  setAMQPChannel: chan => {
    amqpChannel = chan
  },
  setENV: obj => {
    env = obj
  },
  setLND: l => {
    lightning = l
  },
  setInvoice: i => {
    invoiceClient = i
  }
}
