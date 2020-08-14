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
const logger = require('../logger.js')
const utils = require('../utils.js')

/**
 * GET / handler
 *
 * Root path handler with default message.
 *
 */
function getV1(req, res, next) {
  let submittingIP = utils.getClientIP(req)
  logger.info('Client IP: ' + submittingIP)
  return next(new errors.ImATeapotError('This is an API endpoint. Please consult https://chainpoint.org'))
}

module.exports = {
  getV1: getV1
}
