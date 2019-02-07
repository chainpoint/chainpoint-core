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
 * env required for tnt to credit rate
 */
const env = require('./parse-env.js')()
const BigNumber = require('bignumber.js')

/**
 * Format TNT Amount for Token Transfer
 * @return {number}
 */
function tntToGrains (tntAmount) {
  return new BigNumber(tntAmount).times(10 ** 8).toNumber()
}

/**
 * Format TNT Amount from Token Transfer
 * @return {number}
 */
function grainsToTNT (grainsAmount) {
  return new BigNumber(grainsAmount).dividedBy(10 ** 8).toNumber()
}

/**
 * Format TNT Amount from Token Transfer to TNT Credit
 * @return {number}
 */
function grainsToCredits (grainsAmount) {
  return new BigNumber(grainsAmount).times(env.TNT_TO_CREDIT_RATE).dividedBy(10 ** 8).toNumber()
}

module.exports = {
  tntToGrains: tntToGrains,
  grainsToTNT: grainsToTNT,
  grainsToCredits: grainsToCredits
}
