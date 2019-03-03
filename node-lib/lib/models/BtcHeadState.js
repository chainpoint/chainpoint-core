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
  BTC_HEAD_STATE_TABLE_NAME: envalid.str({
    default: 'btchead_states'
  })
})

function defineFor(sqlz) {
  let BtcHeadState = sqlz.define(
    env.BTC_HEAD_STATE_TABLE_NAME,
    {
      btctx_id: { type: Sequelize.STRING, primaryKey: true },
      btchead_height: { type: Sequelize.INTEGER },
      btchead_state: { type: Sequelize.TEXT }
    },
    {
      indexes: [
        {
          unique: false,
          fields: ['btchead_height']
        },
        {
          unique: false,
          fields: ['created_at']
        }
      ],
      // enable timestamps
      timestamps: true,
      // don't use camelcase for automatically added attributes but underscore style
      // so updatedAt will be updated_at
      underscored: true
    }
  )

  return BtcHeadState
}

module.exports = {
  defineFor: defineFor
}
