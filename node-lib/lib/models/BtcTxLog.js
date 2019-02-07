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
  COCKROACH_BTC_TX_LOG_TABLE_NAME: envalid.str({ default: 'chainpoint_btc_tx_log', desc: 'CockroachDB table name' })
})

function defineFor (sqlz) {
  let BtxTxLog = sqlz.define(env.COCKROACH_BTC_TX_LOG_TABLE_NAME,
    {
      txId: {
        comment: 'The bitcoin transaction id hash.',
        primaryKey: true,
        type: Sequelize.STRING,
        validate: {
          is: ['^[a-fA-F0-9:]{1,255}$', 'i']
        },
        field: 'tx_id',
        allowNull: false
      },
      publishDate: {
        comment: 'Transaction publish time in milliseconds since unix epoch',
        type: Sequelize.INTEGER, // is 64 bit in CockroachDB
        validate: {
          isInt: true
        },
        field: 'publish_date',
        allowNull: false,
        unique: true
      },
      rawTx: {
        comment: 'The raw transaction body hex',
        type: Sequelize.TEXT,
        validate: {
          is: ['^([a-f0-9]{2})+$', 'i']
        },
        field: 'raw_tx',
        allowNull: false
      },
      feeSatoshiPerByte: {
        comment: 'The fee expressed in Satoshi per byte',
        type: Sequelize.INTEGER,
        validate: {
          isInt: true
        },
        field: 'fee_satoshi_per_byte',
        allowNull: false
      },
      feePaidSatoshi: {
        comment: 'The final fee paid for this transaction expressed in Satoshi',
        type: Sequelize.INTEGER,
        validate: {
          isInt: true
        },
        field: 'fee_paid_satoshi',
        allowNull: false
      },
      stackId: {
        comment: 'The unique identifier for the stack in which this service runs',
        type: Sequelize.STRING,
        field: 'stack_id',
        allowNull: false
      }
    },
    {
    // No automatic timestamp fields, we add our own 'timestamp' so it is
    // known prior to save so it can be included in the block signature.
      timestamps: false,
      // Disable the modification of table names; By default, sequelize will automatically
      // transform all passed model names (first parameter of define) into plural.
      // if you don't want that, set the following
      freezeTableName: true
    }
  )

  return BtxTxLog
}

module.exports = {
  defineFor: defineFor
}
