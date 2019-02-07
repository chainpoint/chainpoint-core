
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

/**
 * This class can be used to interact with the TierionNetworkToken token on the blockchain.
 */
class TokenOps {
  /**
   * Constructor - Takes the ERC 20 token obj already initialized to an instance on the blockchain.
   */
  constructor (tokenContract) {
    this.tokenContract = tokenContract
  }

  /**
   * This function will send tokens to the specified address.
   * The "From" address signing the transaction will be the default account set in the web3 object.
   *
   * @param {*} sendToAddr - Target address
   * @param {*} amt - base units to transfer (TNT Grains, 8 decimals), 1 TNT => Math.pow(10, 8) => 100000000 Grains
   * @param {*} callback - Called after the transaction is broadcast
   */
  sendTokens (sendToAddr, amt, callback) {
    return this.tokenContract.transfer(sendToAddr, amt, {gas: 2000000}, callback)
  }

  /**
   * Listener function that will invoke callback whenever a new transaction is found.
   * It will trigger any events from blockstart onwards (pass 0 to see all events in the history of the blockchain.)
   *
   * returns the event object so watch can be canceled with event.stopWatching()
   *
   * @param {*} listenAddr - address to listen for incoming transfers
   * @param {*} blockStart - block to start listening from
   * @param {*} callback - callback invoked whenever a new transfer is received to listenAddr
   */
  watchForTransfers (listenAddrs, blockStart, callback) {
    this.tokenContract.Transfer({'to': listenAddrs}, {'fromBlock': blockStart}, callback)
  }

  /**
   * This function will query the balance of tokens in an address.
   * Note the amt will be specified in base unites.
   *
   * @param {*} address - ETH Address to query
   * @param {*} callback - Called with the amount of tokens
   */
  getBalance (address, callback) {
    return this.tokenContract.balanceOf.call(address, callback)
  }
}
module.exports = TokenOps
