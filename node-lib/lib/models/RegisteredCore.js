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
  COCKROACH_REG_CORE_TABLE_NAME: envalid.str({ default: 'chainpoint_registered_cores', desc: 'CockroachDB table name' })
})

function defineFor (sqlz) {
  let RegisteredCore = sqlz.define(env.COCKROACH_REG_CORE_TABLE_NAME,
    {
      stackId: {
        comment: 'The Chainpoint stack identifier. Should be the domain or IP of the API. e.g. a.chainpoint.org',
        type: Sequelize.STRING,
        field: 'stack_id',
        allowNull: false,
        primaryKey: true
      },
      tntAddr: {
        comment: 'A seemingly valid Ethereum address that the Core may receive Core rewards with.',
        type: Sequelize.STRING,
        validate: {
          is: ['^0x[0-9a-f]{40}$']
        },
        field: 'tnt_addr',
        allowNull: true
      },
      rewardEligible: {
        comment: 'Boolean indicating if the Core is eligible to receive Core rewards.',
        type: Sequelize.BOOLEAN,
        field: 'reward_eligible',
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
      underscored: true
    }
  )

  return RegisteredCore
}

module.exports = {
  defineFor: defineFor
}
