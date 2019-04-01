
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

const Sequelize = require('sequelize')

const envalid = require('envalid')

const env = envalid.cleanEnv(process.env, {
    STAKED_NODE_TABLE_NAME: envalid.str({
        default: 'staked_node'
    })
})

function defineFor (sqlz) {
    let StakedNode = sqlz.define(env.STAKED_NODE_TABLE_NAME,
        {
            ethAddr: {
                comment: 'A seemingly valid Ethereum address that the Node will send TNT from, or receive rewards with.',
                type: Sequelize.STRING,
                validate: {
                    is: ['^0x[0-9a-f]{40}$']
                },
                field: 'eth_addr',
                allowNull: false,
                primaryKey: true
            },
            pubKey: {
                comment: 'The public key of a Node.',
                type: Sequelize.STRING,
                validate: {
                    isAlphanumeric: true
                },
                field: 'public_key',
                allowNull: true
            },
            publicIp: {
                comment: 'The public IP address of a Node, when blank represents a non-public Node.',
                type: Sequelize.STRING,
                validate: {
                    isIP: true
                },
                field: 'public_ip',
                allowNull: true
            },
            amountStaked: {
                comment: 'The balance of token credit they have staked against their node.',
                type: Sequelize.BIGINT,
                field: 'amount_staked',
                defaultValue: 0
            },
            stakeExpiration: {
                comment: 'The end of the staking period in unix epoch time',
                type: Sequelize.INTEGER, // is 64 bit in CockroachDB
                validate: {
                    isInt: true
                },
                field: 'stake_expiration',
                allowNull: true
            },
            activeTokenHash: {
                comment: 'The hash of the active expired auth token',
                type: Sequelize.STRING,
                validate: {
                    isAlphanumeric: true
                },
                field: 'active_token_hash',
                allowNull: true
            },
            activeTokenTimestamp: {
                comment: 'The timestamp in unix epoch time of the last auth token',
                type: Sequelize.INTEGER,
                validate: {
                    isInt: true
                },
                field: 'active_token_timestamp',
                allowNull: true
            },
            balance: {
                comment: 'The balance in time units of usage time remaining',
                type: Sequelize.INTEGER,
                validate: {
                    isInt: true
                },
                field: 'balance',
                allowNull: true
            },
            blockNumber: {
                comment: 'The eth block number where this info was valid. Used for versioning node stake updates',
                type: Sequelize.BIGINT, // is 64 bit in CockroachDB
                validate: {
                    isInt: true
                },
                field: 'block_number',
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
                    fields: ['created_at']
                }
            ]
        }
    )

    return StakedNode
}

module.exports = {
    defineFor: defineFor
}