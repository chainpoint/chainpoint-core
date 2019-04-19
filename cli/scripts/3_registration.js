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
// const chalk = require('chalk')
const { pipeP } = require('ramda')
const ethers = require('ethers')
const { getETHStatsByAddressAsync, broadcastEthTxAsync } = require('../../lib/cores')
// const tap = require('./utils/tap')

// const ChainpointRegistryABI = require('../../artifacts/ethcontracts/ChainpointRegistry.json').abi
const TierionNetworkTokenABI = require('../../go-abci-service/ethcontracts/TierionNetworkToken.json').abi
const tokenAddress = fs.readFileSync(path.resolve('../../go-abci-service/ethcontracts/token.txt', 'utf8'))
const registryAddress = fs.readFileSync(path.resolve('../../go-abci-service/ethcontracts/token.txt', 'utf8'))

// const privateKey = fs.readFileSync(path.resolve('/run/secrets/NODE_ETH_PRIVATE_KEY', 'utf8'))
const privateKey = 'super private key...'
const wallet = new ethers.Wallet(privateKey)

async function approve(
  txData = {
    gasPrice: 2000000000,
    nonce: 0
  }
) {
  const tokenInterface = new ethers.Interface(TierionNetworkTokenABI)
  const funcSigEncoded = tokenInterface.functions.approve(registryAddress, 500000000000)

  const tx = {
    gasPrice: txData.gasPrice,
    gasLimit: 185000,
    data: funcSigEncoded.data,
    to: tokenAddress,
    nonce: txData.nonce
  }

  return wallet.sign(tx)
}

async function register() {
  // const tokenInterface = new ethers.Interface(TierionNetworkTokenABI)
  // const funcSigEncoded = tokenInterface.functions.approve(registryAddress, 500000000000)
  // return wallet.sign(tx)
}

module.exports.register = register
module.exports.approve = pipeP(
  getETHStatsByAddressAsync.bind(null, wallet.address),
  approve,
  broadcastEthTxAsync
)