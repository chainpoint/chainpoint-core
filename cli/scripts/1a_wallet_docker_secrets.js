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

const exec = require('executive')
const chalk = require('chalk')

async function createDockerSecrets(wallet) {
  try {
    await exec.quiet([
      `printf ${wallet.address} | docker secret create ETH_ADDRESS -`,
      `printf ${wallet.privateKey} | docker secret create ETH_PRIVATE_KEY -`
    ])

    return wallet
  } catch (err) {
    console.log(chalk.red('Error creating Docker secrets for ETH_ADDRESS & ETH_PRIVATE_KEY'))

    return Promise.reject(err)
  }
}

module.exports = createDockerSecrets
