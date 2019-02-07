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
  COCKROACH_AGG_STATE_TABLE_NAME: envalid.str({ default: 'chainpoint_proof_agg_states', desc: 'CockroachDB table name' })
})

function defineFor (sqlz) {
  let AggState = sqlz.define(env.COCKROACH_AGG_STATE_TABLE_NAME, {
    hash_id: { type: Sequelize.UUID, primaryKey: true },
    hash: { type: Sequelize.STRING },
    agg_id: { type: Sequelize.UUID },
    agg_state: { type: Sequelize.TEXT },
    agg_root: { type: Sequelize.STRING, allowNull: true }
  }, {
    indexes: [
      {
        unique: false,
        fields: ['agg_id']
      },
      {
        unique: false,
        fields: ['created_at', 'agg_id', 'agg_root']
      }
    ],
    // enable timestamps
    timestamps: true,
    // don't use camelcase for automatically added attributes but underscore style
    // so updatedAt will be updated_at
    underscored: true
  })

  return AggState
}

module.exports = {
  defineFor: defineFor
}
