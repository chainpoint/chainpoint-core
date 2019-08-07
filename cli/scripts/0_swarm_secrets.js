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
  let home = await exec.quiet('/bin/bash -c "$(eval printf ~$USER)"')
  let btcRpc = valuePairs.BTC_RPC_URI_LIST
  let ip = valuePairs.CORE_PUBLIC_IP_ADDRESS
  let wif = valuePairs.BITCOIN_WIF
  let privateNetwork = valuePairs.PRIVATE_NETWORK
  let network = valuePairs.NETWORK
  let peers = valuePairs.PEERS != null ? valuePairs.PEERS : ''
  let privateNodes = valuePairs.PRIVATE_NODE_IPS != null ? valuePairs.PRIVATE_NODE_IPS : ''
  let blockCypher = valuePairs.BLOCKCYPHER_API_TOKEN != null ? valuePairs.BLOCKCYPHER_API_TOKEN : ''

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

  return updateOrCreateEnv({
    BTC_RPC_URI_LIST: btcRpc,
    BLOCKCYPHER_API_TOKEN: blockCypher,
    PRIVATE_NODE_IPS: privateNodes,
    PEERS: peers,
    NETWORK: network,
    CHAINPOINT_CORE_BASE_URI: `http://${ip}`,
    CORE_DATADIR: `${home.stdout}/.chainpoint/core`,
    PRIVATE_NETWORK: privateNetwork
  })
}
module.exports = createSwarmAndSecrets
