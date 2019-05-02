/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const usageToken = require('../lib/endpoints/usage-token.js')

describe('Usage Token Controller', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync()
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('POST /usagetoken/refresh with no token', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
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

  describe('POST /usagetoken/refresh with bad token data', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({ token: 'qweqweqwe' })
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

  describe('POST /usagetoken/refresh with missing kid', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
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

  describe('POST /usagetoken/refresh with missing iss', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
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

  describe('POST /usagetoken/refresh with unknown JWK', () => {
    before(() => {
      usageToken.setRedis({
        get: async () => null
      })
      usageToken.setRP(async () => {
        return { body: null }
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
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

  describe('POST /usagetoken/refresh with bad sig', () => {
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
      usageToken.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
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

  describe('POST /usagetoken/refresh cant determine Node IP', () => {
    let jwk = {
      kty: 'EC',
      kid: '6zwQfNTgA8uEN1ufatlq06VIexOZ8Z1_rOUnsvUBrr4',
      crv: 'P-256',
      x: 'YHkAzNJP6Ro8HtX5BBkVJdsNqsgE-EZIij1OZQMBwWA',
      y: '9sgsYrTvZ7mEF5Bg5dwbseOU2EBij5elLzb-4mv6iHE'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async () => null
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setGetIP(() => null)
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IjZ6d1FmTlRnQTh1RU4xdWZhdGxxMDZWSWV4T1o4WjFfck9VbnN2VUJycjQifQ.eyJqdGkiOiI3YjU3ZTExMC02Yjg1LTExZTktOGQxOC1kMTU1MjkwZTU5YjMiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjY1OTI2NiwiYmFsIjowLCJpYXQiOjE1NTY2NTU2NjV9.6BMhmmSTva9HJklAPeeCf56gnwTIK31m28GtydhKexh3p9NfaMy4r-TZioILh5Tol0wOwa0vxx7WOGGg5lrZqg'
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

  describe('POST /usagetoken/refresh sub missing', () => {
    let jwk = {
      kty: 'EC',
      kid: 'm2Z7IaIP6L747gKRsa9M6Bm2LJSbOpSouJF20pW9LRQ',
      crv: 'P-256',
      x: '_LjJFmsZBrct978_ojhXWyklogCaYmvZKT0sC4JPvmY',
      y: 'JhaiIVpJ8RlLQzyksVz8oQXrJRzATzLX88XPIixsJg8'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async () => null
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setGetIP(() => '66.12.12.12')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
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

  describe('POST /usagetoken/refresh Node IP and JWT sub do not match', () => {
    let jwk = {
      kty: 'EC',
      kid: '6zwQfNTgA8uEN1ufatlq06VIexOZ8Z1_rOUnsvUBrr4',
      crv: 'P-256',
      x: 'YHkAzNJP6Ro8HtX5BBkVJdsNqsgE-EZIij1OZQMBwWA',
      y: '9sgsYrTvZ7mEF5Bg5dwbseOU2EBij5elLzb-4mv6iHE'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async () => null
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setGetIP(() => '66.12.12.12')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IjZ6d1FmTlRnQTh1RU4xdWZhdGxxMDZWSWV4T1o4WjFfck9VbnN2VUJycjQifQ.eyJqdGkiOiI3YjU3ZTExMC02Yjg1LTExZTktOGQxOC1kMTU1MjkwZTU5YjMiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjY1OTI2NiwiYmFsIjowLCJpYXQiOjE1NTY2NTU2NjV9.6BMhmmSTva9HJklAPeeCf56gnwTIK31m28GtydhKexh3p9NfaMy4r-TZioILh5Tol0wOwa0vxx7WOGGg5lrZqg'
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

  describe('POST /usagetoken/refresh with 0 balance', () => {
    let cache = null
    let jwk = {
      kty: 'EC',
      kid: '6zwQfNTgA8uEN1ufatlq06VIexOZ8Z1_rOUnsvUBrr4',
      crv: 'P-256',
      x: 'YHkAzNJP6Ro8HtX5BBkVJdsNqsgE-EZIij1OZQMBwWA',
      y: '9sgsYrTvZ7mEF5Bg5dwbseOU2EBij5elLzb-4mv6iHE'
    }
    let jwkStr = JSON.stringify(jwk)
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setGetIP(() => '24.154.21.11')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IjZ6d1FmTlRnQTh1RU4xdWZhdGxxMDZWSWV4T1o4WjFfck9VbnN2VUJycjQifQ.eyJqdGkiOiI3YjU3ZTExMC02Yjg1LTExZTktOGQxOC1kMTU1MjkwZTU5YjMiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjY1OTI2NiwiYmFsIjowLCJpYXQiOjE1NTY2NTU2NjV9.6BMhmmSTva9HJklAPeeCf56gnwTIK31m28GtydhKexh3p9NfaMy4r-TZioILh5Tol0wOwa0vxx7WOGGg5lrZqg'
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
            .and.to.equal('invalid request, token with 0 balance cannot be refreshed')
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /usagetoken/refresh with no active tokens', () => {
    let jwk = {
      kty: 'EC',
      kid: 'eID3NTfsmZvVxbbe4e5l5PvfDrNcwFO8Ty5yrybq-Og',
      crv: 'P-256',
      x: 'cLpdT8KTlI7H9mkBX18UjWPrbABa117h6ECw3BFlv8A',
      y: 'DTUHigEeqsQ-zWuCOYHgU5QOpKgPPsqNGITkAT-7lSI'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async () => null
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setAT({
        getActiveTokenByNodeIPAsync: async () => null
      })
      usageToken.setGetIP(() => '24.154.21.11')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImVJRDNOVGZzbVp2VnhiYmU0ZTVsNVB2ZkRyTmN3Rk84VHk1eXJ5YnEtT2cifQ.eyJqdGkiOiIzYzY4MDgyMC02YzEzLTExZTktOTY5Ny0yNWRkMzEwNGJkZDQiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjcyMDE0OSwiYmFsIjoxMCwiaWF0IjoxNTU2NzE2NTQ4fQ.sh0W19XH52qDGMWNvXfa53njPvn93W9VNO-RH1uWQzee7a_ncTmnbARPESNG9S1SRYkjES6GzOF1xgcfqu7j8w'
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
            .and.to.equal('invalid request, no active token available to be refreshed')
          done()
        })
    })
  })

  describe('POST /usagetoken/refresh - cant read active tokens', () => {
    let jwk = {
      kty: 'EC',
      kid: 'eID3NTfsmZvVxbbe4e5l5PvfDrNcwFO8Ty5yrybq-Og',
      crv: 'P-256',
      x: 'cLpdT8KTlI7H9mkBX18UjWPrbABa117h6ECw3BFlv8A',
      y: 'DTUHigEeqsQ-zWuCOYHgU5QOpKgPPsqNGITkAT-7lSI'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async () => null
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setAT({
        getActiveTokenByNodeIPAsync: async () => {
          throw new Error()
        }
      })
      usageToken.setGetIP(() => '24.154.21.11')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImVJRDNOVGZzbVp2VnhiYmU0ZTVsNVB2ZkRyTmN3Rk84VHk1eXJ5YnEtT2cifQ.eyJqdGkiOiIzYzY4MDgyMC02YzEzLTExZTktOTY5Ny0yNWRkMzEwNGJkZDQiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjcyMDE0OSwiYmFsIjoxMCwiaWF0IjoxNTU2NzE2NTQ4fQ.sh0W19XH52qDGMWNvXfa53njPvn93W9VNO-RH1uWQzee7a_ncTmnbARPESNG9S1SRYkjES6GzOF1xgcfqu7j8w'
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
            .and.to.equal('server error, unable to read active token data')
          done()
        })
    })
  })

  describe('POST /usagetoken/refresh with wrong active token', () => {
    let jwk = {
      kty: 'EC',
      kid: 'eID3NTfsmZvVxbbe4e5l5PvfDrNcwFO8Ty5yrybq-Og',
      crv: 'P-256',
      x: 'cLpdT8KTlI7H9mkBX18UjWPrbABa117h6ECw3BFlv8A',
      y: 'DTUHigEeqsQ-zWuCOYHgU5QOpKgPPsqNGITkAT-7lSI'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async () => null
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setAT({
        getActiveTokenByNodeIPAsync: async () => {
          return { tokenHash: '18ee24150dcb1d96752a4d6dd0f20dfd8ba8c38527e40aa8509b7adecf78f9c6' }
        }
      })
      usageToken.setGetIP(() => '24.154.21.11')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImVJRDNOVGZzbVp2VnhiYmU0ZTVsNVB2ZkRyTmN3Rk84VHk1eXJ5YnEtT2cifQ.eyJqdGkiOiIzYzY4MDgyMC02YzEzLTExZTktOTY5Ny0yNWRkMzEwNGJkZDQiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjcyMDE0OSwiYmFsIjoxMCwiaWF0IjoxNTU2NzE2NTQ4fQ.sh0W19XH52qDGMWNvXfa53njPvn93W9VNO-RH1uWQzee7a_ncTmnbARPESNG9S1SRYkjES6GzOF1xgcfqu7j8w'
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
            .and.to.equal('invalid request, supplied token is not an active token')
          done()
        })
    })
  })

  describe('POST /usagetoken/refresh with bad PEM in env, cant sign', () => {
    let jwk = {
      kty: 'EC',
      kid: 'eID3NTfsmZvVxbbe4e5l5PvfDrNcwFO8Ty5yrybq-Og',
      crv: 'P-256',
      x: 'cLpdT8KTlI7H9mkBX18UjWPrbABa117h6ECw3BFlv8A',
      y: 'DTUHigEeqsQ-zWuCOYHgU5QOpKgPPsqNGITkAT-7lSI'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async () => null
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setAT({
        getActiveTokenByNodeIPAsync: async () => {
          return { tokenHash: 'dd8ebea5abbd264b2098caa5b7aef92e899859c67cbfd2d2ed8c655f7d462171' }
        }
      })
      usageToken.setENV({
        ECDSA_PKPEM: 'badPEM'
      })
      usageToken.setGetIP(() => '24.154.21.11')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImVJRDNOVGZzbVp2VnhiYmU0ZTVsNVB2ZkRyTmN3Rk84VHk1eXJ5YnEtT2cifQ.eyJqdGkiOiIzYzY4MDgyMC02YzEzLTExZTktOTY5Ny0yNWRkMzEwNGJkZDQiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjcyMDE0OSwiYmFsIjoxMCwiaWF0IjoxNTU2NzE2NTQ4fQ.sh0W19XH52qDGMWNvXfa53njPvn93W9VNO-RH1uWQzee7a_ncTmnbARPESNG9S1SRYkjES6GzOF1xgcfqu7j8w'
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
            .and.to.equal('server error, could not sign refreshed token')
          done()
        })
    })
  })

  describe('POST /usagetoken/refresh with broadcast error', () => {
    let coreIdCache = ''
    let jwk = {
      kty: 'EC',
      kid: 'eID3NTfsmZvVxbbe4e5l5PvfDrNcwFO8Ty5yrybq-Og',
      crv: 'P-256',
      x: 'cLpdT8KTlI7H9mkBX18UjWPrbABa117h6ECw3BFlv8A',
      y: 'DTUHigEeqsQ-zWuCOYHgU5QOpKgPPsqNGITkAT-7lSI'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async (k, v) => {
          coreIdCache = [k, v]
        }
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setAT({
        getActiveTokenByNodeIPAsync: async () => {
          return { tokenHash: 'dd8ebea5abbd264b2098caa5b7aef92e899859c67cbfd2d2ed8c655f7d462171' }
        }
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://localhost'
      })
      usageToken.setStatus({
        buildStatusObjectAsync: async () => {
          return { status: { node_info: { id: 'myId!' } } }
        }
      })
      usageToken.setTMRPC({
        broadcastTxAsync: async () => {
          throw new Error('tm error!')
        }
      })
      usageToken.setGetIP(() => '24.154.21.11')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImVJRDNOVGZzbVp2VnhiYmU0ZTVsNVB2ZkRyTmN3Rk84VHk1eXJ5YnEtT2cifQ.eyJqdGkiOiIzYzY4MDgyMC02YzEzLTExZTktOTY5Ny0yNWRkMzEwNGJkZDQiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjcyMDE0OSwiYmFsIjoxMCwiaWF0IjoxNTU2NzE2NTQ4fQ.sh0W19XH52qDGMWNvXfa53njPvn93W9VNO-RH1uWQzee7a_ncTmnbARPESNG9S1SRYkjES6GzOF1xgcfqu7j8w'
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
            .and.to.equal('server error, server error on transaction broadcast, tm error!')
          expect(coreIdCache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(coreIdCache[1]).to.equal('myId!')
          done()
        })
    })
  })

  describe('POST /usagetoken/refresh with broadcast error - bad token tx arg', () => {
    let coreIdCache = ''
    let jwk = {
      kty: 'EC',
      kid: 'eID3NTfsmZvVxbbe4e5l5PvfDrNcwFO8Ty5yrybq-Og',
      crv: 'P-256',
      x: 'cLpdT8KTlI7H9mkBX18UjWPrbABa117h6ECw3BFlv8A',
      y: 'DTUHigEeqsQ-zWuCOYHgU5QOpKgPPsqNGITkAT-7lSI'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async (k, v) => {
          coreIdCache = [k, v]
        }
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setAT({
        getActiveTokenByNodeIPAsync: async () => {
          return { tokenHash: 'dd8ebea5abbd264b2098caa5b7aef92e899859c67cbfd2d2ed8c655f7d462171' }
        }
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://localhost'
      })
      usageToken.setStatus({
        buildStatusObjectAsync: async () => {
          return { status: { node_info: { id: 'myId!' } } }
        }
      })
      usageToken.setTMRPC({
        broadcastTxAsync: async () => {
          return { error: { responseCode: 409, message: 'badarg' } }
        }
      })
      usageToken.setGetIP(() => '24.154.21.11')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImVJRDNOVGZzbVp2VnhiYmU0ZTVsNVB2ZkRyTmN3Rk84VHk1eXJ5YnEtT2cifQ.eyJqdGkiOiIzYzY4MDgyMC02YzEzLTExZTktOTY5Ny0yNWRkMzEwNGJkZDQiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjcyMDE0OSwiYmFsIjoxMCwiaWF0IjoxNTU2NzE2NTQ4fQ.sh0W19XH52qDGMWNvXfa53njPvn93W9VNO-RH1uWQzee7a_ncTmnbARPESNG9S1SRYkjES6GzOF1xgcfqu7j8w'
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
            .and.to.equal('server error, server error on transaction broadcast, badarg')
          expect(coreIdCache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(coreIdCache[1]).to.equal('myId!')
          done()
        })
    })
  })

  describe('POST /usagetoken/refresh with broadcast error - TM comm error', () => {
    let coreIdCache = ''
    let jwk = {
      kty: 'EC',
      kid: 'eID3NTfsmZvVxbbe4e5l5PvfDrNcwFO8Ty5yrybq-Og',
      crv: 'P-256',
      x: 'cLpdT8KTlI7H9mkBX18UjWPrbABa117h6ECw3BFlv8A',
      y: 'DTUHigEeqsQ-zWuCOYHgU5QOpKgPPsqNGITkAT-7lSI'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async (k, v) => {
          coreIdCache = [k, v]
        }
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setAT({
        getActiveTokenByNodeIPAsync: async () => {
          return { tokenHash: 'dd8ebea5abbd264b2098caa5b7aef92e899859c67cbfd2d2ed8c655f7d462171' }
        }
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://localhost'
      })
      usageToken.setStatus({
        buildStatusObjectAsync: async () => {
          return { status: { node_info: { id: 'myId!' } } }
        }
      })
      usageToken.setTMRPC({
        broadcastTxAsync: async () => {
          return { error: { responseCode: 500, message: 'commerr' } }
        }
      })
      usageToken.setGetIP(() => '24.154.21.11')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImVJRDNOVGZzbVp2VnhiYmU0ZTVsNVB2ZkRyTmN3Rk84VHk1eXJ5YnEtT2cifQ.eyJqdGkiOiIzYzY4MDgyMC02YzEzLTExZTktOTY5Ny0yNWRkMzEwNGJkZDQiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjcyMDE0OSwiYmFsIjoxMCwiaWF0IjoxNTU2NzE2NTQ4fQ.sh0W19XH52qDGMWNvXfa53njPvn93W9VNO-RH1uWQzee7a_ncTmnbARPESNG9S1SRYkjES6GzOF1xgcfqu7j8w'
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
            .and.to.equal('server error, server error on transaction broadcast, Could not broadcast transaction')
          expect(coreIdCache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(coreIdCache[1]).to.equal('myId!')
          done()
        })
    })
  })

  describe('POST /usagetoken/refresh success', () => {
    let jwk = {
      kty: 'EC',
      kid: 'eID3NTfsmZvVxbbe4e5l5PvfDrNcwFO8Ty5yrybq-Og',
      crv: 'P-256',
      x: 'cLpdT8KTlI7H9mkBX18UjWPrbABa117h6ECw3BFlv8A',
      y: 'DTUHigEeqsQ-zWuCOYHgU5QOpKgPPsqNGITkAT-7lSI'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async () => null
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setAT({
        getActiveTokenByNodeIPAsync: async () => {
          return { tokenHash: 'dd8ebea5abbd264b2098caa5b7aef92e899859c67cbfd2d2ed8c655f7d462171' }
        }
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----`,
        CHAINPOINT_CORE_BASE_URI: 'http://localhost'
      })
      usageToken.setStatus({
        buildStatusObjectAsync: async () => {
          return { status: { node_info: { id: 'myId!' } } }
        }
      })
      usageToken.setTMRPC({
        broadcastTxAsync: async () => {
          return {}
        }
      })
      usageToken.setGetIP(() => '24.154.21.11')
    })
    it('should return proper refresh token', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImVJRDNOVGZzbVp2VnhiYmU0ZTVsNVB2ZkRyTmN3Rk84VHk1eXJ5YnEtT2cifQ.eyJqdGkiOiIzYzY4MDgyMC02YzEzLTExZTktOTY5Ny0yNWRkMzEwNGJkZDQiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjI0LjE1NC4yMS4xMSIsImV4cCI6MTU1NjcyMDE0OSwiYmFsIjoxMCwiaWF0IjoxNTU2NzE2NTQ4fQ.sh0W19XH52qDGMWNvXfa53njPvn93W9VNO-RH1uWQzee7a_ncTmnbARPESNG9S1SRYkjES6GzOF1xgcfqu7j8w'
        })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('token')
            .and.to.be.a('string')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with no tx', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
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
            .and.to.equal('invalid request, tx must be supplied')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with bad tx no 0x', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({ tx: 'deadbeef' })
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
            .and.to.equal('invalid request, tx must begin with 0x')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with bad tx non hex', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({ tx: '0xnothex' })
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
            .and.to.equal('invalid request, non hex tx value supplied')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with bad tx invalid', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({ tx: '0xdeadbeefcafe' })
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
            .and.to.equal('invalid request, invalid ethereum tx body supplied')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with wrong contract address', () => {
    before(() => {
      usageToken.setTA('0xbadc0de1971122f30998bf2ba76d8cf19ff73ae1')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4bd6ff20b000000000000000000000000000000000000000000000000000000003b9aca001ca064d1e9fcbd45fb666996232481aea69e12b59982097deb8fe6632a06accf0632a032352440244d001856014b7381e7cf23ec51ef941388d30fabc9beb8fd65d1a8'
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
            .and.to.equal('invalid request, transaction must interact with Chainpoint token contract')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with wrong method call', () => {
    before(() => {
      usageToken.setTA('0x684e7D2B54D2fc9fef0138ce00702445cEAd9cEA')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          tx:
            '0xf8ac82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80b844a9059cbb0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003b9aca001ca0c34363aa26528b00c407cc04bf1b595afd87fc45c60e87be8a7387402b8094a1a0300c115718d7271b215e459c811167c318b3dfd0f4b56f0ccbb51054ee35727e'
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
            .and.to.equal(`invalid request, transaction may only make a call to 'purchaseUsage'`)
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with too low amount', () => {
    before(() => {
      usageToken.setTA('0x684e7D2B54D2fc9fef0138ce00702445cEAd9cEA')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4b55968ec000000000000000000000000000000000000000000000000000000000000000a1ba0204e52d00175f6a45c23a3ab600aa04af244b12b5e8ad5957a2eca8377f3e06aa020dda534949c5cf94a703b74f90297509368202a14c504fdcd27790906be1157'
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
            .and.to.equal(`invalid request, must purchase with at least ${10 ** 8} $TKN`)
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with send error', () => {
    before(() => {
      usageToken.setTA('0x684e7D2B54D2fc9fef0138ce00702445cEAd9cEA')
      usageToken.setFP({
        sendTransaction: async () => {
          throw new Error('err!')
        }
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4b55968ec000000000000000000000000000000000000000000000000000000003b9aca001ca0519ab9c09202342a0b6e125ecaeb59ffa139bf02274ad0d3233ac5a8ba8dd217a01eca0f0f2386bd543065a17d1f627aed42885cf160e93a720cb45defc0a4da8a'
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
            .and.to.equal('err!')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with bad sig, cant sign', () => {
    before(() => {
      usageToken.setTA('0x684e7D2B54D2fc9fef0138ce00702445cEAd9cEA')
      usageToken.setFP({
        sendTransaction: async () => {
          return { hash: 'deadbeef' }
        },
        waitForTransaction: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: 'badPEM'
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4b55968ec000000000000000000000000000000000000000000000000000000003b9aca001ca0519ab9c09202342a0b6e125ecaeb59ffa139bf02274ad0d3233ac5a8ba8dd217a01eca0f0f2386bd543065a17d1f627aed42885cf160e93a720cb45defc0a4da8a'
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
            .and.to.equal('server error, could not sign new token')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with broadcast error', () => {
    let coreIdCache = ''
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async (k, v) => {
          coreIdCache = [k, v]
        }
      })
      usageToken.setTA('0x684e7D2B54D2fc9fef0138ce00702445cEAd9cEA')
      usageToken.setFP({
        sendTransaction: async () => {
          return { hash: 'deadbeef' }
        },
        waitForTransaction: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://localhost'
      })
      usageToken.setStatus({
        buildStatusObjectAsync: async () => {
          return { status: { node_info: { id: 'myId!' } } }
        }
      })
      usageToken.setTMRPC({
        broadcastTxAsync: async () => {
          throw new Error('tm error!')
        }
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4b55968ec000000000000000000000000000000000000000000000000000000003b9aca001ca0519ab9c09202342a0b6e125ecaeb59ffa139bf02274ad0d3233ac5a8ba8dd217a01eca0f0f2386bd543065a17d1f627aed42885cf160e93a720cb45defc0a4da8a'
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
            .and.to.equal('server error, server error on transaction broadcast, tm error!')
          expect(coreIdCache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(coreIdCache[1]).to.equal('myId!')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with broadcast error - bad arg', () => {
    let coreIdCache = ''
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async (k, v) => {
          coreIdCache = [k, v]
        }
      })
      usageToken.setTA('0x684e7D2B54D2fc9fef0138ce00702445cEAd9cEA')
      usageToken.setFP({
        sendTransaction: async () => {
          return { hash: 'deadbeef' }
        },
        waitForTransaction: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://localhost'
      })
      usageToken.setStatus({
        buildStatusObjectAsync: async () => {
          return { status: { node_info: { id: 'myId!' } } }
        }
      })
      usageToken.setTMRPC({
        broadcastTxAsync: async () => {
          return { error: { responseCode: 409, message: 'badarg' } }
        }
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4b55968ec000000000000000000000000000000000000000000000000000000003b9aca001ca0519ab9c09202342a0b6e125ecaeb59ffa139bf02274ad0d3233ac5a8ba8dd217a01eca0f0f2386bd543065a17d1f627aed42885cf160e93a720cb45defc0a4da8a'
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
            .and.to.equal('server error, server error on transaction broadcast, badarg')
          expect(coreIdCache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(coreIdCache[1]).to.equal('myId!')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with broadcast error - TM comm error', () => {
    let coreIdCache = ''
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async (k, v) => {
          coreIdCache = [k, v]
        }
      })
      usageToken.setTA('0x684e7D2B54D2fc9fef0138ce00702445cEAd9cEA')
      usageToken.setFP({
        sendTransaction: async () => {
          return { hash: 'deadbeef' }
        },
        waitForTransaction: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://localhost'
      })
      usageToken.setStatus({
        buildStatusObjectAsync: async () => {
          return { status: { node_info: { id: 'myId!' } } }
        }
      })
      usageToken.setTMRPC({
        broadcastTxAsync: async () => {
          return { error: { responseCode: 500, message: 'commerr' } }
        }
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4b55968ec000000000000000000000000000000000000000000000000000000003b9aca001ca0519ab9c09202342a0b6e125ecaeb59ffa139bf02274ad0d3233ac5a8ba8dd217a01eca0f0f2386bd543065a17d1f627aed42885cf160e93a720cb45defc0a4da8a'
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
            .and.to.equal('server error, server error on transaction broadcast, Could not broadcast transaction')
          expect(coreIdCache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(coreIdCache[1]).to.equal('myId!')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with success', () => {
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async () => null
      })
      usageToken.setTA('0x684e7D2B54D2fc9fef0138ce00702445cEAd9cEA')
      usageToken.setFP({
        sendTransaction: async () => {
          return { hash: 'deadbeef' }
        },
        waitForTransaction: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://localhost'
      })
      usageToken.setStatus({
        buildStatusObjectAsync: async () => {
          return { status: { node_info: { id: 'myId!' } } }
        }
      })
      usageToken.setTMRPC({
        broadcastTxAsync: async () => {
          return {}
        }
      })
    })
    it('should return proper new token', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4b55968ec000000000000000000000000000000000000000000000000000000003b9aca001ca0519ab9c09202342a0b6e125ecaeb59ffa139bf02274ad0d3233ac5a8ba8dd217a01eca0f0f2386bd543065a17d1f627aed42885cf160e93a720cb45defc0a4da8a'
        })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('token')
            .and.to.be.a('string')
          done()
        })
    })
  })
})
