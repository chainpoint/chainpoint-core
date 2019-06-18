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

const Sequelize = require('sequelize')

const envalid = require('envalid')

const env = envalid.cleanEnv(process.env, {
  STAKED_CORE_TABLE_NAME: envalid.str({
    default: 'staked_cores'
  })
})

let StakedCore

function defineFor(sqlz) {
  let StakedCore = sqlz.define(
    env.STAKED_CORE_TABLE_NAME,
    {
      ethAddr: {
        comment: 'A seemingly valid Ethereum address that the Core will send TNT from, or receive rewards with.',
        type: Sequelize.STRING,
        validate: {
          is: ['^0x[0-9a-f]{40}$']
        },
        field: 'eth_addr',
        allowNull: false,
        primaryKey: true
      },
      coreId: {
        comment: 'A base64 Tendermint ID of a given Core.',
        type: Sequelize.STRING,
        validate: {
          is: ['^0x[0-9a-f]{40}$']
        },
        field: 'core_id',
        allowNull: true
      },
      publicIp: {
        comment: 'The public IP address of a Core',
        type: Sequelize.STRING,
        validate: {
          isIP: true
        },
        field: 'public_ip',
        allowNull: false
      },
      blockNumber: {
        comment: 'The eth block number where this info was valid. Used for versioning core stake updates',
        type: Sequelize.BIGINT, // is 64 bit in CockroachDB
        validate: {
          isInt: true
        },
        field: 'block_number',
        allowNull: true
      }
    },
    {
      // Disable the modification of table names; By default, sequelize will automatically
      // transform all passed model names (first parameter of define) into plural.
      // if you don't want that, set the following
      freezeTableName: true,
      // enable timestamps
      timestamps: true,
      // don't use camelcase for automatically added attributes but underscore style
      // so updatedAt will be updated_at
      underscored: true,
      indexes: [
        {
          unique: false,
          fields: ['created_at']
        },
        {
          unique: true,
          fields: ['public_ip']
        }
      ]
    }
  )

  return StakedCore
}

async function getRandomCoresAsync() {
  let results = await StakedCore.findAll({ order: Sequelize.literal('random()'), limit: 25, raw: true })
  return results
}

async function hasMemberIPAsync(ip) {
  let count = await StakedCore.count({ where: { public_ip: ip }, raw: true })
  return count > 0
}

module.exports = {
  defineFor: defineFor,
  getRandomCoresAsync: getRandomCoresAsync,
  hasMemberIPAsync: hasMemberIPAsync,
  setDatabase: (sqlz, stakedCore) => {
    // sequelize = sqlz
    StakedCore = stakedCore
  }
}
