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
const uuidValidate = require('uuid-validate')
const _ = require('lodash')
let proof = require('../models/Proof.js')

/**
 * GET /proofs handler
 *
 * Expects a 'hashids' parameter in the header in the form of a CSV of Version 1 UUIDs
 *
 * Returns chainpoint proofs for the requested Hash IDs
 */
async function getProofsByIDsAsync(req, res, next) {
  res.contentType = 'application/json'

  let proofIds = []

  // check if hash_id parameter was included
  if (req.headers && req.headers.proofids) {
    // read from headers.hashids
    proofIds = req.headers.proofids.split(',').map(_.trim)
  }

  // ensure at least one hash_id was submitted
  if (proofIds.length === 0) {
    return next(new errors.InvalidArgumentError('invalid request, at least one hash id required'))
  }

  // ensure that the request count does not exceed the maximum setting
  if (proofIds.length > 250) {
    return next(new errors.InvalidArgumentError('invalid request, too many hash ids (250 max)'))
  }

  // ensure all hash_ids are valid
  for (let proofId of proofIds) {
    if (!uuidValidate(proofId, 1)) {
      return next(new errors.InvalidArgumentError(`invalid request, bad hash_id: ${proofId}`))
    }
  }

  // retrieve all the proofs from postgres
  let queryResults = []
  try {
    queryResults = await proof.getProofsByProofIdsAsync(proofIds)
  } catch (error) {
    return next(new errors.InternalServerError('error retrieving proofs'))
  }

  // create proof lookup table keyed by proofId
  let proofsReturned = queryResults.reduce((result, item) => {
    result[item.hash_id] = item.proof
    return result
  }, {})

  // construct result array for each proofId submitted
  let finalResults = proofIds.map(proofId => {
    if (proofsReturned[proofId]) {
      return {
        hash_id: proofId,
        proof: JSON.parse(proofsReturned[proofId])
      }
    } else {
      return {
        hash_id: proofId,
        proof: null
      }
    }
  })

  res.send(finalResults)
  return next()
}

module.exports = {
  getProofsByIDsAsync: getProofsByIDsAsync,
  // additional functions for testing purposes
  setProof: p => {
    proof = p
  }
}
