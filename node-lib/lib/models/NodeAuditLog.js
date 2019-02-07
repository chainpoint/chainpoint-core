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
  COCKROACH_AUDIT_TABLE_NAME: envalid.str({ default: 'chainpoint_node_audit_log', desc: 'CockroachDB table name' })
})

function defineFor (sqlz) {
  let NodeAuditLog = sqlz.define(env.COCKROACH_AUDIT_TABLE_NAME,
    {
      tntAddr: {
        comment: 'A seemingly valid Ethereum address that the Node will send TNT from, or receive rewards with.',
        type: Sequelize.STRING,
        validate: {
          is: ['^0x[0-9a-f]{40}$']
        },
        field: 'tnt_addr',
        allowNull: false
      },
      publicUri: {
        comment: 'The public URI of the Node at the time of the audit.',
        type: Sequelize.STRING,
        validate: {
          isUrl: true
        },
        field: 'public_uri',
        allowNull: true
      },
      auditAt: {
        comment: 'The time the audit was performed, in MS since EPOCH.',
        type: Sequelize.INTEGER, // is 64 bit in CockroachDB
        validate: {
          isInt: true
        },
        field: 'audit_at',
        allowNull: false
      },
      publicIPPass: {
        comment: 'Boolean logging if the Node was publicly reachable over HTTP by Core.',
        type: Sequelize.BOOLEAN,
        field: 'public_ip_pass',
        allowNull: false
      },
      nodeMSDelta: {
        comment: 'The number of milliseconds difference between Node time and Core time.',
        type: Sequelize.INTEGER, // is 64 bit in CockroachDB
        validate: {
          isInt: true
        },
        field: 'node_ms_delta',
        allowNull: true
      },
      timePass: {
        comment: 'Boolean logging if the Node reported time was verified to be in tolerance by Core.',
        type: Sequelize.BOOLEAN,
        field: 'time_pass',
        allowNull: false
      },
      calStatePass: {
        comment: 'Boolean logging if the Node Calendar was verified by Core.',
        type: Sequelize.BOOLEAN,
        field: 'cal_state_pass',
        allowNull: false
      },
      minCreditsPass: {
        comment: 'Boolean logging if the Node has the minimum credit balance for reward eligibility.',
        type: Sequelize.BOOLEAN,
        field: 'min_credits_pass',
        allowNull: false
      },
      nodeVersion: {
        comment: 'The reported version of the Node.',
        type: Sequelize.STRING,
        field: 'node_version',
        allowNull: true
      },
      nodeVersionPass: {
        comment: 'Boolean logging if the reported Node version was equal to or above the minimum required version.',
        type: Sequelize.BOOLEAN,
        field: 'node_version_pass',
        allowNull: false
      },
      tntBalanceGrains: {
        comment: 'The TNT balance for this Node at the time of audit in Grains.',
        type: Sequelize.INTEGER,
        field: 'tnt_balance_grains',
        allowNull: true
      },
      tntBalancePass: {
        comment: 'Boolean logging if the TNT balance was sufficient to pass this audit.',
        type: Sequelize.BOOLEAN,
        field: 'tnt_balance_pass',
        allowNull: false
      }
    },
    {
    // No automatic timestamp fields, we add our own 'audit_at'
      timestamps: false,
      // Disable the modification of table names; By default, sequelize will automatically
      // transform all passed model names (first parameter of define) into plural.
      // if you don't want that, set the following
      freezeTableName: true,
      indexes: [
        {
          unique: false,
          fields: ['tnt_addr']
        },
        {
          unique: false,
          fields: ['audit_at']
        }
      ]
    }
  )

  return NodeAuditLog
}

module.exports = {
  defineFor: defineFor
}
