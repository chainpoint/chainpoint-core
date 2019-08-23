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
let tmRpc = require('../tendermint-rpc.js')
const { version } = require('../../package.json')
let env = require('../parse-env.js')('api')
const logger = require('../logger.js')
const lnService = require('ln-service')
const utils = require('../utils.js')

// initialize lightning grpc object
let macaroon = utils.toBase64(`/root/.lnd/data/chain/bitcoin/${env.NETWORK}/admin.macaroon`)
let tlsCert = utils.toBase64(`/root/.lnd/tls.cert`)
let { lnd } = lnService.authenticatedLndGrpc({
  cert: tlsCert,
  macaroon: macaroon,
  socket: env.LND_SOCKET
})

async function getCoreStatusAsync(req, res, next) {
  let result = await buildStatusObjectAsync()
  if (result.errorCode) {
    switch (result.errorCode) {
      case 404:
        return next(new errors.NotFoundError(result.errorMessage))
      case 409:
        return next(new errors.InvalidArgumentError(result.errorMessage))
      default:
        return next(new errors.InternalServerError(result.errorMessage))
    }
  }

  res.contentType = 'application/json'
  res.send(result.status)
  return next()
}

async function buildStatusObjectAsync() {
  let walletInfo
  let statusResponse = await tmRpc.getStatusAsync()
  if (statusResponse.error) {
    switch (statusResponse.error.responseCode) {
      case 404:
        return { status: null, errorCode: 404, errorMessage: `Resource not found` }
      case 409:
        return { status: null, errorCode: 409, errorMessage: statusResponse.error.message }
      default:
        logger.error(`RPC error communicating with Tendermint : ${statusResponse.error.message}`)
        return { status: null, errorCode: 500, errorMessage: 'Could not query for status' }
    }
  }
  try {
    walletInfo = await lnService.getWalletInfo({ lnd })
  } catch (error) {
    return { status: null, errorCode: 500, errorMessage: 'Could not query for status' }
  }

  let coreInfo = {
    version: version,
    time: new Date().toISOString(),
    base_uri: env.CHAINPOINT_CORE_BASE_URI,
    network: env.NETWORK,
    public_key: walletInfo.identity_pubkey,
    uris: walletInfo.uris,
    active_channels_count: walletInfo.num_active_channels,
    alias: walletInfo.alias
  }

  return {
    status: Object.assign(coreInfo, statusResponse.result)
  }
}

module.exports = {
  getCoreStatusAsync: getCoreStatusAsync,
  buildStatusObjectAsync: buildStatusObjectAsync,
  // additional functions for testing purposes
  setENV: obj => {
    env = obj
  },
  setTmRpc: rpc => {
    tmRpc = rpc
  }
}
