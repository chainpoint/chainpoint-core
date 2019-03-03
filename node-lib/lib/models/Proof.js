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

let sequelize
let Proof

// How many hours a proof is retained before pruning
const PROOF_EXPIRE_HOURS = 24

const env = envalid.cleanEnv(process.env, {
  PROOFS_TABLE_NAME: envalid.str({
    default: 'proofs',
    desc: 'Table name for stored proof data'
  })
})

function defineFor(sqlz) {
  let Proof = sqlz.define(
    env.PROOFS_TABLE_NAME,
    {
      hash_id: { type: Sequelize.UUID, primaryKey: true },
      proof: { type: Sequelize.TEXT }
    },
    {
      indexes: [
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

  return Proof
}

async function writeProofsBulkAsync(proofs) {
  let insertCmd = 'INSERT INTO proofs (hash_id, proof, created_at, updated_at) VALUES '

  let insertValues = proofs.map(proof => {
    // use sequelize.escape() to sanitize input values just to be safe
    let hashId = sequelize.escape(proof.hash_id_core)
    let proofString = sequelize.escape(JSON.stringify(proof))
    return `(${hashId}, ${proofString}, clock_timestamp(), clock_timestamp())`
  })

  insertCmd = insertCmd + insertValues.join(', ') + ' ON CONFLICT (hash_id) DO UPDATE SET proof = EXCLUDED.proof'

  await sequelize.query(insertCmd, { type: sequelize.QueryTypes.INSERT })
  return true
}

async function getProofsByHashIdsAsync(hashIds) {
  let results = await Proof.findAll({
    where: {
      hash_id: { [sequelize.Op.in]: hashIds }
    },
    raw: true
  })
  return results
}

async function pruneExpiredProofsAsync() {
  let pruneCutoffDate = new Date(Date.now() - PROOF_EXPIRE_HOURS * 60 * 60 * 1000)
  let deleteCount = await Proof.destroy({
    where: { created_at: { [sequelize.Op.lte]: pruneCutoffDate } }
  })
  return deleteCount
}

module.exports = {
  defineFor: defineFor,
  writeProofsBulkAsync: writeProofsBulkAsync,
  getProofsByHashIdsAsync: getProofsByHashIdsAsync,
  pruneExpiredProofsAsync: pruneExpiredProofsAsync,
  setDatabase: (sqlz, proof) => {
    sequelize = sqlz
    Proof = proof
  }
}
