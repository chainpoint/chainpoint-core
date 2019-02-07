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

const env = require('../parse-env.js')('api')

const cachedAuditChallenge = require('../models/cachedAuditChallenge.js')
const restify = require('restify')
const crypto = require('crypto')

let CalendarBlock

// TweetNaCl.js
// see: http://ed25519.cr.yp.to
// see: https://github.com/dchest/tweetnacl-js#signatures
const nacl = require('tweetnacl')
nacl.util = require('tweetnacl-util')

let NODE_AGGREGATION_INTERVAL_SECONDS

// Pass SIGNING_SECRET_KEY as Base64 encoded bytes
const signingSecretKeyBytes = nacl.util.decodeBase64(env.SIGNING_SECRET_KEY)
const signingKeypair = nacl.sign.keyPair.fromSecretKey(signingSecretKeyBytes)

// Calculate the hash of the signing public key bytes
// to allow lookup of which pubkey was used to sign
// a block. Handles different organizations signing blocks
// with different keys and key rotation by those orgs.
// When a Base64 pubKey is published publicly it should also
// be accompanied by this hash of its bytes to serve
// as a fingerprint.
function calcSigningPubKeyHashHex (pubKey) {
  return crypto.createHash('sha256').update(pubKey).digest('hex')
}
const signingPubKeyHashHex = calcSigningPubKeyHashHex(signingKeypair.publicKey)

function getCorePublicKeyList () {
  let pubKeyList = {}
  pubKeyList['09b0ec65fa25'] = 'Q88brO55SfkY5S0Rbnyh3gh1s6izAj9v4BSWVF1dce0='
  pubKeyList['fcbc2ba6c808'] = 'UWJSQwBjlvlkSirJcdFKP4zGQIq1mfrk7j0xV0CZ9yI='

  let currentPubKeyPrefix = signingPubKeyHashHex.slice(0, 12)
  // If the current keypair in use as configured with env.SIGNING_SECRET_KEY is not among the
  // hard coded public key history list above, add this public key information to the results object.
  if (pubKeyList[currentPubKeyPrefix] === undefined) {
    pubKeyList[currentPubKeyPrefix] = nacl.util.encodeBase64(signingKeypair.publicKey)
  }

  return pubKeyList
}

// the minimum audit passing Node version for existing registered Nodes, set by consul
let minNodeVersionExisting = null

// get the first entry in the ETH_TNT_LISTEN_ADDRS CSV to publicize
let coreEthAddress = env.ETH_TNT_LISTEN_ADDRS.split(',')[0]

/**
 * GET /config handler
 *
 * Returns a configuration information object
 */
async function getConfigInfoV1Async (req, res, next) {
  let result
  try {
    let topCoreBlock = await CalendarBlock.findOne({ attributes: ['id'], order: [['id', 'DESC']] })
    if (!topCoreBlock) throw new Error('no blocks found on calendar')

    let mostRecentChallenge = await cachedAuditChallenge.getMostRecentChallengeDataSolutionRemovedAsync()
    let node_aggregation_interval_seconds = NODE_AGGREGATION_INTERVAL_SECONDS // eslint-disable-line

    result = {
      chainpoint_core_base_uri: env.CHAINPOINT_CORE_BASE_URI,
      public_keys: getCorePublicKeyList(),
      calendar: {
        height: parseInt(topCoreBlock.id),
        audit_challenge: mostRecentChallenge || undefined
      },
      node_aggregation_interval_seconds: node_aggregation_interval_seconds,
      core_eth_address: coreEthAddress,
      node_min_version: minNodeVersionExisting
    }
  } catch (error) {
    console.error(`getConfigInfoV1Async failed : Could not generate config object : ${error.message}`)
    return next(new restify.InternalServerError('Could not generate config object'))
  }

  res.cache('public', { maxAge: 60 })
  res.send(result)
  return next()
}

module.exports = {
  getConfigInfoV1Async: getConfigInfoV1Async,
  setRedis: (r) => { cachedAuditChallenge.setRedis(r) },
  setConsul: async (c) => {
    cachedAuditChallenge.setConsul(c)
  },
  setNodeAggregationInterval: (val) => { NODE_AGGREGATION_INTERVAL_SECONDS = val },
  setMostRecentChallengeKey: (key) => { cachedAuditChallenge.setMostRecentChallengeKey(key) },
  setMinNodeVersionExisting: (v) => { minNodeVersionExisting = v },
  setDatabase: (sqlz, calBlock, auditChal) => { CalendarBlock = calBlock; cachedAuditChallenge.setDatabase(sqlz, auditChal) }
}
