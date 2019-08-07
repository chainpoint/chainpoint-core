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
    message: 'Will this Core use Bitcoin mainnet or testnet)?',
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
  },
  AUTO_REFILL_ENABLED: {
    type: 'list',
    name: 'AUTO_REFILL_ENABLED',
    message: 'Enable automatic acquisition of credit when balance reaches 0?',
    choices: [
      {
        name: 'Enable',
        value: true
      },
      {
        name: 'Disable',
        value: false
      }
    ],
    default: true
  },
  AUTO_REFILL_AMOUNT: {
    type: 'number',
    name: 'AUTO_REFILL_AMOUNT',
    message: 'Enter Auto Refill Amount - specify in number of Credits (optional: specify if auto refill is enabled)',
    default: 720,
    validate: (val, answers) => {
      if (answers['AUTO_REFILL_ENABLED'] == true) {
        return val >= 1 && val <= 8760
      } else {
        return val >= 0 && val <= 8760
      }
    }
  }
}
