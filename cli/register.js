const fs = require('fs')
const path = require('path')
const chalk = require('chalk')
const { pipeP } = require('ramda')
const inquirer = require('inquirer')
const cliHelloLogger = require('./utils/cliHelloLogger')
const stakingQuestions = require('./utils/stakingQuestions')
const updateOrCreateEnv = require('./scripts/2_update_env')

async function main() {
  cliHelloLogger()

  console.log(fs.readFileSync(path.resolve('/run/secrets/ETH_ADDRESS'), 'utf-8'))

  console.log(chalk.bold.yellow('Stake your Core:'))

  try {
    await pipeP(
      () => inquirer.prompt([stakingQuestions['ETH_REWARDS_ADDRESS']]),
      updateOrCreateEnv
      // TODO: /eth/:addr/txdata
      // TODO: /eth/broadcast
    )()

    console.log(chalk.green('\n===================================='))
    console.log(chalk.green('==   SUCCESSFULLY STAKED CORE!    =='))
    console.log(chalk.green('====================================', '\n'))
  } catch (error) {
    console.log(chalk.red('Failed to Stake Core to Chainpoint Network. Please try again.'))
  }
}

main().then(() => {
  process.exit(0)
})
