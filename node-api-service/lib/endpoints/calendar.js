/* Copyright (C) 2018 Tierion
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

const _ = require('lodash')
const restify = require('restify')
const connections = require('./lib/connections.js')

/**
 * GET /calendar/:txid handler
 *
 * Expects a path parameter 'txid' as a string
 *
 * Returns a calendar tx by tx hash
 */
async function getCalTxAsync (req, res, next) {
    let txID = req.params.txid
    if (!txID.includes("0x")){
        txID = "0x" + txID
    }
    let tx
    try {
      rpc = connections.openTendermintConnection()
      tx = await rpc.tx({hash: txID, prove: false})
    }catch (error){
      console.error('rpc error')
      return next(new restify.InternalServerError('Could not query for tx by hash'))
    }
    if (!tx) {
        res.status(404)
        res.noCache()
        res.send({ code: 'NotFoundError', message: '' })
        return next()
    }
    res.contentType = 'application/json'
    res.cache('public', { maxAge: 2592000 })
    res.send(tx)
    return next()
}

/**
 * GET /calendar/:txid/data handler
 *
 * Expects a path parameter 'txid' as a string
 *
 * Returns a calendar tx by tx hash
 */
async function getCalTxDataAsync (req, res, next) {
    let txID = req.params.txid
    if (!txID.includes("0x")){
        txID = "0x" + txID
    }
    let tx
    try {
        rpc = connections.openTendermintConnection()
        tx = await rpc.tx({hash: txID, prove: false})
    }catch (error){
        console.error('rpc error')
        return next(new restify.InternalServerError('Could not query for tx by hash'))
    }
    if (!tx) {
        res.status(404)
        res.noCache()
        res.send({ code: 'NotFoundError', message: '' })
        return next()
    }
    let txData = new Buffer(tx.tx, 'base64').toString('ascii')
    jsonData = JSON.parse(txData)
    res.contentType = 'application/json'
    res.cache('public', { maxAge: 2592000 })
    res.send(jsonData.Data)
    return next()
}

async function getCoreProofsAsync (req, res, next) {
    //TODO
    return next()
}

/**
 * GET /calendar/:height handler
 *
 * Expects a path parameter 'height' as an integer
 *
 * Returns a calendar block by calendar height
 */
async function getCalBlockByHeightV1Async (req, res, next) {
  let height = parseInt(req.params.height, 10)

  // ensure that :height is an integer
  if (!_.isInteger(height) || height < 0) {
    return next(new restify.InvalidArgumentError('invalid request, height must be a positive integer'))
  }
  let block
  try {
    block = await CalendarBlock.findOne({ where: { id: height } })
  } catch (error) {
    console.error(`getCalBlockByHeightV1Async failed : Could not query for block by height : ${error.message}`)
    return next(new restify.InternalServerError('Could not query for block by height'))
  }

  if (!block) {
    res.status(404)
    res.noCache()
    res.send({ code: 'NotFoundError', message: '' })
    return next()
  }

  block = block.get({ plain: true })
  res.contentType = 'application/json'
  block.id = parseInt(block.id, 10)
  block.time = parseInt(block.time, 10)
  block.version = parseInt(block.version, 10)
  res.cache('public', { maxAge: 2592000 })
  res.send(block)
  return next()
}

/**
 * GET /calendar/blockrange/:index handler
 *
 * Expects path parameter index as an integer to represent a block range to retrieve
 *
 * Returns an array of calendar blocks
 */
async function getCalBlockRangeV2Async (req, res, next) {
  let blockRangeIndex = parseInt(req.params.index, 10)

  // ensure that :index is an integer
  if (!_.isInteger(blockRangeIndex) || blockRangeIndex < 0) {
    return next(new restify.InvalidArgumentError('invalid request, index must be a positive integer'))
  }

  let fromHeight = blockRangeIndex * BLOCKRANGE_SIZE
  let toHeight = fromHeight + BLOCKRANGE_SIZE - 1

  let topBlock
  try {
    topBlock = await CalendarBlock.findOne({ attributes: ['id'], order: [['id', 'DESC']] })
  } catch (error) {
    console.error(`getCalBlockRangeV2Async failed : Could not query for top block : ${error.message}`)
    return next(new restify.InternalServerError('Could not query for top block'))
  }

  let maxBlockRangeReady = Math.floor((parseInt(topBlock.id) + 1) / BLOCKRANGE_SIZE) - 1
  if (blockRangeIndex > maxBlockRangeReady) {
    res.status(404)
    // cache the 404 for a short time to allow edge cache to remember that for a short while
    res.cache('public', { maxAge: 30 })
    res.send({ code: 'NotFoundError', message: 'block is not complete yet, check back soon' })
    return next()
  }

  let blocks
  try {
    blocks = await CalendarBlock.findAll({ where: { id: { [sequelize.Op.between]: [fromHeight, toHeight] } }, order: [['id', 'ASC']], raw: true })
  } catch (error) {
    console.error(`getCalBlockRangeV2Async failed : Could not query for block range : ${error.message}`)
    return next(new restify.InternalServerError('Could not query for block range'))
  }
  if (!blocks || blocks.length === 0) blocks = []

  // convert requisite fields to integers
  blocks = blocks.map((block) => {
    block.id = parseInt(block.id, 10)
    block.time = parseInt(block.time, 10)
    block.version = parseInt(block.version, 10)
    return block
  })

  let results = {}
  results.blocks = blocks
  res.cache('public', { maxAge: 2592000 })
  res.send(results)
  return next()
}

/**
 * GET /calendar/:height/data handler
 *
 * Expects a path parameter 'height' as an integer
 *
 * Returns dataVal property for calendar block by calendar height
 */
async function getCalBlockDataByHeightV1Async (req, res, next) {
  let height = parseInt(req.params.height, 10)

  // ensure that :height is an integer
  if (!_.isInteger(height) || height < 0) {
    return next(new restify.InvalidArgumentError('invalid request, height must be a positive integer'))
  }
  let block
  try {
    block = await CalendarBlock.findOne({ where: { id: height } })
  } catch (error) {
    console.error(`getCalBlockDataByHeightV1Async failed : Could not query for block by height : ${error.message}`)
    return next(new restify.InternalServerError('Could not query for block by height'))
  }

  if (!block) {
    res.status(404)
    // cache the 404 for a short time to allow edge cache to remember that for a short while
    res.cache('public', { maxAge: 30 })
    res.send({ code: 'NotFoundError', message: 'block not found' })
    return next()
  }

  block = block.get({ plain: true })
  res.contentType = 'text/plain'
  res.cache('public', { maxAge: 2592000 })
  res.send(block.dataVal)
  return next()
}

/**
 * GET /calendar/:height/hash handler
 *
 * Expects a path parameter 'height' as an integer
 *
 * Returns hash property for calendar block by calendar height
 */
async function getCalBlockHashByHeightV1Async (req, res, next) {
  let height = parseInt(req.params.height, 10)

  // ensure that :height is an integer
  if (!_.isInteger(height) || height < 0) {
    return next(new restify.InvalidArgumentError('invalid request, height must be a positive integer'))
  }
  let block
  try {
    block = await CalendarBlock.findOne({ where: { id: height } })
  } catch (error) {
    console.error(`getCalBlockHashByHeightV1Async failed : Could not query for block by height : ${error.message}`)
    return next(new restify.InternalServerError('Could not query for block by height'))
  }

  if (!block) {
    res.status(404)
    // cache the 404 for a short time to allow edge cache to remember that for a short while
    res.cache('public', { maxAge: 30 })
    res.send({ code: 'NotFoundError', message: 'block not found' })
    return next()
  }

  block = block.get({ plain: true })
  res.contentType = 'text/plain'
  res.cache('public', { maxAge: 2592000 })
  res.send(block.hash)
  return next()
}

module.exports = {
  getCalTxAsync: getCalTxAsync,
  getCalTxDataAsync: getCalTxDataAsync,
  getCoreProofsAsync: getCoreProofsAsync,
  getCalBlockByHeightV1Async: getCalBlockByHeightV1Async,
  getCalBlockRangeV2Async: getCalBlockRangeV2Async,
  getCalBlockDataByHeightV1Async: getCalBlockDataByHeightV1Async,
  getCalBlockHashByHeightV1Async: getCalBlockHashByHeightV1Async,
  setDatabase: (sqlz, calBlock) => { sequelize = sqlz; CalendarBlock = calBlock }
}
