const invoice = {
  payreq:
    'lntb10u1pw7kfm8pp50nhe8uk9r2n9yz97c9z8lsu0ckxehnsnwkjn9mdsmnf' +
    'fpgkrxzhqdq5w3jhxapqd9h8vmmfvdjscqzpgllq2qvdlgkllc27kpd87lz8p' +
    'dfsfmtteyc3kwq734jpwnvqt96e4nuy0yauzdrtkumxsvawgda8dlljxu3nnj' +
    'lhs6w75390wy7ukj6cpfmygah',
  secret: '2ca931a1c36b48f54948b898a271a53ed91ff7d0081939a5fa511249e81cba5c',
  id: '7cef93f2c51aa65208bec1447fc38fc58d9bce1375a532edb0dcd290a2c330ae',
  description: 'test invoice',
  createdAt: '2016-08-29T09:12:33.001Z',
  amount: 1000,
  is_confirmed: false
}

const lnServiceInvoice = {
  tokens: invoice.amount,
  request: invoice.payreq,
  created_at: invoice.created_at,
  secret: invoice.secret,
  description: invoice.description,
  id: invoice.id
}

const challenge =
  'LSAT macaroon="MDAxY2xvY2F0aW9uIDEyNy4wLjAuMTo4MDgwCjAwOTRpZGVudGlmaWVyIDAwMDA3Y2VmOTNmMmM1MWFhNjUyMDhiZWMxNDQ3ZmMzOGZjNThkOWJjZTEzNzVhNTMyZWRiMGRjZDI5MGEyYzMzMGFlMmNhOTMxYTFjMzZiNDhmNTQ5NDhiODk4YTI3MWE1M2VkOTFmZjdkMDA4MTkzOWE1ZmE1MTEyNDllODFjYmE1YwowMDJmc2lnbmF0dXJlIFAvS7iENFK0Z7Hc0GBM3wLOu0zB5Ino6DoXosjg4cpcCg", invoice="lntb10u1pw7kfm8pp50nhe8uk9r2n9yz97c9z8lsu0ckxehnsnwkjn9mdsmnffpgkrxzhqdq5w3jhxapqd9h8vmmfvdjscqzpgllq2qvdlgkllc27kpd87lz8pdfsfmtteyc3kwq734jpwnvqt96e4nuy0yauzdrtkumxsvawgda8dlljxu3nnjlhs6w75390wy7ukj6cpfmygah"'

exports.challenge = challenge
exports.invoice = invoice
exports.lnServiceInvoice = lnServiceInvoice
