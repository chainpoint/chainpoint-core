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

const validator = require('validator')

module.exports = {
  NETWORK: {
    type: 'list',
    name: 'NETWORK',
    message: 'Will this Core use Bitcoin mainnet or testnet?',
    choices: [
      {
        name: 'Mainnet',
        value: 'mainnet'
      },
      {
        name: 'Testnet',
        value: 'testnet'
      }
    ],
    default: 'mainnet'
  },
  CORE_PUBLIC_IP_ADDRESS: {
    type: 'input',
    name: 'CORE_PUBLIC_IP_ADDRESS',
    message: "Enter your Instance's Public IP Address:",
    validate: input => {
      if (input) {
        return validator.isIP(input, 4)
      } else {
        return true
      }
    }
  },
  BITCOIN_WIF: {
    type: 'input',
    name: 'BITCOIN_WIF',
    message: 'Enter the Bitcoin private key for your hotwallet:'
  },
  BTC_RPC_URI_LIST: {
    type: 'input',
    name: 'BTC_RPC_URI_LIST',
    message: "Enter the full URL (including protocol and port) to your bitcoin node's RPC endpoint:"
  }
}
