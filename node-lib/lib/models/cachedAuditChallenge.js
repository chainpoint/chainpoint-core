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

const CHALLENGE_CACHE_EXPIRE_MINUTES = 60 * 24
const AUDIT_CHALLENGE_KEY_PREFIX = 'AuditChallenge'

const env = require('../parse-env.js')()

let AuditChallenge

// The most recent challenge redis key, supplied by consul
let MostRecentChallengeKey = null

// The redis connection used for all redis communication
// This value is set once the connection has been established
let redis = null

// The consul connection used for all consul communication
// This value is set once the connection has been established
let consul = null

async function getMostRecentChallengeDataAsync () {
  let mostRecentChallengeText = (redis && MostRecentChallengeKey) ? await redis.get(MostRecentChallengeKey) : null
  // if nothing was found, it is not cached, retrieve from the database and add to cache for future requests
  if (mostRecentChallengeText === null) {
    // get the most recent challenge record
    let mostRecentChallenge = await AuditChallenge.findOne({ order: [['time', 'DESC']] })
    if (!mostRecentChallenge) return null
    // build the most recent challenge string
    mostRecentChallengeText = `${mostRecentChallenge.time}:${mostRecentChallenge.minBlock}:${mostRecentChallenge.maxBlock}:${mostRecentChallenge.nonce}:${mostRecentChallenge.solution}`
    // build the most recent challenge key
    let mostRecentChallengeKey = `${AUDIT_CHALLENGE_KEY_PREFIX}:${mostRecentChallenge.time}`
    // write this most recent challenge to redis
    if (redis) await redis.set(mostRecentChallengeKey, mostRecentChallengeText, 'EX', CHALLENGE_CACHE_EXPIRE_MINUTES * 60)
    // this value should automatically be set from consul, but in case it has not ben set yet, set it here
    MostRecentChallengeKey = mostRecentChallengeKey
  }

  return mostRecentChallengeText
}

async function getMostRecentChallengeDataSolutionRemovedAsync () {
  let mostRecentChallengeText = await getMostRecentChallengeDataAsync()
  let mostRecentChallengeTextNoSolution = mostRecentChallengeText ? mostRecentChallengeText.split(':').slice(0, 4).join(':') : null
  return mostRecentChallengeTextNoSolution
}

async function getChallengeDataByTimeAsync (challengeTime) {
  let challengeKey = `${AUDIT_CHALLENGE_KEY_PREFIX}:${challengeTime}`
  let challengeText = (redis) ? await redis.get(challengeKey) : null
  // if nothing was found, it is not cached, retrieve from the database and add to cache for future requests
  if (challengeText === null) {
    // get the challenge record by time
    let challengeByTime = await AuditChallenge.findOne({ where: { time: challengeTime } })
    if (!challengeByTime) return null
    // build the challenge string
    challengeText = `${challengeByTime.time}:${challengeByTime.minBlock}:${challengeByTime.maxBlock}:${challengeByTime.nonce}:${challengeByTime.solution}`
    // write this challenge to redis
    if (redis) await redis.set(challengeKey, challengeText, 'EX', CHALLENGE_CACHE_EXPIRE_MINUTES * 60)
  }

  return challengeText
}

async function setNewAuditChallengeAsync (challengeTime, challengeMinBlockHeight, challengeMaxBlockHeight, challengeNonce, challengeSolution) {
  // write the new challenge to the database
  let newChallenge = await AuditChallenge.create({
    time: challengeTime,
    minBlock: challengeMinBlockHeight,
    maxBlock: challengeMaxBlockHeight,
    nonce: challengeNonce,
    solution: challengeSolution
  })
  // construct the challenge string
  let auditChallengeText = `${newChallenge.time}:${newChallenge.minBlock}:${newChallenge.maxBlock}:${newChallenge.nonce}:${newChallenge.solution}`
  // build the new challenge redis key
  let newChallengeKey = `${AUDIT_CHALLENGE_KEY_PREFIX}:${newChallenge.time}`
  // write this new challenge to redis
  if (redis) await redis.set(newChallengeKey, auditChallengeText, 'EX', CHALLENGE_CACHE_EXPIRE_MINUTES * 60)
  // update the most recent key in consul with the new challenge key value
  return new Promise((resolve, reject) => {
    consul.kv.set(env.AUDIT_CHALLENGE_RECENT_KEY, newChallengeKey, function (err, result) {
      if (err) return reject(err)
      return resolve(newChallengeKey)
    })
  })
}

module.exports = {
  setRedis: (r) => { redis = r },
  setConsul: (c) => { consul = c },
  setMostRecentChallengeKey: (key) => { MostRecentChallengeKey = key },
  getMostRecentChallengeDataAsync: getMostRecentChallengeDataAsync,
  getMostRecentChallengeDataSolutionRemovedAsync: getMostRecentChallengeDataSolutionRemovedAsync,
  setNewAuditChallengeAsync: setNewAuditChallengeAsync,
  getChallengeDataByTimeAsync: getChallengeDataByTimeAsync,
  setDatabase: (sqlz, auditChal) => { AuditChallenge = auditChal }
}
