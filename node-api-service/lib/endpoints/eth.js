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

const fs = require('fs')
const path = require('path')
const ethers = require('ethers')
const env = require('../parse-env.js')('api')
const errors = require('restify-errors')

const network = env.NODE_ENV === 'production' ? 'homestead' : 'ropsten'
const infuraProvider = new ethers.providers.InfuraProvider(network, env.ETH_INFURA_API_KEY)
const etherscanProvider = new ethers.providers.EtherscanProvider(network, env.ETH_ETHERSCAN_API_KEY)
const fallbackProvider = new ethers.providers.FallbackProvider([infuraProvider, etherscanProvider])

const privKey = fs.readFileSync(path.resolve('/run/secrets/ETH_PRIVATE_KEY'), 'utf-8')
let wallet = new ethers.Wallet(privKey)
wallet = wallet.connect(fallbackProvider)

const regDefinition = require('../../artifacts/ethcontracts/ChainpointRegistry.json')
const registryAddress = regDefinition.networks[network === 'homestead' ? '1' : '3'].address
const registryContract = new ethers.Contract(registryAddress, regDefinition.abi, wallet)

async function getEthStatsAsync(req, res, next) {
  const ethAddress = req.params.addr
  // ensure that addr represents a valid, well formatted ETH address
  if (!/^0x[0-9a-fA-F]{40}$/i.test(ethAddress)) {
    return next(new errors.InvalidArgumentError('invalid request, invalid ethereum address supplied'))
  }

  let result = {}
  try {
    let creditPrice = 0.1 // TODO: Build and request from exchange rate service
    result.creditPrice = creditPrice
  } catch (error) {
    console.error(`Error when attempting to retrieve credit price : ${error.message}`)
    return next(new errors.InternalServerError('Error when attempting to retrieve credit price'))
  }
  try {
    let gasPrice = await fallbackProvider.getGasPrice()
    result.gasPrice = gasPrice.toNumber()
  } catch (error) {
    console.error(`Error when attempting to retrieve gas price : ${error.message}`)
    return next(new errors.InternalServerError('Error when attempting to retrieve gas price'))
  }
  try {
    let transactionCount = await fallbackProvider.getTransactionCount(ethAddress)
    result.transactionCount = transactionCount
  } catch (error) {
    console.error(`Error when attempting to retrieve transaction count : ${ethAddress} : ${error.message}`)
    return next(new errors.InternalServerError('Error when attempting to retrieve transaction count'))
  }

  if (req.query.verbose && (req.query.verbose === 'true' || req.query.verbose === true)) {
    try {
      let registrationResult = await registryContract.nodes(ethAddress)

      result.registration = {
        isStaked: registrationResult.isStaked,
        amountStaked: registrationResult.amountStaked.toNumber(),
        stakeLockedUntil: registrationResult.stakeLockedUntil.toNumber()
      }
    } catch (error) {
      console.error(`Error when attempting to retrieve Chainpoint Registry info : ${ethAddress} : ${error.message}`)
      return next(new errors.InternalServerError('Error when attempting to retrieve Chainpoint Registry info'))
    }
  }

  res.contentType = 'application/json'
  res.send({ [ethAddress]: result })
  return next()
}

async function postEthBroadcastAsync(req, res, next) {
  const rawTx = req.params.tx

  let result
  try {
    let sendResponse = await fallbackProvider.sendTransaction(rawTx)
    let txReceipt = await fallbackProvider.waitForTransaction(sendResponse.hash)
    let transactionHash = txReceipt.transactionHash
    let blockHash = txReceipt.blockHash
    let blockNumber = txReceipt.blockNumber
    let gasUsed = txReceipt.gasUsed.toNumber() // convert from BigNumber to native number
    result = { transactionHash, blockHash, blockNumber, gasUsed }
  } catch (error) {
    console.error(`Error when attempting to broadcast ETH Tx : ${error.message}`)
    return next(new errors.InternalServerError(error.message))
  }

  res.contentType = 'application/json'
  res.send(result)
  return next()
}

module.exports = {
  getEthStatsAsync: getEthStatsAsync,
  postEthBroadcastAsync: postEthBroadcastAsync
}
