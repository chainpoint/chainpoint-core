/* Copyright (C) 2018 Tierion
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

var Web3 = require('web3')
const contract = require('truffle-contract')
/*
 * This function will load the TierionNetworkToken token ABI from the build directory and initialize
 * it so that it can be used to interact with the blockchain. It will use env variable
 * set for the address of the contract on the blockchain.
*/
module.exports = async (provider, tokenAddr) => {
  // Create the provider
  console.log('Creating web3')
  let web3 = new Web3(provider)

  // Load the token json obj
  console.log('loading TierionNetworkToken token')
  let tokenDef = require('../../contracts/TierionNetworkToken.json')

  // If the token addr is specified in the environment var, use that as highest priority
  if (tokenAddr && tokenAddr !== '') {
    console.log('Using token addr: ' + tokenAddr)

    // Load the ABI for the contract and initialize a contract interface
    let tokenABI = tokenDef.abi
    let tokenDefinition = web3.eth.contract(tokenABI)

    // Set the actual instance from the address on the blockchain, so we can communicate with it.
    return tokenDefinition.at(tokenAddr)
  }

  // If the env var was not set, see if the token definition has been deployed.
  const token = contract(tokenDef)
  token.setProvider(provider)
  let deployedToken = await token.deployed()

  // Didn't find it there either... bail out
  if (!deployedToken) {
    console.error('TierionNetworkToken Token ERC20 Contract Address is not found deployed or set as env var (ETH_TNT_TOKEN_ADDR) - Exiting...')
    process.exit(-1)
  }

  // Dumb workaround for bug - https://github.com/ethereum/web3.js/issues/925
  console.log('Using TNT Token contract at ', deployedToken.address)
  let tokenABI = tokenDef.abi
  let tokenDefinition = web3.eth.contract(tokenABI)
  return tokenDefinition.at(deployedToken.address)
}
