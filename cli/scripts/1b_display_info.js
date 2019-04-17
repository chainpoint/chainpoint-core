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

async function displayWalletInfo(wallet) {
  console.log(chalk.bold.yellow('\n\nNew Ethereum Account:'))
  console.log(chalk.bold.white('\n    Address:'))
  console.log(chalk.gray('   ', wallet.address))

  console.log(chalk.bold.white('\n    Private Key:'))
  console.log(chalk.gray('   ', wallet.privateKey))

  console.log(chalk.bold.yellow('\n\nNext Steps:'))
  console.log(
    chalk.white(
      `  a) PLEASE SEND 5000 TNT TO YOUR NEW ADDRESS (${chalk.bold.gray(
        wallet.address
      )}) IN ORDER TO STAKE, THEN RUN "make stake"\n`
    )
  )
}

module.exports = displayWalletInfo
module.exports.displayWalletInfo = displayWalletInfo
