const chalk = require('chalk')
const inquirer = require('inquirer')
const { pipe, pipeP } = require('ramda')
const ora = require('ora')
const tap = require('./utils/tap')
const createSwarmAndSecrets = require('./scripts/0_swarm_secrets')
const cliHelloLogger = require('./utils/cliHelloLogger')
const createWallet = require('./scripts/1_create_wallet')
const createDockerSecrets = require('./scripts/1a_wallet_docker_secrets')
const displayWalletInfo = require('./scripts/1b_display_info')
const updateOrCreateEnv = require('./scripts/2_update_env')
const stakingQuestions = require('./utils/stakingQuestions')

const resolve = Promise.resolve.bind(Promise)

async function main() {
  cliHelloLogger()

  await pipeP(() =>
    inquirer.prompt([
      stakingQuestions['CORE_PUBLIC_IP_ADDRESS'],
      stakingQuestions['INSIGHT_API_URI'],
      stakingQuestions['BITCOIN_WIF'],
      stakingQuestions['INFURA_API_KEY'],
      stakingQuestions['ETHERSCAN_API_KEY']
      
    ]),
    createSwarmAndSecrets,
    createWallet,
    createDockerSecrets,
    tap(
      pipe(
        currVal => ({
          ETH_ADDRESS: currVal.address
        }),
        updateOrCreateEnv
      ),
      resolve
    ),
    //tap(spinner.succeed.bind(spinner, chalk.bold.yellow('New Wallet:\n')), resolve),
    displayWalletInfo
  )()
}

main().then(() => {
    process.exit(0)
})