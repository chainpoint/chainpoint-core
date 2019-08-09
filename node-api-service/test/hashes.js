/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const hashes = require('../lib/endpoints/hashes.js')
const uuidTime = require('uuid-time')
const BLAKE2s = require('blake2s-js')

describe('Hashes Controller', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(false)
    hashes.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.100' })
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('POST /hash', () => {
    it('should return proper error with invalid content type', done => {
      request(insecureServer)
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
  })

  describe('POST /hash', () => {
    it('should return proper error with missing hash', done => {
      request(insecureServer)
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
  })

  describe('POST /hash', () => {
    it('should return proper error with hash not a string', done => {
      request(insecureServer)
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
  })

  describe('POST /hash', () => {
    it('should return proper error with invalid hash', done => {
      request(insecureServer)
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
  })

  describe('POST /hash', () => {
    it('should return proper error with missing invoice_id', done => {
      request(insecureServer)
        .post('/hash')
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
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
            .and.to.equal('invalid JSON body: missing invoice_id')
          done()
        })
    })
  })

  describe('POST /hash', () => {
    it('should return proper error with invoice_id not a string', done => {
      request(insecureServer)
        .post('/hash')
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12', invoice_id: [1] })
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
            .and.to.equal('invalid JSON body: bad invoice_id submitted')
          done()
        })
    })
  })

  describe('POST /hash', () => {
    it('should return proper error with invalid invoice_id', done => {
      request(insecureServer)
        .post('/hash')
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12', invoice_id: 'deadbeef' })
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
            .and.to.equal('invalid JSON body: bad invoice_id submitted')
          done()
        })
    })
  })

  describe('POST /hash', () => {
    before(() => {
      hashes.setRedis({
        get: () => null
      })
    })
    it('should return proper error with unpaid invoice_id', done => {
      let invoiceId = 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
      request(insecureServer)
        .post('/hash')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          invoice_id: invoiceId
        })
        .expect('Content-type', /json/)
        .expect(402)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('PaymentRequired')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal(`invoice ${invoiceId} has not been paid`)
          done()
        })
    })
  })

  describe('POST /hash', () => {
    before(() => {
      hashes.setRedis({
        get: () => '1'
      })
    })
    it('should return proper error with no AMQP connection', done => {
      request(insecureServer)
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
    before(() => {
      hashes.setRedis({
        get: () => '1',
        del: () => null
      })
    })
    it('should return a matched set of metadata and UUID embedded timestamps', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(insecureServer)
        .post('/hash')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          invoice_id: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
        })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('hash_id')
          expect(res.body).to.have.property('submitted_at')
          // The UUID timestamp has ms level precision, ISO8601 only to the second.
          // Check that they are within 1000ms of each other.
          expect(parseInt(uuidTime.v1(res.body.hash_id)) - Date.parse(res.body.submitted_at)).to.be.within(0, 1000)
          done()
        })
    })
  })

  describe('POST /hash', () => {
    before(() => {
      hashes.setRedis({
        get: () => '1',
        del: () => null
      })
    })
    it('should return a v1 UUID node embedded with a partial SHA256 over timestamp and hash', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(insecureServer)
        .post('/hash')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          invoice_id: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
        })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('hash_id')
          // Knowing the original hash, the timestamp from the UUID,
          // and the personalization bytes,
          // you should be able to calculate whether the UUID 'Node ID'
          // data segment is the 5 byte BLAKE2s hash of the timestamp
          // embedded in the UUID and the hash submitted to get this UUID.
          let t = parseInt(uuidTime.v1(res.body.hash_id))

          // 5 byte length BLAKE2s hash w/ personalization
          let h = new BLAKE2s(5, { personalization: Buffer.from('CHAINPNT') })
          let hashStr = [t.toString(), t.toString().length, res.body.hash, res.body.hash.length].join(':')

          h.update(Buffer.from(hashStr))
          let shortHashNodeBuf = Buffer.concat([Buffer.from([0x01]), h.digest()])
          // Last segment of UUIDv1 contains BLAKE2s hash to be matched
          expect(res.body.hash_id.split('-')[4]).to.equal(shortHashNodeBuf.toString('hex'))
          done()
        })
    })
  })

  describe('POST /hash', () => {
    before(() => {
      hashes.setRedis({
        get: () => '1',
        del: () => null
      })
    })
    it('should return proper result with valid call', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(insecureServer)
        .post('/hash')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          invoice_id: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
        })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res).to.have.property('body')
          expect(res.body).to.have.property('hash_id')
          expect(res.body)
            .to.have.property('hash')
            .and.to.equal('ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12')
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
  })
})

describe('Functions', () => {
  describe('calling generatePostHashResponse with one hash', () => {
    it('should return proper response object', done => {
      let res = hashes.generatePostHashResponse('ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12')
      expect(res).to.have.property('hash_id')
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
