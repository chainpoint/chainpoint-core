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

const chalk = require('chalk')
const fs = require('fs')
const path = require('path')
const ethers = require('ethers')
const env = require('../../lib/parse-env.js')()

let tknDefinition = require('../../go-abci-service/ethcontracts/TierionNetworkToken.json')
let regDefinition = require('../../go-abci-service/ethcontracts/ChainpointRegistry.json')

const network = env.NODE_ENV === 'production' ? 'homestead' : 'ropsten'

const tokenAddress = tknDefinition.networks[network === 'homestead' ? '1' : '3'].address
const registryAddress = regDefinition.networks[network === 'homestead' ? '1' : '3'].address
const provider = ethers.getDefaultProvider(network)

const privateKey = fs.readFileSync(path.resolve('/run/secrets/ETH_PRIVATE_KEY', 'utf8'))
const wallet = new ethers.Wallet(privateKey)
const CORE_TNT_STAKE_AMOUNT = 2500000000000

async function approve() {
  let tokenContract = new ethers.Contract(tokenAddress, tknDefinition.abi, wallet)
  console.log(chalk.gray('-> Approving Allowance: ' + wallet.address))
  let approval = await tokenContract.approve(registryAddress, CORE_TNT_STAKE_AMOUNT)
  await approval.wait()
  let txReceipt = await provider.getTransactionReceipt(approval.hash)
  return txReceipt
}

async function register(ip) {
  let registryContract = new ethers.Contract(registryAddress, regDefinition.abi, wallet)
  let stakeResult = await registryContract.stakeCore(ip)
  await stakeResult.wait()
  let txReceipt = await provider.getTransactionReceipt(stakeResult.hash)
  return txReceipt
}

module.exports.register = register
module.exports.approve = approve
