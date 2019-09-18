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

const inquirer = require('inquirer')
const commandLineArgs = require('command-line-args')
const { pipeP } = require('ramda')
const createSwarmAndSecrets = require('./scripts/0_swarm_secrets')
const cliHelloLogger = require('./utils/cliHelloLogger')
const stakingQuestions = require('./utils/stakingQuestions')

const argsDefinitions = [
  { name: 'NETWORK' },
  { name: 'CORE_PUBLIC_IP_ADDRESS' },
  { name: 'BITCOIN_WIF' },
  { name: 'PEERS' },
  { name: 'BTC_RPC_URI_LIST' },
  { name: 'HOT_WALLET_PASS' },
  { name: 'HOT_WALLET_SEED' },
  { name: 'HOT_WALLET_ADDRESS' }
]
const args = commandLineArgs(argsDefinitions)
console.log(args)
async function main() {
  cliHelloLogger()
  if (Object.keys(args).length > 1) {
    await createSwarmAndSecrets(args)
  } else {
    await pipeP(
      () =>
        inquirer.prompt([
          stakingQuestions['NETWORK'],
          stakingQuestions['CORE_PUBLIC_IP_ADDRESS'],
          stakingQuestions['BTC_RPC_URI_LIST'],
          stakingQuestions['BITCOIN_WIF']
        ]),
      createSwarmAndSecrets
    )()
  }
}

main().then(() => {
  process.exit(0)
})
