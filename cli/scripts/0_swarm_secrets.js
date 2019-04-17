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
const updateOrCreateEnv = require('./2_update_env')

async function createSwarmAndSecrets(valuePairs) {
  let ip = valuePairs.CORE_PUBLIC_IP_ADDRESS
  let wif = valuePairs.BITCOIN_WIF
  let infuraApiKey = valuePairs.INFURA_API_KEY
  let etherscanApiKey = valuePairs.ETHERSCAN_API_KEY
  let insightUri = valuePairs.INSIGHT_API_URI
  let sed = `sed -i 's#external_address = .*#external_address = "${ip}:26656"#' config/node_1/config.toml`
  try {
    await exec([
      sed, //sed line needs to be first for some reason
      `docker swarm init --advertise-addr=${ip} || echo "Swarm already initialized"`,
      `openssl ecparam -genkey -name secp256r1 -noout -out data/keys/ecdsa_key.pem`,
      `cat data/keys/ecdsa_key.pem | docker secret create ECDSA_KEYPAIR -`,
      `printf ${wif} | docker secret create BITCOIN_WIF -`,
      `printf ${infuraApiKey} | docker secret create ETH_INFURA_API_KEY -`,
      `printf ${etherscanApiKey} | docker secret create ETH_ETHERSCAN_API_KEY -`
    ])
    console.log(chalk.yellow('Secrets saved to Docker Secrets'))
  } catch (err) {
    console.log(chalk.red('Setting secrets failed (is docker installed?)'))
  }
  return updateOrCreateEnv({ CHAINPOINT_CORE_BASE_URI: `http://${ip}`, INSIGHT_API_BASE_URI: insightUri })
}
module.exports = createSwarmAndSecrets
