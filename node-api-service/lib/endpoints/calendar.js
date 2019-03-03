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
const env = require('../parse-env.js')('api')
const restify = require('restify')
const connections = require('../connections.js')

async function getTransactionAsync(txID) {
  if (!txID.startsWith('0x')) txID = `0x${txID}`
  let tx
  try {
    let rpc = connections.openTendermintConnection(env.TENDERMINT_URI)
    tx = await rpc.tx({ hash: txID, prove: false })
  } catch (error) {
    // check for 404
    if (error.code === -32603) {
      return {
        tx: null,
        error: new restify.NotFoundError(`Could not find transaction with id = '${txID}'`)
      }
    }
    console.error(`RPC error communicating with Tendermint : ${error.data}`)
    return {
      tx: null,
      error: new restify.InternalServerError('Could not query for tx by hash')
    }
  }
  // Txs are double-encoded in base64 when returned from this RPC client
  tx.tx = JSON.parse(new Buffer(new Buffer(tx.tx, 'base64').toString('ascii'), 'base64').toString('ascii'))

  return {
    tx: tx,
    error: null
  }
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
  res.send(result.tx.hash.toLowerCase()) // uppercase hashes are uglyAF
  return next()
}

module.exports = {
  getCalTxAsync: getCalTxAsync,
  getCalTxDataAsync: getCalTxDataAsync
}
