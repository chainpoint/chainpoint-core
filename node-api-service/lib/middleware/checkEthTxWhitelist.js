const ethers = require('ethers')
const errors = require('restify-errors')
const utils = require('../utils.js')
const env = require('../parse-env.js')

const tknDefinition = require('../../artifacts/ethcontracts/TierionNetworkToken.json')
const regDefinition = require('../../artifacts/ethcontracts/ChainpointRegistry.json')

const network = env.NODE_ENV === 'production' ? 'homestead' : 'ropsten'
const tokenAddress = tknDefinition.networks[network === 'homestead' ? '1' : '3'].address
const registryAddress = regDefinition.networks[network === 'homestead' ? '1' : '3'].address

module.exports = function(req, res, next) {
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
  if (decodedTx.to !== tokenAddress && decodedTx.to !== registryAddress) {
    return next(
      new errors.InvalidArgumentError(
        'invalid request, transaction must interact with Chainpoint token or registry contract'
      )
    )
  }

  return next()
}
