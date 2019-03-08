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

const restify = require('restify')
const tmRpc = require('../tendermint-rpc.js')
const { version } = require('../../package.json')
const env = require('../parse-env.js')('api')

let coreEthAddress = env.ETH_TNT_LISTEN_ADDR

async function getCoreStatusAsync(req, res, next) {
  let statusResponse = await tmRpc.getStatusAsync()
  if (statusResponse.error) {
    switch (statusResponse.error.responseCode) {
      case 404:
        return { tx: null, error: new restify.NotFoundError(`Resource not found`) }
      case 409:
        return { tx: null, error: new restify.InvalidArgumentError(statusResponse.error.message) }
      default:
        console.error(`RPC error communicating with Tendermint : ${statusResponse.error.message}`)
        return { tx: null, error: new restify.InternalServerError('Could not query for status') }
    }
  }

  let result = Object.assign(
    {
      version: version,
      time: new Date().toISOString(),
      base_uri: env.CHAINPOINT_CORE_BASE_URI,
      eth_address: coreEthAddress
    },
    statusResponse.result
  )
  res.contentType = 'application/json'
  res.send(result)
  return next()
}

module.exports = {
  getCoreStatusAsync: getCoreStatusAsync
}
