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

let rpcClient = null

function setRpcClient(rpc) {
  rpcClient = rpc
}

function parseRpcError(error) {
  switch (error.code) {
    case -32603: // check for not found
      return {
        result: null,
        error: { responseCode: 404, message: 'Not Found' }
      }
    case -32602: // check for invalid parameters
      return {
        result: null,
        error: { responseCode: 409, message: error.data }
      }
    default:
      return {
        result: null,
        error: { responseCode: 500, message: error.data }
      }
  }
}

async function getTransactionAsync(txID) {
  if (!txID.startsWith('0x')) txID = `0x${txID}`
  let tx
  try {
    tx = await rpcClient.tx({ hash: txID, prove: false })
    // Txs are double-encoded in base64 when returned from this RPC client
    tx.tx = JSON.parse(new Buffer(new Buffer(tx.tx, 'base64').toString('ascii'), 'base64').toString('ascii'))
  } catch (error) {
    return parseRpcError(error)
  }
  return { result: tx, error: null }
}

async function getStatusAsync() {
  let status
  try {
    status = await rpcClient.status({})
  } catch (error) {
    return parseRpcError(error)
  }
  return { result: status, error: null }
}

async function getNetInfoAsync() {
  let netInfo
  try {
    netInfo = await rpcClient.netInfo({})
  } catch (error) {
    return parseRpcError(error)
  }
  return { result: netInfo, error: null }
}

module.exports = {
  setRpcClient: setRpcClient,
  getTransactionAsync: getTransactionAsync,
  getNetInfoAsync: getNetInfoAsync,
  getStatusAsync: getStatusAsync
}
