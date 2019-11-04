/* global describe, it, before, beforeEach, afterEach, after, xit */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const hashes = require('../lib/endpoints/hashes.js')
const uuidTime = require('uuid-time')
const BLAKE2s = require('blake2s-js')
const crypto = require('crypto')
const lnService = require('ln-service')
const sinon = require('sinon')

const nodeInfo = {
  alias: 'alice',
  best_header_timestamp: Math.round(Date.now() / 1000),
  block_hash: Buffer.alloc(32).toString('hex'),
  block_height: 100,
  chains: [],
  color: '#000000',
  public_key: Buffer.alloc(33).toString('hex'),
  active_channels_count: 1,
  peers_count: 1,
  num_pending_channels: 1,
  synced_to_chain: true,
  version: 'version'
}

nodeInfo.uris = [`${nodeInfo.identity_pubkey}@127.0.0.1:19735`]

function createInvoiceResponse(id) {
  let now = new Date(Date.now())
  return {
    created_at: now.toISOString(),
    description: 'test hodl',
    id,
    request:
      'lnsb100n1pwurlz8pp5n6eyk4t953r0rez50hm0ysupkv7ccwumxvwnjxye2wfnd420nzuqdz8fp85gnpqd9h8vmmfvdjjqcmjv4shgetyyphkugrjv4ch2etnwssx6ctyv5sxy7fqwdjkcescqzpgjv9semtflddfy3rqssq9t7kaxfvnpasvh6p9kd4mxr92zwy3zhuqr307lhkc7gauverp00rvwplpvreul49df0gknqawf3kxcjxmntgqs80qv7',
    tokens: 10
  }
}

function isValidHex(str) {
  return /^([a-fA-F0-9]{2}){20,64}$/.test(str)
}

function getSessionFromResp(res) {
  const cookies = res.headers['set-cookie']
  let sessionId = cookies.find(cookie => cookie.includes('sessionId='))
  sessionId = sessionId.slice(sessionId.indexOf('=') + 1, sessionId.indexOf(';'))
  return sessionId
}

describe.only('Hashes Controller', () => {
  let apiServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    apiServer = await app.startAPIServerAsync()
    hashes.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.100', CAVEAT_KEY: '12345' })
  })
  afterEach(() => {
    apiServer.close()
  })

  describe('GET /boltwall/node', () => {
    let getInfoStub
    before(() => {
      getInfoStub = sinon.stub(lnService, 'getWalletInfo')
      getInfoStub.returns(nodeInfo)
    })
    after(() => {
      getInfoStub.restore()
    })
    it('should return information about the boltwall node', done => {
      request(apiServer)
        .get('/boltwall/node')
        .expect(
          200,
          {
            pubKey: nodeInfo.public_key,
            alias: nodeInfo.alias,
            uris: nodeInfo.uris,
            activeChannelsCount: nodeInfo.active_channels_count,
            peersCount: nodeInfo.peers_count
          },
          done
        )
    })
  })

  describe('POST /boltwall/hodl', () => {
    const reqPath = '/boltwall/hodl'
    let createHodlInvoiceStub, secret

    beforeEach(() => {
      secret = crypto.randomBytes(32).toString('hex')
      createHodlInvoiceStub = sinon.stub(lnService, 'createHodlInvoice')
    })

    afterEach(() => {
      if (createHodlInvoiceStub) {
        createHodlInvoiceStub.restore()
        createHodlInvoiceStub = null
      }
    })

    it('should return with valid sessionID/preimage in a cookie when none present in request', done => {
      createHodlInvoiceStub.returns(createInvoiceResponse(secret))

      request(apiServer)
        .post(reqPath)
        .end((err, res) => {
          expect(err).to.equal(null)
          const sessionId = getSessionFromResp(res)
          expect(sessionId).to.have.lengthOf(64)
          expect(isValidHex(sessionId)).to.be.true
          createHodlInvoiceStub.restore()
          done()
        })
    })

    it('should respond with a payment information when requested from a new session', done => {
      const invoiceResp = createInvoiceResponse(secret)
      createHodlInvoiceStub.returns(invoiceResp)
      const expectedResponse = {
        id: invoiceResp.id,
        payreq: invoiceResp.request,
        description: invoiceResp.description,
        createdAt: invoiceResp['created_at'],
        amount: invoiceResp.tokens
      }
      request(apiServer)
        .post(reqPath)
        .expect(200, expectedResponse, done)
    })

    it('should return an invoice whose id is a sha256 hash of the session id', done => {
      // need a spy for this one
      // createHodlInvoiceStub.restore()
      // createHodlInvoiceStub = sinon.spy(lnService, 'createHodlInvoice')
      const expectedPaymentHash = crypto
        .createHash('sha256')
        .update(Buffer.from(secret, 'hex'))
        .digest()
        .toString('hex')
      let calledArgs
      createHodlInvoiceStub.callsFake(args => {
        calledArgs = args
        return createInvoiceResponse(expectedPaymentHash)
      })
      request(apiServer)
        .post(reqPath)
        .set('Cookie', [`sessionId=${secret};`])
        .end((err, res) => {
          expect(err).to.equal(null)
          const paymentHash = createHodlInvoiceStub.getCall(0).args[0].id
          expect(paymentHash).to.equal(expectedPaymentHash)
          expect(calledArgs.id).to.equal(expectedPaymentHash)
          expect(res.body)
            .to.have.property('id')
            .and.to.equal(expectedPaymentHash)
          done()
        })
    })
  })

  describe('GET /boltwall/invoice', () => {
    xit('should return with valid sessionID/preimage in a cookie when none present in request', done => {
      done()
    })
    xit('should return invoice information and status for invoice associated with session', done => {
      done()
    })
  })

  describe('POST /hash request validation', () => {
    it('should return proper error with invalid content type', done => {
      request(apiServer)
        .post('/hash')
        .set('Content-type', 'text/plain')
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid content type')
          done()
        })
    })

    it('should return proper error with missing hash', done => {
      request(apiServer)
        .post('/hash')
        .send({ name: 'Manny' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body: missing hash')
          done()
        })
    })

    it('should return proper error with hash not a string', done => {
      request(apiServer)
        .post('/hash')
        .send({ hash: ['badhash'] })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body: bad hash submitted')
          done()
        })
    })

    it('should return proper error with invalid hash', done => {
      request(apiServer)
        .post('/hash')
        .send({ hash: 'badhash' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body: bad hash submitted')
          done()
        })
    })

    it('should return proper error with no AMQP connection', done => {
      request(apiServer)
        .post('/hash')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          invoice_id: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
        })
        .expect('Content-type', /json/)
        .expect(500)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InternalServer')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('Message could not be delivered')
          done()
        })
    })
  })

  describe('POST /hash', () => {
    let hash
    before(() => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })
      hash = 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
    })

    xit('should return with valid sessionID/preimage in a cookie when none present in request', done => {
      done()
    })

    xit('should return a 402 response if request not made with an existing session', done => {
      done()
    })

    xit('should return a 402 response if request is made for a session with an unpaid invoice', done => {
      done()
    })

    xit('should settle the associated invoice if request is made successfully', done => {
      done()
    })

    xit('should return proper result with valid call', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(apiServer)
        .post('/hash')
        .send({
          hash
        })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res).to.have.property('body')
          expect(res.body).to.have.property('hash_id')
          expect(res.body)
            .to.have.property('hash')
            .and.to.equal(hash)
          expect(res.body).to.have.property('submitted_at')
          expect(res.body).to.have.property('processing_hints')
          expect(res.body.processing_hints)
            .to.have.property('cal')
            .and.to.be.a('string')
          expect(res.body.processing_hints)
            .to.have.property('btc')
            .and.to.be.a('string')
          done()
        })
    })

    xit('should return a v1 UUID node embedded with a partial SHA256 over timestamp and hash', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(apiServer)
        .post('/hash')
        .send({
          hash
        })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('proof_id')
          // Knowing the original hash, the timestamp from the UUID,
          // and the personalization bytes,
          // you should be able to calculate whether the UUID 'Node ID'
          // data segment is the 5 byte BLAKE2s hash of the timestamp
          // embedded in the UUID and the hash submitted to get this UUID.
          let t = parseInt(uuidTime.v1(res.body.proof_id))

          // 5 byte length BLAKE2s hash w/ personalization
          let h = new BLAKE2s(5, { personalization: Buffer.from('CHAINPNT') })
          let hashStr = [t.toString(), t.toString().length, res.body.hash, res.body.hash.length].join(':')

          h.update(Buffer.from(hashStr))
          let shortHashNodeBuf = Buffer.concat([Buffer.from([0x01]), h.digest()])
          // Last segment of UUIDv1 contains BLAKE2s hash to be matched
          expect(res.body.proof_id.split('-')[4]).to.equal(shortHashNodeBuf.toString('hex'))
          done()
        })
    })
  })

  // describe('GET /hash/invoice', () => {
  //   before(() => {
  //     hashes.setLND({
  //       services: {
  //         Lightning: {
  //           addInvoice: () => {
  //             throw new Error('err!')
  //           }
  //         }
  //       }
  //     })
  //   })
  //   it('should return proper error with lnd error', done => {
  //     request(apiServer)
  //       .get('/hash/invoice')
  //       .expect('Content-type', /json/)
  //       .expect(500)
  //       .end((err, res) => {
  //         expect(err).to.equal(null)
  //         expect(res.body)
  //           .to.have.property('code')
  //           .and.to.be.a('string')
  //           .and.to.equal('InternalServer')
  //         expect(res.body)
  //           .to.have.property('message')
  //           .and.to.be.a('string')
  //           .and.to.equal('Unable to generate invoice')
  //         done()
  //       })
  //   })
  // })

  // describe('GET /hash/invoice', () => {
  //   before(() => {
  //     hashes.setLND({
  //       services: {
  //         Lightning: {
  //           addInvoice: () => {
  //             return { payment_request: 'pr' }
  //           }
  //         }
  //       }
  //     })
  //   })
  //   it('should return proper invoice data', done => {
  //     request(apiServer)
  //       .get('/hash/invoice')
  //       .expect('Content-type', /json/)
  //       .expect(200)
  //       .end((err, res) => {
  //         expect(err).to.equal(null)
  //         expect(res.body)
  //           .to.have.property('invoice')
  //           .and.to.be.a('string')
  //           .and.to.equal('pr')
  //         done()
  //       })
  //   })
  // })

  // describe('POST /hash', () => {
  //   it('should return proper error with missing invoice_id', done => {
  //     request(apiServer)
  //       .post('/hash')
  //       .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
  //       .expect('Content-type', /json/)
  //       .expect(409)
  //       .end((err, res) => {
  //         expect(err).to.equal(null)
  //         expect(res.body)
  //           .to.have.property('code')
  //           .and.to.be.a('string')
  //           .and.to.equal('InvalidArgument')
  //         expect(res.body)
  //           .to.have.property('message')
  //           .and.to.be.a('string')
  //           .and.to.equal('invalid JSON body: missing invoice_id')
  //         done()
  //       })
  //   })
  // })

  // describe('POST /hash', () => {
  //   it('should return proper error with invoice_id not a string', done => {
  //     request(apiServer)
  //       .post('/hash')
  //       .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12', invoice_id: [1] })
  //       .expect('Content-type', /json/)
  //       .expect(409)
  //       .end((err, res) => {
  //         expect(err).to.equal(null)
  //         expect(res.body)
  //           .to.have.property('code')
  //           .and.to.be.a('string')
  //           .and.to.equal('InvalidArgument')
  //         expect(res.body)
  //           .to.have.property('message')
  //           .and.to.be.a('string')
  //           .and.to.equal('invalid JSON body: bad invoice_id submitted')
  //         done()
  //       })
  //   })
  // })

  // describe('POST /hash', () => {
  //   it('should return proper error with invalid invoice_id', done => {
  //     request(apiServer)
  //       .post('/hash')
  //       .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12', invoice_id: 'deadbeef' })
  //       .expect('Content-type', /json/)
  //       .expect(409)
  //       .end((err, res) => {
  //         expect(err).to.equal(null)
  //         expect(res.body)
  //           .to.have.property('code')
  //           .and.to.be.a('string')
  //           .and.to.equal('InvalidArgument')
  //         expect(res.body)
  //           .to.have.property('message')
  //           .and.to.be.a('string')
  //           .and.to.equal('invalid JSON body: bad invoice_id submitted')
  //         done()
  //       })
  //   })
  // })

  // describe('POST /hash', () => {
  //   it('should return proper error with unpaid invoice_id', done => {
  //     let invoiceId = 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
  //     request(apiServer)
  //       .post('/hash')
  //       .send({
  //         hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
  //         invoice_id: invoiceId
  //       })
  //       .expect('Content-type', /json/)
  //       .expect(402)
  //       .end((err, res) => {
  //         expect(err).to.equal(null)
  //         expect(res.body)
  //           .to.have.property('code')
  //           .and.to.be.a('string')
  //           .and.to.equal('PaymentRequired')
  //         expect(res.body)
  //           .to.have.property('message')
  //           .and.to.be.a('string')
  //           .and.to.equal(`invoice ${invoiceId} has not been paid`)
  //         done()
  //       })
  //   })
  // })

  // describe('POST /hash', () => {
  //   it('should return a matched set of metadata and UUID embedded timestamps', done => {
  //     app.setAMQPChannel({
  //       sendToQueue: function() {}
  //     })

  //     request(apiServer)
  //       .post('/hash')
  //       .send({
  //         hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
  //         invoice_id: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
  //       })
  //       .expect('Content-type', /json/)
  //       .expect(200)
  //       .end((err, res) => {
  //         expect(err).to.equal(null)
  //         expect(res.body).to.have.property('hash_id')
  //         expect(res.body).to.have.property('submitted_at')
  //         // The UUID timestamp has ms level precision, ISO8601 only to the second.
  //         // Check that they are within 1000ms of each other.
  //         expect(parseInt(uuidTime.v1(res.body.hash_id)) - Date.parse(res.body.submitted_at)).to.be.within(0, 1000)
  //         done()
  //       })
  //   })
  // })

  // describe('POST /hash', () => {

  // })
})

describe('Functions', () => {
  describe('calling generatePostHashResponse with one hash', () => {
    it('should return proper response object', done => {
      let res = hashes.generatePostHashResponse('ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12')
      expect(res).to.have.property('proof_id')
      expect(res)
        .to.have.property('hash')
        .and.to.equal('ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12')
      expect(res).to.have.property('submitted_at')
      expect(res).to.have.property('processing_hints')
      expect(res.processing_hints)
        .to.have.property('cal')
        .and.to.be.a('string')
      expect(res.processing_hints)
        .to.have.property('btc')
        .and.to.be.a('string')
      done()
    })
  })
})
