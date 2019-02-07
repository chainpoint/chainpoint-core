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
  COCKROACH_E2E_AUDIT_TABLE_NAME: envalid.str({ default: 'chainpoint_node_e2e_audit_log', desc: 'CockroachDB table name' })
})

function defineFor (sqlz) {
  let E2ENodeAuditLog = sqlz.define(env.COCKROACH_E2E_AUDIT_TABLE_NAME,
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
      auditDate: {
        comment: 'The time the audit was performed, (yyyymmdd).',
        type: Sequelize.DATE,
        field: 'audit_date',
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
      stage: {
        comment: 'Enum-like field with the following possible values ("hash_submission", "proof_retrieval", "proof_verification").',
        type: Sequelize.STRING,
        validate: {
          is: ['(hash_submission|proof_retrieval|proof_verification)']
        },
        field: 'stage',
        allowNull: false
      },
      status: {
        comment: 'Enum-like field with the following possible values ("pending", "passed", "submission_failure", "hash_mismatch_failure", "hash_id_node_validation_failure", "null_proof_failure", "invalid_cal_branch_failure").',
        type: Sequelize.STRING,
        validate: {
          is: ['(pending|passed|submission_failure|retrieval_failure|verification_failure|hash_mismatch_failure|hash_id_node_validation_failure|null_proof_failure|invalid_cal_branch_failure)']
        },
        field: 'status',
        allowNull: false
      },
      auditAt: {
        comment: 'The time that a "hash_submission" status was captured, in MS since EPOCH.',
        type: Sequelize.INTEGER,
        field: 'audit_at',
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
          fields: ['audit_date']
        },
        {
          unique: false,
          fields: ['stage']
        }
      ]
    }
  )

  return E2ENodeAuditLog
}

module.exports = {
  defineFor: defineFor
}
