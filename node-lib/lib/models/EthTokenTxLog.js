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
  COCKROACH_ETH_TNT_TX_LOG_TABLE_NAME: envalid.str({ default: 'chainpoint_eth_tnt_tx_log', desc: 'CockroachDB table name' })
})

function defineFor (sqlz) {
  let EthTokenLog = sqlz.define(env.COCKROACH_ETH_TNT_TX_LOG_TABLE_NAME,
    {
      txId: {
        comment: 'The ethereum transaction id hash.',
        primaryKey: true,
        type: Sequelize.STRING,
        field: 'tx_id',
        allowNull: false
      },
      transactionIndex: {
        comment: 'Integer of the transactions index position log was created from',
        type: Sequelize.INTEGER,
        validate: {
          isInt: true
        },
        field: 'tx_index',
        allowNull: false
      },
      blockNumber: {
        comment: 'Block number where this log was in',
        type: Sequelize.INTEGER,
        validate: {
          isInt: true
        },
        field: 'tx_block',
        allowNull: false
      },
      fromAddress: {
        comment: 'The ethereum address where tokens were transferred from',
        type: Sequelize.STRING,
        field: 'from_address',
        allowNull: false
      },
      toAddress: {
        comment: 'The ethereum address where tokens were transferred to',
        type: Sequelize.STRING,
        field: 'to_address',
        allowNull: false
      },
      amount: {
        comment: 'The amount of TNT tokens sent in the transaction - in base units',
        type: Sequelize.BIGINT,
        field: 'amount',
        allowNull: false
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
          fields: ['to_address', { attribute: 'created_at', order: 'DESC' }]
        }
      ]
    }
  )

  return EthTokenLog
}

module.exports = {
  defineFor: defineFor
}
