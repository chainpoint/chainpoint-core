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
  COCKROACH_CAL_TABLE_NAME: envalid.str({ default: 'chainpoint_calendar_blockchain', desc: 'CockroachDB table name' })
})

function defineFor (sqlz) {
  let CalendarBlock = sqlz.define(env.COCKROACH_CAL_TABLE_NAME,
    {
      id: {
        comment: 'Sequential monotonically incrementing Integer ID representing block height.',
        primaryKey: true,
        type: Sequelize.INTEGER,
        validate: {
          isInt: true
        },
        allowNull: false
      },
      time: {
        comment: 'Block creation time in seconds since unix epoch',
        type: Sequelize.INTEGER,
        validate: {
          isInt: true
        },
        allowNull: false
      },
      version: {
        comment: 'Block version number, for future use.',
        type: Sequelize.INTEGER,
        defaultValue: function () {
          return 1
        },
        validate: {
          isInt: true
        },
        allowNull: false
      },
      stackId: {
        comment: 'The Chainpoint stack identifier. Should be the domain or IP of the API. e.g. a.chainpoint.org',
        type: Sequelize.STRING,
        field: 'stack_id',
        allowNull: false
      },
      type: {
        comment: 'Block type.',
        type: Sequelize.STRING,
        validate: {
          isIn: [['gen', 'cal', 'nist', 'btc-a', 'btc-c', 'eth-a', 'eth-c', 'reward']]
        },
        allowNull: false
      },
      dataId: {
        comment: 'The identifier for the data to be anchored to this block, data identifier meaning is determined by block type.',
        type: Sequelize.STRING,
        validate: {
          is: ['^[a-fA-F0-9:x]{0,255}$', 'i']
        },
        field: 'data_id',
        allowNull: false
      },
      dataVal: {
        comment: 'The data to be anchored to this block, data value meaning is determined by block type.',
        type: Sequelize.STRING,
        validate: {
          is: ['^[a-fA-F0-9:x]{1,255}$', 'i']
        },
        field: 'data_val',
        allowNull: false
      },
      prevHash: {
        comment: 'Block hash of previous block',
        type: Sequelize.STRING,
        validate: {
          is: ['^[a-f0-9]{64}$', 'i']
        },
        field: 'prev_hash',
        allowNull: false,
        unique: true
      },
      hash: {
        comment: 'The block hash, a hex encoded SHA-256 over canonical values',
        type: Sequelize.STRING,
        validate: {
          is: ['^[a-f0-9]{64}$', 'i']
        },
        allowNull: false,
        unique: true
      },
      sig: {
        comment: 'Truncated SHA256 hash of Signing PubKey bytes, colon separated, plus Base64 encoded signature over block hash',
        type: Sequelize.STRING,
        validate: {
          is: ['^[a-zA-Z0-9:=+/]{1,255}$', 'i']
        },
        allowNull: false,
        unique: true
      }
    },
    {
    // No automatic timestamp fields, we add our own 'timestamp' so it is
    // known prior to save so it can be included in the block signature.
      timestamps: false,
      // Disable the modification of table names; By default, sequelize will automatically
      // transform all passed model names (first parameter of define) into plural.
      // if you don't want that, set the following
      freezeTableName: true,
      indexes: [
        {
          unique: true,
          fields: [{ attribute: 'id', order: 'DESC' }, 'hash']
        },
        {
          unique: false,
          fields: ['type', 'data_id']
        },
        {
          unique: false,
          fields: ['type', 'stack_id']
        }
      ]
    }
  )

  return CalendarBlock
}

module.exports = {
  defineFor: defineFor
}
