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
  COCKROACH_REG_NODE_TABLE_NAME: envalid.str({ default: 'chainpoint_registered_nodes', desc: 'CockroachDB table name' })
})

function defineFor (sqlz) {
  let RegisteredNode = sqlz.define(env.COCKROACH_REG_NODE_TABLE_NAME,
    {
      tntAddr: {
        comment: 'A seemingly valid Ethereum address that the Node will send TNT from, or receive rewards with.',
        type: Sequelize.STRING,
        validate: {
          is: ['^0x[0-9a-f]{40}$']
        },
        field: 'tnt_addr',
        allowNull: false,
        primaryKey: true
      },
      publicUri: {
        comment: 'The public URI address of a Node, when blank represents a non-public Node.',
        type: Sequelize.STRING,
        validate: {
          isUrl: true
        },
        field: 'public_uri',
        allowNull: true
      },
      hmacKey: {
        comment: 'The HMAC secret for this Node. Needed for Node data updates.',
        type: Sequelize.STRING,
        validate: {
          is: ['^[a-f0-9]{64}$', 'i']
        },
        field: 'hmac_key',
        allowNull: false,
        unique: true
      },
      tntCredit: {
        comment: 'The balance of token credit they have against their address.',
        type: Sequelize.DOUBLE,
        field: 'tnt_credit',
        defaultValue: 0
      },
      auditScore: {
        comment: 'The current score for this Node as calculated by the audit processes and used in the reward queue.',
        type: Sequelize.INTEGER,
        field: 'audit_score',
        defaultValue: 0
      },
      passCount: {
        comment: 'The total number of times the Node has passed an audit.',
        type: Sequelize.INTEGER,
        field: 'pass_count',
        defaultValue: 0
      },
      failCount: {
        comment: 'The total number of times the Node has failed an audit.',
        type: Sequelize.INTEGER,
        field: 'fail_count',
        defaultValue: 0
      },
      consecutivePasses: {
        comment: 'The number of consecutive times the Node has passed an audit.',
        type: Sequelize.INTEGER,
        field: 'consecutive_passes',
        defaultValue: 0
      },
      consecutiveFails: {
        comment: 'The number of consecutive times the Node has failed an audit.',
        type: Sequelize.INTEGER,
        field: 'consecutive_fails',
        defaultValue: 0
      },
      createdFromIp: {
        comment: 'The IP origin of the request to register this Node',
        type: Sequelize.STRING,
        field: 'created_from_ip',
        allowNull: true
      },
      verifyE2EPassedAt: {
        comment: 'The time the audit was performed, in MS since EPOCH.',
        type: Sequelize.INTEGER, // is 64 bit in CockroachDB
        validate: {
          isInt: true
        },
        field: 'verify_e2e_passed_at',
        allowNull: true
      },
      verifyE2EFailedAt: {
        comment: 'The time the audit was performed, in MS since EPOCH.',
        type: Sequelize.INTEGER, // is 64 bit in CockroachDB
        validate: {
          isInt: true
        },
        field: 'verify_e2e_failed_at',
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
          fields: ['tnt_credit']
        },
        {
          unique: false,
          fields: ['public_uri', 'tnt_addr', { attribute: 'audit_score', order: 'DESC' }, 'created_at']
        },
        {
          unique: false,
          fields: ['consecutive_passes', 'public_uri']
        },
        {
          unique: false,
          fields: ['created_from_ip', 'created_at']
        }
      ]
    }
  )

  return RegisteredNode
}

module.exports = {
  defineFor: defineFor
}
