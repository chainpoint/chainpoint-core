const fs = require('fs')
const path = require('path')
const ethers = require('ethers')
const errors = require('restify-errors')
const utils = require('../utils.js')

const tokenAddress = fs.readFileSync(path.resolve(__dirname + '../../../artifacts/ethcontracts/token.txt'), 'utf8')
const registryAddress = fs.readFileSync(
  path.resolve(__dirname + '../../../artifacts/ethcontracts/registry.txt'),
  'utf8'
)

module.exports = function(req, res, next) {
  const rawTx = req.params.tx

  // ensure that rawTx was supplied
  if (!rawTx) {
    return next(new errors.InvalidArgumentError('invalid request, tx must be supplied'))
  }
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
