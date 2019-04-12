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
const Web3 = require('web3')
const env = require('../parse-env.js')('api')

const web3 = new Web3()
web3.setProvider(new web3.providers.HttpProvider(`https://ropsten.infura.io/v3/${env.ETH_INFURA_API_KEY}`))
const infuraProvider = new ethers.providers.InfuraProvider('ropsten', env.ETH_INFURA_API_KEY)

// const tokenContractAddr = '0xB439eBe79cAeaA92C8E8813cEF14411B80bB8ef0'
// const tokenContract = web3.eth.contract(abiArray).at(contractAddress)

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

async function postEthBroadcastAsync(req, res, next) {
  const rawTx = req.body.tx

  try {
    let result = await infuraProvider.sendTransaction(rawTx)
    await result.wait()

    delete result.wait

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
