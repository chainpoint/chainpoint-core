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

const ethers = require('ethers')
const env = require('../parse-env.js')('api')
const errors = require('restify-errors')
const utils = require('../utils.js')
const logger = require('../logger.js')

const network = env.NETWORK === 'mainnet' ? 'homestead' : 'ropsten'

const infuraProvider = new ethers.providers.InfuraProvider(network, env.ETH_INFURA_API_KEY)
const etherscanProvider = new ethers.providers.EtherscanProvider(network, env.ETH_ETHERSCAN_API_KEY)
let fallbackProvider = new ethers.providers.FallbackProvider([infuraProvider, etherscanProvider])

const tknDefinition = require('../../artifacts/ethcontracts/TierionNetworkToken.json')
let tokenAddress = tknDefinition.networks[network === 'homestead' ? '1' : '3'].address
let tokenContractInterface = new ethers.utils.Interface(tknDefinition.abi)

const regDefinition = require('../../artifacts/ethcontracts/ChainpointRegistry.json')
let registryAddress = regDefinition.networks[network === 'homestead' ? '1' : '3'].address
let registryContractInterface = new ethers.utils.Interface(regDefinition.abi)
const registryContract = new ethers.Contract(registryAddress, regDefinition.abi, fallbackProvider)

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
    logger.error(`Error when attempting to retrieve credit price : ${error.message}`)
    return next(new errors.InternalServerError('Error when attempting to retrieve credit price'))
  }
  try {
    let gasPrice = await fallbackProvider.getGasPrice()
    result.gasPrice = gasPrice.toNumber()
  } catch (error) {
    logger.error(`Error when attempting to retrieve gas price : ${error.message}`)
    return next(new errors.InternalServerError('Error when attempting to retrieve gas price'))
  }
  try {
    let transactionCount = await fallbackProvider.getTransactionCount(ethAddress, 'pending')
    result.transactionCount = transactionCount
  } catch (error) {
    logger.error(`Error when attempting to retrieve transaction count : ${ethAddress} : ${error.message}`)
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
      logger.error(`Error when attempting to retrieve Chainpoint Registry info : ${ethAddress} : ${error.message}`)
      return next(new errors.InternalServerError('Error when attempting to retrieve Chainpoint Registry info'))
    }
  }

  res.contentType = 'application/json'
  res.send({ [ethAddress]: result })
  return next()
}

async function postEthBroadcastAsync(req, res, next) {
  // ensure that tx was supplied
  if (!req.params.tx) {
    return next(new errors.InvalidArgumentError('invalid request, tx must be supplied'))
  }

  const rawTx = req.params.tx.toString()
  // ensure that rawTx represents a valid hex value starting wiht 0x
  if (!rawTx.startsWith('0x')) {
    return next(new errors.InvalidArgumentError('invalid request, tx must begin with 0x'))
  }
  // ensure that rawTx represents a valid hex value
  let txContent = rawTx.slice(2)
  if (!utils.isHex(txContent)) {
    return next(new errors.InvalidArgumentError('invalid request, non hex tx value supplied'))
  }

  // ensure that rawTx represents a valid ethereum transaction
  let decodedTx = null
  try {
    decodedTx = ethers.utils.parseTransaction(rawTx)
  } catch (error) {
    return next(new errors.InvalidArgumentError('invalid request, invalid ethereum tx body supplied'))
  }

  // Ensure that the raw Eth Tx provided is interacting with either the Chainpoint Token or Registry Contracts
  // Ensure that the raw Eth Tx provided is invoking an allowed method on our Contracts
  let allowedMethods, parsedTx
  switch (decodedTx.to) {
    case tokenAddress: {
      allowedMethods = ['approve']
      parsedTx = tokenContractInterface.parseTransaction(decodedTx)
      break
    }
    case registryAddress: {
      allowedMethods = ['stake', 'unStake', 'updateStake']
      parsedTx = registryContractInterface.parseTransaction(decodedTx)
      break
    }
    default: {
      return next(
        new errors.InvalidArgumentError(
          'invalid request, transaction must interact with Chainpoint token or registry contract'
        )
      )
    }
  }

  if (!allowedMethods.includes(parsedTx.name)) {
    return next(
      new errors.InvalidArgumentError(
        `invalid request, transaction may only call '${allowedMethods.join(',')}' method(s) on that contract`
      )
    )
  }

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
    logger.error(`Error when attempting to broadcast ETH Tx : ${error.message}`)
    return next(new errors.InternalServerError(error.message))
  }

  res.contentType = 'application/json'
  res.send(result)
  return next()
}

module.exports = {
  getEthStatsAsync: getEthStatsAsync,
  postEthBroadcastAsync: postEthBroadcastAsync,
  // additional functions for testing purposes
  setFP: fp => {
    fallbackProvider = fp
  },
  setTA: ta => {
    tokenAddress = ta
  },
  setRA: ra => {
    registryAddress = ra
  },
  setTCI: tci => {
    tokenContractInterface = tci
  },
  setRCI: rci => {
    registryContractInterface = rci
  }
}
