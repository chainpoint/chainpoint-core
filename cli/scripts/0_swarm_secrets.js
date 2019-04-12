const exec = require('executive')
const chalk = require('chalk')
var fs = require('fs')
const updateOrCreateEnv = require('./2_update_env')

async function createSwarmAndSecrets(valuePairs) {
    let ip = valuePairs.CORE_PUBLIC_IP_ADDRESS
    let wif = valuePairs.BITCOIN_WIF
    let apiKey = valuePairs.INFURA_API_KEY
    let sed = `sed -i 's#external_address = .*#external_address = "${ip}:26656"#' config/node_1/config.toml`
    try {
        await exec([
            sed, //sed line needs to be first for some reason
            `docker swarm init --advertise-addr=${ip} || echo "Swarm already initialized"`,
            `openssl ecparam -genkey -name secp256r1 -noout -out data/keys/ecdsa_key.pem`,
            `cat data/keys/ecdsa_key.pem | docker secret create ECDSA_KEYPAIR -`,
            `printf ${wif} | docker secret create BITCOIN_WIF -`,
            `printf ${apiKey} | docker secret create ETH_INFURA_API_KEY -`
        ])
        console.log(chalk.yellow('Secrets saved to Docker Secrets'))
    } catch (err) {
        console.log(chalk.red('Setting secrets failed (is docker installed?)'))
    }
    return updateOrCreateEnv({'CHAINPOINT_CORE_BASE_URI': `http://${ip}`})
}
module.exports = createSwarmAndSecrets