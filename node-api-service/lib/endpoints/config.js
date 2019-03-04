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

// get the first entry in the ETH_TNT_LISTEN_ADDRS CSV to publicize
let coreEthAddress = env.ETH_TNT_LISTEN_ADDRS.split(',')[0]

/**
 * GET /config handler
 *
 * Returns a configuration information object
 */
async function getConfigInfoV1Async(req, res, next) {
  let result = {
    chainpoint_core_base_uri: env.CHAINPOINT_CORE_BASE_URI,
    core_eth_address: coreEthAddress
  }
  res.cache('public', { maxAge: 60 })
  res.send(result)
  return next()
}

module.exports = {
  getConfigInfoV1Async: getConfigInfoV1Async
}
