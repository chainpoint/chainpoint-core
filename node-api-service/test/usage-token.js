/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const usageToken = require('../lib/endpoints/usage-token.js')
const jwt = require('jsonwebtoken')

describe('Usage Token Controller - Public Mode', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(false)
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

  describe('POST /usagetoken/refresh aud missing', () => {
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiI4ZTJjY2YzMC03NWMxLTExZTktOTQ2Yy04MzhhOTRmNDM5YjIiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQwOTc5LCJiYWwiOjEwLCJpYXQiOjE1NTc3ODA5Nzh9.ds2wfyD5n6RiT1QoacANhgTK5JJOt6yR6kCFEPqiQuGJt9FlKsma8H7ANwz2lxgecmpjp43U7MFVeHP5IOh5mg'
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
          done()
        })
    })
  })

  describe('POST /usagetoken/refresh aud invalid', () => {
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiI2NTdkMjdmMC03NWNjLTExZTktYTQ0Mi1kZjZlNTc4NTZjMWYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ1NjM1LCJiYWwiOjEwLCJhdWQiOiJiYWQiLCJpYXQiOjE1NTc3ODU2MzR9.YNdh7-5ChdU0HiGCLRqenJWp0ayVSOyi-GUTJw3WJbfMxSXUvX1ff2ydEMLFGY8QkghxUzeJXM7sqKJ3QSOwAg'
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
          done()
        })
    })
  })

  describe('POST /usagetoken/refresh aud too few ips', () => {
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiJiOGJkMWI1MC03NWNjLTExZTktYTA5OS02ZDEzZjAzMjRiMWEiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ1Nzc1LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSIsImlhdCI6MTU1Nzc4NTc3NH0.kOl7IST1vSVyKXVlKZHX4roKth_uEE3-_JiLNa_jODiZnLUhQLShTUD9_iqrUDh4T2rM05A9auPWw11HSkZQQA'
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
          done()
        })
    })
  })

  describe('POST /usagetoken/refresh aud contains bad ip value', () => {
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiI4Nzc0MGUzMC03NWQzLTExZTktYWU5OC02NThiZTMzZDVhZjMiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ4Njk5LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSxiYWRpcCIsImlhdCI6MTU1Nzc4ODY5OH0.EYuUzIT948yBOZ712ONzAgybv1fd3AjTXNSnD3ilv_d3VY8YsyM77hnDuFDuk4DFGkZ4qTSawSWP9XYT4EG7dQ'
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
          done()
        })
    })
  })

  describe('POST /usagetoken/refresh sub missing', () => {
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.1' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIyOTAwNjhjMC03NWQ0LTExZTktYTM0NC1kM2ExMDMxMWY1YmMiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsImV4cCI6MTg3MzE0ODk3MCwiYmFsIjoxMCwiYXVkIjoiNjUuMTIuMTIuNDUsNjUuMTMuMTMuNTUsNjYuMS4xLjEiLCJpYXQiOjE1NTc3ODg5Njl9.AV54PSh2mbLzgQhhMNYhnkpnNsA2uafGmErmtINuGqwBehkaBMgsxgWZXpjDRHj5qRF_9OnNofB6zn_0LVjlWw'
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
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async () => null
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setGetIP(() => '66.12.12.13')
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.1' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiI4YmZmMWYyMC03NWQ0LTExZTktOTU1My01MWFjMjNiNmI5MTUiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ5MTM2LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2Ni4xLjEuMSIsImlhdCI6MTU1Nzc4OTEzNX0.R3n9LQknVXgtjRoHmzQ5f0BkK6xBnQRpcuzEniHVW5p76GpvliHW0RtdBOWHMkxJOUqn1OXGNn8GsHaAWJykmg'
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
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
      usageToken.setGetIP(() => '66.12.12.12')
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.1' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIzODczNjA0MC03NWQ1LTExZTktOWMzYy0xN2NiMzk1MDQ0MTciLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ5NDI1LCJiYWwiOjAsImF1ZCI6IjY1LjEyLjEyLjQ1LDY1LjEzLjEzLjU1LDY2LjEuMS4xIiwiaWF0IjoxNTU3Nzg5NDI0fQ.g7kWZNtAixfi9knrVJL2zH-qELhLZBnAFcwtR4XIQLR9uDurMan1nqr5AOAK41k5GEU--BJxkCVMO1CtJfvgWQ'
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
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
      usageToken.setGetIP(() => '66.12.12.12')
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.1' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIzNzBiZWI5MC03NWQ2LTExZTktOGY3Yy1kOTdmYTk3YmFiYzYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ5ODUyLCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2Ni4xLjEuMSIsImlhdCI6MTU1Nzc4OTg1MX0.drgsrF1Kvf7LHeTCcQwjSVsT1VFHpYzYr4wu2SkDXRH2ZKvOafN_aQxwokXBpMkuJdIItJgSgQj-23S1VhMrdg'
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
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
      usageToken.setGetIP(() => '66.12.12.12')
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.1' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIzNzBiZWI5MC03NWQ2LTExZTktOGY3Yy1kOTdmYTk3YmFiYzYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ5ODUyLCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2Ni4xLjEuMSIsImlhdCI6MTU1Nzc4OTg1MX0.drgsrF1Kvf7LHeTCcQwjSVsT1VFHpYzYr4wu2SkDXRH2ZKvOafN_aQxwokXBpMkuJdIItJgSgQj-23S1VhMrdg'
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
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
      usageToken.setGetIP(() => '66.12.12.12')
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.1' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIzNzBiZWI5MC03NWQ2LTExZTktOGY3Yy1kOTdmYTk3YmFiYzYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ5ODUyLCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2Ni4xLjEuMSIsImlhdCI6MTU1Nzc4OTg1MX0.drgsrF1Kvf7LHeTCcQwjSVsT1VFHpYzYr4wu2SkDXRH2ZKvOafN_aQxwokXBpMkuJdIItJgSgQj-23S1VhMrdg'
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
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
          return { tokenHash: '82fde371ab5507a54d43cb963855cef4ac9e6057b10f66e1fb972c26a5fade74' }
        }
      })
      usageToken.setENV({
        ECDSA_PKPEM: 'badPEM',
        CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.1'
      })
      usageToken.setGetIP(() => '66.12.12.12')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIzNzBiZWI5MC03NWQ2LTExZTktOGY3Yy1kOTdmYTk3YmFiYzYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ5ODUyLCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2Ni4xLjEuMSIsImlhdCI6MTU1Nzc4OTg1MX0.drgsrF1Kvf7LHeTCcQwjSVsT1VFHpYzYr4wu2SkDXRH2ZKvOafN_aQxwokXBpMkuJdIItJgSgQj-23S1VhMrdg'
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
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
          return { tokenHash: '82fde371ab5507a54d43cb963855cef4ac9e6057b10f66e1fb972c26a5fade74' }
        },
        writeActiveTokenAsync: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.1'
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
      usageToken.setGetIP(() => '66.12.12.12')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIzNzBiZWI5MC03NWQ2LTExZTktOGY3Yy1kOTdmYTk3YmFiYzYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ5ODUyLCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2Ni4xLjEuMSIsImlhdCI6MTU1Nzc4OTg1MX0.drgsrF1Kvf7LHeTCcQwjSVsT1VFHpYzYr4wu2SkDXRH2ZKvOafN_aQxwokXBpMkuJdIItJgSgQj-23S1VhMrdg'
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
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
          return { tokenHash: '82fde371ab5507a54d43cb963855cef4ac9e6057b10f66e1fb972c26a5fade74' }
        },
        writeActiveTokenAsync: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.1'
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
      usageToken.setGetIP(() => '66.12.12.12')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIzNzBiZWI5MC03NWQ2LTExZTktOGY3Yy1kOTdmYTk3YmFiYzYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ5ODUyLCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2Ni4xLjEuMSIsImlhdCI6MTU1Nzc4OTg1MX0.drgsrF1Kvf7LHeTCcQwjSVsT1VFHpYzYr4wu2SkDXRH2ZKvOafN_aQxwokXBpMkuJdIItJgSgQj-23S1VhMrdg'
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
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
          return { tokenHash: '82fde371ab5507a54d43cb963855cef4ac9e6057b10f66e1fb972c26a5fade74' }
        },
        writeActiveTokenAsync: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.1'
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
      usageToken.setGetIP(() => '66.12.12.12')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIzNzBiZWI5MC03NWQ2LTExZTktOGY3Yy1kOTdmYTk3YmFiYzYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ5ODUyLCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2Ni4xLjEuMSIsImlhdCI6MTU1Nzc4OTg1MX0.drgsrF1Kvf7LHeTCcQwjSVsT1VFHpYzYr4wu2SkDXRH2ZKvOafN_aQxwokXBpMkuJdIItJgSgQj-23S1VhMrdg'
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
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
          return { tokenHash: '82fde371ab5507a54d43cb963855cef4ac9e6057b10f66e1fb972c26a5fade74' }
        },
        writeActiveTokenAsync: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----`,
        CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.1'
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
      usageToken.setGetIP(() => '66.12.12.12')
    })
    it('should return proper refresh token', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .send({
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIzNzBiZWI5MC03NWQ2LTExZTktOGY3Yy1kOTdmYTk3YmFiYzYiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ5ODUyLCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2Ni4xLjEuMSIsImlhdCI6MTU1Nzc4OTg1MX0.drgsrF1Kvf7LHeTCcQwjSVsT1VFHpYzYr4wu2SkDXRH2ZKvOafN_aQxwokXBpMkuJdIItJgSgQj-23S1VhMrdg'
        })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          const decoded = jwt.decode(res.body.token, { complete: true })
          expect(res.body)
            .to.have.property('token')
            .and.to.be.a('string')
          expect(decoded.header)
            .to.have.property('kid')
            .and.to.equal('P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU')
          expect(decoded.payload)
            .to.have.property('iss')
            .and.to.equal('http://66.1.1.1')
          expect(decoded.payload)
            .to.have.property('sub')
            .and.to.equal('66.12.12.12')
          expect(decoded.payload)
            .to.have.property('exp')
            .and.to.be.greaterThan(Math.ceil(Date.now() / 1000))
          expect(decoded.payload)
            .to.have.property('bal')
            .and.to.equal(9)
          expect(decoded.payload)
            .to.have.property('aulr')
            .and.to.equal(3)
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with no aud', () => {
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
            .and.to.equal('invalid request, aud must be supplied')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with bad aud', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({ aud: '0xnothex' })
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
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with bad count aud', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({ aud: '64.10.120.11,64.10.120.12' })
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
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with non-ip aud', () => {
    let ip = 'notanip'
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({ aud: '64.10.120.11,64.10.120.12,' + ip })
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
            .and.to.equal(`invalid request, bad IP value in aud - ${ip}`)
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with aud missing core ip', () => {
    before(() => {
      usageToken.setENV({
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({ aud: '64.10.120.11,64.10.120.12,64.10.120.13' })
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
            .and.to.equal(`invalid request, aud must include this Core IP`)
          done()
        })
    })
  })

  describe('POST /usagetoken/credit with no tx', () => {
    before(() => {
      usageToken.setENV({
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({ aud: '64.10.120.11,64.10.120.12,65.1.1.1' })
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
    before(() => {
      usageToken.setENV({
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({ aud: '64.10.120.11,64.10.120.12,65.1.1.1', tx: 'deadbeef' })
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
    before(() => {
      usageToken.setENV({
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({ aud: '64.10.120.11,64.10.120.12,65.1.1.1', tx: '0xnothex' })
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
    before(() => {
      usageToken.setENV({
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({ aud: '64.10.120.11,64.10.120.12,65.1.1.1', tx: '0xdeadbeefcafe' })
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
      usageToken.setENV({
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          aud: '64.10.120.11,64.10.120.12,65.1.1.1',
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
      usageToken.setENV({
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          aud: '64.10.120.11,64.10.120.12,65.1.1.1',
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
      usageToken.setENV({
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          aud: '64.10.120.11,64.10.120.12,65.1.1.1',
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
      usageToken.setENV({
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          aud: '64.10.120.11,64.10.120.12,65.1.1.1',
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
        ECDSA_PKPEM: 'badPEM',
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .send({
          aud: '64.10.120.11,64.10.120.12,65.1.1.1',
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
      usageToken.setAT({
        writeActiveTokenAsync: async () => null
      })
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
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
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
          aud: '64.10.120.11,64.10.120.12,65.1.1.1',
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
      usageToken.setAT({
        writeActiveTokenAsync: async () => null
      })
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
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
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
          aud: '64.10.120.11,64.10.120.12,65.1.1.1',
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
      usageToken.setAT({
        writeActiveTokenAsync: async () => null
      })
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
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
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
          aud: '64.10.120.11,64.10.120.12,65.1.1.1',
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
      usageToken.setAT({
        writeActiveTokenAsync: async () => null
      })
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
        CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.1'
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
          aud: '64.10.120.11,64.10.120.12,65.1.1.1',
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4b55968ec000000000000000000000000000000000000000000000000000000003b9aca001ca0519ab9c09202342a0b6e125ecaeb59ffa139bf02274ad0d3233ac5a8ba8dd217a01eca0f0f2386bd543065a17d1f627aed42885cf160e93a720cb45defc0a4da8a'
        })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          const decoded = jwt.decode(res.body.token, { complete: true })
          expect(res.body)
            .to.have.property('token')
            .and.to.be.a('string')
          expect(decoded.header)
            .to.have.property('kid')
            .and.to.equal('P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU')
          expect(decoded.payload)
            .to.have.property('iss')
            .and.to.equal('http://65.1.1.1')
          expect(decoded.payload).to.have.property('sub')
          expect(decoded.payload)
            .to.have.property('exp')
            .and.to.be.greaterThan(Math.ceil(Date.now() / 1000))
          expect(decoded.payload)
            .to.have.property('bal')
            .and.to.equal(99)
          expect(decoded.payload)
            .to.have.property('aud')
            .and.to.equal('64.10.120.11,64.10.120.12,65.1.1.1')
          expect(decoded.payload)
            .to.have.property('aulr')
            .and.to.equal(3)
          done()
        })
    })
  })

  describe('POST /usagetoken/audience with missing aud', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
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
            .and.to.equal('invalid request, aud must be supplied')
          done()
        })
    })
  })

  describe('POST /usagetoken/audience with bad aud', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({ aud: '0xnothex' })
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
          done()
        })
    })
  })

  describe('POST /usagetoken/audience with bad count aud', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({ aud: '64.10.120.11,64.10.120.12' })
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
          done()
        })
    })
  })

  describe('POST /usagetoken/audience with non-ip aud', () => {
    let ip = 'notanip'
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({ aud: '64.10.120.11,64.10.120.12,' + ip })
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
            .and.to.equal(`invalid request, bad IP value in aud - ${ip}`)
          done()
        })
    })
  })

  describe('POST /usagetoken/audience with no token', () => {
    before(() => {
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.100' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({ aud: '64.10.120.11,64.10.120.12,65.1.1.100' })
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

  describe('POST /usagetoken/audience with bad token data', () => {
    before(() => {
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.100' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({ aud: '64.10.120.11,64.10.120.12,65.1.1.100', token: 'qweqweqwe' })
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

  describe('POST /usagetoken/audience with missing kid', () => {
    before(() => {
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.100' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,65.1.1.100',
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

  describe('POST /usagetoken/audience with missing iss', () => {
    before(() => {
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.100' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,65.1.1.100',
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

  describe('POST /usagetoken/audience with unknown JWK', () => {
    before(() => {
      usageToken.setRedis({
        get: async () => null
      })
      usageToken.setRP(async () => {
        return { body: null }
      })
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.100' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,65.1.1.100',
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

  describe('POST /usagetoken/audience with bad sig', () => {
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
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://65.1.1.100' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,65.1.1.100',
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

  describe('POST /usagetoken/audience cant determine Node IP', () => {
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
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
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

  describe('POST /usagetoken/audience sub missing', () => {
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIyOTAwNjhjMC03NWQ0LTExZTktYTM0NC1kM2ExMDMxMWY1YmMiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsImV4cCI6MTg3MzE0ODk3MCwiYmFsIjoxMCwiYXVkIjoiNjUuMTIuMTIuNDUsNjUuMTMuMTMuNTUsNjYuMS4xLjEiLCJpYXQiOjE1NTc3ODg5Njl9.AV54PSh2mbLzgQhhMNYhnkpnNsA2uafGmErmtINuGqwBehkaBMgsxgWZXpjDRHj5qRF_9OnNofB6zn_0LVjlWw'
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

  describe('POST /usagetoken/audience Node IP and JWT sub do not match', () => {
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
    }
    before(() => {
      usageToken.setRedis({
        get: async () => null,
        set: async () => null
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setGetIP(() => '66.12.12.13')
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiI4YmZmMWYyMC03NWQ0LTExZTktOTU1My01MWFjMjNiNmI5MTUiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMTQ5MTM2LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2Ni4xLjEuMSIsImlhdCI6MTU1Nzc4OTEzNX0.R3n9LQknVXgtjRoHmzQ5f0BkK6xBnQRpcuzEniHVW5p76GpvliHW0RtdBOWHMkxJOUqn1OXGNn8GsHaAWJykmg'
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

  describe('POST /usagetoken/audience exp missing', () => {
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
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100' })
    })
    it('should return proper error', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
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

  describe('POST /usagetoken/audience expired token', () => {
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
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100' })
    })
    it('should return proper error', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
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

  describe('POST /usagetoken/audience missing aulr', () => {
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
      usageToken.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setGetIP(() => '66.12.12.12')
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100' })
    })
    it('should return proper error', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIxN2NkYjY0MC03NjhiLTExZTktYjQ2MS0wYmQwMDJhNDQwOGMiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjI3NTM5LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiaWF0IjoxNTU3ODY3NTM4fQ.M92j3WHkF5Fx4JEPmLMslau_lniQB8Uwo6F6QXGScESYZF9XLUfKi46Uv8QLfFvnJ1JpLG6ZJI4b5PsogeMIgA'
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
            .and.to.equal('invalid request, token missing `aulr` value')
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /usagetoken/audience bad aulr', () => {
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
      usageToken.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setGetIP(() => '66.12.12.12')
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100' })
    })
    it('should return proper error', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiI5MzRhZGYwMC03NjhiLTExZTktYTUzYy1kM2IzNzdlZmI5ZDkiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjI3NzQ2LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiYXVsciI6ImJhZCIsImlhdCI6MTU1Nzg2Nzc0NX0.FRxWfcPnmmcsJr6ZPTV-PB_k7dBrLSPws0V0vpesy3ymRsmWo96Dm9fXQcsy3PexKttdFOIDl7bc3skybIdsQg'
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
            .and.to.equal('invalid request, `aulr` value must be a number')
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /usagetoken/audience bad aulr', () => {
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
      usageToken.setRedis({
        get: async () => null,
        set: async (k, v) => {
          cache = [k, v]
        }
      })
      usageToken.setRP(async () => {
        return { body: { jwk: jwk } }
      })
      usageToken.setGetIP(() => '66.12.12.12')
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100' })
    })
    it('should return proper error', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIzM2NiNmVlMC03NjhjLTExZTktODk4Mi05ZDc4ZGNjYTRlOGMiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjI4MDE1LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiYXVsciI6MCwiaWF0IjoxNTU3ODY4MDE0fQ.KZgqPOYuBUfgsaZJ7rBhZQfSmQiRnohaa2SB5kagBYi1gZs2PMdbZ_vsRhtT8cufKD5cvdy8ZgTD46d7kjiWdA'
        })
        .expect('Content-type', /json/)
        .expect(429)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('TooManyRequests')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('request rejected, aud update rate limit exceeded for this token')
          expect(cache)
            .to.be.a('array')
            .and.to.have.length(2)
          expect(cache[1]).to.equal(jwkStr)
          done()
        })
    })
  })

  describe('POST /usagetoken/audience with no active tokens', () => {
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
      usageToken.setGetIP(() => '66.12.12.12')
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIwYjQ5MjRjMC03NjhkLTExZTktYjJlOS1iNzAyYTc1MjkxM2YiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjI4Mzc3LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiYXVsciI6MiwiaWF0IjoxNTU3ODY4Mzc2fQ.A6GwryTOpaawl1NBe_EOkYabJYyI-cj_UfJCXcGb6yniqlx006rlhh1eIajGpHiagkOgUHzWuRDvq-Aofm06nA'
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
            .and.to.equal('invalid request, no active token available to be updated')
          done()
        })
    })
  })

  describe('POST /usagetoken/audience - cant read active tokens', () => {
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
      usageToken.setGetIP(() => '66.12.12.12')
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIwYjQ5MjRjMC03NjhkLTExZTktYjJlOS1iNzAyYTc1MjkxM2YiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjI4Mzc3LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiYXVsciI6MiwiaWF0IjoxNTU3ODY4Mzc2fQ.A6GwryTOpaawl1NBe_EOkYabJYyI-cj_UfJCXcGb6yniqlx006rlhh1eIajGpHiagkOgUHzWuRDvq-Aofm06nA'
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

  describe('POST /usagetoken/audience with wrong active token', () => {
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
      usageToken.setGetIP(() => '66.12.12.12')
      usageToken.setENV({ CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100' })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIwYjQ5MjRjMC03NjhkLTExZTktYjJlOS1iNzAyYTc1MjkxM2YiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjI4Mzc3LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiYXVsciI6MiwiaWF0IjoxNTU3ODY4Mzc2fQ.A6GwryTOpaawl1NBe_EOkYabJYyI-cj_UfJCXcGb6yniqlx006rlhh1eIajGpHiagkOgUHzWuRDvq-Aofm06nA'
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

  describe('POST /usagetoken/audience with bad PEM in env, cant sign', () => {
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
          return { tokenHash: 'ae582b3778cfa085c215e7ad17b7ba6337bad4eb9ad7c27596aa8cc7f14a965d' }
        }
      })
      usageToken.setENV({
        ECDSA_PKPEM: 'badPEM',
        CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100'
      })
      usageToken.setGetIP(() => '66.12.12.12')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIwYjQ5MjRjMC03NjhkLTExZTktYjJlOS1iNzAyYTc1MjkxM2YiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjI4Mzc3LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiYXVsciI6MiwiaWF0IjoxNTU3ODY4Mzc2fQ.A6GwryTOpaawl1NBe_EOkYabJYyI-cj_UfJCXcGb6yniqlx006rlhh1eIajGpHiagkOgUHzWuRDvq-Aofm06nA'
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
            .and.to.equal('server error, could not sign updated token')
          done()
        })
    })
  })

  describe('POST /usagetoken/audience with broadcast error', () => {
    let coreIdCache = ''
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
          return { tokenHash: 'ae582b3778cfa085c215e7ad17b7ba6337bad4eb9ad7c27596aa8cc7f14a965d' }
        },
        writeActiveTokenAsync: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100'
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
      usageToken.setGetIP(() => '66.12.12.12')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIwYjQ5MjRjMC03NjhkLTExZTktYjJlOS1iNzAyYTc1MjkxM2YiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjI4Mzc3LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiYXVsciI6MiwiaWF0IjoxNTU3ODY4Mzc2fQ.A6GwryTOpaawl1NBe_EOkYabJYyI-cj_UfJCXcGb6yniqlx006rlhh1eIajGpHiagkOgUHzWuRDvq-Aofm06nA'
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

  describe('POST /usagetoken/audience with broadcast error - bad token tx arg', () => {
    let coreIdCache = ''
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
          return { tokenHash: 'ae582b3778cfa085c215e7ad17b7ba6337bad4eb9ad7c27596aa8cc7f14a965d' }
        },
        writeActiveTokenAsync: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100'
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
      usageToken.setGetIP(() => '66.12.12.12')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIwYjQ5MjRjMC03NjhkLTExZTktYjJlOS1iNzAyYTc1MjkxM2YiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjI4Mzc3LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiYXVsciI6MiwiaWF0IjoxNTU3ODY4Mzc2fQ.A6GwryTOpaawl1NBe_EOkYabJYyI-cj_UfJCXcGb6yniqlx006rlhh1eIajGpHiagkOgUHzWuRDvq-Aofm06nA'
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

  describe('POST /usagetoken/audience with broadcast error - TM comm error', () => {
    let coreIdCache = ''
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
          return { tokenHash: 'ae582b3778cfa085c215e7ad17b7ba6337bad4eb9ad7c27596aa8cc7f14a965d' }
        },
        writeActiveTokenAsync: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----
`,
        CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100'
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
      usageToken.setGetIP(() => '66.12.12.12')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIwYjQ5MjRjMC03NjhkLTExZTktYjJlOS1iNzAyYTc1MjkxM2YiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjI4Mzc3LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiYXVsciI6MiwiaWF0IjoxNTU3ODY4Mzc2fQ.A6GwryTOpaawl1NBe_EOkYabJYyI-cj_UfJCXcGb6yniqlx006rlhh1eIajGpHiagkOgUHzWuRDvq-Aofm06nA'
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

  describe('POST /usagetoken/audience success', () => {
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
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
          return { tokenHash: 'ae582b3778cfa085c215e7ad17b7ba6337bad4eb9ad7c27596aa8cc7f14a965d' }
        },
        writeActiveTokenAsync: async () => null
      })
      usageToken.setENV({
        ECDSA_PKPEM: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----`,
        CHAINPOINT_CORE_BASE_URI: 'http://66.1.1.100'
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
      usageToken.setGetIP(() => '66.12.12.12')
    })
    it('should return proper update token', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .send({
          aud: '64.10.120.11,64.10.120.12,66.1.1.100',
          token:
            'eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlA2dVZJcVMwRG5wN1RENXhEWEFaLTV4QnpraHRtdEFBMTNKSWRERVh6U1UifQ.eyJqdGkiOiIwYjQ5MjRjMC03NjhkLTExZTktYjJlOS1iNzAyYTc1MjkxM2YiLCJpc3MiOiJodHRwOi8vMzUuMjQ1LjIxMS45NyIsInN1YiI6IjY2LjEyLjEyLjEyIiwiZXhwIjoxODczMjI4Mzc3LCJiYWwiOjEwLCJhdWQiOiI2NS4xMi4xMi40NSw2NS4xMy4xMy41NSw2NS4xLjEuMTAwIiwiYXVsciI6MiwiaWF0IjoxNTU3ODY4Mzc2fQ.A6GwryTOpaawl1NBe_EOkYabJYyI-cj_UfJCXcGb6yniqlx006rlhh1eIajGpHiagkOgUHzWuRDvq-Aofm06nA'
        })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
          expect(err).to.equal(null)
          const decoded = jwt.decode(res.body.token, { complete: true })
          expect(res.body)
            .to.have.property('token')
            .and.to.be.a('string')
          expect(decoded.header)
            .to.have.property('kid')
            .and.to.equal('P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU')
          expect(decoded.payload)
            .to.have.property('iss')
            .and.to.equal('http://66.1.1.100')
          expect(decoded.payload)
            .to.have.property('sub')
            .and.to.equal('66.12.12.12')
          expect(decoded.payload)
            .to.have.property('exp')
            .and.to.be.greaterThan(Math.ceil(Date.now() / 1000))
          expect(decoded.payload)
            .to.have.property('bal')
            .and.to.equal(10)
          expect(decoded.payload)
            .to.have.property('aud')
            .and.to.equal('64.10.120.11,64.10.120.12,66.1.1.100')
          expect(decoded.payload)
            .to.have.property('aulr')
            .and.to.equal(1)
          done()
        })
    })
  })
})

describe('Usage Token Controller - Private Mode', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(true)
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('POST /usagetoken/refresh', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/refresh')
        .expect('Content-type', /json/)
        .expect(404)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('ResourceNotFound')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('/usagetoken/refresh does not exist')
          done()
        })
    })
  })

  describe('POST /usagetoken/credit', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/credit')
        .expect('Content-type', /json/)
        .expect(404)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('ResourceNotFound')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('/usagetoken/credit does not exist')
          done()
        })
    })
  })

  describe('POST /usagetoken/audience', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/usagetoken/audience')
        .expect('Content-type', /json/)
        .expect(404)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('ResourceNotFound')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('/usagetoken/audience does not exist')
          done()
        })
    })
  })

  describe('POST /eth/broadcast', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .expect('Content-type', /json/)
        .expect(404)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('ResourceNotFound')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('/eth/broadcast does not exist')
          done()
        })
    })
  })

  describe('POST /eth/:addr/stats', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/addr/stats')
        .expect('Content-type', /json/)
        .expect(404)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('ResourceNotFound')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('/eth/addr/stats does not exist')
          done()
        })
    })
  })
})
