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

let tmRpc = require('../tendermint-rpc.js')
const errors = require('restify-errors')

async function getTransactionAsync(txID) {
  let txResponse = await tmRpc.getTransactionAsync(txID)
  if (txResponse.error) {
    switch (txResponse.error.responseCode) {
      case 404:
        return { tx: null, error: new errors.NotFoundError(`Could not find transaction with id = '${txID}'`) }
      case 409:
        return { tx: null, error: new errors.InvalidArgumentError(txResponse.error.message) }
      default:
        console.error(`RPC error communicating with Tendermint : ${txResponse.error.message}`)
        return { tx: null, error: new errors.InternalServerError('Could not query for tx by hash') }
    }
  }
  return { tx: txResponse.result, error: null }
}

/**
 * GET /calendar/:txid handler
 *
 * Expects a path parameter 'txid' as a string
 *
 * Returns a calendar tx by tx hash
 */
async function getCalTxAsync(req, res, next) {
  let result = await getTransactionAsync(req.params.txid)
  if (result.error) return next(result.error)

  res.contentType = 'application/json'
  res.send(result.tx)
  return next()
}

/**
 * GET /calendar/:txid/data handler
 *
 * Expects a path parameter 'txid' as a string
 *
 * Returns a calendar tx's data element
 */
async function getCalTxDataAsync(req, res, next) {
  let result = await getTransactionAsync(req.params.txid)
  if (result.error) return next(result.error)

  res.contentType = 'text/plain'
  res.send(result.tx.tx.data.toLowerCase()) // uppercase hashes are uglyAF
  return next()
}

module.exports = {
  getCalTxAsync: getCalTxAsync,
  getCalTxDataAsync: getCalTxDataAsync,
  // additional functions for testing purposes
  setTmRpc: rpc => {
    tmRpc = rpc
  }
}
