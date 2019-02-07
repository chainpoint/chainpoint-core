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

const restify = require('restify')
const utils = require('../utils.js')

/**
 * GET / handler
 *
 * Root path handler with default message.
 *
 */
function getV1 (req, res, next) {
  return next(new restify.ImATeapotError('This is an API endpoint. Please consult https://chainpoint.org'))
}

/**
 * GET /heartbeat handler
 *
 * Ops heartbeat handler with timestamp.
 *
 */
function getHeartbeatV1 (req, res, next) {
  res.noCache()
  res.send({ code: 'heartbeat', timestamp: utils.formatDateISO8601NoMs(new Date()) })
  return next()
}

module.exports = {
  getV1: getV1,
  getHeartbeatV1: getHeartbeatV1
}
