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

// load all environment variables into env object
const env = require('./lib/parse-env.js')('lnd-mon')
const logger = require('./lib/logger.js')
const lndClient = require('lnrpc-node-client')
const LND_SOCKET = env.LND_SOCKET
const LND_CERTPATH = `/root/.lnd/tls.cert`

async function ensureLndNodeClientWalletUnlockedAsync() {
  lndClient.setTls(LND_SOCKET, LND_CERTPATH)
  let unlocker = lndClient.unlocker()
  try {
    await unlocker.unlockWalletAsync({ wallet_password: env.HOT_WALLET_PASS, recovery_window: 10000 })
    logger.info('Wallet unlocked')
  } catch (error) {
    if (error.code === 12) {
      logger.info('Wallet already unlocked')
      return // already unlocked
    }
    logger.error(`Unable to unlock wallet, retrying in 10 seconds...`)
  }
}

// process all steps need to start the application
async function start() {
  if (env.NODE_ENV === 'test') return
  setInterval(ensureLndNodeClientWalletUnlockedAsync, 10000)
}

// get the whole show started
start()
