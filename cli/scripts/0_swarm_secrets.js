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

const exec = require('executive')
const chalk = require('chalk')
const resolve = require('path').resolve
const generator = require('generate-password')
const lndClient = require('lnrpc-node-client')
const updateOrCreateEnv = require('./2_update_env')
const utils = require(resolve('./node-lib/lib/utils.js'))
const home = require('os').homedir()
const crypto = require('crypto')
const request = require('async-request')

async function numCores(peers) {
  try {
    let arr = peers.split(',')
    let first = arr[0]
    let uriArr = first.split('@')
    let hostportArr = uriArr[1].split(':')
    let host = hostportArr[0]
    let response = await request('http://' + host + ':26657/abci_info')
    let result = JSON.parse(response.body)
    let totalStakePrice = JSON.parse(result.result.response.data).total_stake_price + 1000000
    return totalStakePrice
  } catch (error) {
    console.log(chalk.red(`Could not query peers for total stake amount needed: ${error}`))
  }
  return 0
}

async function createSwarmAndSecrets(valuePairs) {
  let address = { address: valuePairs.HOT_WALLET_ADDRESS }
  let uid = (await exec.quiet('id -u $USER')).stdout.trim()
  let gid = (await exec.quiet('id -g $USER')).stdout.trim()
  let ip = valuePairs.CORE_PUBLIC_IP_ADDRESS
  let network = valuePairs.NETWORK
  let peers
  let lndWalletPass = valuePairs.HOT_WALLET_PASS
  let lndWalletSeed = valuePairs.HOT_WALLET_SEED

  if (valuePairs.PEERS != null) {
    peers = valuePairs.PEERS
  } else if (network == 'testnet') {
    peers = '087186cd1d631c5e709c4afa15a1ce218c6a28c1@3.133.119.65:26656'
  } else if (network == 'mainnet') {
    peers = ''
  }

  let totalStakePrice = await numCores(peers)
  if (totalStakePrice != 0) {
    console.log(
      chalk.green(
        `\nNote: You will need at least ${totalStakePrice} Satoshis (${totalStakePrice /
          100000000} BTC) to join the Chainpoint Network!\n`
      )
    )
  }

  //init swarm and save bitcoin wif
  try {
    await exec.quiet([
      `docker swarm init --advertise-addr=${ip} || echo "Swarm already initialized"`,
      `openssl ecparam -genkey -name secp256r1 -noout -out ${home}/.chainpoint/core/data/keys/ecdsa_key.pem`,
      `cat ${home}/.chainpoint/core/data/keys/ecdsa_key.pem | docker secret create ECDSA_PKPEM -`
    ])
    console.log(chalk.yellow('Secrets saved to Docker Secrets'))
  } catch (err) {
    console.log(chalk.red(`Setting secrets failed (is docker installed?): ${err}`))
  }

  // startup docker compose
  try {
    console.log('initializing LND...')
    await exec.quiet([
      `export NETWORK=${network} && export USERID=${uid} && export GROUPID=${gid} && docker-compose run -d --service-ports lnd`
    ])
    await utils.sleepAsync(30000)
    console.log('LND initialized')
  } catch (err) {
    console.log(chalk.red(`Could not bring up lnd for initialization: ${err}`))
  }

  try {
    lndClient.setTls('127.0.0.1:10009', `${home}/.chainpoint/core/.lnd/tls.cert`)
    let unlocker = lndClient.unlocker()
    if (typeof lndWalletPass !== 'undefined' && typeof lndWalletSeed !== 'undefined') {
      try {
        await unlocker.initWalletAsync({
          wallet_password: lndWalletPass,
          cipher_seed_mnemonic: lndWalletSeed.split(' ')
        })
      } catch (err) {
        console.log(chalk.red(`InitWallet error, likely already initialized: ${err}`))
      }
    } else {
      console.log('Creating a new Lightning wallet...')
      lndWalletPass = generator.generate({
        length: 20,
        numbers: false
      })
      console.log(lndWalletPass)
      console.log('Generating wallet seed...')
      let seed = await unlocker.genSeedAsync({})
      console.log(JSON.stringify(seed))
      let init = await unlocker.initWalletAsync({
        wallet_password: lndWalletPass,
        cipher_seed_mnemonic: seed.cipher_seed_mnemonic
      })
      console.log(`Lightning wallet initialized: ${JSON.stringify(init)}`)
      console.log('Creating bitcoin address for wallet...')
      await utils.sleepAsync(7000)
      lndClient.setCredentials(
        '127.0.0.1:10009',
        `${home}/.chainpoint/core/.lnd/data/chain/bitcoin/${network}/admin.macaroon`,
        `${home}/.chainpoint/core/.lnd/tls.cert`
      )
      let lightning = lndClient.lightning()
      address = await lightning.newAddressAsync({ type: 0 })
      console.log(chalk.green(`\n****************************************************`))
      console.log(chalk.green(`Lightning initialization has completed successfully.`))
      console.log(chalk.green(`****************************************************\n`))
      console.log(chalk.yellow(`\nLightning Wallet Password: ${lndWalletPass}`))
      console.log(chalk.yellow(`\nLightning Wallet Seed: ${seed.cipher_seed_mnemonic.join(' ')}`))
      console.log(chalk.yellow(`\nLightning Wallet Address: ${address.address}\n`))
      console.log(chalk.magenta(`\n******************************************************`))
      console.log(chalk.magenta(`You should back up this information in a secure place.`))
      console.log(chalk.magenta(`******************************************************\n\n`))
      if (totalStakePrice != 0) {
        console.log(
          chalk.green(
            `\nYou will need to fund your Lightning Address with at least ${totalStakePrice} Satoshis (${totalStakePrice /
              100000000} BTC) and wait for 6 confirmations, then run 'make deploy'`
          )
        )
      }
    }
  } catch (err) {
    console.log(chalk.red(`LND setup error: ${err}`))
  }

  //once we know the above password works (whether generated or provided), save it
  if (typeof lndWalletPass !== 'undefined') {
    try {
      await exec.quiet([
        `printf ${lndWalletPass} | docker secret create HOT_WALLET_PASS -`,
        `printf ${address.address} | docker secret create HOT_WALLET_ADDRESS -`
      ])
    } catch (err) {
      console.log(chalk.red(`Could not exec docker secret creation: ${err}`))
    }
  }

  // set session secret to be used for LSAT macaroon signing
  try {
    const secret = crypto.randomBytes(32).toString('hex')
    await exec.quiet([`printf ${secret} | docker secret create SESSION_SECRET -`])
  } catch (err) {
    console.log(chalk.red(`Could not exec docker secret creation: ${err}`))
  }

  try {
    console.log('shutting down LND...')
    await exec.quiet([`docker-compose down && rm ${home}/.chainpoint/core/.lnd/tls.*`])
    console.log('LND shut down')
  } catch (err) {
    console.log(chalk.red(`Could not bring down LND: ${err}`))
  }

  return updateOrCreateEnv({
    HOT_WALLET_ADDRESS: address.address,
    PEERS: peers,
    NETWORK: network,
    CHAINPOINT_CORE_BASE_URI: `http://${ip}`,
    CORE_PUBLIC_IP_ADDRESS: `${ip}`,
    CORE_DATADIR: `${home}/.chainpoint/core`,
    LND_SOCKET: `lnd:10009`
  })
}
module.exports = createSwarmAndSecrets
