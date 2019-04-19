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
const tmRpc = require('../tendermint-rpc.js')

async function getPeersAsync(req, res, next) {
  let netResponse = await tmRpc.getNetInfoAsync()
  if (netResponse.error) {
    switch (netResponse.error.responseCode) {
      case 404:
        return next(new errors.NotFoundError(`Resource not found`))
      case 409:
        return next(new errors.InvalidArgumentError(netResponse.error.message))
      default:
        console.error(`RPC error communicating with Tendermint : ${netResponse.error.message}`)
        return next(new errors.InternalServerError('Could not query for net info'))
    }
  }

  let decodedPeers = netResponse.result.peers
    .map(peer => {
      let ipBytes = Buffer.from(peer.remote_ip, 'base64').slice(-4)
      let remoteIP = ipBytes.join('.')
      let firstOctet = remoteIP.substring(0, remoteIP.indexOf('.'))
      //use listen_addr if there are non-routable peer exchange IPs when behind NATs
      if (firstOctet == '10' || firstOctet == '172' || firstOctet == '192') {
        let listenAddr = peer.node_info.listen_addr
        if (listenAddr.includes('//')) {
          return listenAddr.substring(listenAddr.lastIndexOf('/'), listenAddr.lastIndexOf(':'))
        }
        return listenAddr.substring(0, listenAddr.lastIndexOf(':'))
      }
      return remoteIP
    })
    .filter(ip => {
      //filter out non-routable IPs that slipped through the above fallback
      let firstOctet = ip.substring(0, ip.indexOf('.'))
      return firstOctet != '0' || firstOctet != '10' || firstOctet != '172' || firstOctet != '192'
    })
  res.contentType = 'application/json'
  res.send(decodedPeers)
  return next()
}

module.exports = {
  getPeersAsync: getPeersAsync
}
