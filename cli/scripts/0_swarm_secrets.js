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
const keccak = require('keccak')
const secp256k1 = require('secp256k1')
const createWallet = require('./1_create_wallet')
const updateOrCreateEnv = require('./2_update_env')
const displayInfo = require('./1b_display_info')

async function createSwarmAndSecrets(valuePairs) {
  let home = await exec.quiet('/bin/bash -c "$(eval printf ~$USER)"')
  let ip = valuePairs.CORE_PUBLIC_IP_ADDRESS
  let wif = valuePairs.BITCOIN_WIF
  let insightUri = valuePairs.INSIGHT_API_URI
  let privateNetwork = valuePairs.PRIVATE_NETWORK

  let sed = `sed -i 's#external_address = .*#external_address = "${ip}:26656"#' ${
    home.stdout
  }/.chainpoint/core/config/node_1/config.toml`

  try {
    await exec([
      sed, //sed line needs to be first for some reason
      `docker swarm init --advertise-addr=${ip} || echo "Swarm already initialized"`,
      `openssl ecparam -genkey -name secp256r1 -noout -out ${home.stdout}/.chainpoint/core/data/keys/ecdsa_key.pem`,
      `cat ${home.stdout}/.chainpoint/core/data/keys/ecdsa_key.pem | docker secret create ECDSA_PKPEM -`,
      `printf ${wif} | docker secret create BITCOIN_WIF -`
    ])
    console.log(chalk.yellow('Secrets saved to Docker Secrets'))
  } catch (err) {
    console.log(chalk.red('Setting secrets failed (is docker installed?)'))
  }

  if (!privateNetwork) {
    try {
      let privateKey
      if (!('ETH_PRIVATE_KEY' in valuePairs)) {
        privateKey = (await createWallet()).privateKey
      } else {
        privateKey = valuePairs.ETH_PRIVATE_KEY
      }
      let infuraApiKey = valuePairs.INFURA_API_KEY
      let etherscanApiKey = valuePairs.ETHERSCAN_API_KEY
      /* Derive address from private key */
      let privateKeyBytes = new Buffer(privateKey.slice(2), 'hex')
      let pubKey = secp256k1.publicKeyCreate(privateKeyBytes, false).slice(1)
      let address =
        '0x' +
        keccak('keccak256')
          .update(pubKey)
          .digest()
          .slice(-20)
          .toString('hex')
      await exec.quiet([
        `printf ${address} | docker secret create ETH_ADDRESS -`,
        `printf ${privateKey} | docker secret create ETH_PRIVATE_KEY -`,
        `printf ${infuraApiKey} | docker secret create ETH_INFURA_API_KEY -`,
        `printf ${etherscanApiKey} | docker secret create ETH_ETHERSCAN_API_KEY -`
      ])
      await displayInfo.displayWalletInfo({ address: address, privateKey: privateKey })
    } catch (err) {
      console.log(chalk.red(`Error creating Docker secrets for ETH_ADDRESS & ETH_PRIVATE_KEY: ${err}`))
    }
  }

  return updateOrCreateEnv({
    CHAINPOINT_CORE_BASE_URI: `http://${ip}`,
    INSIGHT_API_BASE_URI: insightUri,
    CORE_DATADIR: `${home.stdout}/.chainpoint/core`,
    PRIVATE_NETWORK: privateNetwork
  })
}
module.exports = createSwarmAndSecrets
