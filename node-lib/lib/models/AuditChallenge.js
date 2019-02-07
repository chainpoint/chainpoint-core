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
  COCKROACH_AUDIT_CHALLENGE_TABLE_NAME: envalid.str({ default: 'chainpoint_audit_challenges', desc: 'CockroachDB table name' })
})

function defineFor (sqlz) {
  let AuditChallenge = sqlz.define(env.COCKROACH_AUDIT_CHALLENGE_TABLE_NAME,
    {
      time: {
        comment: 'Audit time in milliseconds since unix epoch',
        primaryKey: true,
        type: Sequelize.INTEGER, // is 64 bit in CockroachDB
        validate: {
          isInt: true
        },
        field: 'time',
        allowNull: false
      },
      minBlock: {
        comment: 'The minimum block height included in the challenge calculation',
        type: Sequelize.INTEGER, // is 64 bit in CockroachDB
        validate: {
          isInt: true
        },
        field: 'min_block',
        allowNull: false
      },
      maxBlock: {
        comment: 'The maximum block height included in the challenge calculation',
        type: Sequelize.INTEGER, // is 64 bit in CockroachDB
        validate: {
          isInt: true
        },
        field: 'max_block',
        allowNull: false
      },
      nonce: {
        comment: 'The random nonce hex string included in the challenge calculation',
        type: Sequelize.TEXT,
        validate: {
          is: ['^([a-f0-9]{2})+$', 'i']
        },
        field: 'nonce',
        allowNull: false
      },
      solution: {
        comment: 'The solution for this challenge calculation',
        type: Sequelize.TEXT,
        validate: {
          is: ['^([a-f0-9]{2})+$', 'i']
        },
        field: 'solution',
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
      freezeTableName: true,
      indexes: [
        {
          unique: false,
          fields: [{ attribute: 'time', order: 'DESC' }]
        }
      ]
    }
  )

  return AuditChallenge
}

module.exports = {
  defineFor: defineFor
}
