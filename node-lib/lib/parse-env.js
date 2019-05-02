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

const envalid = require('envalid')

const validateMaxHashes = envalid.makeValidator(x => {
  if (x >= 100 && x <= 25000) return x
  else throw new Error('Value must be between 100 and 25000, inclusive')
})
const validateMonitorRange = envalid.makeValidator(x => {
  if (x >= 10 && x <= 600) return x
  else throw new Error('Value must be between 10 and 600, inclusive')
})
const validateMinConfirmRange = envalid.makeValidator(x => {
  if (x >= 1 && x <= 16) return x
  else throw new Error('Value must be between 1 and 16, inclusive')
})
const valETHAddress = envalid.makeValidator(addr => {
  if (!/^0x[0-9a-fA-F]{40}$/i.test(addr)) throw new Error('The Ethereum (TNT) address is invalid')
  return addr.toLowerCase()
})

let envDefinitions = {
  // The following variables are exposed by this stack's /status endpoint
  //
  // CHAINPOINT_CORE_BASE_URI: Base URI for this Chainpoint Core stack of services

  // ***********************************************************************
  // * Global variables with default values
  // ***********************************************************************

  // Chainpoint stack related variables
  NODE_ENV: envalid.str({ default: 'production', desc: 'The type of environment in which the service is running' }),

  // RabbitMQ related variables
  RABBITMQ_CONNECT_URI: envalid.url({
    default: 'amqp://chainpoint:chainpoint@rabbitmq',
    desc: 'Connection string w/ credentials for RabbitMQ'
  }),
  RMQ_WORK_OUT_CAL_QUEUE: envalid.str({
    default: 'work.cal',
    desc: 'The queue name for outgoing message to the calendar service'
  }),
  RMQ_WORK_OUT_AGG_QUEUE: envalid.str({
    default: 'work.agg',
    desc: 'The queue name for outgoing message to the aggregator service'
  }),
  RMQ_WORK_OUT_GEN_QUEUE: envalid.str({
    default: 'work.gen',
    desc: 'The queue name for outgoing message to the proof gen service'
  }),

  // Redis related variables
  REDIS_CONNECT_URIS: envalid.str({
    default: 'redis://redis:6379',
    desc: 'The Redis server connection URI, or a comma separated list of Sentinel URIs'
  }),

  // Service Specific Variables

  // Aggregator service specific variables
  RMQ_WORK_IN_AGG_QUEUE: envalid.str({
    default: 'work.agg',
    desc: 'The queue name for message consumption originating from the api service'
  }),
  HASHES_PER_MERKLE_TREE: validateMaxHashes({
    default: 25000,
    desc: 'The maximum number of hashes to be used when constructing an aggregation tree'
  }),

  // API service specific variables
  MAX_BODY_SIZE: envalid.num({ default: 131072, desc: 'Max body size in bytes for incoming requests' }),

  // BTC Mon service specific variables
  RMQ_PREFETCH_COUNT_BTCMON: envalid.num({
    default: 0,
    desc: 'The maximum number of messages sent over the channel that can be awaiting acknowledgement, 0 = no limit'
  }),
  RMQ_WORK_IN_BTCMON_QUEUE: envalid.str({
    default: 'work.btcmon',
    desc: 'The queue name for message consumption originating from the calendar service'
  }),
  MONITOR_INTERVAL_SECONDS: validateMonitorRange({
    default: 30,
    desc: 'The frequency that transactions are monitored for new confirmations, in seconds'
  }),
  MIN_BTC_CONFIRMS: validateMinConfirmRange({
    default: 6,
    desc: 'The number of confirmations needed before the transaction is considered ready for proof delivery'
  }),

  // BTC Tx service specific variables
  RMQ_PREFETCH_COUNT_BTCTX: envalid.num({
    default: 0,
    desc: 'The maximum number of messages sent over the channel that can be awaiting acknowledgement, 0 = no limit'
  }),
  RMQ_WORK_IN_BTCTX_QUEUE: envalid.str({
    default: 'work.btctx',
    desc: 'The queue name for message consumption originating from the calendar service'
  }),
  // This is to safeguard against the service returning a very high value in error
  // and to impose a common sense limit on the highest fee per byte to allow.
  // MAX BTC to spend = AverageTxSizeBytes * BTC_MAX_FEE_SAT_PER_BYTE / 100000000
  // If we are to limit the maximum fee per transaction to 0.0015 BTC, then
  // 0.0015 = 235 * BTC_MAX_FEE_SAT_PER_BYTE / 100000000
  // BTC_MAX_FEE_SAT_PER_BYTE = 0.0015 * 100000000 / 235
  // BTC_MAX_FEE_SAT_PER_BYTE = 635
  BTC_MAX_FEE_SAT_PER_BYTE: envalid.num({ default: 600, desc: 'The maximum feeRateSatPerByte value accepted' }),

  // Proof Gen service specific variables
  RMQ_PREFETCH_COUNT_GEN: envalid.num({
    default: 1,
    desc: 'The maximum number of messages sent over the channel that can be awaiting acknowledgement, 0 = no limit'
  }),
  RMQ_WORK_IN_GEN_QUEUE: envalid.str({
    default: 'work.gen',
    desc: 'The queue name for message consumption originating from the proof state service'
  }),

  // Proof State / Gen service specific variables
  RMQ_PREFETCH_COUNT_STATE: envalid.num({
    default: 0,
    desc: 'The maximum number of messages sent over the channel that can be awaiting acknowledgement, 0 = no limit'
  }),
  RMQ_WORK_IN_STATE_QUEUE: envalid.str({
    default: 'work.proofstate',
    desc: 'The queue name for message consumption originating from the aggregator, calendar, and proof state services'
  }),
  PRUNE_FREQUENCY_MINUTES: envalid.num({
    default: 1,
    desc: 'The frequency that the proof state and proof tables have their expired data pruned, in minutes'
  }),

  // Tendermint RPC URI
  TENDERMINT_URI: envalid.str({ default: 'http://abci:26657', desc: 'Tendermint RPC URI' })
}

module.exports = service => {
  // Load and validate service specific require variables as needed
  switch (service) {
    case 'api':
      envDefinitions.ETH_PRIVATE_KEY = envalid.str({
        desc: `The private key for this Node's Ethereum wallet`
      })
      envDefinitions.CHAINPOINT_CORE_BASE_URI = envalid.url({
        desc: 'Base URI for this Chainpoint Core stack of services'
      })
      envDefinitions.ETH_TNT_LISTEN_ADDR = valETHAddress({
        default: '0x5702ac6389aa79dedea2b9e816a14a19dd11923f',
        desc: 'The address used to listen for incoming TNT transfers.'
      })
      envDefinitions.ETH_INFURA_API_KEY = envalid.str({ desc: 'Infura API Key' })
      envDefinitions.ETH_ETHERSCAN_API_KEY = envalid.str({ desc: 'Etherscan API Key' })
      envDefinitions.ECDSA_PKPEM = envalid.str({ desc: 'ECDSA private key in PEM format' })
      break
    case 'btc-mon':
      envDefinitions.INSIGHT_API_BASE_URI = envalid.url({ desc: 'The Bitcore Insight-API base URI' })
      break
    case 'btc-tx':
      envDefinitions.CHAINPOINT_CORE_BASE_URI = envalid.url({
        desc: 'Base URI for this Chainpoint Core stack of services'
      })
      envDefinitions.INSIGHT_API_BASE_URI = envalid.url({ desc: 'The Bitcore Insight-API base URI' })
      envDefinitions.BITCOIN_WIF = envalid.str({ desc: 'The Bitcoin private key WIF used for transaction creation' })
      break
    case 'state':
      envDefinitions.CHAINPOINT_CORE_BASE_URI = envalid.url({
        desc: 'Base URI for this Chainpoint Core stack of services'
      })
      break
  }
  return envalid.cleanEnv(process.env, envDefinitions, {
    strict: true
  })
}
