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

const errors = require('restify-errors')
let tmRpc = require('../tendermint-rpc.js')
let stakedNode = require('../models/StakedNode.js')
let env = require('../parse-env.js')('api')
const logger = require('../logger.js')

async function getNodesAsync(req, res, next) {
  let nodes = []

  // Do not attempt to build node list from rewards candidate in Private Mode
  if (env.PRIVATE_NETWORK === false) {
    try {
      //get the minting epoch which will tell us how to retrieve known good nodes from tendermint
      let abciResponse = await tmRpc.getAbciInfo()
      if (abciResponse.error) {
        logger.error(`RPC error communicating with Tendermint : ${abciResponse.error.message}`)
        throw new Error('Could not get abci info')
      }

      //create tx query tag and query TM for transactions containing reward candidate nodes
      let prevEpoch = JSON.parse(abciResponse.result.response.data).prev_mint_block
      if (prevEpoch != 0) {
        let tag = `NODERC=${prevEpoch}`
        let txResponse = await tmRpc.getTxSearch(tag, 1, 25) //get NODE-RC transactions from past 24 hours
        if (txResponse.error) {
          logger.error(`RPC error communicating with Tendermint : ${txResponse.error.message}`)
          throw new Error('Could not get NODE-RC transactions')
        }

        //retrieve IPs from reward candidate arrays
        let nodeArrays = txResponse.result.txs.map(tx => {
          let txText = new Buffer(new Buffer(tx, 'base64').toString('ascii'), 'base64').toString('ascii')
          return JSON.parse(txText).data.map(node => {
            return { public_uri: 'http://' + node.node_ip }
          })
        })
        nodes = [].concat.apply([], nodeArrays) //flatten array
      }
    } catch (error) {
      logger.error(`Tendermint RPC error : falling back to random nodes list : ${error.message}`)
    }
  }

  // If nothing was retrieved from Tendermint (0 results or Private Mode), get random Node list
  if (nodes.length == 0) {
    try {
      let nodesResponse = await stakedNode.getRandomNodes() //get random nodes if we can't get reward-candidates
      nodes = nodesResponse.map(row => {
        return { public_uri: 'http://' + row.publicIp }
      })
    } catch (error) {
      logger.error(`Random Nodes retrieval error : ${error.message}`)
      return next(new errors.InternalServerError('Could not query for nodes'))
    }
  }
  res.contentType = 'application/json'
  res.send(nodes)
  return next()
}

module.exports = {
  getNodesAsync: getNodesAsync,
  // additional functions for testing purposes
  setTmRpc: rpc => {
    tmRpc = rpc
  },
  setStakedNode: sn => {
    stakedNode = sn
  },
  setENV: obj => {
    env = obj
  }
}
