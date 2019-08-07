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

  describe('POST /hashes', () => {
    it('should return proper error with invalid content type', done => {
      request(insecureServer)
        .post('/hashes')
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

  describe('POST /hashes', () => {
    it('should return proper error with missing hash', done => {
      request(insecureServer)
        .post('/hashes')
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

  describe('POST /hashes', () => {
    it('should return proper error with hash not a string', done => {
      request(insecureServer)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
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

  describe('POST /hashes', () => {
    it('should return proper error with invalid hash', done => {
      request(insecureServer)
        .post('/hashes')
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

  describe('POST /hashes', () => {
    it('should return proper error with no AMQP connection', done => {
      request(insecureServer)
        .post('/hashes')
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
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

  describe('POST /hashes with no token', () => {
    it('should return proper error', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })
      request(insecureServer)
        .post('/hashes')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
        })
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
            .and.to.equal('invalid request, token must be supplied')
          done()
        })
    })
  })

  describe('POST /hashes with bad token data', () => {
    it('should return proper error', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })
      request(insecureServer)
        .post('/hashes')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          token: 'qweqweqwe'
        })
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
            .and.to.equal('invalid request, token cannot be decoded')
          done()
        })
    })
  })

  describe('POST /hashes with missing kid', () => {
    it('should return proper error', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })
      request(insecureServer)
        .post('/hashes')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIzNDFmODNkMC02YjdmLTExZTktYWM3Ni1kNTcyYjBmMzllNDgiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjY1NjU3MCwiYmFsIjoyNywiaWF0IjoxNTU2NjUyOTY5fQ.3sAXn6X7qhMXAriDBr470ciqyKTADeplWUN4skvscE9MaNkj6DtXWSw0ZujUqPwlpmAF3mq4kbJn-7SEXUa4JQ'
        })
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
            .and.to.equal('invalid request, token missing `kid` value')
          done()
        })
    })
  })

  describe('POST /hashes with missing iss', () => {
    it('should return proper error', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })
      request(insecureServer)
        .post('/hashes')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IjU1OTI5ODQwLTZiN2YtMTFlOS04Yzk2LWM3NDY4MGE4MDQ4ZiJ9.eyJqdGkiOiI1NTkxYWRlMC02YjdmLTExZTktOGM5Ni1jNzQ2ODBhODA0OGYiLCJzdWIiOiIyNC4xNTQuMjEuMTEiLCJleHAiOjE1NTY2NTY2MjYsImJhbCI6MjcsImlhdCI6MTU1NjY1MzAyNX0.-JbQlyGo7cy5iWJTZhRizjndltpTFbxJCJoSOI5SVtepJuCp5SWHbdL7xdhWE78oKCpy6nVk5IjKTZUCVnK0cQ'
        })
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
            .and.to.equal('invalid request, token missing `iss` value')
          done()
        })
    })
  })

  describe('POST /hashes with non-cached non-peer iss', () => {
    before(() => {
      hashes.setRedis({
        get: async () => null
      })
      hashes.setSC({
        hasMemberIPAsync: async () => false
      })
    })
    it('should return proper error', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })
      request(insecureServer)
        .post('/hashes')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImYxYTg1M2YwLTZiN2YtMTFlOS04MmViLWNiMDRiOGFlYjI4NiJ9.eyJqdGkiOiJmMWE3Njk5MC02YjdmLTExZTktODJlYi1jYjA0YjhhZWIyODYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjY1Njg4OCwiYmFsIjoyNywiaWF0IjoxNTU2NjUzMjg3fQ.rqOklC2mhxWcYyLnfE9jOfr1i7Nx4uIVC7S5AszqxfkLIjts7eniSF1gyvvqZ4BkEvn0qROP9QcwPjUCD5_BaA'
        })
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
            .and.to.equal('invalid request, `iss` not a known network peer')
          done()
        })
    })
  })

  describe('POST /hashes with cached non-peer iss', () => {
    before(() => {
      hashes.setRedis({
        get: async () => 'false'
      })
      hashes.setSC({
        hasMemberIPAsync: async () => false
      })
    })
    it('should return proper error', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })
      request(insecureServer)
        .post('/hashes')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImYxYTg1M2YwLTZiN2YtMTFlOS04MmViLWNiMDRiOGFlYjI4NiJ9.eyJqdGkiOiJmMWE3Njk5MC02YjdmLTExZTktODJlYi1jYjA0YjhhZWIyODYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjY1Njg4OCwiYmFsIjoyNywiaWF0IjoxNTU2NjUzMjg3fQ.rqOklC2mhxWcYyLnfE9jOfr1i7Nx4uIVC7S5AszqxfkLIjts7eniSF1gyvvqZ4BkEvn0qROP9QcwPjUCD5_BaA'
        })
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
            .and.to.equal('invalid request, `iss` not a known network peer')
          done()
        })
    })
  })

  describe('POST /hashes with non-cached peer iss', () => {
    let cache = ''
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async (key, val) => {
          cache = val
        }
      })
      hashes.setSC({
        hasMemberIPAsync: async () => true
      })
    })
    it('should return proper error', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })
      request(insecureServer)
        .post('/hashes')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImYxYTg1M2YwLTZiN2YtMTFlOS04MmViLWNiMDRiOGFlYjI4NiJ9.eyJqdGkiOiJmMWE3Njk5MC02YjdmLTExZTktODJlYi1jYjA0YjhhZWIyODYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjY1Njg4OCwiYmFsIjoyNywiaWF0IjoxNTU2NjUzMjg3fQ.rqOklC2mhxWcYyLnfE9jOfr1i7Nx4uIVC7S5AszqxfkLIjts7eniSF1gyvvqZ4BkEvn0qROP9QcwPjUCD5_BaA'
        })
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
            .and.to.equal('invalid request, unable to find public key for given kid')
          expect(cache).to.equal(true)
          done()
        })
    })
  })

  describe('POST /hashes', () => {
    it('should return a matched set of metadata and UUID embedded timestamps', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(insecureServer)
        .post('/hashes')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiJhNWRhNTc5MC03NjcyLTExZTktODE1Mi1lOTA3YWYzZjRhY2EiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjE3MDQwLCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiaWF0IjoxNTU3ODU3MDM5fQ.nXUiZ_FzWMZGIgHlgf92Jx0jMhPnFfAundv0USEqQKqceeKoXI6YmtgjzuVXoYcmK0b9r6I0vN_20b3mK8oX9w'
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

  describe('POST /hashes', () => {
    it('should return a v1 UUID node embedded with a partial SHA256 over timestamp and hash', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(insecureServer)
        .post('/hashes')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiJhNWRhNTc5MC03NjcyLTExZTktODE1Mi1lOTA3YWYzZjRhY2EiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjE3MDQwLCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiaWF0IjoxNTU3ODU3MDM5fQ.nXUiZ_FzWMZGIgHlgf92Jx0jMhPnFfAundv0USEqQKqceeKoXI6YmtgjzuVXoYcmK0b9r6I0vN_20b3mK8oX9w'
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

  describe('POST /hashes', () => {
    it('should return proper result with valid call', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(insecureServer)
        .post('/hashes')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiJhNWRhNTc5MC03NjcyLTExZTktODE1Mi1lOTA3YWYzZjRhY2EiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjE3MDQwLCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiaWF0IjoxNTU3ODU3MDM5fQ.nXUiZ_FzWMZGIgHlgf92Jx0jMhPnFfAundv0USEqQKqceeKoXI6YmtgjzuVXoYcmK0b9r6I0vN_20b3mK8oX9w'
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
