
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
            publicIp: {
                comment: 'The public IP address of a Node, when blank represents a non-public Node.',
                type: Sequelize.STRING,
                validate: {
                    isIP: true
                },
                field: 'public_ip',
                allowNull: true
            },
            pubKey: {
                comment: 'The public key for this Node.',
                type: Sequelize.STRING,
                validate: {
                    is: ['^[a-f0-9]{64}$', 'i']
                },
                field: 'public_key',
                allowNull: false,
                unique: true
            },
            amountStaked: {
                comment: 'The balance of token credit they have staked against their node.',
                type: Sequelize.DOUBLE,
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
            lastTokenHash: {
                comment: 'The hash of the last expired auth token',
                type: Sequelize.STRING,
                validate: {
                    isAlphanumeric: true
                },
                field: 'last_token_hash',
                allowNull: true
            },
            lastTokenTimestamp: {
                comment: 'The timestamp in unix epoch time of the last auth token',
                type: Sequelize.INTEGER,
                validate: {
                    isInt: true
                },
                field: 'last_token_timestamp',
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
            createdAt: {
                comment: 'The unix epoch time of creation in seconds',
                type: Sequelize.INTEGER,
                validate: {
                    isInt: true
                },
                field: 'created_at',
                allowNull: true
            },
            updatedAt: {
                comment: 'The unix epoch time of an update in seconds',
                type: Sequelize.INTEGER,
                validate: {
                    isInt: true
                },
                field: 'updated_at',
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