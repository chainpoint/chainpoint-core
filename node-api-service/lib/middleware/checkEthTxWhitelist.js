const fs = require('fs')
const path = require('path')
const ethers = require('ethers')
const errors = require('restify-errors')

const tokenAddress = fs.readFileSync(path.resolve(__dirname + '../../../artifacts/ethcontracts/token.txt'), 'utf8')
const registryAddress = fs.readFileSync(
  path.resolve(__dirname + '../../../artifacts/ethcontracts/registry.txt'),
  'utf8'
)

module.exports = function(req, res, next) {
  // Ensure a raw Eth Tx has been included in the request body
  if (!req.body.tx) {
    return next(new errors.BadRequestError())
  }
  // Ensure that the raw Eth Tx provided is interacting with either the Chainpoint Token or Registry Contracts
  let decodedTx
  try {
    decodedTx = ethers.utils.parseTransaction(req.body.tx)

    if (decodedTx.to !== tokenAddress && decodedTx.to !== registryAddress) {
      return next(
        new errors.BadRequestError({
          message:
            'Only Ethereum transactions that interact with either the Chainpoint Token or Registry contracts are allowed.'
        })
      )
    }
  } catch (error) {
    return next(
      new errors.BadRequestError({
        message:
          'Error parsing Ethereum Tx. Only Ethereum transactions that interact with either the Chainpoint Token or Registry contracts are allowed.'
      })
    )
  }

  return next()
}
