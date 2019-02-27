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
const { version } = require('../../package.json')

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

  let result = Object.assign(
    {
      version: version,
      time: new Date().toISOString()
    },
    status
  )
  res.send(result)
  return next()
}

module.exports = {
  getCoreStatusAsync: getCoreStatusAsync
}
