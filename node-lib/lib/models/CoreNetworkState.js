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

const Sequelize = require('sequelize-cockroachdb')

const envalid = require('envalid')

const env = envalid.cleanEnv(process.env, {
  COCKROACH_CORE_NETWORK_STATE_TABLE_NAME: envalid.str({ default: 'chainpoint_core_network_state', desc: 'CockroachDB table name' })
})

let coreNetworkState

function defineFor (sqlz) {
  let CoreNetworkState = sqlz.define(env.COCKROACH_CORE_NETWORK_STATE_TABLE_NAME, {
    stateKey: { type: Sequelize.STRING, primaryKey: true, field: 'state_key', allowNull: false },
    stateValue: { type: Sequelize.STRING, field: 'state_value', allowNull: false }
  }, {
    // enable timestamps
    timestamps: true,
    // don't use camelcase for automatically added attributes but underscore style
    // so updatedAt will be updated_at
    underscored: true,
    indexes: [
      {
        unique: true,
        fields: ['state_key', 'state_value']
      }
    ]
  })

  coreNetworkState = CoreNetworkState
  return CoreNetworkState
}

const LAST_AGG_STATE_PROCESSED_FOR_CAL_BLOCK_TIMESTAMP = 'LAST_AGG_STATE_PROCESSED_FOR_CAL_BLOCK_TIMESTAMP'
const LAST_CAL_BLOCK_HEIGHT_PROCESSED_FOR_BTC_A_BLOCK = 'LAST_CAL_BLOCK_HEIGHT_PROCESSED_FOR_BTC_A_BLOCK'

async function getLastAggStateProcessedForCalBlockTimestamp () {
  let results = await coreNetworkState.find({ where: { stateKey: LAST_AGG_STATE_PROCESSED_FOR_CAL_BLOCK_TIMESTAMP }, raw: true })
  return results ? parseInt(results.stateValue) : null
}

async function setLastAggStateProcessedForCalBlockTimestamp (value) {
  let stateObject = {
    stateKey: LAST_AGG_STATE_PROCESSED_FOR_CAL_BLOCK_TIMESTAMP,
    stateValue: value.toString()
  }
  await coreNetworkState.upsert(stateObject)
}

async function getLastCalBlockHeightProcessedForBtcABlock () {
  let results = await coreNetworkState.find({ where: { stateKey: LAST_CAL_BLOCK_HEIGHT_PROCESSED_FOR_BTC_A_BLOCK }, raw: true })
  return results ? parseInt(results.stateValue) : null
}

async function setLastCalBlockHeightProcessedForBtcABlock (value) {
  let stateObject = {
    stateKey: LAST_CAL_BLOCK_HEIGHT_PROCESSED_FOR_BTC_A_BLOCK,
    stateValue: value.toString()
  }
  await coreNetworkState.upsert(stateObject)
}

module.exports = {
  getLastAggStateProcessedForCalBlockTimestamp: getLastAggStateProcessedForCalBlockTimestamp,
  getLastCalBlockHeightProcessedForBtcABlock: getLastCalBlockHeightProcessedForBtcABlock,
  setLastAggStateProcessedForCalBlockTimestamp: setLastAggStateProcessedForCalBlockTimestamp,
  setLastCalBlockHeightProcessedForBtcABlock: setLastCalBlockHeightProcessedForBtcABlock,
  defineFor: defineFor
}
