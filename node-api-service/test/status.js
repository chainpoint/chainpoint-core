/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const status = require('../lib/endpoints/status.js')
const { version } = require('../package.json')

describe('Status Controller - Public Mode', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(false)
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('GET /status with bad TM connection', () => {
    before(() => {
      status.setTmRpc({
        getStatusAsync: async () => {
          return { error: true }
        }
      })
    })
    it('should return proper error with TM communication error', done => {
      request(insecureServer)
        .get('/status')
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
            .and.to.equal('Could not query for status')
          done()
        })
    })
  })

  describe('GET /status', () => {
    let baseURI = 'http://base.uri'
    let ecdsa = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgnwHQZK/KRmLIlm3l
zfB8ygE3fGv5tKTCVQUg8I/gB8OhRANCAATQppszaJzIR+Ri1L7LgyVqEkUFsfAO
HPZuKph2KdSNn2jrHKWSZCviI9J6REY6H1kM47aFiyrrls9DnXSN1OoB
-----END PRIVATE KEY-----`
    let jwk = {
      kty: 'EC',
      kid: 'P6uVIqS0Dnp7TD5xDXAZ-5xBzkhtmtAA13JIdDEXzSU',
      crv: 'P-256',
      x: '0KabM2icyEfkYtS-y4MlahJFBbHwDhz2biqYdinUjZ8',
      y: 'aOscpZJkK-Ij0npERjofWQzjtoWLKuuWz0OddI3U6gE'
    }
    before(() => {
      status.setENV({ CHAINPOINT_CORE_BASE_URI: baseURI, ECDSA_PKPEM: ecdsa, NODE_ENV: 'test', PRIVATE_NETWORK: false })
      let statusResult = { tmresult: 1 }
      status.setTmRpc({
        getStatusAsync: async () => {
          return { result: statusResult }
        }
      })
    })
    it('should return proper status object', done => {
      request(insecureServer)
        .get('/status')
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('version')
            .and.to.be.a('string')
            .and.to.equal(version)
          expect(res.body)
            .to.have.property('time')
            .and.to.be.a('string')
          expect(res.body)
            .to.have.property('base_uri')
            .and.to.be.a('string')
            .and.to.equal(baseURI)
          expect(res.body)
            .to.have.property('environment')
            .and.to.be.a('string')
            .and.to.equal('test')
          expect(res.body)
            .to.have.property('mode')
            .and.to.be.a('string')
            .and.to.equal('public')
          expect(res.body)
            .to.have.property('jwk')
            .and.to.be.a('object')
            .and.to.deep.equal(jwk)
          expect(res.body)
            .to.have.property('tmresult')
            .and.to.be.a('number')
            .and.to.equal(1)
          done()
        })
    })
  })
})

describe('Status Controller - Private Mode', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(true)
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('GET /status', () => {
    let baseURI = 'http://base.uri'
    before(() => {
      status.setENV({ CHAINPOINT_CORE_BASE_URI: baseURI, NODE_ENV: 'test', PRIVATE_NETWORK: true })
      let statusResult = { tmresult: 1 }
      status.setTmRpc({
        getStatusAsync: async () => {
          return { result: statusResult }
        }
      })
    })
    it('should return proper status object', done => {
      request(insecureServer)
        .get('/status')
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('version')
            .and.to.be.a('string')
            .and.to.equal(version)
          expect(res.body)
            .to.have.property('time')
            .and.to.be.a('string')
          expect(res.body)
            .to.have.property('base_uri')
            .and.to.be.a('string')
            .and.to.equal(baseURI)
          expect(res.body)
            .to.have.property('environment')
            .and.to.be.a('string')
            .and.to.equal('test')
          expect(res.body)
            .to.have.property('mode')
            .and.to.be.a('string')
            .and.to.equal('private')
          expect(res.body).to.not.have.property('jwk')
          expect(res.body)
            .to.have.property('tmresult')
            .and.to.be.a('number')
            .and.to.equal(1)
          done()
        })
    })
  })
})
