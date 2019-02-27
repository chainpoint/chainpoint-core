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

async function getCoresRandomAsync(req, res, next) {
  let netInfo
  try {
    let rpc = connections.openTendermintConnection(env.TENDERMINT_URI)
    netInfo = await rpc.netInfo({})
  } catch (error) {
    console.log(error)
    console.error('rpc error')
    return next(new restify.InternalServerError('Could not get net info'))
  }
  if (!netInfo) {
    res.status(404)
    res.noCache()
    res.send({ code: 'NotFoundError', message: '' })
    return next()
  }
  if (netInfo.peers.length > 0) {
    let decodedPeers = netInfo.peers.map(peer => {
      let byteArray = Array.prototype.slice.call(Buffer.from(peer.remote_ip, 'base64'), 0)
      let newBytes = byteArray.slice(-4)
      return (
        newBytes[0].toString(10) +
        '.' +
        newBytes[1].toString(10) +
        '.' +
        newBytes[2].toString(10) +
        '.' +
        newBytes[3].toString(10)
      )
    })
    res.contentType = 'application/json'
    res.cache('public', { maxAge: 1000 })
    res.send(decodedPeers)
    return next()
  }
  res.noCache()
  res.send([])
  return next()
}

async function getCoreStatusAsync(req, res, next) {
  let status
  try {
    let rpc = connections.openTendermintConnection(env.TENDERMINT_URI)
    status = await rpc.status({})
  } catch (error) {
    console.log(error)
    console.error('rpc error')
    return next(new restify.InternalServerError('Could not query for status'))
  }
  if (!status) {
    res.status(404)
    res.noCache()
    res.send({ code: 'NotFoundError', message: '' })
    return next()
  }
  res.noCache()
  res.contentType = 'application/json'
  res.cache('public', { maxAge: 1000 })
  res.send(status)
  return next()
}

module.exports = {
  getCoresRandomAsync: getCoresRandomAsync,
  getCoreStatusAsync: getCoreStatusAsync
}
