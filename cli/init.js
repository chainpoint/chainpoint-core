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
const { pipe, pipeP } = require('ramda')
const tap = require('./utils/tap')
const createSwarmAndSecrets = require('./scripts/0_swarm_secrets')
const cliHelloLogger = require('./utils/cliHelloLogger')
const createWallet = require('./scripts/1_create_wallet')
const createDockerSecrets = require('./scripts/1a_wallet_docker_secrets')
const displayWalletInfo = require('./scripts/1b_display_info')
const updateOrCreateEnv = require('./scripts/2_update_env')
const stakingQuestions = require('./utils/stakingQuestions')

const resolve = Promise.resolve.bind(Promise)

async function main() {
  cliHelloLogger()

  await pipeP(
    () =>
      inquirer.prompt([
        stakingQuestions['CORE_PUBLIC_IP_ADDRESS'],
        stakingQuestions['INSIGHT_API_URI'],
        stakingQuestions['BITCOIN_WIF'],
        stakingQuestions['INFURA_API_KEY'],
        stakingQuestions['ETHERSCAN_API_KEY']
      ]),
    createSwarmAndSecrets,
    createWallet,
    createDockerSecrets,
    tap(
      pipe(
        currVal => ({
          ETH_ADDRESS: currVal.address
        }),
        updateOrCreateEnv
      ),
      resolve
    ),
    //tap(spinner.succeed.bind(spinner, chalk.bold.yellow('New Wallet:\n')), resolve),
    displayWalletInfo
  )()
}

main().then(() => {
  process.exit(0)
})
