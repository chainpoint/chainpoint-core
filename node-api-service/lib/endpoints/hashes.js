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
const { Lsat, Identifier } = require('lsat-js')
const { MacaroonsBuilder } = require('macaroons.js')
const { boltwall } = require('boltwall')

// Generate a v1 UUID (time-based)
// see: https://github.com/broofa/node-uuid
const uuidv1 = require('uuid/v1')

// The channel used for all amqp communication
// This value is set once the connection has been established
let amqpChannel = null

// The lightning connection used for all lightning communication
// This value is set once the connection has been established
let lightning = null
let invoiceClient = null

/**
 * Converts an array of hash strings to a object suitable to
 * return to HTTP clients.
 *
 * @param {string} hash - A hash string to process
 * @returns {Object} An Object with 'proof_id', 'hash', 'submitted_at' and 'processing_hints' properties
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
  // This value can be checked on receipt of the proof_id UUID
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

  let proofId = uuidv1({
    msecs: timestampMS,
    node: Buffer.concat([Buffer.from([0x01]), h.digest()])
  })

  let result = {}
  result.proof_id = proofId
  result.hash = hash
  result.hash_received = utils.formatDateISO8601NoMs(timestampDate)
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
 * POST /hash handler
 *
 * A validator to validate POST hash requests
 *   {"hash": "11cd8a380e8d5fd3ac47c1f880390341d40b11485e8ae946d8fa3d466f23fe89"}
 *
 * The `hash` key must reference valid hex string representing the hash to anchor.
 *
 * Each hash must be:
 * - in Hexadecimal form [a-fA-F0-9]
 * - minimum 40 chars long (e.g. 20 byte SHA1)
 * - maximum 128 chars long (e.g. 64 byte SHA512)
 * - an even length string
 *
 * This is split into its own middleware so we can reject requests with invalid hashes
 * before even making a request to boltwall which requires extra async requests
 */
function validatePostHashRequest(req, res, next) {
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

  return next()
}

/**
 * POST /hash handler
 *
 * Expects a JSON body with the form:
 *   {"hash": "11cd8a380e8d5fd3ac47c1f880390341d40b11485e8ae946d8fa3d466f23fe89"}
 *
 * The `hash` key must reference valid hex string representing the hash to anchor.
 * Will send a response object containing the hash submission information.
 * We expect only validated LSATs to be allowed to reach this middleware and so
 * should be protected by boltwall when implemented in the server.
 */
async function postHashV1Async(req, res, next) {
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

const boltwallConfigs = {
  minAmount: env.SUBMIT_HASH_PRICE_SAT || 10,
  getInvoiceDescription: req => `HODL invoice payment to submit a hash to chainpoint core from ${req.header('HOST')}`,
  hodl: true
}

/**
 * @description A middleware to parse POST /hash requests
 * If the request has an LSAT, then we check the status of the invoice. If the invoice does not exist,
 * return a 404. If it exists and is not paid, return a 402. If it exists and is paid but not settled,
 * we add the preimage to the LSAT token and forward on to the next route (which should be boltwall).
 * If it is paid and settled, then boltwall will ultimately reject it.
 *
 * If the request doesn't have an LSAT, then we must request a new hodl invoice and generate a new
 * LSAT. We are doing this here instead of boltwall so that we can embed the secret into the
 * LSAT identifier for later retrieval
 * @param {*} req
 * @param {*} res
 * @param {*} next
 */
async function parsePostHashRequest(req, res, next) {
  let token = req.header('Authorization')
  if (!token) {
    logger.verbose('Unauthorized request made to submit hash. Generating LSAT...')
    try {
      token = await generateHodlLsat(req)
      res.set('www-authenticate', token.toChallenge())
      res.status(402)
      return res.send({ error: { message: 'Payment Required' } })
    } catch (e) {
      logger.error('Could not generate HODL lsat:', e)
      return next(new errors.InternalServerError('There was a problem generating your request.'))
    }
  } else {
    try {
      let lsat = Lsat.fromToken(token)
      const invoice = await lightning.lookupInvoiceAsync({
        r_hash_str: lsat.paymentHash
      })

      // no invoice found, then return a 404
      if (!invoice) return next(new errors.NotFoundError({ message: 'Unable to locate invoice for that LSAT' }))

      // determine if the invoice is held. Unpaid invoices, should return a 402
      // since they still require payment before they can be allowed through
      const isHeld = invoice.state === 'ACCEPTED' && !invoice.settled
      if (invoice.state === 'SETTLED') {
        return next(
          new errors.UnauthorizedError(
            'Unauthorized: Invoice has already been settled. Try again with a different LSAT'
          )
        )
      } else if (!isHeld) {
        lsat.addInvoice(invoice.payment_request)
        res.set('www-authenticate', lsat.toChallenge())
        res.status(402)
        return res.send({ error: { message: 'Payment Required' } })
      }

      // if the invoice exists and has been paid, then the LSAT just needs the
      // preimage extracted from it and added to the token. If the invoice
      // was already settled and the preimage was just missing from the LSAT, boltwall
      // will reject the request.
      lsat = setLsatPreimageFromId(lsat)
      req.headers.authorization = lsat.toToken()
      return next()
    } catch (e) {
      let error = ''
      if (typeof e === 'string') error = e
      else if (e.message) error = e.message
      const message = `Invalid LSAT provided in Authorization header: ${error}`
      logger.error(message)
      return next(new errors.InvalidHeaderError(message))
    }
  }
}

// helper function to extract the secret from an lsat's identifier
// and use it to satisfy the LSAT
function setLsatPreimageFromId(lsat) {
  if (!(lsat instanceof Lsat)) throw new Error('Must pass an LSAT object to add reimage')

  const identifier = Identifier.fromString(lsat.id)
  // secret is saved as the tokenId in the identifier
  const secret = identifier.tokenId.toString('hex')
  lsat.setPreimage(secret)
  return lsat
}

/**
 * @description Generate a new HODL-based LSAT.
 * First a preimage and hash pair are generated. The hash is used to request a new
 * hodl invoice. Then an LSAT is created where the Identifier is generated using the secret
 * allowing the LSAT to be settled on demand. Normally this is done in boltwall, but since we
 * want to store the secret in the macaroon identifier, we need to build it ourselves.
 * @param {Object} req - request object from server
 */
async function generateHodlLsat(req) {
  if (!invoiceClient) throw new Error('LND invoices connection not available')

  // getting values required for generating a new invoice
  const preimage = crypto.randomBytes(32).toString('hex')
  const hash = getHash(preimage)

  // set value based on body or config's minAmount
  let value
  if (req.body && req.body.amount && req.body.amount > boltwallConfigs.minAmount) value = req.body.amount
  else value = boltwallConfigs.minAmount

  // Normally used in boltwall, but we can just use the function from the config
  const memo = boltwallConfigs.getInvoiceDescription(req)

  // use lnd clieng to create a new invoice
  const { payment_request: payreq } = await invoiceClient.addHoldInvoiceAsync({
    hash: Buffer.from(hash, 'hex'),
    memo,
    value
  })

  // this is where we save the preimage to later settle the hodl invoice with
  const id = new Identifier({
    paymentHash: Buffer.from(hash, 'hex'),
    tokenId: Buffer.from(preimage, 'hex')
  })

  const macaroon = new MacaroonsBuilder(
    req.header('x-forwarded-for') || req.header('HOST'),
    env.SESSION_SECRET,
    id.toString()
  )
    .getMacaroon()
    .serialize()
  const lsat = Lsat.fromMacaroon(macaroon)
  lsat.invoice = payreq
  return lsat
}

function getHash(preimage) {
  if (typeof preimage !== 'string') throw new Error('Must give a string to convert to a hash')
  return crypto
    .createHash('sha256')
    .update(Buffer.from(preimage, 'hex'))
    .digest('hex')
}

module.exports = {
  postHashV1Async: postHashV1Async,
  validatePostHashRequest: validatePostHashRequest,
  boltwallConfigs: boltwallConfigs,
  boltwall: boltwall(boltwallConfigs, logger),
  generatePostHashResponse: generatePostHashResponse,
  parsePostHashRequest: parsePostHashRequest,
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
  setInvoiceClient: i => {
    invoiceClient = i
  }
}
