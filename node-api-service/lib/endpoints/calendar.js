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

/**
 * GET /calendar/:txid handler
 *
 * Expects a path parameter 'txid' as a string
 *
 * Returns a calendar tx by tx hash
 */
async function getCalTxAsync(req, res, next) {
  let txID = req.params.txid
  if (!txID.includes('0x')) {
    txID = '0x' + txID
  }
  let tx
  try {
    let rpc = connections.openTendermintConnection(env.TENDERMINT_URI)
    tx = await rpc.tx({ hash: txID, prove: false })
  } catch (error) {
    console.error('rpc error')
    return next(new restify.InternalServerError('Could not query for tx by hash'))
  }
  if (!tx) {
    res.status(404)
    res.noCache()
    res.send({ code: 'NotFoundError', message: '' })
    return next()
  }
  res.contentType = 'application/json'
  res.cache('public', { maxAge: 2592000 })
  res.send(tx)
  return next()
}

/**
 * GET /calendar/:txid/data handler
 *
 * Expects a path parameter 'txid' as a string
 *
 * Returns a calendar tx by tx hash
 */
async function getCalTxDataAsync(req, res, next) {
  let txID = req.params.txid
  if (!txID.includes('0x')) {
    txID = '0x' + txID
  }
  let tx
  try {
    let rpc = connections.openTendermintConnection(env.TENDERMINT_URI)
    tx = await rpc.tx({ hash: txID, prove: false })
  } catch (error) {
    console.error('rpc error')
    return next(new restify.InternalServerError('Could not query for tx by hash'))
  }
  if (!tx) {
    res.status(404)
    res.noCache()
    res.send({ code: 'NotFoundError', message: '' })
    return next()
  }
  let txData = new Buffer(tx.tx, 'base64').toString('ascii')
  let jsonData = JSON.parse(txData)
  res.contentType = 'application/json'
  res.cache('public', { maxAge: 2592000 })
  res.send(jsonData.Data)
  return next()
}

module.exports = {
  getCalTxAsync: getCalTxAsync,
  getCalTxDataAsync: getCalTxDataAsync
}
