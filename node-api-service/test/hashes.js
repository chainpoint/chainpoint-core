/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const hashes = require('../lib/endpoints/hashes.js')
const uuidTime = require('uuid-time')
const BLAKE2s = require('blake2s-js')

describe('Hashes Controller - Public Mode', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(false)
    hashes.setENV({ PRIVATE_NETWORK: false, CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.100' })
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

  describe('POST /hashes with unknown JWK', () => {
    before(() => {
      hashes.setRedis({
        get: async () => null
      })
      hashes.setRP(async () => {
        return { body: null }
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
          done()
        })
    })
  })

  describe('POST /hashes with bad sig', () => {
    let cache = null
    let jwk = {
      kty: 'EC',
      kid: '-I1DHE7VP3w7MW3xFEz6x8uXz0ucJ9y-2ukFzhYlAOc',
      crv: 'P-256',
      x: 'bnLffIyK_4cAitZsk38CmpTzSNbKduzzxfWZgspdRcU',
      y: 'Rx27copGfpqO5wswOMQzF2aXYzgBJeVe1hGvO0HtK_0'
    }
    let jwkStr = JSON.stringify(jwk)
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6Ii1JMURIRTdWUDN3N01XM3hGRXo2eDh1WHowdWNKOXktMnVrRnpoWWxBT2MifQ.eyJqdGkiOiI5MjhkMDAwMC02Yjg0LTExZTktYTQzYS03ZjQ1ZTk1MGU0ZjMiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI1LjE1NC4yMS4xMSIsImV4cCI6MTU1NjY1ODg3NiwiYmFsIjoyNywiaWF0IjoxNTU2NjU1Mjc1fQ.uQobSDBxbLOTmmgdipggAu7xzZlhu-SKHqppVxM5SyM_1tQaSQgUPo7VrDu4bFHUVR7AbP_ejM5HigtXE6ferw'
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
            .and.to.equal('invalid request, token signature cannot be verified')
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /hashes cant determine Node IP', () => {
    let jwk = {
      kty: 'EC',
      kid: 'wIf_pOR22DMLqhDqqlUxA2bC0kqpndXh58O6dguUoVY',
      crv: 'P-256',
      x: '2uX3pg85W7sEvvvml-1pWhdZ6FfyXhUWSYbuzkjz5mo',
      y: 'EfT6n1A7g9hcwKh_TL3-iWim7PlxvZfO1SsM68duXBc'
    }
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async () => null
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      hashes.setGetIP(() => null)
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IndJZl9wT1IyMkRNTHFoRHFxbFV4QTJiQzBrcXBuZFhoNThPNmRndVVvVlkifQ.eyJqdGkiOiJkZGFlYTcxMC02Y2Q3LTExZTktYTZjYy1iMTFlZWE2ZTA3MGYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImJhbCI6MTAsImlhdCI6MTU1NjgwMTAwMH0.rYkUwEfLVgDK14nd9sAzCVQwI1WwKDwpjKwVMXudRpxjlksNuKSJ_SXe4vw6SKKRw3PRv9b_gLpTfgrPOlMaBw'
        })
        .expect('Content-type', /json/)
        .expect(400)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('BadRequest')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('bad request, unable to determine Node IP')
          done()
        })
    })
  })

  describe('POST /hashes sub missing', () => {
    let jwk = {
      kty: 'EC',
      kid: 'm2Z7IaIP6L747gKRsa9M6Bm2LJSbOpSouJF20pW9LRQ',
      crv: 'P-256',
      x: '_LjJFmsZBrct978_ojhXWyklogCaYmvZKT0sC4JPvmY',
      y: 'JhaiIVpJ8RlLQzyksVz8oQXrJRzATzLX88XPIixsJg8'
    }
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async () => null
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      hashes.setGetIP(() => '66.12.12.12')
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6Im0yWjdJYUlQNkw3NDdnS1JzYTlNNkJtMkxKU2JPcFNvdUpGMjBwVzlMUlEifQ.eyJqdGkiOiJhOWE4MDNmMC02ZDA3LTExZTktYWFjZC02OWMzOWFhNTRhYjIiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsImV4cCI6MTg3MjE4MTUzMCwiYmFsIjoxMCwiaWF0IjoxNTU2ODIxNTI5fQ.8VeFFh7SNCfCJlSOyovVmeNyaAdgh7V_OZTVvutdRD1y9_5JmPOvuo2xQTRLUAwFZfVWYVOms399TiosmZARww'
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
            .and.to.equal('invalid request, token missing `sub` value')
          done()
        })
    })
  })

  describe('POST /hashes Node IP and JWT sub do not match', () => {
    let jwk = {
      kty: 'EC',
      kid: 'wIf_pOR22DMLqhDqqlUxA2bC0kqpndXh58O6dguUoVY',
      crv: 'P-256',
      x: '2uX3pg85W7sEvvvml-1pWhdZ6FfyXhUWSYbuzkjz5mo',
      y: 'EfT6n1A7g9hcwKh_TL3-iWim7PlxvZfO1SsM68duXBc'
    }
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async () => null
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      hashes.setGetIP(() => '66.12.12.12')
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IndJZl9wT1IyMkRNTHFoRHFxbFV4QTJiQzBrcXBuZFhoNThPNmRndVVvVlkifQ.eyJqdGkiOiJkZGFlYTcxMC02Y2Q3LTExZTktYTZjYy1iMTFlZWE2ZTA3MGYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImJhbCI6MTAsImlhdCI6MTU1NjgwMTAwMH0.rYkUwEfLVgDK14nd9sAzCVQwI1WwKDwpjKwVMXudRpxjlksNuKSJ_SXe4vw6SKKRw3PRv9b_gLpTfgrPOlMaBw'
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
            .and.to.equal('invalid request, token subject does not match Node IP')
          done()
        })
    })
  })

  describe('POST /hashes exp missing', () => {
    let cache = null
    let jwk = {
      kty: 'EC',
      kid: 'wIf_pOR22DMLqhDqqlUxA2bC0kqpndXh58O6dguUoVY',
      crv: 'P-256',
      x: '2uX3pg85W7sEvvvml-1pWhdZ6FfyXhUWSYbuzkjz5mo',
      y: 'EfT6n1A7g9hcwKh_TL3-iWim7PlxvZfO1SsM68duXBc'
    }
    let jwkStr = JSON.stringify(jwk)
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      hashes.setGetIP(() => '24.154.21.11')
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IndJZl9wT1IyMkRNTHFoRHFxbFV4QTJiQzBrcXBuZFhoNThPNmRndVVvVlkifQ.eyJqdGkiOiJkZGFlYTcxMC02Y2Q3LTExZTktYTZjYy1iMTFlZWE2ZTA3MGYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImJhbCI6MTAsImlhdCI6MTU1NjgwMTAwMH0.rYkUwEfLVgDK14nd9sAzCVQwI1WwKDwpjKwVMXudRpxjlksNuKSJ_SXe4vw6SKKRw3PRv9b_gLpTfgrPOlMaBw'
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
            .and.to.equal('invalid request, token missing `exp` value')
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /hashes expired token', () => {
    let cache = null
    let jwk = {
      kty: 'EC',
      kid: 'MCtHoDIYJj3hW7Dhte0drRYP31azdH8wB8Ml-7TbQzg',
      crv: 'P-256',
      x: 'StbDPZ0dU6cNQ6y9cfcYiMF97I6vqxSKtd6e8gboaSY',
      y: 'JaSI56M0dAg1JJke9a-bYBhA18PzZ6DBS9hbJcX9Gvs'
    }
    let jwkStr = JSON.stringify(jwk)
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      hashes.setGetIP(() => '24.154.21.11')
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6Ik1DdEhvRElZSmozaFc3RGh0ZTBkclJZUDMxYXpkSDh3QjhNbC03VGJRemcifQ.eyJqdGkiOiI0MTFkODQ2MC02Y2Q4LTExZTktYmMxMC1mMzYyYzYyMTkxZWMiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjgwMTE2OCwiYmFsIjoxMCwiaWF0IjoxNTU2ODAxMTY3fQ.u-K-UibbYBzeCPL_Ge9qBiwXH8idWLvc5AaTpqpHxHCLo5p2UnoPMtjMk5rLDDom93BilIUsV17hfqgT7UfAAQ'
        })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('Unauthorized')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('not authorized, token has expired')
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /hashes token missing `aud` value', () => {
    let cache = null
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
    }
    let jwkStr = JSON.stringify(jwk)
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      hashes.setGetIP(() => '66.12.12.12')
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiJiYWNiOTY3MC03NjcwLTExZTktOTU0My0xYjA3ZDQ1NDUyODAiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjE2MjE2LCJiYWwiOjEwLCJpYXQiOjE1NTc4NTYyMTV9.RQdN64rkCTVPjk_zS5Ap9nlq2VMtkVY_Qy8N6Y_yjNj-g1GhyWCV1YmJ1pED0cFOXQ2uQ1C6jIzeo5XVlHPAEg'
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
            .and.to.equal('invalid request, token missing `aud` value')
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /hashes aud must contain 3 values', () => {
    let cache = null
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
    }
    let jwkStr = JSON.stringify(jwk)
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      hashes.setGetIP(() => '66.12.12.12')
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiJmYjgwNjA2MC03NjcwLTExZTktYTk1Yy0yMTY5MWRmNDRjYjkiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjE2MzI0LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSIsImlhdCI6MTU1Nzg1NjMyM30.j0OLuvrc-y67daAZfxYaxLD_vP3rSX8firO2j03GkDTEOodbRNTDTLaf_4asnZYlAWJb-leTLd9PWRDiJpBwQQ'
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
            .and.to.equal('invalid request, aud must contain 3 values')
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /hashes bad IP value in aud', () => {
    let cache = null
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
    }
    let jwkStr = JSON.stringify(jwk)
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      hashes.setGetIP(() => '66.12.12.12')
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIxMmRkNWQzMC03NjcxLTExZTktOTMyYS00NTA3YjBjYzhkMTQiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjE2MzY0LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSxiYWRpcCIsImlhdCI6MTU1Nzg1NjM2M30.ovxjzztWOayd1-vzdQS-BYMc8WQ-oMMAjTOyz9ZtWBL-A-0BzDlDAAxeKm7xSIimI8d7naiqi-uCt928CRng7w'
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
            .and.to.equal(`invalid request, bad IP value in aud - badip`)
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /hashes aud must include this Core IP', () => {
    let cache = null
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
    }
    let jwkStr = JSON.stringify(jwk)
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      hashes.setGetIP(() => '66.12.12.12')
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIyYTAzOTI5MC03NjcxLTExZTktOTc2Yy0wYmQyOGVlNTc4ZDYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjE2NDAyLCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMSIsImlhdCI6MTU1Nzg1NjQwMX0.0Gk3D78Nydg2p6jkdIBvLFKb-QURCGlCWtJXFCM1RXID9xR2Hz9dzIKcLfqJLqr7c_ul6qO68ys2VPqpP2ccKg'
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
            .and.to.equal('invalid request, aud must include this Core IP')
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /hashes', () => {
    let cache = null
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
    }
    let jwkStr = JSON.stringify(jwk)
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      hashes.setGetIP(() => '66.12.12.12')
    })
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
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /hashes', () => {
    let cache = null
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
    }
    let jwkStr = JSON.stringify(jwk)
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      hashes.setGetIP(() => '66.12.12.12')
    })
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
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /hashes', () => {
    let cache = null
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
    }
    let jwkStr = JSON.stringify(jwk)
    before(() => {
      hashes.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      hashes.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      hashes.setGetIP(() => '66.12.12.12')
    })
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
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })
})

describe('Hashes Controller - Private Mode', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(true)
    hashes.setENV({ PRIVATE_NETWORK: true })
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('POST /hashes', () => {
    it('should return a matched set of metadata and UUID embedded timestamps', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(insecureServer)
        .post('/hashes')
        .send({
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
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
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
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
          hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12'
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
