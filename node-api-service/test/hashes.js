/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const { invoice, lnServiceInvoice, challenge } = require('./sample_data/hashes_tests')
const hashes = require('../lib/endpoints/hashes.js')
const uuidTime = require('uuid-time')
const BLAKE2s = require('blake2s-js')
const crypto = require('crypto')
const lnService = require('ln-service')
const sinon = require('sinon')
const { Lsat, Identifier } = require('lsat-js')

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

describe.only('Hashes Controller', () => {
  let apiServer = null,
    lndGrpcStub,
    lnd
  beforeEach(async () => {
    app.setThrottle(() => (_req, _res, next) => next())
    apiServer = await app.startAPIServerAsync()
    hashes.setENV({
      CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.100',
      SESSION_SECRET: process.env.SESSION_SECRET
    })
    // need a reference to the authenticated lnd object for checking if other spies
    // are called with expected args
    lnd = {}
    lndGrpcStub = sinon.stub(lnService, 'authenticatedLndGrpc').returns({ lnd })
  })
  afterEach(() => {
    apiServer.close()
    lndGrpcStub.restore()
  })

  describe('GET /boltwall/node', () => {
    let getInfoStub
    beforeEach(() => {
      getInfoStub = sinon.stub(lnService, 'getWalletInfo')
      getInfoStub.returns(nodeInfo)
    })
    afterEach(() => {
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

  describe('GET /boltwall/invoice', () => {
    let getInvoiceStub

    beforeEach(() => {
      getInvoiceStub = sinon.stub(lnService, 'getInvoice')
      getInvoiceStub.returns({ ...invoice, request: invoice.payreq })
    })

    afterEach(() => {
      getInvoiceStub.restore()
    })

    it('should return invoice information and status for invoice associated with LSAT', done => {
      const lsat = Lsat.fromChallenge(challenge)
      request(apiServer)
        .get('/boltwall/invoice')
        .set('Authorization', lsat.toToken())
        .expect(200)
        .end((_err, res) => {
          expect(res.body).to.eql({
            status: invoice.is_confirmed ? 'paid' : 'unpaid',
            payreq: invoice.payreq,
            id: invoice.id,
            description: invoice.description
          })
          expect(res.body.id).to.equal(lsat.paymentHash)
          done()
        })
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
    let hash, getInvoiceStub, settleInvoiceStub, randomBytesStub, lsat
    before(() => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      hash = 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
      lsat = Lsat.fromChallenge(challenge)
    })

    beforeEach(() => {
      hashes.setInvoiceClient({
        addHoldInvoiceAsync: sinon.spy(() => ({
          payment_request: invoice.payreq
        }))
      })

      // stub the get invoice call that is made by boltwall's ln-service
      // return should be set in the individual tests
      getInvoiceStub = sinon.stub(lnService, 'getInvoice')
      settleInvoiceStub = sinon.stub(lnService, 'settleHodlInvoice')
      getInvoiceStub.returns({ ...lnServiceInvoice, is_confirmed: false, is_held: true })

      // used to generate a random secret, but we need this to be predictable
      randomBytesStub = sinon.stub(crypto, 'randomBytes').callThrough()
      randomBytesStub.withArgs(32).returns(Buffer.from(invoice.secret, 'hex'))
    })

    afterEach(() => {
      getInvoiceStub.restore()
      settleInvoiceStub.restore()
      randomBytesStub.restore()
    })

    it('should return 402 with WWW-Authenticate header if made without LSAT', done => {
      request(apiServer)
        .post('/hash')
        .send({
          hash
        })
        .expect(402)
        .expect('WWW-Authenticate')
        .end((err, res) => {
          const challenge = res.header['www-authenticate']
          expect(challenge, 'No challenge found in the response header').to.exist
          const getLsat = () => Lsat.fromChallenge(challenge)
          expect(getLsat).to.not.throw()
          done()
        })
    })

    it('should return an LSAT challenge whose Identifier is the paymentHash secret', done => {
      request(apiServer)
        .post('/hash')
        .send({
          hash
        })
        .expect(402)
        .expect('WWW-Authenticate')
        .end((err, res) => {
          const challenge = res.header['www-authenticate']
          expect(challenge).to.exist
          const lsat = Lsat.fromChallenge(challenge)

          const identifier = Identifier.fromString(lsat.id)

          expect(identifier.tokenId.toString('hex'), 'LSAT id should be generated from the invoice secret').to.equal(
            invoice.secret
          )

          const hash = crypto
            .createHash('sha256')
            .update(identifier.tokenId)
            .digest('hex')

          expect(hash, 'Hash of the LSAT id should be the invoice payment hash').to.equal(invoice.id)
          expect(hash, "Hash of the LSAT id should be the Lsat's payment hash").to.equal(invoice.id)
          done()
        })
    })

    it('should return a 402 response if request is made for a session with an unpaid invoice', done => {
      hashes.setLND({
        lookupInvoiceAsync: () => ({
          settled: false,
          state: 'OPEN',
          payment_request: invoice.payreq
        })
      })

      request(apiServer)
        .post('/hash')
        .send({
          hash
        })
        .set('Authorization', lsat.toToken())
        .expect(402)
        .end((_err, res) => {
          const getLsat = () => Lsat.fromChallenge(res.header['www-authenticate'])
          expect(getLsat, 'Should be able to retrieve lsat from challenge for an unpaid LSAT request').to.not.throw()
          const secondLsat = getLsat()
          expect(secondLsat.id).to.equal(lsat.id)
          expect(secondLsat.paymentHash).to.equal(lsat.paymentHash)
          done()
        })
    })

    it('should settle the associated invoice if request is made successfully', done => {
      // return a paid but held invoice when invoice status is checked in chainpoint
      hashes.setLND({
        lookupInvoiceAsync: () => ({
          settled: false,
          state: 'ACCEPTED',
          payment_request: invoice.payreq
        })
      })

      // make sure it's not called before running the submission
      expect(settleInvoiceStub.called, 'settle invoice should not have been called het').to.be.false

      // hash submission with an lsat and an invoice that is paid should settle invoice
      request(apiServer)
        .post('/hash')
        .send({
          hash
        })
        .set('Authorization', lsat.toToken())
        .expect(200)
        .end(() => {
          expect(settleInvoiceStub.called, 'boltwall should have settled invoice for satisfied lsat').to.be.true
          expect(settleInvoiceStub.getCall(0).args[0].secret).to.equal(invoice.secret)
          done()
        })
    })

    it('should fail when submitted with a settled LSAT invoice', done => {
      hashes.setLND({
        lookupInvoiceAsync: () => ({
          settled: true,
          state: 'SETTLED',
          payment_request: invoice.payreq
        })
      })
      getInvoiceStub.returns({ ...lnServiceInvoice, is_confirmed: true, is_held: false })

      request(apiServer)
        .post('/hash')
        .send({ hash })
        .set('Authorization', lsat.toToken())
        .expect(401)
        .end((err, res) => {
          expect(res.status, 'Should be unauthorized when using a settled LSAT').to.equal(401)
          done()
        })
    })

    it('should return proper result with valid call', done => {
      // return a paid but held invoice when invoice status is checked in chainpoint
      hashes.setLND({
        lookupInvoiceAsync: () => ({
          settled: false,
          state: 'ACCEPTED',
          payment_request: invoice.payreq
        })
      })

      request(apiServer)
        .post('/hash')
        .send({
          hash
        })
        .set('Authorization', lsat.toToken())
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res).to.have.property('body')
          expect(res.body)
            .to.have.property('hash')
            .and.to.equal(hash)
          expect(res.body).to.have.property('hash_received')
          expect(res.body).to.have.property('processing_hints')
          expect(res.body.processing_hints)
            .to.have.property('cal')
            .and.to.be.a('string')
          expect(res.body.processing_hints)
            .to.have.property('btc')
            .and.to.be.a('string')
          expect(res.body).to.have.property('proof_id')
          expect(res.body).to.have.property('hash_received')
          // The UUID timestamp has ms level precision, ISO8601 only to the second.
          // Check that they are within 1000ms of each other.
          expect(parseInt(uuidTime.v1(res.body.proof_id), 10) - Date.parse(res.body.hash_received)).to.be.within(
            0,
            1000
          )
          done()
        })
    })

    it('should return a v1 UUID node embedded with a partial SHA256 over timestamp and hash', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(apiServer)
        .post('/hash')
        .send({
          hash
        })
        .set('Authorization', lsat.toToken())
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
})

describe('Functions', () => {
  describe('calling generatePostHashResponse with one hash', () => {
    it('should return proper response object', done => {
      let res = hashes.generatePostHashResponse('ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12')
      expect(res).to.have.property('proof_id')
      expect(res)
        .to.have.property('hash')
        .and.to.equal('ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12')
      expect(res).to.have.property('hash_received')
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
