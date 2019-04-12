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
const ethers = require('ethers')
const env = require('../parse-env.js')('api')

let infuraProvider = new ethers.providers.InfuraProvider('ropsten', env.ETH_INFURA_API_KEY)

async function getEthStatsAsync(req, res, next) {
  const ethAddress = req.params.addr

  try {
    let nonce = await infuraProvider.getTransactionCount(ethAddress)
    let gasPrice = await infuraProvider.getGasPrice()

    res.contentType = 'application/json'
    res.send({ nonce, gasPrice: parseInt(gasPrice.toString(), 10) })
  } catch (error) {
    console.error(`Error communicating with Infura attempting to retrieve {nonce, gasPrice} - ${error.message}`)
    return next(new restify.InternalServerError('Error fetching and delivering {nonce, gasPrice}'))
  }
  return next()
}

module.exports = {
  getEthStatsAsync: getEthStatsAsync
}
