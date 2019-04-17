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
const chalk = require('chalk')
const { pipeP } = require('ramda')
const inquirer = require('inquirer')
const cliHelloLogger = require('./utils/cliHelloLogger')
const stakingQuestions = require('./utils/stakingQuestions')
const updateOrCreateEnv = require('./scripts/2_update_env')

async function main() {
  cliHelloLogger()

  console.log(fs.readFileSync(path.resolve('/run/secrets/ETH_ADDRESS'), 'utf-8'))

  console.log(chalk.bold.yellow('Stake your Core:'))

  try {
    await pipeP(
      () => inquirer.prompt([stakingQuestions['ETH_REWARDS_ADDRESS']]),
      updateOrCreateEnv
      // TODO: /eth/:addr/txdata
      // TODO: /eth/broadcast
    )()

    console.log(chalk.green('\n===================================='))
    console.log(chalk.green('==   SUCCESSFULLY STAKED CORE!    =='))
    console.log(chalk.green('====================================', '\n'))
  } catch (error) {
    console.log(chalk.red('Failed to Stake Core to Chainpoint Network. Please try again.'))
  }
}

main().then(() => {
  process.exit(0)
})
