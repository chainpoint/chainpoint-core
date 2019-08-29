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

const fs = require('fs')

/**
 * Sleep for a specified number of milliseconds
 *
 * @param {number} ms - The number of milliseconds to sleep
 */
function sleepAsync(ms) {
  return new Promise(resolve => setTimeout(resolve, ms))
}

/**
 * Add specified seconds to a Date object
 *
 * @param {Date} date - The starting date
 * @param {number} seconds - The seconds of seconds to add to the date
 * @returns {Date}
 */
function addSeconds(date, seconds) {
  return new Date(date.getTime() + seconds * 1000)
}

/**
 * Add specified minutes to a Date object
 *
 * @param {Date} date - The starting date
 * @param {number} minutes - The number of minutes to add to the date
 * @returns {Date}
 */
function addMinutes(date, minutes) {
  return new Date(date.getTime() + minutes * 60000)
}

/**
 * Convert Date to ISO8601 string, stripping milliseconds
 * '2017-03-19T23:24:32Z'
 *
 * @param {Date} date - The date to convert
 * @returns {string} An ISO8601 formatted time string
 */
function formatDateISO8601NoMs(date) {
  return date.toISOString().slice(0, 19) + 'Z'
}

/**
 * Checks if value is a hexadecimal string
 *
 * @param {string} value - The value to check
 * @returns {bool} true if value is a hexadecimal string, otherwise false
 */
function isHex(value) {
  var hexRegex = /^[0-9a-f]{2,}$/i
  var isHex = hexRegex.test(value) && !(value.length % 2)
  return isHex
}

/**
 * Checks if value is a valid IP
 *
 * @param {string} value - The value to check
 * @returns {bool} true if value is a valid IP, otherwise false
 */
function isIP(value) {
  var ipRegex = /\b(?:(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])\.){3}(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])\b/
  var isIP = ipRegex.test(value)
  return isIP
}

/**
 * Converts proof path array output from the merkle-tools package
 * to a Chainpoint v3 ops array
 *
 * @param {proof object array} proof - The proof array generated by merkle-tools
 * @param {string} op - The hash type performed throughout merkle tree construction (sha-256, sha-512, sha-256-x2, etc.)
 * @returns {ops object array}
 */
function formatAsChainpointV3Ops(proof, op) {
  let ChainpointV3Ops = proof.reduce((result, item) => {
    if (item.left) {
      item = { l: item.left }
    } else {
      item = { r: item.right }
    }
    result.push(item, { op: op })
    return result
  }, [])

  return ChainpointV3Ops
}

/**
 * Reads a file and converts content to base64 string
 *
 * @param {file} file - The file to be converted to base64
 * @returns {string}
 */
function toBase64(file) {
  var body = fs.readFileSync(file)
  return body.toString('base64').replace(/\s/g, '')
}

module.exports = {
  sleepAsync: sleepAsync,
  addMinutes: addMinutes,
  addSeconds: addSeconds,
  formatDateISO8601NoMs: formatDateISO8601NoMs,
  isHex: isHex,
  isIP: isIP,
  formatAsChainpointV3Ops: formatAsChainpointV3Ops,
  toBase64: toBase64
}
