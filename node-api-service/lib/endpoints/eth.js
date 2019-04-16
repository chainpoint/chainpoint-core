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
const utils = require('../utils.js')
const env = require('../parse-env.js')('api')
const infuraProvider = new ethers.providers.InfuraProvider('ropsten', env.ETH_INFURA_API_KEY)

async function getEthStatsAsync(req, res, next) {
  const ethAddress = req.params.addr
  // ensure that addr represents a valid, well formatted ETH address
  if (!/^0x[0-9a-fA-F]{40}$/i.test(ethAddress)) {
    return next(new restify.InvalidArgumentError('invalid request, invalid ethereum address supplied'))
  }

  let result = {}
  try {
    let creditPrice = 0.001 // TODO: Build and request from exchange rate service
    result.creditPrice = creditPrice
  } catch (error) {
    console.error(`Error when attempting to retrieve credit price : ${error.message}`)
    return next(new restify.InternalServerError('Error when attempting to retrieve credit price'))
  }
  try {
    let gasPrice = await infuraProvider.getGasPrice()
    result.gasPrice = gasPrice.toNumber()
  } catch (error) {
    console.error(`Error when attempting to retrieve gas price : ${error.message}`)
    return next(new restify.InternalServerError('Error when attempting to retrieve gas price'))
  }
  try {
    let transactionCount = await infuraProvider.getTransactionCount(ethAddress)
    result.transactionCount = transactionCount.toNumber()
  } catch (error) {
    console.error(`Error when attempting to retrieve transaction count : ${ethAddress} : ${error.message}`)
    return next(new restify.InternalServerError('Error when attempting to retrieve transaction count'))
  }

  res.contentType = 'application/json'
  res.send(result)
  return next()
}

async function postEthBroadcastAsync(req, res, next) {
  const rawTx = req.params.tx
  // ensure that rawTx represents a valid hex value starting wiht 0x
  if (!rawTx.startsWith('0x')) {
    return next(new restify.InvalidArgumentError('invalid request, transaction must begin with 0x'))
  }
  // ensure that rawTx represents a valid hex value
  let txContent = rawTx.slice(2)
  if (!utils.isHex(txContent)) {
    return next(new restify.InvalidArgumentError('invalid request, non hex value supplied'))
  }

  // ensure that rawTx represents a valid ethereum transaction
  try {
    ethers.utils.parseTransaction(rawTx)
  } catch (error) {
    return next(new restify.InvalidArgumentError('invalid request, invalid ethereum transaction body supplied'))
  }

    res.send(result)
  } catch (error) {
    console.error(`Error communicating with Infura attempting to broadcast ETH Tx - ${error.message}`)
    return next(new restify.InternalServerError(error.message))
  }
  return next()
}

module.exports = {
  getEthStatsAsync: getEthStatsAsync,
  postEthBroadcastAsync: postEthBroadcastAsync
}
