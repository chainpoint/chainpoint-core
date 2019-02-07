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

const crypto = require('crypto')
const restify = require('restify')
const _ = require('lodash')
const moment = require('moment')
var validUrl = require('valid-url')
const url = require('url')
const ip = require('ip')
const utils = require('../utils.js')
const semver = require('semver')
const rp = require('request-promise-native')
const tntUnits = require('../tntUnits.js')

const env = require('../parse-env.js')('api')

let RegisteredNode
let sequelize

// The redis connection used for all redis communication
// This value is set once the connection has been established
let redis = null

// The number of results to return when responding to a random nodes query
const RANDOM_NODES_RESULT_LIMIT = 25

// The minimium TNT grains required to operate a Node
const minGrainsBalanceNeeded = env.MIN_TNT_GRAINS_BALANCE_FOR_REWARD

// the minimum audit passing Node version for existing registered Nodes, set by consul
let minNodeVersionExisting = null

// the minimum audit passing Node version for newly registering Nodes, set by consul
let minNodeVersionNew = null

// The lifespan of balance pass redis entries
const BALANCE_PASS_EXPIRE_MINUTES = 60 * 24 // 1 day

// validate eth address is well formed
let isEthereumAddr = (address) => {
  return /^0x[0-9a-fA-F]{40}$/i.test(address)
}

let isHMAC = (hmac) => {
  return /^[0-9a-fA-F]{64}$/i.test(hmac)
}

/**
 * GET /nodes/random retrieve handler
 *
 * Retrieve a random subset of registered and healthy Nodes
 */
async function getNodesRandomV1Async (req, res, next) {
  // get a list of random healthy Nodes
  // Produce a weighted health score and select 25 Nodes from the top 1000
  // cp_health_score counts for 75% of the score, pf_health_score for 25%
  let sqlQuery = `SELECT public_uri FROM 
                  (
                    SELECT public_uri, ROUND(((cp_health_score * 3 + pf_health_score * 1) / 4), 4) AS weighted_score FROM 
                    (
                      SELECT public_uri, 
                      CASE (pass_count+fail_count) WHEN 0 THEN 0 ELSE ROUND(pass_count/(pass_count+fail_count), 4) END AS pf_health_score,
                      LEAST(ROUND(consecutive_passes / 48, 4), 1) AS cp_health_score
                      FROM chainpoint_registered_nodes 
                    )
                    ORDER BY weighted_score DESC LIMIT 1000
                  )
                  ORDER BY RANDOM() LIMIT ${RANDOM_NODES_RESULT_LIMIT}`

  let rndNodes
  try {
    rndNodes = await sequelize.query(sqlQuery, { type: sequelize.QueryTypes.SELECT })
  } catch (error) {
    console.error(`getNodesRandomV1Async failed : Unable to query for random nodes : ${error.message}`)
    return next(new restify.InternalServerError(`Unable to query for random nodes`))
  }

  // build well formatted result array
  rndNodes = rndNodes.map((rndNode) => {
    return {
      public_uri: rndNode.public_uri
    }
  })

  res.cache('public', { maxAge: 60 })

  // randomize results order, limit, and send
  res.send(rndNodes)
  return next()
}

/**
 * GET /nodes/blacklist retrieve handler
 *
 * Retrieve an IP blacklist that can be pulled by Nodes to
 * block connections from abusive IPs
 */
async function getNodesBlacklistV1Async (req, res, next) {
  let list = { blacklist: [] }
  res.cache('public', { maxAge: 600 })
  res.send(list)
  return next()
}

/**
 * POST /node create handler
 *
 * Create a new registered Node
 */
async function postNodeV1Async (req, res, next) {
  if (req.contentType() !== 'application/json') {
    return next(new restify.InvalidArgumentError('invalid content type'))
  }

  let minNodeVersionOK = false
  if (req.headers && req.headers['x-node-version']) {
    let nodeVersion = req.headers['x-node-version']
    try {
      minNodeVersionOK = semver.satisfies(nodeVersion, `>=${minNodeVersionNew}`)
    } catch (error) {
      return next(new restify.UpgradeRequiredError(`Node version ${minNodeVersionNew} or greater required`))
    }
  }

  if (!minNodeVersionOK) {
    return next(new restify.UpgradeRequiredError(`Node version ${minNodeVersionNew} or greater required`))
  }

  if (!req.params.hasOwnProperty('tnt_addr')) {
    return next(new restify.InvalidArgumentError('invalid JSON body, missing tnt_addr'))
  }

  if (_.isEmpty(req.params.tnt_addr)) {
    return next(new restify.InvalidArgumentError('invalid JSON body, empty tnt_addr'))
  }

  let lowerCasedTntAddrParam
  if (!isEthereumAddr(req.params.tnt_addr)) {
    return next(new restify.InvalidArgumentError('invalid JSON body, malformed tnt_addr'))
  } else {
    lowerCasedTntAddrParam = req.params.tnt_addr.toLowerCase()
  }

  // Return formatted Public URI, omit port number as nodes are only allowed to run on default: Port 80
  // using url.parse() will implicitly lowercase the uri provided
  let lowerCasedPublicUri = (() => {
    if (!req.params.public_uri) return null

    let parsedURI = url.parse(req.params.public_uri)

    return `${parsedURI.protocol}//${parsedURI.hostname}`
  })()
  // if an public_uri is provided, it must be valid
  if (lowerCasedPublicUri && !_.isEmpty(lowerCasedPublicUri)) {
    if (!validUrl.isHttpUri(lowerCasedPublicUri)) {
      return next(new restify.InvalidArgumentError('invalid JSON body, invalid public_uri'))
    }

    let parsedPublicUri = url.parse(lowerCasedPublicUri)
    // ensure that hostname is an IP
    if (!utils.isIP(parsedPublicUri.hostname)) return next(new restify.InvalidArgumentError('public_uri hostname must be an IP'))
    // ensure that it is not a private IP
    if (ip.isPrivate(parsedPublicUri.hostname)) return next(new restify.InvalidArgumentError('public_uri hostname must not be a private IP'))
    // disallow 0.0.0.0
    if (parsedPublicUri.hostname === '0.0.0.0') return next(new restify.InvalidArgumentError('0.0.0.0 not allowed in public_uri'))
    // disallow any port that is not 80
    if (req.params.public_uri && url.parse(req.params.public_uri).port && url.parse(req.params.public_uri).port !== '80') return next(new restify.InvalidArgumentError('public_uri hostname must specify port 80 or omit the port number to have it be implicitly set to 80'))
  }

  try {
    let whereClause
    if (lowerCasedPublicUri && !_.isEmpty(lowerCasedPublicUri)) {
      whereClause = { [sequelize.Op.or]: [{ tntAddr: lowerCasedTntAddrParam }, { publicUri: lowerCasedPublicUri }] }
    } else {
      whereClause = { tntAddr: lowerCasedTntAddrParam }
    }
    let result = await RegisteredNode.findOne({ where: whereClause, raw: true, attributes: ['tntAddr', 'publicUri'] })
    if (result) {
      // a result was found, so some element of vaidation failed. Identify and return.
      if (lowerCasedTntAddrParam === result.tntAddr) {
        // the tnt address is already registered
        return next(new restify.ConflictError('the Ethereum address provided is already registered'))
      } else {
        // the public uri is already in use
        return next(new restify.ConflictError('the public URI provided is already registered'))
      }
    }
  } catch (error) {
    console.error(`Unable to query registered Nodes: ${error.message}`)
    return next(new restify.InternalServerError('Unable to query registered Nodes'))
  }

  // ensure a unique source ip
  let createdFromIp = getSourceIp(req)
  if (createdFromIp) {
    let thirtyDaysAgo = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000)
    let matches = await RegisteredNode.count({ where: { createdFromIp: createdFromIp, created_at: { [sequelize.Op.gte]: thirtyDaysAgo } } })
    if (matches > 0) return next(new restify.ConflictError('a Node has already been registered from this IP'))
  }

  // check to see if the Node has the min balance required for Node operation
  try {
    let nodeBalance = await getTNTGrainsBalanceForAddressAsync(lowerCasedTntAddrParam)
    if (nodeBalance < minGrainsBalanceNeeded) {
      let minTNTBalanceNeeded = tntUnits.grainsToTNT(minGrainsBalanceNeeded)
      return next(new restify.ForbiddenError(`TNT address ${lowerCasedTntAddrParam} does not have the minimum balance of ${minTNTBalanceNeeded} TNT for Node operation`))
    }
    // create a balance check entry for this tnt address
    await redis.set(`${env.BALANCE_CHECK_KEY_PREFIX}:${lowerCasedTntAddrParam}`, nodeBalance, 'EX', BALANCE_PASS_EXPIRE_MINUTES * 60)
  } catch (error) {
    console.error(`Unable to check address balance: ${error.message}`)
    return next(new restify.InternalServerError(`Unable to check address balance`))
  }

  let randHMACKey = crypto.randomBytes(32).toString('hex')

  let newNode
  try {
    newNode = await RegisteredNode.create({
      tntAddr: lowerCasedTntAddrParam,
      publicUri: lowerCasedPublicUri,
      hmacKey: randHMACKey,
      tntCredit: 86400,
      createdFromIp: createdFromIp
    })
  } catch (error) {
    console.error(`Could not create RegisteredNode for ${lowerCasedTntAddrParam} at ${lowerCasedPublicUri}: ${error.message}`)
    return next(new restify.InternalServerError(`Could not create RegisteredNode for ${lowerCasedTntAddrParam} at ${lowerCasedPublicUri}`))
  }

  res.send({
    tnt_addr: newNode.tntAddr,
    public_uri: newNode.publicUri,
    hmac_key: newNode.hmacKey
  })
  return next()
}

/**
 * PUT /node/:tnt_addr update handler
 *
 * Updates an existing registered Node
 */
async function putNodeV1Async (req, res, next) {
  if (req.contentType() !== 'application/json') {
    return next(new restify.InvalidArgumentError('invalid content type'))
  }

  let minNodeVersionOK = false
  if (req.headers && req.headers['x-node-version']) {
    let nodeVersion = req.headers['x-node-version']
    try {
      minNodeVersionOK = semver.satisfies(nodeVersion, `>=${minNodeVersionExisting}`)
    } catch (error) {
      return next(new restify.UpgradeRequiredError(`Node version ${minNodeVersionExisting} or greater required`))
    }
  }
  if (!minNodeVersionOK) {
    return next(new restify.UpgradeRequiredError(`Node version ${minNodeVersionExisting} or greater required`))
  }

  if (!req.params.hasOwnProperty('tnt_addr')) {
    return next(new restify.InvalidArgumentError('invalid JSON body, missing tnt_addr'))
  }

  if (_.isEmpty(req.params.tnt_addr)) {
    return next(new restify.InvalidArgumentError('invalid JSON body, empty tnt_addr'))
  }

  let lowerCasedTntAddrParam
  if (!isEthereumAddr(req.params.tnt_addr)) {
    return next(new restify.InvalidArgumentError('invalid JSON body, malformed tnt_addr'))
  } else {
    lowerCasedTntAddrParam = req.params.tnt_addr.toLowerCase()
  }

  if (!req.params.hasOwnProperty('hmac')) {
    return next(new restify.InvalidArgumentError('invalid JSON body, missing hmac'))
  }

  if (_.isEmpty(req.params.hmac)) {
    return next(new restify.InvalidArgumentError('invalid JSON body, empty hmac'))
  }

  if (!isHMAC(req.params.hmac)) {
    return next(new restify.InvalidArgumentError('invalid JSON body, invalid hmac'))
  }

  // Return formatted Public URI, omit port number as nodes are only allowed to run on default: Port 80
  let lowerCasedPublicUri = (() => {
    if (!req.params.public_uri) return null

    let parsedURI = url.parse(req.params.public_uri)

    return `${parsedURI.protocol}//${parsedURI.hostname}`
  })()
  // if an public_uri is provided, it must be valid
  if (lowerCasedPublicUri && !_.isEmpty(lowerCasedPublicUri)) {
    if (!validUrl.isHttpUri(lowerCasedPublicUri)) {
      return next(new restify.InvalidArgumentError('invalid JSON body, invalid public_uri'))
    }
    let parsedPublicUri = url.parse(lowerCasedPublicUri)
    // ensure that hostname is an IP
    if (!utils.isIP(parsedPublicUri.hostname)) return next(new restify.InvalidArgumentError('public_uri hostname must be an IP'))
    // ensure that it is not a private IP
    if (ip.isPrivate(parsedPublicUri.hostname)) return next(new restify.InvalidArgumentError('public_uri hostname must not be a private IP'))
    // disallow 0.0.0.0
    if (parsedPublicUri.hostname === '0.0.0.0') return next(new restify.InvalidArgumentError('0.0.0.0 not allowed in public_uri'))
    // disallow any port that is not 80
    if (req.params.public_uri && url.parse(req.params.public_uri).port && url.parse(req.params.public_uri).port !== '80') return next(new restify.InvalidArgumentError('public_uri hostname must specify port 80 or omit the port number to have it be implicitly set to 80'))
  }

  let regNode
  try {
    let whereClause
    if (lowerCasedPublicUri && !_.isEmpty(lowerCasedPublicUri)) {
      whereClause = { [sequelize.Op.or]: [{ tntAddr: lowerCasedTntAddrParam }, { publicUri: lowerCasedPublicUri }] }
    } else {
      whereClause = { tntAddr: lowerCasedTntAddrParam }
    }
    let results = await RegisteredNode.findAll({ where: whereClause, attributes: ['tntAddr', 'publicUri', 'hmacKey'] })
    if (results.length === 0) {
      // no results found, a node with this tntAddr does not exist
      res.status(404)
      res.noCache()
      res.send({ code: 'NotFoundError', message: 'could not find registered Node' })
      return next()
    } else if (results.length === 1) {
      // One results found. If the tntAddr doesn't match, it was just a
      // publicUri match, but a node with this tntAddr does not exist
      if (results[0].tntAddr !== lowerCasedTntAddrParam) {
        res.status(404)
        res.noCache()
        res.send({ code: 'NotFoundError', message: 'could not find registered Node' })
        return next()
      }
      // a matching register node was found
      regNode = results[0]
    } else {
      // two results found, a matching tntAddr and matching publicUri in different records
      // this means we have a registered node that is attempting to change its publicUri
      // to a value already held by a different registered Node at a different tntAddr
      // the public uri is already in use
      return next(new restify.ConflictError('the public URI provided is already registered'))
    }
  } catch (error) {
    console.error(`Unable to query registered Nodes: ${error.message}`)
    return next(new restify.InternalServerError('Unable to query registered Nodes'))
  }

  // HMAC-SHA256(hmac-key, TNT_ADDRESS|IP|YYYYMMDDHHmm)
  // Forces Nodes to be within +/- 1 min of Core to generate a valid HMAC
  let formattedDateInt = parseInt(moment().utc().format('YYYYMMDDHHmm'))
  // build an array af acceptable hmac values with -1 minute, current minute, +1 minute
  let acceptableHMACs = [-1, 0, 1].map((addend) => {
    // use req.params.tnt_addr below instead of lowerCasedTntAddrParam and
    // to req.params.public_uri below instead of lowerCasedPublicUri preserve
    // formatting submitted from Node and used in that Node's calculation
    let formattedTimeString = (formattedDateInt + addend).toString()
    let hmacTxt = [req.params.tnt_addr, req.params.public_uri, formattedTimeString].join('')
    let calculatedHMAC = crypto.createHmac('sha256', regNode.hmacKey).update(hmacTxt).digest('hex')
    return calculatedHMAC
  })
  if (!_.includes(acceptableHMACs, req.params.hmac)) {
    return next(new restify.InvalidArgumentError('Invalid authentication HMAC provided - Try NTP sync'))
  }

  if (lowerCasedPublicUri == null || _.isEmpty(lowerCasedPublicUri)) {
    regNode.publicUri = null
  } else {
    regNode.publicUri = lowerCasedPublicUri
  }

  // check to see if the Node has the min balance required for Node operation
  try {
    let nodeBalance = await getTNTGrainsBalanceForAddressAsync(lowerCasedTntAddrParam)
    if (nodeBalance < minGrainsBalanceNeeded) {
      let minTNTBalanceNeeded = tntUnits.grainsToTNT(minGrainsBalanceNeeded)
      return next(new restify.ForbiddenError(`TNT address ${lowerCasedTntAddrParam} does not have the minimum balance of ${minTNTBalanceNeeded} TNT for Node operation`))
    }
    // create a balance check entry for this tnt address
    await redis.set(`${env.BALANCE_CHECK_KEY_PREFIX}:${lowerCasedTntAddrParam}`, nodeBalance, 'EX', BALANCE_PASS_EXPIRE_MINUTES * 60)
  } catch (error) {
    console.error(`Unable to check address balance: ${error.message}`)
    return next(new restify.InternalServerError(`Unable to check address balance`))
  }

  try {
    await regNode.save()
  } catch (error) {
    console.error(`Could not update RegisteredNode: ${error.message}`)
    return next(new restify.InternalServerError('Could not update RegisteredNode'))
  }

  res.send({
    tnt_addr: regNode.tntAddr,
    public_uri: regNode.publicUri || undefined
  })
  return next()
}

function updateMinNodeVersionNew (ver) {
  try {
    if (!semver.valid(ver) || ver === null) throw new Error(`Bad minNodeVersionNew semver value : ${ver}`)
    minNodeVersionNew = ver
    console.log(`Minimum Node version for *new* Nodes updated to ${ver}`)
  } catch (error) {
    console.error(`Could not update minNodeVersionNew : ${error.message}`)
  }
}

function updateMinNodeVersionExisting (ver) {
  try {
    if (!semver.valid(ver) || ver === null) throw new Error(`Bad minNodeVersionExisting semver value : ${ver}`)
    minNodeVersionExisting = ver
    console.log(`Minimum Node version for *existing* Nodes updated to ${ver}`)
  } catch (error) {
    console.error(`Could not update minNodeVersionExisting : ${error.message}`)
  }
}

let getTNTGrainsBalanceForAddressAsync = async (tntAddress) => {
  let options = {
    headers: [
      {
        name: 'Content-Type',
        value: 'application/json'
      }
    ],
    method: 'GET',
    uri: `${env.ETH_TNT_TX_CONNECT_URI}/balance/${tntAddress}`,
    json: true,
    gzip: true,
    timeout: 60000,
    resolveWithFullResponse: true
  }

  try {
    let balanceResponse = await rp(options)
    let balanceTNTGrains = balanceResponse.body.balance
    let intBalance = parseInt(balanceTNTGrains)
    if (intBalance >= 0) {
      return intBalance
    } else {
      throw new Error(`Bad TNT balance value: ${balanceTNTGrains}`)
    }
  } catch (error) {
    throw new Error(`TNT balance read error: ${error.message}`)
  }
}

function getSourceIp (req) {
  let reqIp = null
  if (req.headers['CF-Connecting-IP']) {
    // Cloudflare
    reqIp = req.headers['CF-Connecting-IP']
    console.log(`getSourceIp : extracted source IP from CF-Connecting-IP : ${reqIp}`)
  } else if (req.headers['x-forwarded-for']) {
    let fwdIPs = req.headers['x-forwarded-for'].split(',')
    reqIp = fwdIPs[0]
    console.log(`getSourceIp : extracted source IP from x-forwarded-for : ${req.headers['x-forwarded-for']} : ${reqIp}`)
  } else {
    reqIp = req.connection.remoteAddress || null
    console.log(`getSourceIp : extracted source IP from remoteAddress : ${reqIp}`)
  }
  if (reqIp) reqIp = reqIp.replace(/^.*:/, '')

  return reqIp
}

module.exports = {
  getNodesRandomV1Async: getNodesRandomV1Async,
  getNodesBlacklistV1Async: getNodesBlacklistV1Async,
  postNodeV1Async: postNodeV1Async,
  putNodeV1Async: putNodeV1Async,
  overrideGetTNTGrainsBalanceForAddressAsync: (func) => { getTNTGrainsBalanceForAddressAsync = func },
  setMinNodeVersionExisting: (ver) => { updateMinNodeVersionExisting(ver) },
  setMinNodeVersionNew: (ver) => { updateMinNodeVersionNew(ver) },
  setRedis: (redisClient) => { redis = redisClient },
  setDatabase: (sqlz, regNode) => { sequelize = sqlz; RegisteredNode = regNode }
}
