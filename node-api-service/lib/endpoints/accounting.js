const lnAccounting = require('ln-accounting')
const errors = require('restify-errors')
const logger = require('../logger.js')

let lnd = null

async function getAccountingReport(req, res, next) {
  if (!lnd) return next(new errors.InternalServerError('Missing authenticated lnd gRPC API Object'))
  const { query } = req
  const options = { lnd }

  options.currency = 'BTC'
  options.fiat = 'USD'

  if (query.provider && lnAccounting.rateProviders.includes(query.provider)) options.rate_provider = query.provider
  else options.rate_provider = 'coincap'

  if (query.category) options.category = query.category
  if (query.before && query.after && query.before < query.after)
    return next(new errors.BadRequestError('Request made with a start date after end date'))
  if (query.before) options.before = query.before
  if (query.after) options.after = query.after

  try {
    const report = await lnAccounting.getAccountingReport(options)
    res.status(200)
    return res.send(report)
  } catch (e) {
    logger.error(e)
  }
}

module.exports = {
  getAccountingReport: getAccountingReport,
  setLND: l => {
    lnd = l
  }
}
