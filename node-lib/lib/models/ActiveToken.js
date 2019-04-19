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

// let sequelize
let ActiveToken

const env = envalid.cleanEnv(process.env, {
  ACTIVE_TOKEN_TABLE_NAME: envalid.str({
    default: 'active_tokens'
  })
})

function defineFor(sqlz) {
  let ActiveToken = sqlz.define(
    env.ACTIVE_TOKEN_TABLE_NAME,
    {
      nodeIp: {
        comment: 'The registered IP address of a Node',
        type: Sequelize.STRING,
        validate: {
          isIP: true
        },
        field: 'node_ip',
        allowNull: false,
        primaryKey: true
      },
      tokenHash: {
        comment: 'The SHA-256 hash of the active token',
        type: Sequelize.STRING,
        validate: {
          is: ['^[0-9a-f]{64}$']
        },
        field: 'token_hash',
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
          fields: ['updated_at']
        }
      ]
    }
  )

  return ActiveToken
}

async function getActiveTokenByNodeIPAsync(ip) {
  let results = await ActiveToken.findOne({ where: { node_ip: ip }, raw: true })
  return results
}

module.exports = {
  defineFor: defineFor,
  getActiveTokenByNodeIPAsync: getActiveTokenByNodeIPAsync,
  setDatabase: (sqlz, activeToken) => {
    // sequelize = sqlz
    ActiveToken = activeToken
  }
}
