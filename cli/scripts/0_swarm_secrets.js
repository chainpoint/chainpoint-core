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
const lnService = require('ln-service')
const updateOrCreateEnv = require('./2_update_env')
const utils = require(resolve('./node-lib/lib/utils.js'))
const homedir = require('os').homedir()

async function createSwarmAndSecrets(valuePairs) {
  let home = await exec.quiet('/bin/bash -c "$(eval printf ~$USER)"')
  let uid = (await exec.quiet('id -u $USER')).stdout.trim()
  let gid = (await exec.quiet('id -g $USER')).stdout.trim()
  console.log(uid)
  console.log(gid)
  let btcRpc = valuePairs.BTC_RPC_URI_LIST
  let ip = valuePairs.CORE_PUBLIC_IP_ADDRESS
  let wif = valuePairs.BITCOIN_WIF
  let network = valuePairs.NETWORK
  let peers = valuePairs.PEERS != null ? valuePairs.PEERS : ''
  let blockCypher = valuePairs.BLOCKCYPHER_API_TOKEN != null ? valuePairs.BLOCKCYPHER_API_TOKEN : ''
  let lndWalletPass = valuePairs.HOT_WALLET_PASS
  let lndWalletSeed = valuePairs.HOT_WALLET_SEED

  let sed = `sed -i 's#external_address = .*#external_address = "${ip}:26656"#' ${
    home.stdout
  }/.chainpoint/core/config/node_1/config.toml`

  try {
    await exec([
      sed, //sed line needs to be first for some reason
      `docker swarm init --advertise-addr=${ip} || echo "Swarm already initialized"`,
      `printf ${wif} | docker secret create BITCOIN_WIF -`
    ])
    console.log(chalk.yellow('Secrets saved to Docker Secrets'))
  } catch (err) {
    console.log(chalk.red('Setting secrets failed (is docker installed?)'))
  }

  try {
    console.log('initializing LND...')
    await exec([`export USERID=${uid} && export GROUPID=${gid} && docker-compose run -d lnd`])
    await utils.sleep(5000)
    console.log('LND initialized')
  } catch (err) {
    console.log(chalk.red(`Could not bring up lnd for initialization: ${err}`))
    return
  }
  let tlsCert = utils.toBase64(`${homedir}/.lnd/tls.cert`)
  console.log(tlsCert)
  let { lnd } = lnService.unauthenticatedLndGrpc({
    cert: tlsCert,
    socket: '127.0.0.1:10009'
  })

  try {
    let create
    if (typeof lndWalletPass !== 'undefined' && typeof lndWalletSeed !== 'undefined') {
      create = await lnService.createWallet({ lnd, lndWalletSeed, password: lndWalletPass })
    } else {
      lndWalletPass = generator.generate({
        length: 20,
        numbers: true
      })
      const { seed } = await lnService.createSeed({ lnd })
      create = await lnService.createWallet({ lnd, seed, password: lndWalletPass })
      console.log(chalk.green(`LND Wallet Password: ${lndWalletPass}`))
      console.log(chalk.green(`LND Wallet Seed: ${seed}`))
    }
    console.log(create)
    let unlock = await lnService.unlockWallet({ lnd, password: lndWalletPass })
    console.log(unlock)
  } catch (err) {
    console.log(chalk.red(`Could not unlock the lnd wallet: ${err}`))
    return
  }

  return updateOrCreateEnv({
    BTC_RPC_URI_LIST: btcRpc,
    BLOCKCYPHER_API_TOKEN: blockCypher,
    PEERS: peers,
    NETWORK: network,
    CHAINPOINT_CORE_BASE_URI: `http://${ip}`,
    CORE_DATADIR: `${home.stdout}/.chainpoint/core`
  })
}
module.exports = createSwarmAndSecrets
