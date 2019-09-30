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

async function createSwarmAndSecrets(valuePairs) {
  let address = { value: { address: valuePairs.HOT_WALLET_ADDRESS } }
  let uid = (await exec.quiet('id -u $USER')).stdout.trim()
  let gid = (await exec.quiet('id -g $USER')).stdout.trim()
  let ip = valuePairs.CORE_PUBLIC_IP_ADDRESS
  let network = valuePairs.NETWORK
  let peers = valuePairs.PEERS != null ? valuePairs.PEERS : ''
  let lndWalletPass = valuePairs.HOT_WALLET_PASS
  let lndWalletSeed = valuePairs.HOT_WALLET_SEED

  //init swarm and save bitcoin wif
  try {
    await exec([
      `docker swarm init --advertise-addr=${ip} || echo "Swarm already initialized"`,
      `openssl ecparam -genkey -name secp256r1 -noout -out ${home}/.chainpoint/core/data/keys/ecdsa_key.pem`,
      `cat ${home}/.chainpoint/core/data/keys/ecdsa_key.pem | docker secret create ECDSA_PKPEM -`
    ])
    console.log(chalk.yellow('Secrets saved to Docker Secrets'))
  } catch (err) {
    console.log(chalk.red('Setting secrets failed (is docker installed?)'))
  }

  // startup docker compose
  try {
    console.log('initializing LND...')
    await exec([`export USERID=${uid} && export GROUPID=${gid} && docker-compose run -d --service-ports lnd`])
    await utils.sleepAsync(5000)
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
      console.log('Creating a new LND wallet...')
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
        cipher_seed_mnemonic: seed.value.cipher_seed_mnemonic
      })
      console.log(`LND wallet initialized: ${JSON.stringify(init)}`)
      console.log('Creating bitcoin address for wallet...')
      await utils.sleepAsync(7000)
      lndClient.setCredentials(
        '127.0.0.1:10009',
        `${home}/.chainpoint/core/.lnd/data/chain/bitcoin/${network}/admin.macaroon`,
        `${home}/.chainpoint/core/.lnd/tls.cert`
      )
      let client = lndClient.lightning()
      address = await client.newAddressAsync({ type: 0 })
      console.log(address)
      console.log(chalk.yellow(`\nLND Wallet Password: ${lndWalletPass}`))
      console.log(chalk.yellow(`\nLND Wallet Seed: ${seed.value.cipher_seed_mnemonic.join(' ')}`))
      console.log(chalk.yellow(`\nLND Wallet Address: ${address.value.address}\n`))
    }
  } catch (err) {
    console.log(chalk.red(`LND setup error: ${err}`))
  }

  //once we know the above password works (whether generated or provided), save it
  if (typeof lndWalletPass !== 'undefined') {
    try {
      await exec.quiet([
        `printf ${lndWalletPass} | docker secret create HOT_WALLET_PASS -`,
        `printf ${address.value.address} | docker secret create HOT_WALLET_ADDRESS -`
      ])
    } catch (err) {
      console.log(chalk.red(`Could not exec docker secret creation: ${err}`))
      return
    }
  }

  try {
    console.log('shutting down LND...')
    await exec([`docker-compose down && rm ${home}/chainpoint-core/.lnd/tls.*`])
    console.log('LND shut down')
  } catch (err) {
    console.log(chalk.red(`Could not bring down LND: ${err}`))
  }

  return updateOrCreateEnv({
    HOT_WALLET_ADDRESS: address.value.address,
    PEERS: peers,
    NETWORK: network,
    CHAINPOINT_CORE_BASE_URI: `http://${ip}`,
    CORE_PUBLIC_IP_ADDRESS: `${ip}`,
    CORE_DATADIR: `${home}/.chainpoint/core`
  })
}
module.exports = createSwarmAndSecrets
